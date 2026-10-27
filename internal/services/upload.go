package services

import (
	"bytes"
	"context"
	"crypto/sha1" // #nosec G505 -- SHA-1 is the Upload API's own integrity hash, not a security control
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	bshttp "github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// The Upload API (2019/03) is a separate service from the Content REST API: a
// sessioned, chunked, hash-verified protocol with vendor media types. The flow,
// and why each field is what it is, is written up in docs/content-api.md.
const (
	uploadAPIPath = "/Upload/2019/03/REST"

	// uploadNullDate is sent as fileLastModifiedDate on start. It is deliberately
	// not a real date: it keeps BSN's duplicate match on name+path+size+hash only.
	uploadNullDate = "0001-01-01 00:00:00Z"

	// uploadDateLayout is the "T" ISO form carried on complete.
	uploadDateLayout = "2006-01-02T15:04:05.000Z"

	// Vendor media types, used exactly as the reference client sends them.
	uploadStartContentType    = "application/vnd.bsn.start.content.upload.arguments.201903+json"
	uploadCompleteContentType = "application/vnd.bsn.complete.content.upload.arguments.201903+json"
	uploadStatusAccept        = "application/vnd.bsn.content.upload.status.201903+json"
	uploadNegotiationAccept   = "application/vnd.bsn.content.upload.negotiation.status.201903+json"
	uploadErrorAccept         = "application/vnd.bsn.error+json"
	uploadStartAccept         = uploadStatusAccept + ", " + uploadNegotiationAccept + ", " + uploadErrorAccept

	defaultChunkSize    = 5 * 1024 * 1024
	defaultPollInterval = 500 * time.Millisecond
	defaultPollTimeout  = 2 * time.Minute
	defaultRetryDelay   = time.Second
	chunkAttempts       = 3
)

// UploadService pushes files into the content library through the Upload API.
type UploadService interface {
	// Upload sends the file at filePath, using its base name as the file name.
	// virtualPath is the library folder (either slash style; "" is the root).
	Upload(ctx context.Context, filePath, virtualPath string, opts ...UploadOption) (*types.UploadResponse, error)

	// UploadReader sends fileSize bytes from r. The size must be exact: BSN
	// pre-allocates the file from it. A reader that is also an io.Seeker is read
	// twice (hash, then send) without buffering; any other reader is buffered in
	// memory.
	UploadReader(ctx context.Context, fileName string, fileSize int64, r io.Reader, virtualPath string, opts ...UploadOption) (*types.UploadResponse, error)

	// UploadBytes sends an in-memory file to the library root and returns its
	// content ID.
	UploadBytes(ctx context.Context, fileName string, data []byte, opts ...UploadOption) (int, error)
}

// UploadOption tunes a single upload.
type UploadOption func(*uploadOptions)

type uploadOptions struct {
	mediaType string
	progress  func(types.UploadProgress)
	modTime   time.Time
}

// WithUploadMediaType overrides the media type that is otherwise detected from
// the file extension. Use one of the types.MediaType* values.
func WithUploadMediaType(mediaType string) UploadOption {
	return func(o *uploadOptions) { o.mediaType = mediaType }
}

// WithUploadProgress registers a callback invoked after every chunk is sent. It
// runs on the uploading goroutine and should not block.
func WithUploadProgress(fn func(types.UploadProgress)) UploadOption {
	return func(o *uploadOptions) { o.progress = fn }
}

// WithUploadModTime sets the file modification time recorded on completion. The
// default is the file's own mtime for Upload and the current time otherwise.
func WithUploadModTime(t time.Time) UploadOption {
	return func(o *uploadOptions) { o.modTime = t }
}

// uploadService implements the UploadService interface.
type uploadService struct {
	config      *config.Config
	httpClient  *bshttp.HTTPClient
	authManager *auth.AuthManager

	// Tunables. Production uses the defaults; tests shrink them.
	chunkSize    int
	pollInterval time.Duration
	pollTimeout  time.Duration
	retryDelay   time.Duration
	now          func() time.Time
}

// NewUploadService creates a new upload service.
func NewUploadService(cfg *config.Config, httpClient *bshttp.HTTPClient, authManager *auth.AuthManager) UploadService {
	return &uploadService{
		config:       cfg,
		httpClient:   httpClient,
		authManager:  authManager,
		chunkSize:    defaultChunkSize,
		pollInterval: defaultPollInterval,
		pollTimeout:  defaultPollTimeout,
		retryDelay:   defaultRetryDelay,
		now:          time.Now,
	}
}

func (s *uploadService) baseURL() string {
	return s.config.BSNBaseURL + uploadAPIPath
}

// sessionURL is the (single-file) upload session root; the literal "None" stands
// in for a session token, which only multi-file uploads use.
func (s *uploadService) sessionURL() string { return s.baseURL() + "/Sessions/None/Uploads/" }

func (s *uploadService) uploadURL(uploadToken string) string {
	return s.sessionURL() + escSegment(uploadToken) + "/"
}

// Upload sends a file from disk.
func (s *uploadService) Upload(ctx context.Context, filePath, virtualPath string, opts ...UploadOption) (*types.UploadResponse, error) {
	if filePath == "" {
		return nil, errors.NewValidationError("filePath", filePath, "file path cannot be empty")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return nil, errors.NewValidationError("filePath", filePath, "path is a directory")
	}

	// The file's own mtime is the default; a caller-supplied option comes later
	// in the list and wins.
	withDefault := append([]UploadOption{WithUploadModTime(info.ModTime())}, opts...)
	return s.UploadReader(ctx, filepath.Base(filePath), info.Size(), f, virtualPath, withDefault...)
}

// UploadBytes sends an in-memory file to the library root.
func (s *uploadService) UploadBytes(ctx context.Context, fileName string, data []byte, opts ...UploadOption) (int, error) {
	resp, err := s.UploadReader(ctx, fileName, int64(len(data)), bytes.NewReader(data), "", opts...)
	if err != nil {
		return 0, err
	}
	return resp.ContentID, nil
}

// UploadReader runs the whole upload: hash, start, (dedup | chunks, poll,
// complete).
func (s *uploadService) UploadReader(ctx context.Context, fileName string, fileSize int64, r io.Reader, virtualPath string, opts ...UploadOption) (*types.UploadResponse, error) {
	var o uploadOptions
	for _, opt := range opts {
		opt(&o)
	}

	// Everything we can check without the network comes first.
	if fileName == "" {
		return nil, errors.NewValidationError("fileName", fileName, "file name cannot be empty")
	}
	if strings.ContainsAny(fileName, `/\`) {
		return nil, errors.NewValidationError("fileName", fileName, "file name must not contain a path; use the virtual path for folders")
	}
	if fileSize <= 0 {
		// NOTE: BSN's behaviour for a zero-byte upload is not documented (there
		// would be no chunk to poll for), so empty files are refused up front.
		return nil, errors.NewValidationError("fileSize", fileSize, "file size must be positive")
	}
	if r == nil {
		return nil, errors.NewValidationError("reader", r, "reader cannot be nil")
	}
	vp, err := ToBSNVirtualPath(virtualPath)
	if err != nil {
		return nil, err
	}
	mediaType := o.mediaType
	if mediaType == "" {
		mediaType = DetectMediaType(fileName)
	} else if !validMediaTypes[mediaType] {
		return nil, errors.NewValidationError("mediaType", mediaType, "not a recognised upload media type")
	}
	if o.modTime.IsZero() {
		o.modTime = s.now()
	}

	// Fail on auth/network before spending time hashing a large file.
	if _, err := ready(ctx, s.authManager); err != nil {
		return nil, err
	}

	src, sha1Hex, err := newUploadSource(ctx, r, fileSize)
	if err != nil {
		return nil, err
	}

	args := types.ContentUploadArguments{
		FileName:             fileName,
		VirtualPath:          vp,
		MediaType:            mediaType,
		FileSize:             fileSize,
		FileLastModifiedDate: uploadNullDate,
		SHA1Hash:             sha1Hex,
	}

	// Step 1: start. BSN answers with the upload token and any matching files.
	started, err := s.start(ctx, args)
	if err != nil {
		return nil, err
	}

	// Step 2: a true duplicate needs no bytes. Cancel the session and reuse it.
	if dup := findDuplicate(started.MatchingFiles, fileName, vp); dup != nil {
		s.cancel(ctx, started.UploadToken)
		return &types.UploadResponse{
			ContentID:      dup.ID,
			FileName:       dup.FileName,
			FileSize:       fileSize,
			VirtualPath:    vp,
			MediaType:      mediaType,
			FileHash:       sha1Hex,
			UploadDate:     dup.UploadDate,
			UploadComplete: true,
			Deduplicated:   true,
		}, nil
	}

	// Step 3: send the chunks.
	if err := s.sendChunks(ctx, started.UploadToken, src, fileName, fileSize, o.progress); err != nil {
		return nil, err
	}

	// Step 4: the server lags behind the last chunk POST. Completing before the
	// chunks read Uploaded yields an incomplete-assembly error, so wait.
	if _, err := s.await(ctx, started.UploadToken, fileName, chunksUploaded); err != nil {
		return nil, err
	}

	// Step 5: complete, with the real modification date and the token filled in.
	final, err := s.complete(ctx, started.UploadToken, args, o.modTime)
	if err != nil {
		return nil, err
	}

	return &types.UploadResponse{
		ContentID:      final.ContentID,
		FileName:       fileName,
		FileSize:       fileSize,
		VirtualPath:    vp,
		MediaType:      mediaType,
		FileHash:       sha1Hex,
		UploadDate:     final.EndTime,
		UploadComplete: final.State == types.UploadStateVerified || final.State == types.UploadStateCompleted,
	}, nil
}

// start opens the upload session.
func (s *uploadService) start(ctx context.Context, args types.ContentUploadArguments) (*types.ContentUploadStatus, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}
	var status types.ContentUploadStatus
	err = s.httpClient.DoWithAuth(ctx, token, &bshttp.Request{
		Method: http.MethodPost,
		URL:    s.sessionURL(),
		Body:   args,
		Result: &status,
		Headers: map[string]string{
			"Content-Type": uploadStartContentType,
			"Accept":       uploadStartAccept,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start upload of %q: %w", args.FileName, err)
	}
	if status.UploadToken == "" {
		return nil, errors.NewAppError("upload_no_token", "BSN did not return an upload token",
			fmt.Sprintf("starting upload of %q", args.FileName))
	}
	return &status, nil
}

// cancel abandons a session. It is best effort: the only caller is the dedup
// path, where the upload has already been satisfied, and BSN reaps abandoned
// sessions on its own, so a failed cancel is not worth failing the upload for.
func (s *uploadService) cancel(ctx context.Context, uploadToken string) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return
	}
	_ = s.httpClient.DeleteWithAuth(ctx, token, s.uploadURL(uploadToken), nil)
}

// findDuplicate returns the existing item that is the same file in the same
// folder, or nil. BSN already matched on name, size and hash; it is only a true
// duplicate when it also sits at our virtual path.
//
// NOTE: the spec compares virtualPath only. We also require the same file name
// so a hash match under a different name is uploaded rather than silently reused
// under the wrong name. That can only cause an extra upload, never a wrong reuse.
func findDuplicate(matches []types.UploadMatchingFile, fileName, virtualPath string) *types.UploadMatchingFile {
	for i := range matches {
		m := &matches[i]
		if m.ID <= 0 || m.FileName != fileName {
			continue
		}
		mp, err := ToBSNVirtualPath(m.VirtualPath)
		if err != nil || mp != virtualPath {
			continue
		}
		return m
	}
	return nil
}

// sendChunks posts the file in order, each chunk at its byte offset.
func (s *uploadService) sendChunks(ctx context.Context, uploadToken string, src uploadSource, fileName string, size int64, progress func(types.UploadProgress)) error {
	chunkSize := int64(s.chunkSize)
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	for offset := int64(0); offset < size; {
		n := chunkSize
		if size-offset < n {
			n = size - offset
		}
		data, err := src.chunk(offset, int(n))
		if err != nil {
			return fmt.Errorf("failed to read %q at offset %d: %w", fileName, offset, err)
		}
		if err := s.sendChunk(ctx, uploadToken, offset, data); err != nil {
			return fmt.Errorf("failed to upload chunk of %q at offset %d: %w", fileName, offset, err)
		}
		offset += n
		if progress != nil {
			progress(types.UploadProgress{FileName: fileName, Sent: offset, Total: size})
		}
	}
	return nil
}

// sendChunk posts one chunk. Chunks are written at an absolute offset, so
// repeating one is harmless; that lets us retry transient failures here even
// though the HTTP client never retries a POST on its own.
func (s *uploadService) sendChunk(ctx context.Context, uploadToken string, offset int64, data []byte) error {
	chunkURL := fmt.Sprintf("%s%s/Chunks/?offset=%d", s.sessionURL(), escSegment(uploadToken), offset)

	var lastErr error
	for attempt := 1; attempt <= chunkAttempts; attempt++ {
		token, err := ready(ctx, s.authManager)
		if err != nil {
			return err
		}
		lastErr = s.httpClient.DoWithAuth(ctx, token, &bshttp.Request{
			Method:  http.MethodPost,
			URL:     chunkURL,
			Body:    data,
			Headers: map[string]string{"Content-Type": "application/octet-stream"},
		})
		if lastErr == nil {
			return nil
		}
		if !errors.IsRetryableError(lastErr) || attempt == chunkAttempts {
			return lastErr
		}
		if err := sleepCtx(ctx, s.retryDelay); err != nil {
			return err
		}
	}
	return lastErr
}

// status reads the upload's current status.
func (s *uploadService) status(ctx context.Context, uploadToken string) (*types.ContentUploadStatus, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}
	var st types.ContentUploadStatus
	err = s.httpClient.DoWithAuth(ctx, token, &bshttp.Request{
		Method:  http.MethodGet,
		URL:     s.uploadURL(uploadToken),
		Result:  &st,
		Headers: map[string]string{"Accept": uploadStatusAccept + ", " + uploadErrorAccept},
	})
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// failedStates end an upload for good.
//
// NOTE: the spec names Corrupted, Cancelled and Terminated as terminal
// failures. Collected (the server reaped the session) can never recover either.
// Stopped, Queued and Unknown are not treated as final: they keep polling until
// the timeout, which reports the last state seen.
func failedState(st types.UploadState) bool {
	switch st {
	case types.UploadStateCorrupted, types.UploadStateCancelled, types.UploadStateTerminated, types.UploadStateCollected:
		return true
	}
	return false
}

// chunksUploaded reports that the server has every chunk.
func chunksUploaded(st *types.ContentUploadStatus) bool {
	switch st.State {
	case types.UploadStateUploaded, types.UploadStateVerified, types.UploadStateCompleted:
		return true
	}
	if len(st.Chunks) == 0 {
		return false
	}
	for _, c := range st.Chunks {
		if c.State != types.UploadStateUploaded {
			return false
		}
	}
	return true
}

// finished reports the upload's final, successful state.
func finished(st *types.ContentUploadStatus) bool {
	return st.State == types.UploadStateVerified || st.State == types.UploadStateCompleted
}

// await polls until done reports true, a terminal failure appears, the poll
// budget runs out or ctx ends.
func (s *uploadService) await(ctx context.Context, uploadToken, fileName string, done func(*types.ContentUploadStatus) bool) (*types.ContentUploadStatus, error) {
	interval := s.pollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	attempts := int(s.pollTimeout / interval)
	if attempts < 1 {
		attempts = 1
	}

	var last types.UploadState
	for i := 0; i < attempts; i++ {
		st, err := s.status(ctx, uploadToken)
		if err != nil {
			return nil, fmt.Errorf("failed to check upload status of %q: %w", fileName, err)
		}
		last = st.State
		if failedState(st.State) {
			return nil, errors.NewAppError("upload_failed",
				fmt.Sprintf("upload of %q ended in state %s", fileName, st.State), "")
		}
		if done(st) {
			return st, nil
		}
		if i == attempts-1 {
			break
		}
		if err := sleepCtx(ctx, interval); err != nil {
			return nil, err
		}
	}
	return nil, errors.NewAppError("upload_timeout",
		fmt.Sprintf("upload of %q did not finish processing in time", fileName),
		fmt.Sprintf("last state: %s", last))
}

// complete finalises the upload and returns its final status.
func (s *uploadService) complete(ctx context.Context, uploadToken string, startArgs types.ContentUploadArguments, modTime time.Time) (*types.ContentUploadStatus, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	args := startArgs
	args.UploadToken = &uploadToken
	args.FileLastModifiedDate = modTime.UTC().Format(uploadDateLayout)

	var st types.ContentUploadStatus
	err = s.httpClient.DoWithAuth(ctx, token, &bshttp.Request{
		Method: http.MethodPut,
		URL:    s.uploadURL(uploadToken),
		Body:   args,
		Result: &st,
		Headers: map[string]string{
			"Content-Type": uploadCompleteContentType,
			"Accept":       uploadStartAccept,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to complete upload of %q: %w", args.FileName, err)
	}

	switch {
	case st.State == types.UploadStateCorrupted:
		// The assembled file's SHA-1 did not match the one we sent.
		return nil, errors.NewAppError("upload_corrupted",
			fmt.Sprintf("BSN rejected %q: the uploaded file failed hash verification", args.FileName), "")
	case failedState(st.State):
		return nil, errors.NewAppError("upload_failed",
			fmt.Sprintf("upload of %q ended in state %s", args.FileName, st.State), "")
	case !finished(&st):
		// Still verifying. Wait for the final state rather than guessing.
		final, err := s.await(ctx, uploadToken, args.FileName, finished)
		if err != nil {
			return nil, err
		}
		st = *final
	}
	if st.ContentID <= 0 {
		return nil, noContentID(args.FileName)
	}
	return &st, nil
}

func noContentID(fileName string) error {
	return errors.NewAppError("upload_no_content_id", "upload finished but BSN returned no content ID",
		fmt.Sprintf("file %q", fileName))
}

// sleepCtx waits for d or until ctx ends, whichever is first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Sources: where chunk bytes come from
// ---------------------------------------------------------------------------

// uploadSource yields the file's bytes by offset, so a chunk can be re-read.
type uploadSource interface {
	chunk(offset int64, n int) ([]byte, error)
}

// seekSource reads chunks from a seekable reader, never holding the whole file.
type seekSource struct {
	rs   io.ReadSeeker
	base int64 // where the file starts within rs
}

func (s *seekSource) chunk(offset int64, n int) ([]byte, error) {
	if _, err := s.rs.Seek(s.base+offset, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(s.rs, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// memSource serves chunks from a buffered copy.
type memSource []byte

func (m memSource) chunk(offset int64, n int) ([]byte, error) {
	if offset < 0 || offset+int64(n) > int64(len(m)) {
		return nil, io.ErrUnexpectedEOF
	}
	return m[offset : offset+int64(n)], nil
}

// newUploadSource hashes the file and returns a source for its chunks. The
// reader must hold exactly size bytes: BSN pre-allocates the file from the
// declared size, so a mismatch would corrupt assembly.
func newUploadSource(ctx context.Context, r io.Reader, size int64) (uploadSource, string, error) {
	h := sha1.New() // #nosec G401 -- required by the Upload API
	if rs, ok := r.(io.ReadSeeker); ok {
		base, err := rs.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, "", fmt.Errorf("failed to locate file start: %w", err)
		}
		n, err := copyHash(ctx, h, io.LimitReader(rs, size+1))
		if err != nil {
			return nil, "", fmt.Errorf("failed to read file: %w", err)
		}
		if n != size {
			return nil, "", sizeMismatch(size, n)
		}
		return &seekSource{rs: rs, base: base}, hex.EncodeToString(h.Sum(nil)), nil
	}

	data, err := io.ReadAll(&ctxReader{ctx: ctx, r: io.LimitReader(r, size+1)})
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}
	if int64(len(data)) != size {
		return nil, "", sizeMismatch(size, int64(len(data)))
	}
	h.Write(data)
	return memSource(data), hex.EncodeToString(h.Sum(nil)), nil
}

func sizeMismatch(declared, actual int64) error {
	// A reader capped at size+1 can only prove "at least" when it is too long.
	if actual > declared {
		return errors.NewValidationError("fileSize", declared, "reader holds more bytes than the declared size")
	}
	return errors.NewValidationError("fileSize", declared,
		fmt.Sprintf("reader holds only %d bytes", actual))
}

func copyHash(ctx context.Context, h hash.Hash, r io.Reader) (int64, error) {
	return io.Copy(h, &ctxReader{ctx: ctx, r: r})
}

// ctxReader stops a long read when its context ends.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
