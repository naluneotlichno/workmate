package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	fileutil "workmate/internal/back/file"
)

type TaskStore interface {
	SaveTask(ctx context.Context, t *Task) error
	LoadTasks(ctx context.Context) ([]*Task, error)
	EnsureTaskDir(ctx context.Context, taskID string) (string, error)
	ArchivePath(taskID string) string
}

type fileStore struct {
	dataDir string
}

func NewFileStore(dataDir string) TaskStore {
	if dataDir == "" {
		dataDir = "storage/data"
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

func (s *fileStore) EnsureTaskDir(ctx context.Context, taskID string) (string, error) {
	dir := s.taskDir(taskID)
	if err := fileutil.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("ensure task dir: %w", err)
	}
	return dir, nil
}

func (s *fileStore) SaveTask(ctx context.Context, t *Task) error {
	if _, err := s.EnsureTaskDir(ctx, t.ID); err != nil {
		return err
	}
	if err := fileutil.WriteJSONAtomic(s.statusPath(t.ID), t); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	return nil
}

func (s *fileStore) LoadTasks(ctx context.Context) ([]*Task, error) {
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
		b, err := os.ReadFile(s.statusPath(e.Name()))
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
