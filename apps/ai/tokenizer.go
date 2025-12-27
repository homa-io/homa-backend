package ai

import (
	"regexp"
	"strings"
	"unicode"
)

// Tokenizer provides text tokenization and chunking utilities for RAG
type Tokenizer struct {
	maxChunkTokens  int
	overlapTokens   int
	sentencePattern *regexp.Regexp
}

// NewTokenizer creates a new tokenizer with default settings
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		maxChunkTokens:  500,  // ~2000 characters
		overlapTokens:   50,   // ~200 characters overlap
		sentencePattern: regexp.MustCompile(`[.!?]+\s+`),
	}
}

// NewTokenizerWithConfig creates a tokenizer with custom configuration
func NewTokenizerWithConfig(maxChunkTokens, overlapTokens int) *Tokenizer {
	return &Tokenizer{
		maxChunkTokens:  maxChunkTokens,
		overlapTokens:   overlapTokens,
		sentencePattern: regexp.MustCompile(`[.!?]+\s+`),
	}
}

// EstimateTokens estimates the number of tokens in a text
// Uses a simple approximation: ~4 characters per token for English
// This is a rough estimate - for accurate counts, use tiktoken
func (t *Tokenizer) EstimateTokens(text string) int {
	// Remove extra whitespace
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	// Count words and punctuation as rough token estimate
	words := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				words++
				inWord = false
			}
		} else {
			inWord = true
		}
	}

	if inWord {
		words++
	}

	// Estimate: average of 1.3 tokens per word (accounts for subword tokenization)
	tokens := int(float64(words) * 1.3)

	// Also add tokens for punctuation and special characters
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			tokens++
		}
	}

	return tokens
}

// ChunkText splits text into chunks suitable for embedding
func (t *Tokenizer) ChunkText(text string) []TextChunk {
	// Clean up text
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Split into sentences
	sentences := t.splitIntoSentences(text)

	var chunks []TextChunk
	var currentChunk strings.Builder
	currentTokens := 0
	chunkIndex := 0

	for i, sentence := range sentences {
		sentenceTokens := t.EstimateTokens(sentence)

		// If single sentence exceeds max, split it further
		if sentenceTokens > t.maxChunkTokens {
			// Flush current chunk if not empty
			if currentChunk.Len() > 0 {
				chunks = append(chunks, TextChunk{
					Content:    strings.TrimSpace(currentChunk.String()),
					TokenCount: currentTokens,
					Index:      chunkIndex,
				})
				chunkIndex++
				currentChunk.Reset()
				currentTokens = 0
			}

			// Split long sentence into smaller pieces
			subChunks := t.splitLongText(sentence)
			for _, sub := range subChunks {
				chunks = append(chunks, TextChunk{
					Content:    sub.Content,
					TokenCount: sub.TokenCount,
					Index:      chunkIndex,
				})
				chunkIndex++
			}
			continue
		}

		// Check if adding this sentence would exceed the limit
		if currentTokens+sentenceTokens > t.maxChunkTokens && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, TextChunk{
				Content:    strings.TrimSpace(currentChunk.String()),
				TokenCount: currentTokens,
				Index:      chunkIndex,
			})
			chunkIndex++

			// Start new chunk with overlap
			currentChunk.Reset()
			currentTokens = 0

			// Add overlap from previous sentences
			overlapStart := i - 1
			overlapTokens := 0
			for overlapStart >= 0 && overlapTokens < t.overlapTokens {
				overlapTokens += t.EstimateTokens(sentences[overlapStart])
				overlapStart--
			}

			// Add overlap sentences
			for j := overlapStart + 1; j < i; j++ {
				if j >= 0 {
					currentChunk.WriteString(sentences[j])
					currentChunk.WriteString(" ")
					currentTokens += t.EstimateTokens(sentences[j])
				}
			}
		}

		currentChunk.WriteString(sentence)
		currentChunk.WriteString(" ")
		currentTokens += sentenceTokens
	}

	// Add remaining content as final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, TextChunk{
			Content:    strings.TrimSpace(currentChunk.String()),
			TokenCount: currentTokens,
			Index:      chunkIndex,
		})
	}

	return chunks
}

// TextChunk represents a chunk of text with metadata
type TextChunk struct {
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
	Index      int    `json:"index"`
}

// splitIntoSentences splits text into sentences
func (t *Tokenizer) splitIntoSentences(text string) []string {
	// Split by sentence-ending punctuation
	parts := t.sentencePattern.Split(text, -1)
	matches := t.sentencePattern.FindAllString(text, -1)

	var sentences []string
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		sentence := strings.TrimSpace(part)
		// Add back the punctuation
		if i < len(matches) {
			sentence += strings.TrimSpace(matches[i])
		}

		sentences = append(sentences, sentence)
	}

	return sentences
}

// splitLongText splits text that's too long for a single chunk
func (t *Tokenizer) splitLongText(text string) []TextChunk {
	var chunks []TextChunk

	// Split by words
	words := strings.Fields(text)
	var currentChunk strings.Builder
	currentTokens := 0
	chunkIndex := 0

	for _, word := range words {
		wordTokens := t.EstimateTokens(word)

		if currentTokens+wordTokens > t.maxChunkTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, TextChunk{
				Content:    strings.TrimSpace(currentChunk.String()),
				TokenCount: currentTokens,
				Index:      chunkIndex,
			})
			chunkIndex++
			currentChunk.Reset()
			currentTokens = 0
		}

		currentChunk.WriteString(word)
		currentChunk.WriteString(" ")
		currentTokens += wordTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, TextChunk{
			Content:    strings.TrimSpace(currentChunk.String()),
			TokenCount: currentTokens,
			Index:      chunkIndex,
		})
	}

	return chunks
}

// CleanText removes HTML tags and normalizes whitespace
func (t *Tokenizer) CleanText(text string) string {
	// Remove HTML tags
	htmlPattern := regexp.MustCompile(`<[^>]*>`)
	text = htmlPattern.ReplaceAllString(text, " ")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	whitespacePattern := regexp.MustCompile(`\s+`)
	text = whitespacePattern.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
