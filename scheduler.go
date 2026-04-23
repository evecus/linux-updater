package main

import (
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron and maps task IDs to entry IDs.
type Scheduler struct {
	mu      sync.Mutex
	cr      *cron.Cron
	entries map[string]cron.EntryID // taskID -> cron entry ID
	store   *Store
}

func NewScheduler(store *Store) *Scheduler {
	return &Scheduler{
		cr:      cron.New(),
		entries: make(map[string]cron.EntryID),
		store:   store,
	}
}

// Start launches the cron runner and registers existing tasks.
func (s *Scheduler) Start() {
	for _, task := range s.store.ListTasks() {
		if task.Cron != "" {
			s.Register(task)
		}
	}
	s.cr.Start()
	log.Println("scheduler started")
}

// Register adds or replaces a cron job for the task.
func (s *Scheduler) Register(task *Task) {
	if task.Cron == "" {
		s.Unregister(task.ID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old entry if present
	if id, ok := s.entries[task.ID]; ok {
		s.cr.Remove(id)
	}

	taskID := task.ID
	store := s.store
	entryID, err := s.cr.AddFunc(task.Cron, func() {
		t, ok := store.GetTask(taskID)
		if !ok {
			return
		}
		log.Printf("[cron] running update for task %s (%s)", t.ID, t.Name)
		RunUpdate(t, store)
	})
	if err != nil {
		log.Printf("[scheduler] invalid cron %q for task %s: %v", task.Cron, task.ID, err)
		return
	}
	s.entries[task.ID] = entryID
	log.Printf("[scheduler] registered task %s with cron %q", task.ID, task.Cron)
}

// Unregister removes the cron job for the given task ID.
func (s *Scheduler) Unregister(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[taskID]; ok {
		s.cr.Remove(id)
		delete(s.entries, taskID)
	}
}
