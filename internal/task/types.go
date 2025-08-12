package task

import "time"

// Status represents the lifecycle state of a task
type Status string

const (
	StatusCreated    Status = "created"
	StatusInProgress Status = "in_progress"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

// FileState represents processing state of a file inside a task
type FileState string

const (
	FilePending FileState = "pending"
	FileOK      FileState = "ok"
	FileFailed  FileState = "failed"
)

// FileRef describes an input URL and its processing result
type FileRef struct {
	URL      string    `json:"url"`
	State    FileState `json:"state"`
	Error    string    `json:"error,omitempty"`
	Filename string    `json:"filename,omitempty"`
}

// Task represents a user request to build an archive
type Task struct {
	ID          string    `json:"id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Title       string    `json:"title"`
	Files       []FileRef `json:"files"`
	ArchivePath string    `json:"archive_path,omitempty"`
}

// Options configures the Manager
type Options struct {
	DataDir            string
	AllowedExtensions  []string
	MaxConcurrentTasks int
}

const (
	// MaxFilesPerTask defines how many files are required to start processing
	// and is used by API layer to expose when archive link should appear.
	MaxFilesPerTask      = 3
	defaultMaxConcurrent = 3
)
