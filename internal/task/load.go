package task

import (
	"context"
	"fmt"
)

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
