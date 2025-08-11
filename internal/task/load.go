package task

import (
	"context"
	"fmt"
)

// LoadFromDisk scans data/tasks and loads tasks into memory.
// If a task has StatusInProgress (from a previous run), it is marked as failed.
func (m *Manager) LoadFromDisk() error {
	if m.store == nil {
		return nil
	}
	loadedTasks, err := m.store.LoadTasks(context.Background())
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	for _, taskEntity := range loadedTasks {
		if taskEntity.Status == StatusInProgress {
			taskEntity.Status = StatusFailed
			_ = m.persistTask(taskEntity)
		}
		m.mu.Lock()
		m.tasks[taskEntity.ID] = taskEntity
		m.mu.Unlock()
	}
	return nil
}
