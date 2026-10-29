package types

import "time"

// Wire types for presentations (2022/06 Presentations API) and group schedules
// (Groups/Regular/{name}/schedule). Presentations and schedules keep time.Time
// for their dates: those shapes are what gopurple has been decoding against the
// live service, and the canary depends on them.

// PresentationFileRef is a file attached to a presentation: the project file
// (.bpfx), the autoplay JSON, the resources file or a thumbnail. Type is "New"
// when the client supplies the bytes in Body, or the server's own marker when it
// reads one back. Every field except ID and Path is always written, null and
// empty values included, because the service expects the full shape.
type PresentationFileRef struct {
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	Body             any     `json:"body"` // JSON document for .bpfx/autoplay, nil for resources.txt
	TransferEncoding string  `json:"transferEncoding"`
	Size             int64   `json:"size"`
	Hash             string  `json:"hash"`
	CreationDate     *string `json:"creationDate"`
	LastModifiedDate string  `json:"lastModifiedDate"`
	ID               int     `json:"id,omitempty"`
	Path             string  `json:"path,omitempty"`
}

// PresentationFile is a content item a presentation plays (an entry in its
// Files list). Type is "Media" for library content.
type PresentationFile struct {
	Type      string `json:"type"`
	ID        int    `json:"id"` // content ID from the library
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
	Path      string `json:"path,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// PresentationAutorun selects the autorun the player boots with.
type PresentationAutorun struct {
	Version  string `json:"version,omitempty"` // e.g. "10.0.110"
	IsCustom bool   `json:"isCustom,omitempty"`
}

// PresentationDeviceWebPage references the device web page a presentation uses.
type PresentationDeviceWebPage struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// PresentationScreenSettings is the display configuration of a presentation.
type PresentationScreenSettings struct {
	VideoMode       string `json:"videoMode,omitempty"`       // e.g. "1920x1080x60p"
	Orientation     string `json:"orientation,omitempty"`     // "Landscape" or "Portrait"
	Connector       string `json:"connector,omitempty"`       // e.g. "HDMI"
	BackgroundColor string `json:"backgroundColor,omitempty"` // e.g. "RGB:000000"
	Overscan        string `json:"overscan,omitempty"`        // e.g. "NoOverscan"
}

// Presentation is a presentation on the network.
type Presentation struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type,omitempty"` // e.g. "Complete"
	Description      string    `json:"description,omitempty"`
	LastModifiedDate time.Time `json:"lastModifiedDate"`
	CreationDate     time.Time `json:"creationDate"`
	Status           string    `json:"status,omitempty"` // e.g. "Draft", "Published"
	DeviceModel      string    `json:"deviceModel,omitempty"`
	Language         string    `json:"language,omitempty"`

	ProjectFile           *PresentationFileRef `json:"projectFile,omitempty"`
	AutoplayFile          *PresentationFileRef `json:"autoplayFile,omitempty"`
	ResourcesFile         *PresentationFileRef `json:"resourcesFile,omitempty"`
	UserDefinedEventsFile *PresentationFileRef `json:"userDefinedEventsFile,omitempty"`
	ThumbnailFile         *PresentationFileRef `json:"thumbnailFile,omitempty"`

	Files          []PresentationFile `json:"files,omitempty"`
	AutorunPlugins any                `json:"autorunPlugins,omitempty"` // shape not pinned down; kept raw
	Applications   []any              `json:"applications,omitempty"`
	Dependencies   []any              `json:"dependencies,omitempty"`
	Groups         any                `json:"groups,omitempty"`
	Permissions    []any              `json:"permissions,omitempty"`

	Autorun        *PresentationAutorun        `json:"autorun,omitempty"`
	DeviceWebPage  *PresentationDeviceWebPage  `json:"deviceWebPage,omitempty"`
	ScreenSettings *PresentationScreenSettings `json:"screenSettings,omitempty"`
}

// PresentationCount is the number of presentations on the network.
type PresentationCount struct {
	Count int `json:"count"`
}

