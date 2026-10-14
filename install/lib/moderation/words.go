// Package moderation screens text against an operator-maintained word list.
// The list is a JSON file the moderation service owns, so operations can
// change what is screened without a redeploy; the matcher reports the words
// that hit and the category each belongs to, so a reviewer sees why a message
// was held back.
package moderation

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"
)

// DefaultWordsFile is the word list path used when MODERATION_WORDS_FILE is
// unset. plugin:install drops the starter list at this path in a host
// project.
const DefaultWordsFile = "config/moderation_words.json"

//go:embed default_words.json
var defaultWordsFS embed.FS

// WordList is the word list file: categories of words, so one list can be
// extended without touching the matcher.
type WordList struct {
	Categories map[string][]string `json:"categories"`
}

// Category is one named group of words, as reported by the words endpoint.
type Category struct {
	Name  string   `json:"name"`
	Words []string `json:"words"`
}

// Entry is a single word with the category it belongs to.
type Entry struct {
	Word     string
	Category string
}

// SourceInfo records where the list in use came from, so an operator can tell
// a loaded file apart from the built-in fallback.
type SourceInfo struct {
	Path     string    `json:"path"`
	Embedded bool      `json:"embedded"`
	Words    int       `json:"words"`
	LoadedAt time.Time `json:"loaded_at"`
}

// WordsFile reports the configured word list path.
func WordsFile() string {
	if value := strings.TrimSpace(os.Getenv("MODERATION_WORDS_FILE")); value != "" {
		return value
	}
	return DefaultWordsFile
}

// DefaultWords returns the built-in starter list.
func DefaultWords() WordList {
	data, err := defaultWordsFS.ReadFile("default_words.json")
	if err != nil {
		panic("moderation: embedded word list is missing: " + err.Error())
	}
	var list WordList
	if err := json.Unmarshal(data, &list); err != nil {
		panic("moderation: embedded word list is invalid: " + err.Error())
	}
	return list.normalized()
}

// Load reads the word list at path. An empty path, or a file that is not
// there, yields the built-in list — the service still answers on a checkout
// that never ran plugin:install. A file that is present but unusable is an
// error, never a silent fallback: screening must not run on a list nobody
// asked for.
func Load(path string) (WordList, SourceInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return fallback(SourceInfo{})
	}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fallback(SourceInfo{Path: path})
	case err != nil:
		return WordList{}, SourceInfo{}, fmt.Errorf("moderation: read word list %s: %w", path, err)
	}
	var list WordList
	if err := json.Unmarshal(data, &list); err != nil {
		return WordList{}, SourceInfo{}, fmt.Errorf("moderation: parse word list %s: %w", path, err)
	}
	normalized := list.normalized()
	if normalized.Count() == 0 {
		return WordList{}, SourceInfo{}, fmt.Errorf("moderation: word list %s holds no words", path)
	}
	return normalized, SourceInfo{Path: path, Words: normalized.Count(), LoadedAt: time.Now().UTC()}, nil
}

func fallback(info SourceInfo) (WordList, SourceInfo, error) {
	list := DefaultWords()
	info.Embedded = true
	info.Words = list.Count()
	info.LoadedAt = time.Now().UTC()
	return list, info, nil
}

// Count reports how many words the list holds.
func (l WordList) Count() int {
	total := 0
	for _, words := range l.Categories {
		total += len(words)
	}
	return total
}

// Grouped returns the list as categories, ordered by name so the same list
// always renders the same way.
func (l WordList) Grouped() []Category {
	names := make([]string, 0, len(l.Categories))
	for name := range l.Categories {
		names = append(names, name)
	}
	sort.Strings(names)
	categories := make([]Category, 0, len(names))
	for _, name := range names {
		categories = append(categories, Category{Name: name, Words: l.Categories[name]})
	}
	return categories
}

// entries returns every word with its category, ordered by word, so the
// matcher is built the same way for the same list.
func (l WordList) entries() []Entry {
	var entries []Entry
	for _, category := range l.Grouped() {
		for _, word := range category.Words {
			entries = append(entries, Entry{Word: word, Category: category.Name})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Word != entries[j].Word {
			return entries[i].Word < entries[j].Word
		}
		return entries[i].Category < entries[j].Category
	})
	return entries
}

// normalized lower-cases and trims every word, drops blanks, duplicate words
// and empty categories, and orders words within a category. A word listed
// under two categories stays with the first one alphabetically, so a hit is
// reported against exactly one category.
func (l WordList) normalized() WordList {
	normalized := WordList{Categories: map[string][]string{}}
	seen := map[string]bool{}
	for _, category := range l.Grouped() {
		words := make([]string, 0, len(category.Words))
		for _, word := range category.Words {
			trimmed := strings.ToLower(strings.TrimSpace(word))
			if trimmed == "" || seen[trimmed] {
				continue
			}
			seen[trimmed] = true
			words = append(words, trimmed)
		}
		if len(words) == 0 {
			continue
		}
		sort.Strings(words)
		normalized.Categories[category.Name] = words
	}
	return normalized
}
