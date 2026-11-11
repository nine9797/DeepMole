package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"deepmole/internal/strategy"
)

type Entry struct {
	Name   string        `json:"name"`
	Plan   strategy.Plan `json:"plan"`
	Saved  time.Time     `json:"saved"`
	LastOK time.Time     `json:"last_ok"`
}

type State struct {
	Networks map[string]Entry `json:"networks"`

	path string
	mu   sync.Mutex
}

func Load(path string) *State {
	s := &State{Networks: map[string]Entry{}, path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, s)
		if s.Networks == nil {
			s.Networks = map[string]Entry{}
		}
	}
	return s
}

func (s *State) Get(id string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.Networks[id]
	return e, ok
}

func (s *State) Put(id string, e Entry) {
	s.mu.Lock()
	s.Networks[id] = e
	s.mu.Unlock()
	s.save()
}

func (s *State) Touch(id string) {
	s.mu.Lock()
	if e, ok := s.Networks[id]; ok {
		e.LastOK = time.Now()
		s.Networks[id] = e
	}
	s.mu.Unlock()
	s.save()
}

func (s *State) save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, b, 0o644)
}
