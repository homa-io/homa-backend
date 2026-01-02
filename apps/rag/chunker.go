package rag

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Pre-compiled regex patterns
var (
	scriptTagRe     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleTagRe      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlTagRe       = regexp.MustCompile(`<[^>]*>`)
	numericEntityRe = regexp.MustCompile(`&#\d+;`)
	multiSpaceRe    = regexp.MustCompile(` +`)
	multiNewlineRe  = regexp.MustCompile(`\n{3,}`)
)

// Chunk represents a text chunk
type Chunk struct {
	Content    string
	Index      int
	StartPos   int
	EndPos     int
	TokenCount int
}

// ChunkText splits text into chunks with overlap
func ChunkText(content string, chunkSize, overlap, minChunkSize int) []Chunk {
	// Clean the content
	content = cleanText(content)

	if len(content) == 0 {
		return nil
	}

	// If content is smaller than chunk size, return as single chunk
	if len(content) <= chunkSize {
		return []Chunk{
			{
				Content:    content,
				Index:      0,
				StartPos:   0,
				EndPos:     len(content),
				TokenCount: estimateTokens(content),
			},
		}
	}

	var chunks []Chunk
	startPos := 0
	chunkIndex := 0

	for startPos < len(content) {
		endPos := startPos + chunkSize

		// Don't go past the end
		if endPos > len(content) {
			endPos = len(content)
		}

		// Try to find a natural break point (sentence end, paragraph)
		if endPos < len(content) {
			endPos = findBreakPoint(content, startPos, endPos)
		}

		chunkContent := strings.TrimSpace(content[startPos:endPos])

		// Skip empty or too small chunks (unless it's the last one)
		if len(chunkContent) >= minChunkSize || startPos+len(chunkContent) >= len(content) {
			chunks = append(chunks, Chunk{
				Content:    chunkContent,
				Index:      chunkIndex,
				StartPos:   startPos,
				EndPos:     endPos,
				TokenCount: estimateTokens(chunkContent),
			})
			chunkIndex++
		}

		// Move start position, accounting for overlap
		nextStart := endPos - overlap
		if nextStart <= startPos {
			nextStart = startPos + 1
		}
		startPos = nextStart

		// If we're at the end, break
		if endPos >= len(content) {
			break
		}
	}

	return chunks
}

// findBreakPoint finds a natural break point near the target position
func findBreakPoint(content string, startPos, targetPos int) int {
	// Look for break points in order of preference
	searchStart := targetPos - 100 // Look back up to 100 chars
	if searchStart < startPos {
		searchStart = startPos
	}

	searchContent := content[searchStart:targetPos]

	// Priority 1: Paragraph break (double newline)
	if idx := strings.LastIndex(searchContent, "\n\n"); idx != -1 {
		return searchStart + idx + 2
	}

	// Priority 2: Single newline
	if idx := strings.LastIndex(searchContent, "\n"); idx != -1 {
		return searchStart + idx + 1
	}

	// Priority 3: Sentence end (. ! ?)
	for i := len(searchContent) - 1; i >= 0; i-- {
		if searchContent[i] == '.' || searchContent[i] == '!' || searchContent[i] == '?' {
			// Make sure it's followed by a space or end
			if i == len(searchContent)-1 || searchContent[i+1] == ' ' {
				return searchStart + i + 1
			}
		}
	}

	// Priority 4: Space
	if idx := strings.LastIndex(searchContent, " "); idx != -1 {
		return searchStart + idx + 1
	}

	// No good break point found, use original position
	return targetPos
}

// cleanText cleans and normalizes text
func cleanText(text string) string {
	// Strip HTML tags
	text = stripHTMLTags(text)

	// Normalize whitespace
	text = normalizeWhitespace(text)

	// Trim
	text = strings.TrimSpace(text)

	return text
}

// stripHTMLTags removes HTML tags from text
func stripHTMLTags(html string) string {
	// Remove script elements entirely
	html = scriptTagRe.ReplaceAllString(html, "")

	// Remove style elements entirely
	html = styleTagRe.ReplaceAllString(html, "")

	// Remove HTML tags
	text := htmlTagRe.ReplaceAllString(html, " ")

	// Decode common HTML entities
	replacements := map[string]string{
		"&nbsp;":  " ",
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&#39;":   "'",
		"&apos;":  "'",
		"&mdash;": "—",
		"&ndash;": "–",
		"&bull;":  "•",
		"&copy;":  "©",
		"&reg;":   "®",
		"&trade;": "™",
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Remove numeric entities
	text = numericEntityRe.ReplaceAllString(text, " ")

	return text
}

// normalizeWhitespace normalizes whitespace in text
func normalizeWhitespace(text string) string {
	// Replace tabs with spaces
	text = strings.ReplaceAll(text, "\t", " ")

	// Replace multiple spaces with single space
	text = multiSpaceRe.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline
	text = multiNewlineRe.ReplaceAllString(text, "\n\n")

	return text
}

// estimateTokens provides a rough estimate of token count
// OpenAI uses roughly 4 characters per token for English text
func estimateTokens(text string) int {
	charCount := utf8.RuneCountInString(text)
	return (charCount + 3) / 4 // Rough estimate: ~4 chars per token
}
