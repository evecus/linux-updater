package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UpdateType distinguishes binary-core updates from plain file updates.
type UpdateType string

const (
	UpdateTypeCore UpdateType = "core"
	UpdateTypeFile UpdateType = "file"
)

// Task represents one auto-update job.
type Task struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	UpdateType     UpdateType `json:"update_type"`
	RepoURL        string     `json:"repo_url"`   // https://github.com/owner/repo
	CurrentVersion string     `json:"current_version"`
	FileKeyword    string     `json:"file_keyword"`
	Rename         string     `json:"rename"`      // optional
	TargetPath     string     `json:"target_path"` // absolute path
	PreCmd         string     `json:"pre_cmd"`
	PostCmd        string     `json:"post_cmd"`
	Cron           string     `json:"cron"` // empty = manual only
	LastCheck      time.Time  `json:"last_check"`
	LastUpdate     time.Time  `json:"last_update"`
	Status         string     `json:"status"` // idle / checking / updating / ok / error
	LastError      string     `json:"last_error"`
}

// Store holds all tasks in memory and persists them as JSON.
type Store struct {
	mu      sync.RWMutex
	dataDir string
	Tasks   map[string]*Task `json:"tasks"`
}

func NewStore(dataDir string) *Store {
	return &Store{
		dataDir: dataDir,
		Tasks:   make(map[string]*Task),
	}
}

func (s *Store) dbPath() string {
	return filepath.Join(s.dataDir, "tasks.json")
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.dbPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Tasks)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.Tasks, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.dbPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.dbPath())
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

func (s *Store) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.Tasks[id]
	return t, ok
}

func (s *Store) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Task, 0, len(s.Tasks))
	for _, t := range s.Tasks {
		list = append(list, t)
	}
	return list
}

func (s *Store) UpsertTask(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tasks[t.ID] = t
	return s.save()
}

func (s *Store) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Tasks, id)
	return s.save()
}

func (s *Store) UpdateTaskField(id string, fn func(*Task)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.Tasks[id]
	if !ok {
		return nil
	}
	fn(t)
	return s.save()
}

// LogPath returns the log file path for a task.
func (s *Store) LogPath(taskID string) string {
	return filepath.Join(s.dataDir, "logs", taskID+".log")
}
