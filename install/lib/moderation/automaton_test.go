package moderation

import "testing"

func hitsToWords(hits []Hit) []string {
	words := make([]string, 0, len(hits))
	for _, hit := range hits {
		words = append(words, hit.Word)
	}
	return words
}

func TestMatchReportsEveryCategory(t *testing.T) {
	matcher := NewMatcher(DefaultWords())
	for content, want := range map[string]string{
		"台独":   "political",
		"裸聊":   "porn",
		"买卖枪支": "illegal",
	} {
		hits := matcher.Match(content)
		if len(hits) != 1 || hits[0].Word != content || hits[0].Category != want {
			t.Fatalf("Match(%q) = %#v, want one %s hit", content, hits, want)
		}
	}
}

func TestMatchFindsWordsInsideSurroundingText(t *testing.T) {
	hits := NewMatcher(DefaultWords()).Match("有人在群里发广告，还提到网络赌博，大家小心")
	if len(hits) != 1 || hits[0].Word != "网络赌博" {
		t.Fatalf("hits = %#v", hits)
	}
}

func TestMatchIgnoresCaseAndSurroundingSpace(t *testing.T) {
	matcher := NewMatcher(WordList{Categories: map[string][]string{"illegal": {"Money Laundering"}}})
	hits := matcher.Match("  ABC money LAUNDERING DEF  ")
	if len(hits) != 1 || hits[0].Word != "money laundering" {
		t.Fatalf("hits = %#v", hits)
	}
}

func TestMatchAllowsOrdinaryContent(t *testing.T) {
	matcher := NewMatcher(DefaultWords())
	for _, content := range []string{
		"",
		"   ",
		"新版本什么时候发？",
		"这周五，争取。",
	} {
		if hits := matcher.Match(content); len(hits) != 0 {
			t.Fatalf("Match(%q) = %#v", content, hits)
		}
	}
}

// A short word and a long word ending at the same place are both reported,
// longest first, and a word repeated in the content is reported once.
func TestMatchReportsOverlappingAndRepeatedWords(t *testing.T) {
	matcher := NewMatcher(WordList{Categories: map[string][]string{"illegal": {"赌博", "网络赌博"}}})
	hits := matcher.Match("网络赌博，还是网络赌博")
	words := hitsToWords(hits)
	if len(words) != 2 || words[0] != "网络赌博" || words[1] != "赌博" {
		t.Fatalf("hits = %#v", hits)
	}
}

// A word that is a suffix of another word is found without a second pass:
// the failure link carries the shorter word's hit into the longer one's node.
func TestMatchFindsSuffixWordInCompoundText(t *testing.T) {
	matcher := NewMatcher(WordList{Categories: map[string][]string{"illegal": {"冰毒", "毒"}}})
	words := hitsToWords(matcher.Match("他提到了冰毒"))
	if len(words) != 2 || words[0] != "冰毒" || words[1] != "毒" {
		t.Fatalf("words = %#v", words)
	}
}

func TestMatcherBuildsFromEmptyList(t *testing.T) {
	if hits := NewMatcher(WordList{}).Match("网络赌博"); len(hits) != 0 {
		t.Fatalf("hits = %#v", hits)
	}
}

// A word listed under two categories is reported against the first one
// alphabetically, so a hit never names two categories at once.
func TestMatcherAssignsWordToFirstCategory(t *testing.T) {
	matcher := NewMatcher(WordList{Categories: map[string][]string{
		"porn":    {"赌博"},
		"illegal": {"赌博"},
	}})
	hits := matcher.Match("赌博")
	if len(hits) != 1 || hits[0].Category != "illegal" {
		t.Fatalf("hits = %#v", hits)
	}
}
