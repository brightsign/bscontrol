package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/types"
)

// ready is the preamble every network-scoped call shares: a valid token and a
// network on the session. It returns the token to use for the request.
func ready(ctx context.Context, am *auth.AuthManager) (string, error) {
	if err := am.EnsureValid(ctx); err != nil {
		return "", err
	}
	if err := am.EnsureNetworkSet(ctx); err != nil {
		return "", err
	}
	return am.GetToken()
}

// listQuery turns list options into the query string every marker-paginated
// collection shares. The result is already encoded and carries no leading "?".
func listQuery(opts []ListOption) string {
	cfg := &listConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	params := url.Values{}
	if cfg.pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(cfg.pageSize))
	}
	if cfg.marker != "" {
		params.Set("marker", cfg.marker)
	}
	if cfg.filter != "" {
		params.Set("filter", cfg.filter)
	}
	if cfg.sort != "" {
		params.Set("sort", cfg.sort)
	}
	return params.Encode()
}

// withQuery appends an already-encoded query string to a URL.
func withQuery(u, query string) string {
	if query == "" {
		return u
	}
	return u + "?" + query
}

// maxListPages is a backstop for ListAll: a server that keeps handing back
// fresh markers forever would otherwise loop us forever.
const maxListPages = 10000

// collectPages follows marker pagination until the server says it is done.
// fetch returns one page and the marker for the next. A truncated page with no
// marker, or a marker we have already used, is a server fault and ends the loop
// with an error rather than a spin.
func collectPages[T any](fetch func(marker string) (items []T, truncated bool, next string, err error)) ([]T, error) {
	var all []T
	seen := map[string]bool{}
	marker := ""
	for page := 0; page < maxListPages; page++ {
		items, truncated, next, err := fetch(marker)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !truncated {
			return all, nil
		}
		if next == "" {
			return nil, errors.NewAppError("pagination_no_marker",
				"list response was truncated but gave no next marker", "")
		}
		if seen[next] {
			return nil, errors.NewAppError("pagination_loop",
				"list response repeated a marker", "")
		}
		seen[next] = true
		marker = next
	}
	return nil, errors.NewAppError("pagination_limit", "list exceeded the page limit", "")
}

// withMarker returns opts with the marker replaced.
func withMarker(opts []ListOption, marker string) []ListOption {
	out := make([]ListOption, 0, len(opts)+1)
	out = append(out, opts...)
	if marker != "" {
		out = append(out, WithMarker(marker))
	}
	return out
}

// flexCount decodes the shapes BSN uses for "how many": a bare number or an
// object carrying a count. Count endpoints and bulk deletes are not consistent
// about it.
type flexCount struct {
	N      int
	IDs    []int
	Errors []string
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *flexCount) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		c.N = n
		return nil
	}
	var obj struct {
		Count        *int     `json:"count"`
		TotalCount   *int     `json:"totalCount"`
		DeletedCount *int     `json:"deletedCount"`
		DeletedIDs   []int    `json:"deletedIds"`
		Errors       []string `json:"errors"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("expected a count (number or object): %w", err)
	}
	switch {
	case obj.DeletedCount != nil:
		c.N = *obj.DeletedCount
	case obj.Count != nil:
		c.N = *obj.Count
	case obj.TotalCount != nil:
		c.N = *obj.TotalCount
	}
	c.IDs = obj.DeletedIDs
	c.Errors = obj.Errors
	return nil
}

// ---------------------------------------------------------------------------
// Virtual paths and media types
// ---------------------------------------------------------------------------

// ToBSNVirtualPath converts a path as a person would write it (forward slashes,
// any leading or trailing slash, or already backslashed) into the form the
// Upload API wants: backslash separated, with a leading and a trailing
// backslash. The root is a single backslash.
//
//	"videos/promos"  -> \videos\promos\
//	"/videos/"       -> \videos\
//	""               -> \
//
// Empty and "." segments are dropped. ".." and control characters are rejected,
// since a virtual path that escapes its folder is never what the caller meant.
func ToBSNVirtualPath(p string) (string, error) {
	segs, err := virtualPathSegments(p)
	if err != nil {
		return "", err
	}
	if len(segs) == 0 {
		return `\`, nil
	}
	return `\` + strings.Join(segs, `\`) + `\`, nil
}

// virtualPathSegments splits a virtual path, in either slash style, into clean
// segments.
func virtualPathSegments(p string) ([]string, error) {
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return nil, errors.NewValidationError("virtualPath", p, "path contains a control character")
		}
	}
	var segs []string
	for _, seg := range strings.Split(strings.ReplaceAll(p, `\`, "/"), "/") {
		switch seg {
		case "", ".":
			continue
		case "..":
			return nil, errors.NewValidationError("virtualPath", p, `path may not contain ".." segments`)
		}
		segs = append(segs, seg)
	}
	return segs, nil
}

// EscapeFilterValue escapes a string for use inside a quoted value of a BSN
// filter expression: backslashes are doubled and single quotes are doubled. Use
// it for anything that did not come from a literal you wrote yourself.
//
//	[FileName] IS '` + EscapeFilterValue(name) + `'
func EscapeFilterValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `''`)
}

// mediaTypeByExtension is the reference client's extension table
// (bscGetFileMediaType). Anything not listed is Auxiliary.
var mediaTypeByExtension = map[string]string{
	"jpg": types.MediaTypeImage, "jpeg": types.MediaTypeImage, "png": types.MediaTypeImage, "bmp": types.MediaTypeImage,
	"mov": types.MediaTypeVideo, "mp4": types.MediaTypeVideo, "mpg": types.MediaTypeVideo, "ts": types.MediaTypeVideo,
	"vob": types.MediaTypeVideo, "wmv": types.MediaTypeVideo, "m2ts": types.MediaTypeVideo, "mkv": types.MediaTypeVideo,
	"webm": types.MediaTypeVideo,
	"mp3":  types.MediaTypeAudio, "wav": types.MediaTypeAudio, "ogg": types.MediaTypeAudio, "flac": types.MediaTypeAudio,
	"m4a": types.MediaTypeAudio, "aac": types.MediaTypeAudio, "ac3": types.MediaTypeAudio, "eac3": types.MediaTypeAudio,
	"ec3": types.MediaTypeAudio,
	"txt": types.MediaTypeText,
	"ttf": types.MediaTypeFont, "otf": types.MediaTypeFont, "ttc": types.MediaTypeFont,
	"css": types.MediaTypeStylesheet,
}

// DetectMediaType picks the concrete upload mediaType from a file name's
// extension, case-insensitively. Unknown or missing extensions are Auxiliary, as
// in the reference client; "Auto" is never returned.
func DetectMediaType(fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(fileName), "."))
	if mt, ok := mediaTypeByExtension[ext]; ok {
		return mt
	}
	return types.MediaTypeAuxiliary
}

// validMediaTypes are the values the Upload API accepts for mediaType.
var validMediaTypes = map[string]bool{
	types.MediaTypeAuto: true, types.MediaTypeText: true, types.MediaTypeImage: true, types.MediaTypeVideo: true,
	types.MediaTypeAudio: true, types.MediaTypeFont: true, types.MediaTypeStylesheet: true,
	types.MediaTypeAuxiliary: true, types.MediaTypeWebPage: true, types.MediaTypeDeviceWebPage: true,
}
