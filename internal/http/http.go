package http

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"time"

	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/types"
	"github.com/go-resty/resty/v2"
)

// classifyExecuteError turns a resty request failure into the right typed
// error. When resp has a real status code, a response was received and the
// failure is resty failing to decode its body into the caller's result (or
// error) type — deterministic, so it must be a DecodeError, never a
// NetworkError: retrying would only re-fetch the same unparseable body. This
// mirrors the check the client's own retry condition already uses (see
// AddRetryCondition below) to decide the same question.
func classifyExecuteError(operation string, resp *resty.Response, err error) error {
	if resp != nil && resp.StatusCode() != 0 {
		return errors.NewDecodeError(operation, err)
	}
	return errors.NewNetworkError(operation, err)
}

// isIdempotent reports whether an HTTP method is safe to retry. POST and PATCH
// are not: retrying them can duplicate a non-idempotent side effect.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// HTTPClient wraps the resty client with BSN.cloud-specific functionality.
type HTTPClient struct {
	client *resty.Client
	config *config.Config
}

// NewHTTPClient creates a new HTTP client with the given configuration.
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	client := resty.New().
		SetTimeout(cfg.Timeout).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
			// Exponential backoff for retries
			return time.Duration(resp.Request.Attempt) * time.Second, nil
		}).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			// Only idempotent methods may be retried. Retrying a POST or PATCH
			// can duplicate a non-idempotent side effect — a timed-out reboot or
			// create could run twice — so those are never retried, not even on a
			// transport error, because the request may have reached the server.
			if resp == nil || resp.Request == nil || !isIdempotent(resp.Request.Method) {
				return false
			}

			// A non-nil err with no HTTP status is a transport-level failure
			// (connection refused, timeout, TLS) — worth retrying.
			//
			// A non-nil err WITH a status means the response arrived and then
			// failed to DECODE into the caller's result type. That is
			// deterministic: retrying re-fetches an identically-unparseable body,
			// so RetryCount rounds of backoff buy nothing and turn a sub-second
			// call into several seconds of dead waiting. This was MEASURED on
			// GET /v1/system, where firmware that reports a version component as a
			// JSON string (e.g. "9") against a numeric field failed the decode and
			// drove a ~0.3s call to ~7s of retries. Do not retry a decode failure.
			if err != nil {
				return resp.StatusCode() == 0
			}
			// Retry on server errors and rate limiting.
			return resp.StatusCode() >= 500 || resp.StatusCode() == http.StatusTooManyRequests
		}).
		SetHeaders(map[string]string{
			"Accept":     "application/json",
			"User-Agent": "bscontrol-sdk/1.0",
		}).
		SetAuthScheme("Bearer"). // Required for SetAuthToken() to work correctly
		SetDebug(cfg.Debug)      // Enable debug logging if configured

	// Transport tracing is wired only when a tracer is registered, so the common
	// path pays nothing. EnableTrace turns on httptrace per request; the two hooks
	// are mutually exclusive per request — OnAfterResponse fires when a response is
	// received, OnError after all retries when none is.
	if cfg.RequestTracer != nil {
		tracer := cfg.RequestTracer
		client.EnableTrace()
		client.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			tracer(newRequestTrace(resp.Request, resp, nil))
			return nil
		})
		client.OnError(func(req *resty.Request, err error) {
			// A ResponseError carries the response that was received before the
			// failure (for example a decode failure on a 200), so its status and
			// size are still reported alongside the error.
			var respErr *resty.ResponseError
			var resp *resty.Response
			if stderrors.As(err, &respErr) {
				resp = respErr.Response
			}
			tracer(newRequestTrace(req, resp, err))
		})
	}

	return &HTTPClient{
		client: client,
		config: cfg,
	}
}

