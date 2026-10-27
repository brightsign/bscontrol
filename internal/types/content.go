package types

import "encoding/json"

// This file holds the wire types for the Content REST API (2022/06) and the
// Upload API (2019/03). See docs/content-api.md for how the two fit together.

// Media types, as the Upload API spells them (enum BsnUploadMediaType in the
// reference client). CSS is "Stylesheet", not "Css".
const (
	MediaTypeAuto          = "Auto"
	MediaTypeText          = "Text"
	MediaTypeImage         = "Image"
	MediaTypeVideo         = "Video"
	MediaTypeAudio         = "Audio"
	MediaTypeFont          = "Font"
	MediaTypeStylesheet    = "Stylesheet"
	MediaTypeAuxiliary     = "Auxiliary"
	MediaTypeWebPage       = "WebPage"
	MediaTypeDeviceWebPage = "DeviceWebPage"
	MediaTypeFolder        = "Folder"
)

// ContentFile is one entry in the network's content library: a file or, when
// MediaType is "Folder", a folder. Most fields are computed by BSN after an
// upload; the client never sends them.
type ContentFile struct {
	ID               int     `json:"id"`
	FileName         string  `json:"fileName"`
	PhysicalPath     string  `json:"physicalPath,omitempty"`
	VirtualPath      string  `json:"virtualPath,omitempty"` // backslash form, e.g. \videos\
	ThumbPath        string  `json:"thumbPath,omitempty"`
	FileURL          string  `json:"fileUrl,omitempty"` // direct download URL, when the API provides one
	MediaType        string  `json:"mediaType,omitempty"`
	FileSize         int64   `json:"fileSize,omitempty"`
	FileHash         string  `json:"fileHash,omitempty"` // may carry a "SHA1:" prefix
	UploadDate       BSNTime `json:"uploadDate"`
	LastModifiedDate BSNTime `json:"lastModifiedDate"`
	// FileLastModifiedDate is the source file's own modification time.
	FileLastModifiedDate BSNTime `json:"fileLastModifiedDate"`

	// Metadata and ProbeData are computed by BSN (codec, dimensions, duration,
	// ffprobe output). Their shape varies by media type, so they are kept raw.
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	ProbeData json.RawMessage `json:"probeData,omitempty"`
}

// ContentFileList is one page of content files.
type ContentFileList struct {
	Items       []ContentFile `json:"items"`
	IsTruncated bool          `json:"isTruncated"`
	NextMarker  string        `json:"nextMarker,omitempty"`
	TotalCount  int           `json:"totalCount,omitempty"`
}

// ContentFileCount is the number of content items on the network.
type ContentFileCount struct {
	Count int `json:"count"`
}

// ContentDeleteResult is the outcome of a bulk delete by filter.
type ContentDeleteResult struct {
	DeletedCount int      `json:"deletedCount"`
	DeletedIDs   []int    `json:"deletedIds,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// ContentUploadArguments is the body of the Upload API's start and complete
// calls. Key casing matters: the hash key is "shA1Hash" (capital A only) exactly
// as the reference client sends it, and "tags" is deliberately absent.
type ContentUploadArguments struct {
	SessionToken *string `json:"sessionToken"`
	UploadToken  *string `json:"uploadToken"`
	FileName     string  `json:"fileName"`
	VirtualPath  string  `json:"virtualPath"` // backslash form, leading and trailing \
	MediaType    string  `json:"mediaType"`
	FileSize     int64   `json:"fileSize"`
	// FileLastModifiedDate is the "0001-01-01 00:00:00Z" sentinel on start and a real ISO date on complete.
	FileLastModifiedDate string  `json:"fileLastModifiedDate"`
	SHA1Hash             string  `json:"shA1Hash"` // lowercase hex SHA-1 of the whole file
	FileThumb            *string `json:"fileThumb"`
}

// UploadState is where an upload session is in its life, as BSN reports it.
type UploadState string

// Upload states (enum FileUploadState in the reference client).
const (
	UploadStateUnknown    UploadState = "Unknown"
	UploadStateQueued     UploadState = "Queued"
	UploadStateStarted    UploadState = "Started"
	UploadStateUploading  UploadState = "Uploading"
	UploadStateStopped    UploadState = "Stopped"
	UploadStateUploaded   UploadState = "Uploaded"
	UploadStateVerified   UploadState = "Verified"
	UploadStateCancelled  UploadState = "Cancelled"
	UploadStateCorrupted  UploadState = "Corrupted"
	UploadStateCollected  UploadState = "Collected"
	UploadStateTerminated UploadState = "Terminated"
	UploadStateCompleted  UploadState = "Completed"
)

// UploadChunkStatus summarises a run of chunks that share a state.
type UploadChunkStatus struct {
	Count  int         `json:"count"`
	Length int64       `json:"length"`
	State  UploadState `json:"state"`
}

// UploadMatchingFile is an existing library item BSN thinks is the same file
// (matched on name, size and hash). It is only a true duplicate when its
// VirtualPath equals the upload's.
type UploadMatchingFile struct {
	ID          int     `json:"id"`
	FileName    string  `json:"fileName"`
	VirtualPath string  `json:"virtualPath"`
	MediaType   string  `json:"mediaType,omitempty"`
	FileSize    int64   `json:"fileSize,omitempty"`
	SHA1Hash    string  `json:"shA1Hash,omitempty"`
	UploadDate  BSNTime `json:"uploadDate"`
}

// ContentUploadStatus is the Upload API's status object (BsnFileUploadStatus in
// the reference client). Decoding is case-insensitive, so "shA1Hash" and
// "SHA1Hash" both land in SHA1Hash.
type ContentUploadStatus struct {
	UploadToken      string               `json:"uploadToken"`
	ContentID        int                  `json:"contentId,omitempty"`
	FileName         string               `json:"fileName"`
	FileSize         int64                `json:"fileSize"`
	SHA1Hash         string               `json:"shA1Hash,omitempty"`
	State            UploadState          `json:"state"`
	StartTime        BSNTime              `json:"startTime"`
	EndTime          BSNTime              `json:"endTime"`
	LastActivityTime BSNTime              `json:"lastActivityTime"`
	Chunks           []UploadChunkStatus  `json:"chunks,omitempty"`
	MatchingFiles    []UploadMatchingFile `json:"matchingFiles,omitempty"`
}

// UploadResponse is what a finished Upload* call hands back.
type UploadResponse struct {
	ContentID   int     `json:"contentId"`
	FileName    string  `json:"fileName"`
	FileSize    int64   `json:"fileSize"`
	VirtualPath string  `json:"virtualPath"` // as sent: backslash form
	MediaType   string  `json:"mediaType,omitempty"`
	FileHash    string  `json:"fileHash,omitempty"` // lowercase hex SHA-1
	UploadDate  BSNTime `json:"uploadDate"`
	// UploadComplete is true when BSN reported Verified or Completed, or when the
	// file already existed.
	UploadComplete bool `json:"uploadComplete"`
	// Deduplicated is true when BSN already held an identical file at the same
	// virtual path: no bytes were sent and ContentID is the existing item.
	Deduplicated bool `json:"deduplicated,omitempty"`
}

// UploadProgress is reported to a WithUploadProgress callback after each chunk.
type UploadProgress struct {
	FileName string
	Sent     int64
	Total    int64
}
