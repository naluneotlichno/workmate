package task

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	archive "workmate/internal/archive"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	tempDir := t.TempDir()
	return NewManagerWithOptions(Options{
		DataDir:            tempDir,
		AllowedExtensions:  []string{".pdf", ".jpeg"},
		MaxConcurrentTasks: 1,
	})
}

func TestCreateTaskAndUpdateTitle(t *testing.T) {
	m := newTestManager(t)
	taskEntity := m.CreateTask()
	if taskEntity.Status != StatusCreated {
		t.Fatalf("expected status created, got %s", taskEntity.Status)
	}

	_, err := m.AddFiles(taskEntity.ID, []string{
		"https://www.example.org/a.pdf",
		"https://e.org/b.jpeg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(taskEntity.Title, "example.org") || !strings.Contains(taskEntity.Title, "e.org") {
		t.Fatalf("title should contain hosts, got %q", taskEntity.Title)
	}
	if strings.Contains(taskEntity.Title, "www.") {
		t.Fatalf("title should not contain www prefix: %q", taskEntity.Title)
	}
}

func TestAddFilesValidationErrors(t *testing.T) {
	m := newTestManager(t)
	taskEntity := m.CreateTask()

	if _, err := m.AddFiles(taskEntity.ID, nil); !errors.Is(err, ErrNoURLs) {
		t.Fatalf("expected ErrNoURLs, got %v", err)
	}

	if _, err := m.AddFiles(taskEntity.ID, []string{"https://e.org/a.exe"}); err == nil || !strings.Contains(err.Error(), "extension not allowed") {
		t.Fatalf("expected extension not allowed error, got %v", err)
	}

	if _, err := m.AddFiles(taskEntity.ID, []string{"https://e.org/a.pdf", "https://e.org/b.jpeg", "https://e.org/c.pdf", "https://e.org/d.pdf"}); !errors.Is(err, ErrTooManyFiles) {
		t.Fatalf("expected ErrTooManyFiles, got %v", err)
	}
}

func TestProcessingFlowReadyAndArchivePath(t *testing.T) {
	m := newTestManager(t)

	m.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {

		f, err := os.Create(dest)
		if err != nil {
			return nil, err
		}
		zw := zip.NewWriter(f)
		_, _ = zw.Create("dummy.txt")
		_ = zw.Close()
		_ = f.Close()

		res := make([]archive.Result, len(urls))
		for i := range res {
			res[i].Filename = "f.pdf"
			if i == len(res)-1 {
				res[i].Err = "boom"
			}
		}
		return res, nil
	})

	tsk := m.CreateTask()
	if _, err := m.AddFiles(tsk.ID, []string{"https://e.org/a.pdf", "https://e.org/b.jpeg", "https://e.org/c.pdf"}); err != nil {
		t.Fatalf("add files: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got, ok := m.GetTask(tsk.ID); ok {
			if got.Status == StatusReady || got.Status == StatusFailed {
				if got.Status != StatusReady {
					t.Fatalf("expected ready, got %s", got.Status)
				}
				if got.ArchivePath == "" {
					t.Fatalf("expected archive path set")
				}
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for processing")
}

func TestIsBusyWhileProcessing(t *testing.T) {
	m := NewManagerWithOptions(Options{DataDir: t.TempDir(), AllowedExtensions: []string{".pdf"}, MaxConcurrentTasks: 1})
	blocker := make(chan struct{})
	m.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {
		<-blocker

		f, _ := os.Create(dest)
		_ = f.Close()
		r := make([]archive.Result, len(urls))
		return r, nil
	})

	tsk := m.CreateTask()
	if _, err := m.AddFiles(tsk.ID, []string{"https://e.org/a.pdf", "https://e.org/b.pdf", "https://e.org/c.pdf"}); err != nil {
		t.Fatalf("add files: %v", err)
	}

	if !m.IsBusy() {
		t.Fatalf("expected manager to be busy while processing")
	}
	close(blocker)

	ok := m.WaitAll(context.Background())
	if !ok {
		t.Fatalf("expected workers to finish")
	}
}

func TestPersistAndLoadFromDisk(t *testing.T) {
	dataDir := t.TempDir()
	m := NewManagerWithOptions(Options{DataDir: dataDir, AllowedExtensions: []string{".pdf"}, MaxConcurrentTasks: 1})

	t1 := &Task{ID: "t1", Status: StatusInProgress, CreatedAt: time.Now()}
	t2 := &Task{ID: "t2", Status: StatusReady, CreatedAt: time.Now()}
	if err := m.persistTask(t1); err != nil {
		t.Fatalf("persist t1: %v", err)
	}
	if err := m.persistTask(t2); err != nil {
		t.Fatalf("persist t2: %v", err)
	}

	m2 := NewManagerWithOptions(Options{DataDir: dataDir, AllowedExtensions: []string{".pdf"}, MaxConcurrentTasks: 1})
	if err := m2.LoadFromDisk(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got, ok := m2.GetTask("t1"); !ok || got.Status != StatusFailed {
		t.Fatalf("expected t1 failed after load, got: %+v, ok=%v", got, ok)
	}
	if got, ok := m2.GetTask("t2"); !ok || got.Status != StatusReady {
		t.Fatalf("expected t2 ready after load, got: %+v, ok=%v", got, ok)
	}
}