// PresentationList is one page of presentations.
type PresentationList struct {
	Items       []Presentation `json:"items"`
	IsTruncated bool           `json:"isTruncated"`
	NextMarker  string         `json:"nextMarker,omitempty"`
	TotalCount  int            `json:"totalCount,omitempty"`
}

// PresentationNullDate is the date BSN expects on a brand-new presentation.
const PresentationNullDate = "0001-01-01T00:00:00"

// PresentationCreateRequest is the body for creating or updating a presentation.
// The service wants every key present, with null/empty placeholders for the parts
// a new presentation does not have yet. Create and Update fill those in for any
// field left unset (dates, empty lists), so a minimal request needs only Name.
type PresentationCreateRequest struct {
	ID               int    `json:"id"`           // 0 for a new presentation
	Name             string `json:"name"`         // required
	CreationDate     string `json:"creationDate"` // PresentationNullDate for new
	LastModifiedDate string `json:"lastModifiedDate"`

	ProjectFile           *PresentationFileRef `json:"projectFile"`
	AutoplayFile          *PresentationFileRef `json:"autoplayFile"`
	ResourcesFile         *PresentationFileRef `json:"resourcesFile"`
	UserDefinedEventsFile *PresentationFileRef `json:"userDefinedEventsFile"`
	ThumbnailFile         *PresentationFileRef `json:"thumbnailFile"`

	Autorun        *PresentationAutorun        `json:"autorun,omitempty"`
	DeviceWebPage  *PresentationDeviceWebPage  `json:"deviceWebPage,omitempty"`
	DeviceModel    string                      `json:"deviceModel,omitempty"` // e.g. "XT1144"
	ScreenSettings *PresentationScreenSettings `json:"screenSettings,omitempty"`
	Language       string                      `json:"language,omitempty"` // e.g. "English"
	Status         string                      `json:"status,omitempty"`

	Files          []PresentationFile `json:"files"`
	AutorunPlugins any                `json:"autorunPlugins"`
	Applications   []any              `json:"applications"`
	Dependencies   []any              `json:"dependencies"`
	Groups         any                `json:"groups"`
	Permissions    []any              `json:"permissions"`
	Tags           []string           `json:"tags,omitempty"`
}

// PresentationDeleteResult is the outcome of deleting presentations by filter.
type PresentationDeleteResult struct {
	DeletedCount int      `json:"deletedCount"`
	DeletedIDs   []int    `json:"deletedIds,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// Days-of-week values the schedule API accepts as strings.
const (
	DaysOfWeekEveryDay = "EveryDay"
)

// ScheduledPresentation is a presentation scheduled on a group.
type ScheduledPresentation struct {
	ID               int    `json:"id,omitempty"`
	PresentationID   int    `json:"presentationId"`
	PresentationName string `json:"presentationName,omitempty"`
	IsRecurrent      bool   `json:"isRecurrent"`
	// EventDate applies to non-recurrent schedules.
	EventDate *time.Time `json:"eventDate,omitempty"`
	StartTime string     `json:"startTime"` // TimeSpan, HH:MM:SS
	Duration  string     `json:"duration"`  // TimeSpan, HH:MM:SS
	// RecurrenceStartDate and RecurrenceEndDate apply to recurrent schedules.
	RecurrenceStartDate *time.Time `json:"recurrenceStartDate,omitempty"`
	RecurrenceEndDate   *time.Time `json:"recurrenceEndDate,omitempty"`
	// DaysOfWeek is a number (bit mask) or a string such as "EveryDay"; the
	// service returns either, so it stays untyped.
	DaysOfWeek          any        `json:"daysOfWeek,omitempty"`
	InterruptScheduling bool       `json:"interruptScheduling,omitempty"`
	ExpirationDate      *time.Time `json:"expirationDate,omitempty"`
}

// ScheduledPresentationList is a group's schedule.
type ScheduledPresentationList struct {
	Items       []ScheduledPresentation `json:"items"`
	IsTruncated bool                    `json:"isTruncated"`
	NextMarker  string                  `json:"nextMarker,omitempty"`
	TotalCount  int                     `json:"totalCount,omitempty"`
}
