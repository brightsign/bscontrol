package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	bshttp "github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// ContentService manages the network's content library through the Content REST
// API (2022/06). It handles metadata and CRUD; bytes go in through UploadService.
// See docs/content-api.md.
type ContentService interface {
	// List returns one page of content. Use WithMarker to continue.
	List(ctx context.Context, opts ...ListOption) (*types.ContentFileList, error)

	// ListAll follows pagination and returns every matching item.
	ListAll(ctx context.Context, opts ...ListOption) ([]types.ContentFile, error)

	// ListFolder lists the items directly in a virtual folder. The path may be
	// written with either slash style; "" is the root.
	ListFolder(ctx context.Context, virtualPath string, opts ...ListOption) (*types.ContentFileList, error)

	// GetCount returns the number of content items (respects no filter).
	GetCount(ctx context.Context) (*types.ContentFileCount, error)

	// GetByID returns one item's metadata.
	GetByID(ctx context.Context, id int) (*types.ContentFile, error)

	// Delete removes every item matching a filter expression. The filter is
	// required; build values with EscapeFilterValue. Folders are not deleted.
	Delete(ctx context.Context, filter string) (*types.ContentDeleteResult, error)

	// DeleteByID removes one item. Folders are rejected by the service.
	DeleteByID(ctx context.Context, id int) error

	// Download returns a file's bytes. It is the two-step download: read the
	// entity's direct URL, then fetch it. Prefer DownloadTo for large files.
	Download(ctx context.Context, id int) ([]byte, error)

	// DownloadTo streams a file's bytes into w and returns how many were written.
	// On an error part-way through, w may hold a partial file.
	DownloadTo(ctx context.Context, id int, w io.Writer) (int64, error)

	// GetThumbnail returns the server-generated thumbnail. A zero maxWidth or
	// maxHeight leaves the server default (100x100).
	GetThumbnail(ctx context.Context, id, maxWidth, maxHeight int) ([]byte, error)

	// CreateFolder creates a folder named name inside parentPath.
	CreateFolder(ctx context.Context, parentPath, name string) (*types.ContentFile, error)

	// Move changes an item's virtual path (JSON-Patch replace of /virtualPath/).
	Move(ctx context.Context, id int, newVirtualPath string) error
}

// contentService implements the ContentService interface.
type contentService struct {
	config      *config.Config
	httpClient  *bshttp.HTTPClient
	authManager *auth.AuthManager
}

// NewContentService creates a new content service.
func NewContentService(cfg *config.Config, httpClient *bshttp.HTTPClient, authManager *auth.AuthManager) ContentService {
	return &contentService{config: cfg, httpClient: httpClient, authManager: authManager}
}

func (s *contentService) contentURL() string {
	return fmt.Sprintf("%s/%s/Content", s.config.BSNBaseURL, s.config.APIVersion)
}

func validateContentID(id int) error {
	if id <= 0 {
		return errors.NewValidationError("id", id, "content ID must be positive")
	}
	return nil
}

// List retrieves one page of content.
func (s *contentService) List(ctx context.Context, opts ...ListOption) (*types.ContentFileList, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ContentFileList
	if err := s.httpClient.GetWithAuth(ctx, token, withQuery(s.contentURL(), listQuery(opts)), &result); err != nil {
		return nil, fmt.Errorf("failed to list content: %w", err)
	}
	return &result, nil
}

// ListAll retrieves every matching content item across pages.
func (s *contentService) ListAll(ctx context.Context, opts ...ListOption) ([]types.ContentFile, error) {
	return collectPages(func(marker string) ([]types.ContentFile, bool, string, error) {
		page, err := s.List(ctx, withMarker(opts, marker)...)
		if err != nil {
			return nil, false, "", err
		}
		return page.Items, page.IsTruncated, page.NextMarker, nil
	})
}

// rootURL builds /Content/Root/{segments} with every segment escaped.
func (s *contentService) rootURL(virtualPath string) (string, error) {
	segs, err := virtualPathSegments(virtualPath)
	if err != nil {
		return "", err
	}
	escaped := make([]string, len(segs))
	for i, seg := range segs {
		escaped[i] = escSegment(seg)
	}
	return s.contentURL() + "/Root/" + strings.Join(escaped, "/"), nil
}