// newRequestTrace translates resty's per-request TraceInfo into the SDK's public
// RequestTrace. resp is nil when the request failed before a response arrived.
func newRequestTrace(req *resty.Request, resp *resty.Response, err error) types.RequestTrace {
	info := req.TraceInfo()
	trace := types.RequestTrace{
		Method:       req.Method,
		URL:          req.URL,
		Attempt:      info.RequestAttempt,
		DNSLookup:    info.DNSLookup,
		TLSHandshake: info.TLSHandshake,
		ServerTime:   info.ServerTime,
		TotalTime:    info.TotalTime,
		ConnReused:   info.IsConnReused,
		Err:          err,
	}
	if resp != nil {
		trace.StatusCode = resp.StatusCode()
		trace.BodyBytes = resp.Size()
	}
	return trace
}

// Request represents an HTTP request to be made.
type Request struct {
	Method      string
	URL         string
	Body        interface{}
	Result      interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// Do executes an HTTP request with error handling and response parsing.
func (h *HTTPClient) Do(ctx context.Context, req *Request) error {
	request := h.client.R().SetContext(ctx)

	// Set headers
	if req.Headers != nil {
		request.SetHeaders(req.Headers)
	}

	// Set query parameters
	if req.QueryParams != nil {
		request.SetQueryParams(req.QueryParams)
	}

	// Set body
	if req.Body != nil {
		request.SetBody(req.Body)
		request.SetHeader("Content-Type", "application/json")
	}

	// Set result structure
	if req.Result != nil {
		request.SetResult(req.Result)
	}

	// Set error structure to capture API errors
	var apiError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Details          string `json:"details"`
	}
	request.SetError(&apiError)

	// Execute request
	resp, err := request.Execute(req.Method, req.URL)
	if err != nil {
		return classifyExecuteError(fmt.Sprintf("%s %s", req.Method, req.URL), resp, err)
	}

	// Check for API errors
	if !resp.IsSuccess() {
		return h.handleAPIError(resp, &apiError)
	}

	return nil
}

// DoWithAuth executes an HTTP request with authentication headers.
func (h *HTTPClient) DoWithAuth(ctx context.Context, token string, req *Request) error {
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers["Authorization"] = "Bearer " + token
	return h.Do(ctx, req)
}

// handleAPIError converts HTTP error responses to appropriate error types.
func (h *HTTPClient) handleAPIError(resp *resty.Response, apiError *struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Details          string `json:"details"`
}) error {
	statusCode := resp.StatusCode()

	// Try to get error details from response
	errorCode := apiError.Error
	errorMessage := apiError.ErrorDescription
	errorDetails := apiError.Details

	// If we couldn't parse the error, include the raw response body
	if errorCode == "" && errorMessage == "" {
		bodyStr := string(resp.Body())
		if bodyStr != "" && len(bodyStr) < 500 {
			errorDetails = bodyStr
		}
	}

	// Fallback to status text if no error details
	if errorCode == "" {
		errorCode = http.StatusText(statusCode)
	}
	if errorMessage == "" {
		errorMessage = "Request failed"
	}

	// Create specific error types based on status code
	switch statusCode {
	case http.StatusUnauthorized:
		return errors.NewAuthError("invalid or expired token", errors.NewAPIError(statusCode, errorCode, errorMessage, errorDetails))
	case http.StatusForbidden:
		return errors.NewAuthError("insufficient permissions", errors.NewAPIError(statusCode, errorCode, errorMessage, errorDetails))
	default:
		return errors.NewAPIError(statusCode, errorCode, errorMessage, errorDetails)
	}
}

// Get performs a GET request.
func (h *HTTPClient) Get(ctx context.Context, url string, result interface{}) error {
	return h.Do(ctx, &Request{
		Method: "GET",
		URL:    url,
		Result: result,
	})
}

// GetWithAuth performs a GET request with authentication.
func (h *HTTPClient) GetWithAuth(ctx context.Context, token, url string, result interface{}) error {
	return h.DoWithAuth(ctx, token, &Request{
		Method: "GET",
		URL:    url,
		Result: result,
	})
}

