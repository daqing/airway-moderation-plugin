package moderation

import (
	"os"
	"path/filepath"
	"testing"
)

func writeWordList(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "words.json")
	if err := writeWordListAt(t, path, body); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	return path
}

func writeWordListAt(t *testing.T, path, body string) error {
	t.Helper()
	return os.WriteFile(path, []byte(body), 0o644)
}

func TestDefaultWordsHoldEveryCategory(t *testing.T) {
	list := DefaultWords()
	for _, name := range []string{
		"corruption", "illegal", "political", "porn", "spam", "url", "violent", "weapons",
	} {
		if len(list.Categories[name]) == 0 {
			t.Fatalf("category %q is empty: %#v", name, list.Categories)
		}
	}
	// The floor is deliberately loose: it is here to catch a regeneration that
	// loses most of the list, not to pin a number that every edit has to chase.
	if list.Count() < 15000 {
		t.Fatalf("word count = %d, want at least 15000", list.Count())
	}
}

func TestLoadReadsTheFile(t *testing.T) {
	path := writeWordList(t, `{"categories":{"custom":["  Foo Bar  ","foo bar",""]}}`)
	list, info, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if info.Embedded || info.Path != path || info.Words != 1 || info.LoadedAt.IsZero() {
		t.Fatalf("source = %#v", info)
	}
	if len(list.Categories) != 1 || list.Categories["custom"][0] != "foo bar" {
		t.Fatalf("list = %#v", list)
	}
}

func TestLoadFallsBackToTheBuiltInList(t *testing.T) {
	for _, path := range []string{"", filepath.Join(t.TempDir(), "missing.json")} {
		list, info, err := Load(path)
		if err != nil {
			t.Fatalf("Load(%q): %v", path, err)
		}
		if !info.Embedded || list.Count() != DefaultWords().Count() {
			t.Fatalf("Load(%q) = %#v, source = %#v", path, list, info)
		}
	}
}

func TestLoadRejectsUnusableFiles(t *testing.T) {
	for name, body := range map[string]string{
		"broken json": `{"categories":`,
		"no words":    `{"categories":{"illegal":[]}}`,
		"blank words": `{"categories":{"illegal":["  "]}}`,
	} {
		if _, _, err := Load(writeWordList(t, body)); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

func TestCategoriesAreOrderedByName(t *testing.T) {
	list := WordList{Categories: map[string][]string{"porn": {"b"}, "illegal": {"a"}}}
	categories := list.Grouped()
	if len(categories) != 2 || categories[0].Name != "illegal" || categories[1].Name != "porn" {
		t.Fatalf("categories = %#v", categories)
	}
}

func TestWordsFileFollowsTheEnvironment(t *testing.T) {
	if got := WordsFile(); got != DefaultWordsFile {
		t.Fatalf("WordsFile() = %q", got)
	}
	t.Setenv("MODERATION_WORDS_FILE", "/tmp/words.json")
	if got := WordsFile(); got != "/tmp/words.json" {
		t.Fatalf("WordsFile() = %q", got)
	}
	t.Setenv("MODERATION_WORDS_FILE", "  ")
	if got := WordsFile(); got != DefaultWordsFile {
		t.Fatalf("WordsFile() = %q", got)
	}
}
