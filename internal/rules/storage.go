package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Storage struct {
	mu   sync.Mutex
	path string
}

func NewStorage(path string) (*Storage, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	return &Storage{path: path}, nil
}

func (s *Storage) Load() ([]Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, err
	}

	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

func (s *Storage) Save(rules []Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}
