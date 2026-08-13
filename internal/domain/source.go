package domain

// SourceKind is the kind of an imported source.
type SourceKind string

const (
	SourceKindFolder SourceKind = "folder"
	SourceKindGit    SourceKind = "git"
	SourceKindZip    SourceKind = "zip"
)

// SourceStatus is the lifecycle status of a source.
type SourceStatus string

const (
	StatusPending    SourceStatus = "pending"
	StatusCloning    SourceStatus = "cloning"
	StatusExtracting SourceStatus = "extracting"
	StatusIndexing   SourceStatus = "indexing"
	StatusReady      SourceStatus = "ready"
	StatusUnavailable SourceStatus = "unavailable"
	StatusError      SourceStatus = "error"
)

// AllSourceStatuses lists every status value for the badge and the status
// guards.
var AllSourceStatuses = []SourceStatus{
	StatusPending, StatusCloning, StatusExtracting, StatusIndexing,
	StatusReady, StatusUnavailable, StatusError,
}

// Source is an imported folder, git repo, or zip — the physical browsing axis.
type Source struct {
	ID            int64
	Kind          SourceKind
	Name          string
	RootPath      string
	OriginURL     string
	GitBranch     string
	GitCommit     string
	IsManaged     bool // 1 = app owns root_path (zip), 0 = referenced in place
	Status        SourceStatus
	ErrorMessage  string
	DocumentCount int64
	IgnoreGlobs   []string
	CreatedAt     string
	UpdatedAt     string
	IndexedAt     string
}

// Managed reports whether deleting this source deletes files on disk (zip
// sources extracted into app storage).
func (s Source) Managed() bool { return s.IsManaged }