// ListFolder lists the items in one virtual folder.
func (s *contentService) ListFolder(ctx context.Context, virtualPath string, opts ...ListOption) (*types.ContentFileList, error) {
	rootURL, err := s.rootURL(virtualPath)
	if err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ContentFileList
	if err := s.httpClient.GetWithAuth(ctx, token, withQuery(rootURL, listQuery(opts)), &result); err != nil {
		return nil, fmt.Errorf("failed to list content folder %q: %w", virtualPath, err)
	}
	return &result, nil
}

// GetCount retrieves the number of content items.
func (s *contentService) GetCount(ctx context.Context) (*types.ContentFileCount, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	// NOTE: the response shape of GET /Content/Count is not pinned down in the
	// spec (a bare number for presentations, possibly an object here), so decode
	// either.
	var count flexCount
	if err := s.httpClient.GetWithAuth(ctx, token, s.contentURL()+"/Count", &count); err != nil {
		return nil, fmt.Errorf("failed to get content count: %w", err)
	}
	return &types.ContentFileCount{Count: count.N}, nil
}

// GetByID retrieves one item's metadata.
func (s *contentService) GetByID(ctx context.Context, id int) (*types.ContentFile, error) {
	if err := validateContentID(id); err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ContentFile
	if err := s.httpClient.GetWithAuth(ctx, token, fmt.Sprintf("%s/%d", s.contentURL(), id), &result); err != nil {
		return nil, fmt.Errorf("failed to get content %d: %w", id, err)
	}
	return &result, nil
}

// Delete removes content matching a filter.
func (s *contentService) Delete(ctx context.Context, filter string) (*types.ContentDeleteResult, error) {
	if filter == "" {
		return nil, errors.NewValidationError("filter", filter, "filter cannot be empty for bulk delete")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	// NOTE: the bulk-delete response is not specified (a count, or an object
	// carrying one); an empty body reads as zero.
	var res flexCount
	deleteURL := withQuery(s.contentURL(), "filter="+escQuery(filter))
	if err := s.httpClient.DeleteWithAuth(ctx, token, deleteURL, &res); err != nil {
		return nil, fmt.Errorf("failed to delete content with filter %q: %w", filter, err)
	}
	return &types.ContentDeleteResult{DeletedCount: res.N, DeletedIDs: res.IDs, Errors: res.Errors}, nil
}

// DeleteByID removes one item.
func (s *contentService) DeleteByID(ctx context.Context, id int) error {
	if err := validateContentID(id); err != nil {
		return err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return err
	}

	if err := s.httpClient.DeleteWithAuth(ctx, token, fmt.Sprintf("%s/%d", s.contentURL(), id), nil); err != nil {
		return fmt.Errorf("failed to delete content %d: %w", id, err)
	}
	return nil
}

// Download returns a file's bytes.
func (s *contentService) Download(ctx context.Context, id int) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := s.DownloadTo(ctx, id, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DownloadTo streams a file's bytes into w.
func (s *contentService) DownloadTo(ctx context.Context, id int, w io.Writer) (int64, error) {
	if w == nil {
		return 0, errors.NewValidationError("w", w, "writer cannot be nil")
	}
	entity, err := s.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}

	target, sendToken, err := s.downloadTarget(entity)
	if err != nil {
		return 0, err
	}
	token := ""
	if sendToken {
		if token, err = s.authManager.GetToken(); err != nil {
			return 0, err
		}
	}

	n, err := s.httpClient.Stream(ctx, token, target, nil, w)
	if err != nil {
		return n, fmt.Errorf("failed to download content %d: %w", id, err)
	}
	return n, nil
}

// downloadTarget decides where the bytes live and whether the bearer token may
// go with the request. The token is only sent to the BSN API host: the file URL
// is normally a pre-signed storage URL, which rejects an extra credential, and a
// token has no business leaving BSN.
func (s *contentService) downloadTarget(entity *types.ContentFile) (target string, sendToken bool, err error) {
	raw := entity.FileURL
	if raw == "" {
		// NOTE: docs/content-api.md names the entity's fileUrl as the direct
		// URL, but the entity schema lists physicalPath (an absolute storage URL
		// in practice). Use fileUrl when present and fall back to physicalPath.
		raw = entity.PhysicalPath
	}
	if raw == "" {
		return "", false, errors.NewAppError("content_no_download_url",
			fmt.Sprintf("content %d has no download URL", entity.ID), "")
	}

	u, perr := url.Parse(raw)
	if perr != nil {
		return "", false, errors.NewAppError("content_bad_download_url",
			fmt.Sprintf("content %d has an unusable download URL", entity.ID), perr.Error())
	}
	base, perr := url.Parse(s.config.BSNBaseURL)
	if perr != nil {
		return "", false, errors.NewConfigError("BSNBaseURL", "is not a valid URL", "")
	}
	if !u.IsAbs() {
		u = base.ResolveReference(u)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", false, errors.NewAppError("content_bad_download_url",
			fmt.Sprintf("content %d download URL has unsupported scheme %q", entity.ID, u.Scheme), "")
	}
	return u.String(), u.Host == base.Host, nil
}

