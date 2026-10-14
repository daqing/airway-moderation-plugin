// Command lexicon builds the moderation word list from the upstream
// Sensitive-lexicon lists.
//
// The upstream project is a keyword collection for search engines and content
// sites. It covers a lot of ground, and it mixes ordinary vocabulary into it —
// search terms such as 兼职 or 自拍, news and legal phrasing such as 有期徒刑,
// generic ASCII tokens such as www or 64 — which has no place in a chat filter
// whose hits go to a human review queue. This command keeps the high-signal
// categories, drops the rest, and filters what is left twice: mechanically
// (see asciiFloor) and against the curated list in exclude.txt.
//
// The result is the plugin's built-in list and the starter list that
// plugin:install drops into a host project; both files are generated from the
// same run, so they never drift.
//
// Usage:
//
//	go run ./tools/lexicon -src /path/to/Sensitive-lexicon \
//	  -out install/lib/moderation/default_words.json
//	go run ./tools/lexicon -src /path/to/Sensitive-lexicon \
//	  -out install/host/config/moderation_words.json
//
// -check compares the file at -out with what the sources generate, instead of
// writing it.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const upstream = "https://github.com/konsheng/Sensitive-lexicon"

// sources lists the upstream files this command reads, and the category each
// one screens under. Every other file in the source tree is ignored: they are
// either broad political or livelihood vocabulary, or keyword lists built for
// a tokenizer rather than for chat. README.md names them and says why.
var sources = []struct {
	file     string
	category string
}{
	{"色情词库.txt", "porn"},
	{"色情类型.txt", "porn"},
	{"暴恐词库.txt", "violent"},
	{"涉枪涉爆.txt", "weapons"},
	{"广告类型.txt", "spam"},
	{"非法网址.txt", "url"},
	{"贪腐词库.txt", "corruption"},
	{"其他词库.txt", "illegal"},
}

// ignored documents the upstream files left out on purpose, so a file that
// disappears upstream shows up in the report as read rather than as ignored.
var ignored = map[string]string{
	"反动词库.txt":       "broad political vocabulary",
	"政治类型.txt":       "broad political vocabulary",
	"民生词库.txt":       "everyday livelihood vocabulary",
	"GFW补充词库.txt":    "network censorship keywords, not chat content",
	"网易前端过滤敏感词库.txt": "tokenizer keywords, mostly ordinary words",
	"零时-Tencent.txt": "tokenizer keywords, mostly ordinary words",
	"COVID-19词库.txt": "pandemic vocabulary",
	"新思想启蒙.txt":      "meme phrases",
	"补充词库.txt":       "segmented sentence fragments",
}

// A word is dropped when it is shorter than shortestWord, or when it is pure
// ASCII and shorter than the floor for its category. Upstream ASCII entries
// are mostly junk ("www", "64", "189", "ma", "CCTV") that a substring matcher
// would hit inside ordinary chat; a domain needs more characters than a word
// before a hit means anything.
const (
	shortestWord  = 2
	asciiFloor    = 6
	asciiFloorURL = 8
)

// excludeFile is the curated exclusion list: ordinary words that survive the
// mechanical filters but have no business holding a message back.
//
//go:embed exclude.txt
var excludeFile string

// starterFile holds the hand-picked words the list shipped with before the
// upstream lists were merged in. They keep their category, and they are where
// an operator adds a word permanently.
//
//go:embed starter_words.json
var starterFile []byte

type list struct {
	Categories map[string][]string `json:"categories"`
}

func main() {
	src := flag.String("src", "", "directory holding the upstream word list files")
	out := flag.String("out", "", "path of the JSON word list to write")
	check := flag.Bool("check", false, "compare -out with the generated list instead of writing it")
	flag.Parse()
	if *src == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}

	categories, report, err := build(*src)
	if err != nil {
		fail(err)
	}
	data, err := render(categories)
	if err != nil {
		fail(err)
	}

	fmt.Print(report)

	if *check {
		current, err := os.ReadFile(*out)
		if err != nil {
			fail(err)
		}
		if !bytes.Equal(current, data) {
			fail(fmt.Errorf("%s is not what these sources generate; run again without -check to write it", *out))
		}
		fmt.Printf("%s is up to date\n", *out)
		return
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s\n", *out)
}

