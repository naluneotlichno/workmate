package task

import (
	"context"
	"path/filepath"

	"workmate/internal/archive"
	fileutil "workmate/internal/file"

	"github.com/rs/zerolog/log"
)

// startProcessing attempts to start processing. If slotAlreadyAcquired is false,
// the function acquires a slot and releases it on return.
func (m *Manager) startProcessing(taskID string, slotAlreadyAcquired bool) {
	if !slotAlreadyAcquired {
		m.semaphore <- struct{}{}
	}
	defer func() { <-m.semaphore }()

	m.mu.Lock()
	taskToProcess, taskFound := m.tasks[taskID]
	if !taskFound {
		m.mu.Unlock()
		return
	}
	taskToProcess.Status = StatusInProgress
	m.mu.Unlock()
	if err := m.persistTask(taskToProcess); err != nil {
		log.Warn().Str("task_id", taskToProcess.ID).Err(err).Msg("persist in_progress failed")
	}

	// Prepare paths
	taskDirectory := filepath.Join(m.dataDir, "tasks", taskToProcess.ID)
	if err := fileutil.EnsureDir(taskDirectory); err != nil {
		m.failTask(taskToProcess, "failed to create task dir: "+err.Error())
		return
	}
	destinationZipPath := filepath.Join(taskDirectory, "archive.zip")

	// Collect URLs
	urlsToProcess := make([]string, 0, len(taskToProcess.Files))
	for _, fileRef := range taskToProcess.Files {
		urlsToProcess = append(urlsToProcess, fileRef.URL)
	}

	builder := m.buildArchive
	if builder == nil {
		builder = archive.BuildArchive
	}
	// allow cancellation on graceful shutdown by using background here
	processingContext := m.baseCtx
	if processingContext == nil {
		processingContext = context.Background()
	}
	archiveResults, err := builder(processingContext, destinationZipPath, urlsToProcess)
	if err != nil {
		m.failTask(taskToProcess, err.Error())
		return
	}

	// Update file states
	m.mu.Lock()
	for i := range taskToProcess.Files {
		archiveResult := archiveResults[i]
		taskToProcess.Files[i].Filename = archiveResult.Filename
		if archiveResult.Err == "" {
			taskToProcess.Files[i].State = FileOK
		} else {
			taskToProcess.Files[i].State = FileFailed
			taskToProcess.Files[i].Error = archiveResult.Err
		}
	}

	// Determine overall status
	anyFilesOK := false
	for _, fileResult := range taskToProcess.Files {
		if fileResult.State == FileOK {
			anyFilesOK = true
			break
		}
	}
	if anyFilesOK {
		taskToProcess.Status = StatusReady
		taskToProcess.ArchivePath = destinationZipPath
	} else {
		taskToProcess.Status = StatusFailed
	}
	m.mu.Unlock()
	if err := m.persistTask(taskToProcess); err != nil {
		log.Warn().Str("task_id", taskToProcess.ID).Err(err).Msg("persist final state failed")
	}
}

func (m *Manager) failTask(taskEntity *Task, msg string) {
	m.mu.Lock()
	taskEntity.Status = StatusFailed
	// propagate error to pending files
	for i := range taskEntity.Files {
		if taskEntity.Files[i].State == FilePending {
			taskEntity.Files[i].State = FileFailed
			taskEntity.Files[i].Error = msg
		}
	}
	m.mu.Unlock()
	if err := m.persistTask(taskEntity); err != nil {
		log.Warn().Str("task_id", taskEntity.ID).Err(err).Msg("persist failed state failed")
	}
}