// GetThumbnail returns the server-generated thumbnail bytes.
func (s *contentService) GetThumbnail(ctx context.Context, id, maxWidth, maxHeight int) ([]byte, error) {
	if err := validateContentID(id); err != nil {
		return nil, err
	}
	if maxWidth < 0 {
		return nil, errors.NewValidationError("maxWidth", maxWidth, "cannot be negative")
	}
	if maxHeight < 0 {
		return nil, errors.NewValidationError("maxHeight", maxHeight, "cannot be negative")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if maxWidth > 0 {
		query.Set("maxWidth", strconv.Itoa(maxWidth))
	}
	if maxHeight > 0 {
		query.Set("maxHeight", strconv.Itoa(maxHeight))
	}
	var buf bytes.Buffer
	_, err = s.httpClient.Stream(ctx, token,
		withQuery(fmt.Sprintf("%s/%d", s.contentURL(), id), query.Encode()),
		map[string]string{"Accept": "image/*"}, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to get thumbnail for content %d: %w", id, err)
	}
	return buf.Bytes(), nil
}

// CreateFolder creates a folder inside parentPath.
func (s *contentService) CreateFolder(ctx context.Context, parentPath, name string) (*types.ContentFile, error) {
	if name == "" {
		return nil, errors.NewValidationError("name", name, "folder name cannot be empty")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return nil, errors.NewValidationError("name", name, "folder name must be a single path segment")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return nil, errors.NewValidationError("name", name, "folder name contains a control character")
		}
	}
	rootURL, err := s.rootURL(parentPath)
	if err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ContentFile
	body := struct {
		Name string `json:"name"`
	}{Name: name}
	if err := s.httpClient.PostWithAuth(ctx, token, rootURL, body, &result); err != nil {
		return nil, fmt.Errorf("failed to create folder %q in %q: %w", name, parentPath, err)
	}
	return &result, nil
}

// jsonPatchOp is one JSON-Patch (RFC 6902) operation.
type jsonPatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// Move changes an item's virtual path.
func (s *contentService) Move(ctx context.Context, id int, newVirtualPath string) error {
	if err := validateContentID(id); err != nil {
		return err
	}
	vp, err := ToBSNVirtualPath(newVirtualPath)
	if err != nil {
		return err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return err
	}

	// op is the lowercase literal "replace" and path carries a trailing slash,
	// both exactly as the reference client sends them.
	patch, err := json.Marshal([]jsonPatchOp{{Op: "replace", Path: "/virtualPath/", Value: vp}})
	if err != nil {
		return fmt.Errorf("failed to encode move patch: %w", err)
	}
	err = s.httpClient.DoWithAuth(ctx, token, &bshttp.Request{
		Method:  http.MethodPatch,
		URL:     fmt.Sprintf("%s/%d/", s.contentURL(), id),
		Body:    patch,
		Headers: map[string]string{"Content-Type": "application/json-patch+json"},
	})
	if err != nil {
		return fmt.Errorf("failed to move content %d to %q: %w", id, vp, err)
	}
	return nil
}
