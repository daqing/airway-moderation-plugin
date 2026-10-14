package moderation

import (
	"sort"
	"strings"
)

// Hit is one word the matcher found in the content.
type Hit struct {
	Word     string `json:"word"`
	Category string `json:"category"`
}

type node struct {
	next map[rune]int32
	fail int32
	hits []int32
}

// Matcher is an Aho-Corasick automaton over a word list: one pass over the
// content finds every word, however many there are.
type Matcher struct {
	nodes   []node
	entries []Entry
	runes   []int32
}

// NewMatcher builds a matcher for the list. The list is normalized on the way
// in — lower-cased, trimmed, de-duplicated — so the same automaton is built
// for the same list however the caller assembled it.
func NewMatcher(list WordList) *Matcher {
	matcher := &Matcher{nodes: []node{{}}}
	for _, entry := range list.normalized().entries() {
		matcher.insert(entry)
	}
	matcher.build()
	return matcher
}

func (m *Matcher) insert(entry Entry) {
	state := int32(0)
	for _, r := range entry.Word {
		next, ok := m.nodes[state].next[r]
		if !ok {
			m.nodes = append(m.nodes, node{})
			next = int32(len(m.nodes) - 1)
			if m.nodes[state].next == nil {
				m.nodes[state].next = map[rune]int32{}
			}
			m.nodes[state].next[r] = next
		}
		state = next
	}
	index := int32(len(m.entries))
	m.entries = append(m.entries, entry)
	m.nodes[state].hits = append(m.nodes[state].hits, index)
}

// build fills in the failure links breadth-first: the failure of a node is
// the longest proper suffix of the node's path that is also a prefix of some
// word. Each node then inherits its failure's hits, so a word that is a
// suffix of another is reported without a second pass over the content.
func (m *Matcher) build() {
	m.runes = make([]int32, len(m.entries))
	for index, entry := range m.entries {
		m.runes[index] = int32(len([]rune(entry.Word)))
	}
	queue := make([]int32, 0, len(m.nodes))
	for _, next := range m.nodes[0].next {
		queue = append(queue, next)
	}
	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		if fail := m.nodes[state].fail; fail != 0 {
			m.nodes[state].hits = append(m.nodes[state].hits, m.nodes[fail].hits...)
		}
		for r, next := range m.nodes[state].next {
			m.nodes[next].fail = m.step(m.nodes[state].fail, r)
			queue = append(queue, next)
		}
	}
}

// step follows the automaton from state on r, walking failure links until the
// transition exists, and reports the root when it does not.
func (m *Matcher) step(state int32, r rune) int32 {
	for state != 0 {
		if next, ok := m.nodes[state].next[r]; ok {
			return next
		}
		state = m.nodes[state].fail
	}
	return m.nodes[0].next[r]
}

// Match returns every word the content hits, de-duplicated and in the order
// the hits end — longest word first when several end together. Matching is
// case-insensitive and ignores surrounding space, mirroring the clients that
// pre-screen a message before sending it.
func (m *Matcher) Match(content string) []Hit {
	normalized := strings.ToLower(strings.TrimSpace(content))
	if normalized == "" || len(m.entries) == 0 {
		return nil
	}

	ends := make(map[int32]int32, len(m.entries))
	order := make([]int32, 0, len(m.entries))
	state := int32(0)
	var offset int32
	for _, r := range normalized {
		state = m.step(state, r)
		for _, index := range m.nodes[state].hits {
			if _, seen := ends[index]; !seen {
				ends[index] = offset
				order = append(order, index)
			}
		}
		offset++
	}
	if len(order) == 0 {
		return nil
	}

	m.sortByPosition(order, ends)
	hits := make([]Hit, 0, len(order))
	for _, index := range order {
		hits = append(hits, Hit{Word: m.entries[index].Word, Category: m.entries[index].Category})
	}
	return hits
}

// sortByPosition orders the hits in place by where they end, longest word
// first, then alphabetically — a total order, so the same content always
// reports the same sequence.
func (m *Matcher) sortByPosition(order []int32, ends map[int32]int32) {
	sort.Slice(order, func(i, j int) bool {
		left, right := order[i], order[j]
		if ends[left] != ends[right] {
			return ends[left] < ends[right]
		}
		if m.runes[left] != m.runes[right] {
			return m.runes[left] > m.runes[right]
		}
		return m.entries[left].Word < m.entries[right].Word
	})
}