func build(src string) (map[string][]string, string, error) {
	excluded, err := exclusion()
	if err != nil {
		return nil, "", err
	}
	starter, err := starterWords()
	if err != nil {
		return nil, "", err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", src, err)
	}
	present := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			present[entry.Name()] = true
		}
	}

	// The starter words claim their category first: they are what a reviewer
	// has been seeing, and they must not change category underneath them.
	claims := map[string]map[string]bool{}
	claim := func(category, word string) {
		if claims[category] == nil {
			claims[category] = map[string]bool{}
		}
		claims[category][word] = true
	}
	starterCount := 0
	for category, words := range starter {
		for _, word := range words {
			claim(category, word)
			starterCount++
		}
	}

	var report strings.Builder
	fmt.Fprintf(&report, "%s\n\n", upstream)
	fmt.Fprintf(&report, "read from %s:\n", src)

	reasons := map[string]int{}
	hit := map[string]int{}
	kept := map[string]int{}
	for _, source := range sources {
		if !present[source.file] {
			return nil, "", fmt.Errorf("%s is missing from %s", source.file, src)
		}
		data, err := os.ReadFile(filepath.Join(src, source.file))
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", source.file, err)
		}
		words := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			word := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "\ufeff")))
			switch {
			case word == "":
				continue
			case utf8.RuneCountInString(word) < shortestWord:
				reasons["shorter than "+strconv.Itoa(shortestWord)+" characters"]++
				continue
			case strings.ContainsAny(word, " \t|"):
				reasons["carries a space or a pipe"]++
				continue
			case isASCII(word) && utf8.RuneCountInString(word) < asciiFloorFor(source.category):
				reasons["ASCII below the category floor"]++
				continue
			}
			if _, found := excluded[word]; found {
				hit[word]++
				reasons["listed in exclude.txt"]++
				continue
			}
			words[word] = true
		}
		for word := range words {
			claim(source.category, word)
		}
		kept[source.category] += len(words)
		fmt.Fprintf(&report, "  %-24s -> %-11s %6d words\n", source.file, source.category, len(words))
	}

	names := make([]string, 0, len(present))
	for name := range present {
		if _, read := sourceFor(name); !read {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) > 0 {
		fmt.Fprintf(&report, "\nignored (on purpose):\n")
		for _, name := range names {
			why := ignored[name]
			if why == "" {
				why = "not part of this list"
			}
			fmt.Fprintf(&report, "  %-24s %s\n", name, why)
		}
	}

	// A word claimed by two categories stays with the first one alphabetically,
	// which is the rule the service itself applies on load: writing it that way
	// here keeps the file and the running list in agreement.
	resolved := map[string][]string{}
	seen := map[string]bool{}
	categories := make([]string, 0, len(claims))
	for category := range claims {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		words := make([]string, 0, len(claims[category]))
		for word := range claims[category] {
			if seen[word] {
				continue
			}
			seen[word] = true
			words = append(words, word)
		}
		sort.Strings(words)
		resolved[category] = words
	}

	fmt.Fprintf(&report, "\nkept:\n")
	total := 0
	for _, category := range categories {
		fmt.Fprintf(&report, "  %-11s %6d words\n", category, len(resolved[category]))
		total += len(resolved[category])
	}
	fmt.Fprintf(&report, "  %-11s %6d words (%d of them from starter_words.json)\n", "total", total, starterCount)

	fmt.Fprintf(&report, "\ndropped:\n")
	for _, reason := range sortedKeys(reasons) {
		fmt.Fprintf(&report, "  %-32s %6d\n", reason, reasons[reason])
	}

	unused := make([]string, 0)
	for word := range excluded {
		if hit[word] == 0 {
			unused = append(unused, word)
		}
	}
	if len(unused) > 0 {
		sort.Strings(unused)
		fmt.Fprintf(&report, "\nexclude.txt entries that matched nothing — upstream may have renamed them:\n  %s\n",
			strings.Join(unused, ", "))
	}

	if commit := head(src); commit != "" {
		fmt.Fprintf(&report, "\nupstream commit: %s\n", commit)
	} else {
		fmt.Fprintf(&report, "\nupstream commit: unknown — record it by hand\n")
	}

	return resolved, report.String(), nil
}

func render(categories map[string][]string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(list{Categories: categories}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func exclusion() (map[string]bool, error) {
	words := map[string]bool{}
	for _, line := range strings.Split(excludeFile, "\n") {
		word := strings.ToLower(strings.TrimSpace(line))
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words[word] = true
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("exclude.txt holds no words")
	}
	return words, nil
}

func starterWords() (map[string][]string, error) {
	var parsed list
	if err := json.Unmarshal(starterFile, &parsed); err != nil {
		return nil, fmt.Errorf("starter_words.json: %w", err)
	}
	if len(parsed.Categories) == 0 {
		return nil, fmt.Errorf("starter_words.json holds no words")
	}
	return parsed.Categories, nil
}

func sourceFor(name string) (string, bool) {
	for _, source := range sources {
		if source.file == name {
			return source.category, true
		}
	}
	return "", false
}

func asciiFloorFor(category string) int {
	if category == "url" {
		return asciiFloorURL
	}
	return asciiFloor
}

func isASCII(word string) bool {
	for _, r := range word {
		if r > 127 {
			return false
		}
	}
	return true
}

func head(src string) string {
	out, err := exec.Command("git", "-C", src, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sortedKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lexicon:", err)
	os.Exit(1)
}
