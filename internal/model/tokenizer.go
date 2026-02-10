package model

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	sotToken     = "<|startoftext|>"
	eotToken     = "<|endoftext|>"
	contextLen   = 77
	endOfWordSfx = "</w>"
)

// Tokenizer implements CLIP's BPE tokenization.
type Tokenizer struct {
	encoder    map[string]int
	decoder    map[int]string
	bpeRanks   map[[2]string]int
	pat        *regexp.Regexp
	sotTokenID int
	eotTokenID int
}

// LoadTokenizer loads the tokenizer from vocab.json and merges.txt files.
func LoadTokenizer(vocabPath, mergesPath string) (*Tokenizer, error) {
	vocabData, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read vocab file: %w", err)
	}

	var encoder map[string]int
	if err := json.Unmarshal(vocabData, &encoder); err != nil {
		return nil, fmt.Errorf("cannot parse vocab file: %w", err)
	}

	mergesData, err := os.ReadFile(mergesPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read merges file: %w", err)
	}

	lines := strings.Split(string(mergesData), "\n")
	bpeRanks := make(map[[2]string]int)
	for i, line := range lines {
		// Skip header line and empty lines
		if i == 0 && strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		bpeRanks[[2]string{parts[0], parts[1]}] = len(bpeRanks)
	}

	decoder := make(map[int]string, len(encoder))
	for k, v := range encoder {
		decoder[v] = k
	}

	// CLIP tokenizer pattern
	pat := regexp.MustCompile(`<\|startoftext\|>|<\|endoftext\|>|'s|'t|'re|'ve|'m|'ll|'d|[\pL]+|[\pN]|[^\s\pL\pN]+`)

	t := &Tokenizer{
		encoder:    encoder,
		decoder:    decoder,
		bpeRanks:   bpeRanks,
		pat:        pat,
		sotTokenID: encoder[sotToken],
		eotTokenID: encoder[eotToken],
	}
	return t, nil
}

// Encode tokenizes a text string and returns token IDs padded/truncated to contextLen.
func (t *Tokenizer) Encode(text string) []int64 {
	text = strings.ToLower(strings.TrimSpace(text))

	tokens := []int{t.sotTokenID}

	for _, match := range t.pat.FindAllString(text, -1) {
		encoded := t.encodeBytes(match)
		bpeTokens := t.bpe(encoded)
		for _, bt := range bpeTokens {
			if id, ok := t.encoder[bt]; ok {
				tokens = append(tokens, id)
			}
		}
	}

	tokens = append(tokens, t.eotTokenID)

	// Pad or truncate to context length
	result := make([]int64, contextLen)
	for i := 0; i < contextLen && i < len(tokens); i++ {
		result[i] = int64(tokens[i])
	}
	return result
}

// encodeBytes converts a string to byte-level BPE tokens (CLIP uses byte-level encoding).
func (t *Tokenizer) encodeBytes(s string) string {
	var result []rune
	for i := 0; i < len(s); i++ {
		result = append(result, byteEncoder[s[i]])
	}
	return string(result)
}

// bpe applies BPE merges to a word, returning the final subword tokens.
func (t *Tokenizer) bpe(token string) []string {
	if len(token) == 0 {
		return nil
	}

	// Split into individual characters, append </w> to last
	word := make([]string, 0, utf8.RuneCountInString(token))
	runes := []rune(token)
	for i, r := range runes {
		s := string(r)
		if i == len(runes)-1 {
			s += endOfWordSfx
		}
		word = append(word, s)
	}

	if len(word) == 1 {
		return word
	}

	for {
		// Find the pair with the lowest rank
		bestPair := [2]string{}
		bestRank := -1
		for i := 0; i < len(word)-1; i++ {
			pair := [2]string{word[i], word[i+1]}
			if rank, ok := t.bpeRanks[pair]; ok {
				if bestRank == -1 || rank < bestRank {
					bestRank = rank
					bestPair = pair
				}
			}
		}

		if bestRank == -1 {
			break
		}

		// Merge the best pair
		newWord := make([]string, 0, len(word))
		i := 0
		for i < len(word) {
			if i < len(word)-1 && word[i] == bestPair[0] && word[i+1] == bestPair[1] {
				newWord = append(newWord, bestPair[0]+bestPair[1])
				i += 2
			} else {
				newWord = append(newWord, word[i])
				i++
			}
		}
		word = newWord

		if len(word) == 1 {
			break
		}
	}

	return word
}

// byteEncoder maps bytes to unicode characters (CLIP's byte-level BPE encoding table).
var byteEncoder map[byte]rune

func init() {
	byteEncoder = make(map[byte]rune)
	n := 0
	for b := 0; b < 256; b++ {
		c := rune(b)
		if isBasicByte(c) {
			byteEncoder[byte(b)] = c
		} else {
			byteEncoder[byte(b)] = rune(256 + n)
			n++
		}
	}
}

func isBasicByte(r rune) bool {
	// Characters that map to themselves in CLIP's byte encoder
	return (r >= '!' && r <= '~') || (r >= '\u00A1' && r <= '\u00AC') || (r >= '\u00AE' && r <= '\u00FF')
}

// TokenizerFromModelsDir loads the tokenizer from the standard models directory.
func TokenizerFromModelsDir() (*Tokenizer, error) {
	vocabPath, err := FilePath("vocab.json")
	if err != nil {
		return nil, err
	}
	mergesPath, err := FilePath("merges.txt")
	if err != nil {
		return nil, err
	}
	return LoadTokenizer(vocabPath, mergesPath)
}

// EncodeCategories tokenizes a batch of category labels using CLIP's prompt template.
func (t *Tokenizer) EncodeCategories(categories []string) []int64 {
	result := make([]int64, 0, len(categories)*contextLen)
	for _, cat := range categories {
		prompt := fmt.Sprintf("a photo of %s", cat)
		tokens := t.Encode(prompt)
		result = append(result, tokens...)
	}
	return result
}

// IsUnicodeLetter checks if a rune is a unicode letter (exported for testing).
func IsUnicodeLetter(r rune) bool {
	return unicode.IsLetter(r)
}
