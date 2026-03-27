package tdt

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

// stopWords contains stemmed forms of common low-discriminative terms
// that appear in many tool descriptions and inflate BM25 match counts.
var stopWords = map[string]bool{
	"search": true, // search, searching, searches
	"get":    true, // get, gets, getting
	"set":    true, // set, sets, setting
	"list":   true, // list, lists, listing
	"data":   true, // data
	"info":   true, // info, information
	"util":   true, // util, utility, utilities
	"the":    true,
	"and":    true,
	"for":    true,
	"with":   true,
	"from":   true,
	"that":   true,
	"this":   true,
	"all":    true,
}

// tokenize splits text into lowercase tokens, splitting on whitespace,
// punctuation, underscores, and camelCase boundaries.
// Tokens shorter than 2 characters and stop words are dropped.
func tokenize(text string) []string {
	if text == "" {
		return nil
	}
	// Split camelCase on the original text first, then process.
	expanded := splitCamelCase(text)
	// Replace underscores with spaces.
	expanded = strings.ReplaceAll(expanded, "_", " ")
	// Split on non-letter, non-digit characters.
	parts := strings.FieldsFunc(expanded, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	// Lowercase, stem, filter short tokens, and remove stop words.
	var tokens []string
	for _, p := range parts {
		p = strings.ToLower(p)
		if len(p) < 2 {
			continue
		}
		stemmed, err := snowball.Stem(p, "english", false)
		if err == nil && len(stemmed) >= 2 {
			p = stemmed
		}
		if stopWords[p] {
			continue
		}
		tokens = append(tokens, p)
	}
	return tokens
}

// splitCamelCase inserts spaces at camelCase boundaries.
// Handles: "getWeather" -> "get Weather", "HTTPServer" -> "HTTP Server".
func splitCamelCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			// Case 1: uppercase after lowercase (e.g., "getWeather" at 'W')
			if unicode.IsLower(prev) {
				b.WriteRune(' ')
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				// Case 2: uppercase followed by lowercase, after uppercase sequence
				// (e.g., "HTTPServer" at 'S' - prev is 'P', next is 'e')
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// bm25Doc represents a tokenized document in the corpus.
type bm25Doc struct {
	toolName   string
	serverName string
	tokens     []string
	termFreq   map[string]int
}

// bm25Corpus holds the BM25 index for all tools.
type bm25Corpus struct {
	docs   []bm25Doc
	idf    map[string]float64
	avgLen float64
	k1     float64
	b      float64
}

// toolScore holds a BM25 score for a tool.
type toolScore struct {
	toolName   string
	serverName string
	score      float64
}

// buildCompositeText creates a searchable text blob from a server and tool.
// Tags are sorted by key for deterministic output.
func buildCompositeText(s ServerMetadata, t ToolInfo) string {
	var b strings.Builder
	b.WriteString(t.Name)
	b.WriteString(" ")
	b.WriteString(t.Description)
	b.WriteString(" ")
	b.WriteString(s.Category)
	if s.Hint != "" {
		b.WriteString(" ")
		b.WriteString(s.Hint)
	}

	// Sort tag keys for deterministic iteration order.
	keys := make([]string, 0, len(s.Tags))
	for k := range s.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(" ")
		b.WriteString(k)
		b.WriteString(" ")
		b.WriteString(s.Tags[k])
	}
	return b.String()
}

// newBM25Corpus builds a BM25 index from server metadata.
func newBM25Corpus(servers []ServerMetadata) *bm25Corpus {
	var docs []bm25Doc
	for _, s := range servers {
		for _, t := range s.Tools {
			text := buildCompositeText(s, t)
			tokens := tokenize(text)
			tf := make(map[string]int)
			for _, tok := range tokens {
				tf[tok]++
			}
			docs = append(docs, bm25Doc{
				toolName:   t.Name,
				serverName: s.ServerName,
				tokens:     tokens,
				termFreq:   tf,
			})
		}
	}

	n := float64(len(docs))
	docFreq := make(map[string]int)
	totalLen := 0
	for _, d := range docs {
		totalLen += len(d.tokens)
		seen := make(map[string]bool)
		for _, tok := range d.tokens {
			if !seen[tok] {
				docFreq[tok]++
				seen[tok] = true
			}
		}
	}

	idf := make(map[string]float64)
	for term, df := range docFreq {
		idf[term] = math.Log((n-float64(df)+0.5)/(float64(df)+0.5) + 1)
	}

	avgLen := 0.0
	if len(docs) > 0 {
		avgLen = float64(totalLen) / n
	}

	return &bm25Corpus{
		docs:   docs,
		idf:    idf,
		avgLen: avgLen,
		k1:     1.2,
		b:      0.75,
	}
}

// score computes BM25 scores for all documents against the query.
func (c *bm25Corpus) score(query string) []toolScore {
	queryTokens := tokenize(query)
	scores := make([]toolScore, len(c.docs))

	for i, doc := range c.docs {
		s := 0.0
		docLen := float64(len(doc.tokens))
		for _, qt := range queryTokens {
			tf := float64(doc.termFreq[qt])
			if tf == 0 {
				continue
			}
			idf := c.idf[qt]
			norm := 1 - c.b + c.b*(docLen/c.avgLen)
			s += idf * (tf * (c.k1 + 1)) / (tf + c.k1*norm)
		}
		scores[i] = toolScore{
			toolName:   doc.toolName,
			serverName: doc.serverName,
			score:      s,
		}
	}
	return scores
}
