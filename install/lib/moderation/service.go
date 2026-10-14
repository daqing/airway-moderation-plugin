package moderation

import (
	"sync"
	"sync/atomic"
)

// Result is the verdict for one piece of content.
type Result struct {
	Flagged    bool     `json:"flagged"`
	Categories []string `json:"categories"`
	Words      []string `json:"words"`
}

// Service screens content against the word list currently loaded. A reload
// swaps the matcher atomically, so screening requests never wait on an
// operator editing the list.
type Service struct {
	path    string
	matcher atomic.Pointer[Matcher]
	mu      sync.Mutex
	list    WordList
	info    SourceInfo
}

// New loads the word list at path and returns a service screening against it.
func New(path string) (*Service, error) {
	service := &Service{path: path}
	if err := service.Reload(); err != nil {
		return nil, err
	}
	return service, nil
}

// Reload re-reads the word list. A list that cannot be read or parsed leaves
// the one in use untouched: a bad edit must not disarm screening.
func (s *Service) Reload() error {
	list, info, err := Load(s.path)
	if err != nil {
		return err
	}
	s.matcher.Store(NewMatcher(list))
	s.mu.Lock()
	s.list, s.info = list, info
	s.mu.Unlock()
	return nil
}

// Screen reports the words content hits and the categories they belong to.
func (s *Service) Screen(content string) Result {
	result := Result{Categories: []string{}, Words: []string{}}
	matcher := s.matcher.Load()
	if matcher == nil {
		return result
	}
	for _, hit := range matcher.Match(content) {
		result.Flagged = true
		result.Words = append(result.Words, hit.Word)
		if !contains(result.Categories, hit.Category) {
			result.Categories = append(result.Categories, hit.Category)
		}
	}
	return result
}

// Path reports the word list path this service reads.
func (s *Service) Path() string { return s.path }

// Info describes the list currently loaded.
func (s *Service) Info() SourceInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

// Categories returns the list currently loaded, grouped by category.
func (s *Service) Categories() []Category {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list.Grouped()
}

func contains(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

var current atomic.Pointer[Service]

// Setup loads the word list at path and installs it as the service every
// handler screens through. It reports the error instead of falling back, so a
// host can refuse to start on an unusable word list.
func Setup(path string) error {
	service, err := New(path)
	if err != nil {
		return err
	}
	current.Store(service)
	return nil
}

// Current returns the process-wide service. It loads the built-in list on
// first use, so a handler works even before Setup runs or when the configured
// word list is unusable.
func Current() *Service {
	if service := current.Load(); service != nil {
		return service
	}
	service, err := New(WordsFile())
	if err != nil {
		service, _ = New("")
	}
	if current.CompareAndSwap(nil, service) {
		return service
	}
	return current.Load()
}
