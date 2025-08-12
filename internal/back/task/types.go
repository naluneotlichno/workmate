package task

import "time"

type Status string

const (
	StatusCreated    Status = "created"
	StatusInProgress Status = "in_progress"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

type FileState string

const (
	FilePending FileState = "pending"
	FileOK      FileState = "ok"
	FileFailed  FileState = "failed"
)

type FileRef struct {
	URL      string    `json:"url"`
	State    FileState `json:"state"`
	Error    string    `json:"error,omitempty"`
	Filename string    `json:"filename,omitempty"`
}

type Task struct {
	ID          string    `json:"id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	Title       string    `json:"title"`
	Files       []FileRef `json:"files"`
	ArchivePath string    `json:"archive_path,omitempty"`
}

type Options struct {
	DataDir            string
	AllowedExtensions  []string
	MaxConcurrentTasks int
}

const (
	MaxFilesPerTask      = 3
	defaultMaxConcurrent = 3
)
