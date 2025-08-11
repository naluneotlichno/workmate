package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	fileutil "workmate/internal/file"
)

// TaskStore abstracts persistence for tasks and archive destination resolution.
// Default implementation is file-based, but the interface allows plugging a DB-backed store later (e.g., Postgres via pgxpool).
type TaskStore interface {
	SaveTask(ctx context.Context, t *Task) error
	LoadTasks(ctx context.Context) ([]*Task, error)
	EnsureTaskDir(ctx context.Context, taskID string) (string, error)
	ArchivePath(taskID string) string
}

// fileStore implements TaskStore using the local filesystem under dataDir.
type fileStore struct {
	dataDir string
}

func NewFileStore(dataDir string) TaskStore { //nolint:ireturn
	if dataDir == "" {
		dataDir = "data"
	}
	return &fileStore{dataDir: dataDir}
}

func (s *fileStore) taskDir(taskID string) string {
	return filepath.Join(s.dataDir, "tasks", taskID)
}

func (s *fileStore) statusPath(taskID string) string {
	return filepath.Join(s.taskDir(taskID), "status.json")
}

func (s *fileStore) ArchivePath(taskID string) string {
	return filepath.Join(s.taskDir(taskID), "archive.zip")
}

func (s *fileStore) EnsureTaskDir(ctx context.Context, taskID string) (string, error) { //nolint:revive,stylecheck // context reserved for future use
	dir := s.taskDir(taskID)
	if err := fileutil.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("ensure task dir: %w", err)
	}
	return dir, nil
}

func (s *fileStore) SaveTask(ctx context.Context, t *Task) error { //nolint:revive,stylecheck // context reserved for future use
	if _, err := s.EnsureTaskDir(ctx, t.ID); err != nil {
		return err
	}
	return fileutil.WriteJSONAtomic(s.statusPath(t.ID), t) //nolint:wrapcheck
}

func (s *fileStore) LoadTasks(ctx context.Context) ([]*Task, error) { //nolint:revive,stylecheck // context reserved for future use
	root := filepath.Join(s.dataDir, "tasks")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir: %w", err)
	}
	tasks := make([]*Task, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(s.statusPath(e.Name())) //nolint:gosec // path is controlled by application
		if err != nil {
			continue
		}
		var t Task
		if err := json.Unmarshal(b, &t); err != nil {
			continue
		}
		tt := t
		tasks = append(tasks, &tt)
	}
	return tasks, nil
}