// GetBytesWithAuth performs a GET request and returns raw bytes (for downloading files).
func (h *HTTPClient) GetBytesWithAuth(ctx context.Context, token, url string) ([]byte, error) {
	request := h.client.R().
		SetContext(ctx).
		SetAuthToken(token)

	// Set error structure
	var apiError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Details          string `json:"details"`
	}
	request.SetError(&apiError)

	// Execute request. This call sets no Result (it returns raw bytes), so
	// unlike Do/PostForm resty cannot fail Get() with a decode error here — an
	// unparseable error body on a non-2xx response just leaves apiError
	// zero-valued and falls through to handleAPIError below. err != nil is
	// therefore always a genuine transport failure, but classifyExecuteError
	// is still used for consistency should that ever change.
	resp, err := request.Get(url)
	if err != nil {
		return nil, classifyExecuteError(fmt.Sprintf("GET %s", url), resp, err)
	}

	// Check for HTTP errors
	if resp.StatusCode() >= 400 {
		return nil, h.handleAPIError(resp, &apiError)
	}

	return resp.Body(), nil
}

// Post performs a POST request.
func (h *HTTPClient) Post(ctx context.Context, url string, body, result interface{}) error {
	return h.Do(ctx, &Request{
		Method: "POST",
		URL:    url,
		Body:   body,
		Result: result,
	})
}

// PostWithAuth performs a POST request with authentication.
func (h *HTTPClient) PostWithAuth(ctx context.Context, token, url string, body, result interface{}) error {
	return h.DoWithAuth(ctx, token, &Request{
		Method: "POST",
		URL:    url,
		Body:   body,
		Result: result,
	})
}

// Put performs a PUT request.
func (h *HTTPClient) Put(ctx context.Context, url string, body, result interface{}) error {
	return h.Do(ctx, &Request{
		Method: "PUT",
		URL:    url,
		Body:   body,
		Result: result,
	})
}

// PutWithAuth performs a PUT request with authentication.
func (h *HTTPClient) PutWithAuth(ctx context.Context, token, url string, body, result interface{}) error {
	return h.DoWithAuth(ctx, token, &Request{
		Method: "PUT",
		URL:    url,
		Body:   body,
		Result: result,
	})
}

// DeleteWithAuth performs a DELETE request with authentication.
func (h *HTTPClient) DeleteWithAuth(ctx context.Context, token, url string, result interface{}) error {
	return h.DoWithAuth(ctx, token, &Request{
		Method: "DELETE",
		URL:    url,
		Result: result,
	})
}

// PostForm performs a POST request with form data (for OAuth token requests).
func (h *HTTPClient) PostForm(ctx context.Context, url string, data map[string]string, result interface{}) error {
	request := h.client.R().SetContext(ctx)

	// Set form data
	request.SetFormData(data)
	request.SetHeader("Content-Type", "application/x-www-form-urlencoded")

	// Set result
	if result != nil {
		request.SetResult(result)
	}

	// Set error structure
	var apiError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	request.SetError(&apiError)

	resp, err := request.Post(url)
	if err != nil {
		return classifyExecuteError(fmt.Sprintf("POST %s", url), resp, err)
	}

	if !resp.IsSuccess() {
		return h.handleAPIError(resp, &struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			Details          string `json:"details"`
		}{
			Error:            apiError.Error,
			ErrorDescription: apiError.ErrorDescription,
		})
	}

	return nil
}

// PostFormWithAuth performs a POST request with form data and basic auth.
func (h *HTTPClient) PostFormWithAuth(ctx context.Context, clientID, clientSecret, url string, data map[string]string, result interface{}) error {
	request := h.client.R().SetContext(ctx)

	// Set basic auth
	request.SetBasicAuth(clientID, clientSecret)

	// Set form data
	request.SetFormData(data)
	request.SetHeader("Content-Type", "application/x-www-form-urlencoded")

	// Set result
	if result != nil {
		request.SetResult(result)
	}

	// Set error structure
	var apiError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	request.SetError(&apiError)

	resp, err := request.Post(url)
	if err != nil {
		return classifyExecuteError(fmt.Sprintf("POST %s", url), resp, err)
	}

	if !resp.IsSuccess() {
		return h.handleAPIError(resp, &struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			Details          string `json:"details"`
		}{
			Error:            apiError.Error,
			ErrorDescription: apiError.ErrorDescription,
		})
	}

	return nil
}

// GetClient returns the underlying resty client for advanced usage.
func (h *HTTPClient) GetClient() *resty.Client {
	return h.client
}
