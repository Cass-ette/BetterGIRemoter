package store

import (
	"sync"
	"time"
)

type Task struct {
	ID        string
	Type      string // "start", "stop", "status"
	Status    string // "pending", "running", "completed", "failed"
	CreatedAt time.Time
	UpdatedAt time.Time
	Result    string
}

type Store struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func New() *Store {
	return &Store{
		tasks: make(map[string]*Task),
	}
}

func (s *Store) Add(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

func (s *Store) Get(id string) *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tasks[id]
}

func (s *Store) Update(id, status, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[id]; ok {
		t.Status = status
		t.Result = result
		t.UpdatedAt = time.Now()
	}
}

func (s *Store) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

func (s *Store) Recent(n int) []*Task {
	all := s.List()

	// 按创建时间倒序排序
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[i].CreatedAt.Before(all[j].CreatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if len(all) > n {
		return all[:n]
	}
	return all
}
