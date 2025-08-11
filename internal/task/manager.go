package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"workmate/internal/archive"
	fileutil "workmate/internal/file"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Manager provides an in-memory store for tasks and background processing
type Manager struct {
	mu                sync.RWMutex
	tasks             map[string]*Task
	dataDir           string
	allowedExtensions map[string]struct{}
	semaphore         chan struct{}
	buildArchive      func(ctx context.Context, destZipPath string, urls []string) ([]archive.Result, error)
	workersWG         sync.WaitGroup
	baseCtx           context.Context
	store             TaskStore
}

// NewManager creates a manager with default options suitable for tests
func NewManager() *Manager {
	return NewManagerWithOptions(Options{
		DataDir:            "data",
		AllowedExtensions:  []string{".pdf", ".jpeg"},
		MaxConcurrentTasks: defaultMaxConcurrent,
	})
}

// NewManagerWithOptions creates a manager with provided configuration
func NewManagerWithOptions(opts Options) *Manager { //nolint:cyclop
	allowed := make(map[string]struct{}, len(opts.AllowedExtensions))
	for _, ext := range opts.AllowedExtensions {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		allowed[ext] = struct{}{}
	}
	if opts.MaxConcurrentTasks <= 0 {
		opts.MaxConcurrentTasks = 1
	}
	return &Manager{
		tasks:             make(map[string]*Task),
		dataDir:           opts.DataDir,
		allowedExtensions: allowed,
		semaphore:         make(chan struct{}, opts.MaxConcurrentTasks),
		buildArchive:      archive.BuildArchive,
		baseCtx:           context.Background(),
		store:             NewFileStore(opts.DataDir),
	}
}

// IsBusy reports whether the system is currently at max concurrent processing
func (m *Manager) IsBusy() bool {
	return len(m.semaphore) >= cap(m.semaphore)
}

// CreateTask creates a new task and stores it in memory
func (m *Manager) CreateTask() *Task {
	newID := uuid.NewString()
	newTask := &Task{
		ID:        newID,
		Status:    StatusCreated,
		CreatedAt: time.Now(),
		Files:     make([]FileRef, 0, maxFilesPerTask),
	}

	m.mu.Lock()
	m.tasks[newID] = newTask
	m.mu.Unlock()

	if err := m.persistTask(newTask); err != nil { // best-effort
		log.Warn().Str("task_id", newTask.ID).Err(err).Msg("persist task failed")
	}
	return newTask
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*Task, bool) {
	m.mu.RLock()
	foundTask, taskFound := m.tasks[taskID]
	m.mu.RUnlock()
	return foundTask, taskFound
}

// AddFiles adds URLs to a task enforcing extension and count limits.
// Returns the updated task or an error.
func (m *Manager) AddFiles(taskID string, urls []string) (*Task, error) {
	if len(urls) == 0 {
		return nil, ErrNoURLs
	}

	m.mu.Lock()
	currentTask, taskFound := m.tasks[taskID]
	if !taskFound {
		m.mu.Unlock()
		return nil, ErrTaskNotFound
	}
	if len(currentTask.Files)+len(urls) > maxFilesPerTask {
		m.mu.Unlock()
		return nil, ErrTooManyFiles
	}

	// Validate and append
	for _, rawURL := range urls {
		fileExtension := strings.ToLower(filepath.Ext(strings.TrimSpace(rawURL)))
		if _, allowed := m.allowedExtensions[fileExtension]; !allowed {
			m.mu.Unlock()
			return nil, NewErrExtNotAllowed(fileExtension)
		}
		currentTask.Files = append(currentTask.Files, FileRef{URL: rawURL, State: FilePending})
	}
	m.mu.Unlock()

	if err := m.persistTask(currentTask); err != nil {
		log.Warn().Str("task_id", currentTask.ID).Err(err).Msg("persist after add files failed")
		return nil, err
	}

	// If we reached max files, acquire a processing slot synchronously to
	// reflect busy state immediately, then start background processing.
	if len(currentTask.Files) == maxFilesPerTask {
		m.semaphore <- struct{}{}
		m.workersWG.Add(1)
		go func() {
			defer m.workersWG.Done()
			m.startProcessing(taskID, true)
		}()
	}

	return currentTask, nil
}

// SetBaseContext sets the base context used to control long-running operations (e.g., downloads).
// Intended to be set at process startup and cancelled during shutdown.
func (m *Manager) SetBaseContext(ctx context.Context) {
	m.mu.Lock()
	m.baseCtx = ctx
	m.mu.Unlock()
}

// WaitAll blocks until all in-flight task workers finish or the context is done.
// Returns true if all workers finished, false if timed out.
func (m *Manager) WaitAll(ctx context.Context) bool {
	done := make(chan struct{})
	go func() {
		m.workersWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

// UseArchiveBuilder allows tests to inject a fake archive builder.
// Not safe for concurrent mutation with running tasks; intended for test setup only.
func (m *Manager) UseArchiveBuilder(builder func(ctx context.Context, destZipPath string, urls []string) ([]archive.Result, error)) {
	m.mu.Lock()
	m.buildArchive = builder
	m.mu.Unlock()
}

// persistTask writes task state to disk atomically under data/tasks/<id>/status.json
func (m *Manager) persistTask(taskEntity *Task) error {
	if m.store != nil {
		// let store handle persist
		return m.store.SaveTask(context.Background(), taskEntity) //nolint:wrapcheck
	}
	// Fallback to direct file persist (should not happen when store is set)
	taskDirectory := filepath.Join(m.dataDir, "tasks", taskEntity.ID)
	if err := fileutil.EnsureDir(taskDirectory); err != nil {
		return fmt.Errorf("ensure task dir: %w", err)
	}
	statusPath := filepath.Join(taskDirectory, "status.json")
	return fileutil.WriteJSONAtomic(statusPath, taskEntity) //nolint:wrapcheck
}
