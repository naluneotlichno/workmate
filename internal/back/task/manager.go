package task

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"workmate/internal/back/archive"
	fileutil "workmate/internal/back/file"

	"github.com/rs/zerolog/log"
)

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

func NewManager() *Manager {
	return NewManagerWithOptions(Options{
		DataDir:            "bin/data",
		AllowedExtensions:  []string{".pdf", ".jpeg"},
		MaxConcurrentTasks: defaultMaxConcurrent,
	})
}

func NewManagerWithOptions(opts Options) *Manager {
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

func (m *Manager) IsBusy() bool {
	return len(m.semaphore) >= cap(m.semaphore)
}

func (m *Manager) CreateTask() *Task {
	createdAt := time.Now()
	baseID := createdAt.Format("2006-01-02_15-04-05")
	newTask := &Task{
		ID:        baseID,
		Status:    StatusCreated,
		CreatedAt: createdAt,
		Files:     make([]FileRef, 0, MaxFilesPerTask),
	}

	m.updateTaskTitle(newTask)

	m.mu.Lock()
	finalID := baseID
	if _, exists := m.tasks[finalID]; exists {
		suffix := 1
		for {
			candidate := fmt.Sprintf("%s-%02d", baseID, suffix)
			if _, ok := m.tasks[candidate]; !ok {
				finalID = candidate
				break
			}
			suffix++
		}
	}
	newTask.ID = finalID
	m.tasks[finalID] = newTask
	m.mu.Unlock()

	if err := m.persistTask(newTask); err != nil {
		log.Warn().Str("task_id", newTask.ID).Err(err).Msg("persist task failed")
	}
	return newTask
}

func (m *Manager) GetTask(taskID string) (*Task, bool) {
	m.mu.RLock()
	foundTask, taskFound := m.tasks[taskID]
	m.mu.RUnlock()
	return foundTask, taskFound
}

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
	if len(currentTask.Files)+len(urls) > MaxFilesPerTask {
		m.mu.Unlock()
		return nil, ErrTooManyFiles
	}

	for _, rawURL := range urls {
		fileExtension := strings.ToLower(filepath.Ext(strings.TrimSpace(rawURL)))
		if _, allowed := m.allowedExtensions[fileExtension]; !allowed {
			m.mu.Unlock()
			return nil, NewErrExtNotAllowed(fileExtension)
		}
		currentTask.Files = append(currentTask.Files, FileRef{URL: rawURL, State: FilePending})
	}

	m.updateTaskTitle(currentTask)
	m.mu.Unlock()

	if err := m.persistTask(currentTask); err != nil {
		log.Warn().Str("task_id", currentTask.ID).Err(err).Msg("persist after add files failed")
		return nil, err
	}

	if len(currentTask.Files) == MaxFilesPerTask {
		m.semaphore <- struct{}{}
		m.workersWG.Add(1)
		go func() {
			defer m.workersWG.Done()
			m.startProcessing(taskID, true)
		}()
	}

	return currentTask, nil
}

func (m *Manager) SetBaseContext(ctx context.Context) {
	m.mu.Lock()
	m.baseCtx = ctx
	m.mu.Unlock()
}

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

func (m *Manager) UseArchiveBuilder(builder func(ctx context.Context, destZipPath string, urls []string) ([]archive.Result, error)) {
	m.mu.Lock()
	m.buildArchive = builder
	m.mu.Unlock()
}

func (m *Manager) persistTask(taskEntity *Task) error {
	if m.store != nil {
		if err := m.store.SaveTask(context.Background(), taskEntity); err != nil {
			return fmt.Errorf("store save task: %w", err)
		}
		return nil
	}

	taskDirectory := filepath.Join(m.dataDir, "tasks", taskEntity.ID)
	if err := fileutil.EnsureDir(taskDirectory); err != nil {
		return fmt.Errorf("ensure task dir: %w", err)
	}
	statusPath := filepath.Join(taskDirectory, "status.json")
	if err := fileutil.WriteJSONAtomic(statusPath, taskEntity); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	return nil
}

func (m *Manager) updateTaskTitle(t *Task) {
	timestamp := t.CreatedAt.Local().Format("2006-01-02 15:04")

	seen := make(map[string]struct{}, len(t.Files))
	hosts := make([]string, 0, len(t.Files))
	for _, f := range t.Files {
		parsed, err := url.Parse(strings.TrimSpace(f.URL))
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		host := parsed.Hostname()

		host = strings.TrimPrefix(host, "www.")
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}
	if len(hosts) == 0 {
		t.Title = timestamp
		return
	}
	t.Title = timestamp + " — " + strings.Join(hosts, ", ")
}
