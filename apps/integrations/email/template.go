package email

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"time"
)

// RenderTemplate renders an HTML email template with the given data.
func RenderTemplate(templateHTML string, data TemplateData) (string, error) {
	// If no template provided, use default
	if templateHTML == "" {
		templateHTML = DefaultTemplate
	}

	// Create template with custom functions
	tmpl, err := template.New("email").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
	}).Parse(templateHTML)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// BuildTemplateData creates template data from conversation and message info.
func BuildTemplateData(message, displayName, avatar string, conversationID int, conversationNumber, status, department, priority string) TemplateData {
	return TemplateData{
		Message:                message,
		DisplayName:            displayName,
		Avatar:                 avatar,
		Date:                   time.Now().Format("January 2, 2006 at 3:04 PM"),
		ConversationID:         conversationID,
		ConversationNumber:     conversationNumber,
		ConversationStatus:     status,
		ConversationDepartment: department,
		ConversationPriority:   priority,
	}
}

// StripHTML removes HTML tags from a string.
func StripHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Clean up whitespace
	text = strings.TrimSpace(text)
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return text
}

// ExtractPlainText extracts plain text from an email (prefers plain, falls back to HTML).
func ExtractPlainText(email Email) string {
	if email.Body != "" {
		return strings.TrimSpace(email.Body)
	}
	if email.HTMLBody != "" {
		return StripHTML(email.HTMLBody)
	}
	return ""
}

// IsReplyEmail checks if an email is a reply based on subject.
func IsReplyEmail(subject string) bool {
	subject = strings.ToLower(strings.TrimSpace(subject))
	return strings.HasPrefix(subject, "re:") ||
		strings.HasPrefix(subject, "re ") ||
		strings.HasPrefix(subject, "aw:") || // German reply prefix
		strings.HasPrefix(subject, "sv:") || // Swedish reply prefix
		strings.HasPrefix(subject, "odp:") // Polish reply prefix
}

// CleanSubject removes Re:, Fwd:, etc. prefixes from subject.
func CleanSubject(subject string) string {
	// Remove common prefixes
	prefixes := []string{"re:", "re ", "fwd:", "fwd ", "fw:", "fw ", "aw:", "aw ", "sv:", "sv ", "odp:", "odp "}

	cleaned := strings.TrimSpace(subject)
	lower := strings.ToLower(cleaned)

	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			cleaned = strings.TrimSpace(cleaned[len(prefix):])
			lower = strings.ToLower(cleaned)
		}
	}

	return cleaned
}

// ExtractReplyText attempts to extract just the reply portion of an email.
// It removes quoted text and signatures.
func ExtractReplyText(body string) string {
	lines := strings.Split(body, "\n")
	var result []string
	inQuote := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the start
		if len(result) == 0 && trimmed == "" {
			continue
		}

		// Detect start of quoted text
		if strings.HasPrefix(trimmed, ">") ||
			strings.HasPrefix(trimmed, "On ") && strings.Contains(trimmed, " wrote:") ||
			strings.HasPrefix(trimmed, "----") ||
			strings.HasPrefix(trimmed, "From:") ||
			strings.HasPrefix(trimmed, "Sent:") ||
			strings.HasPrefix(trimmed, "To:") ||
			strings.HasPrefix(trimmed, "Subject:") {
			inQuote = true
			continue
		}

		// Detect signature markers
		if trimmed == "--" || trimmed == "-- " || trimmed == "---" {
			break
		}

		if !inQuote {
			result = append(result, line)
		}
	}

	// Remove trailing empty lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}

	return strings.Join(result, "\n")
}

// GenerateReplySubject creates a proper reply subject.
func GenerateReplySubject(originalSubject string) string {
	cleaned := CleanSubject(originalSubject)
	return "Re: " + cleaned
}

// ValidateTemplate checks if a template is valid.
func ValidateTemplate(templateHTML string) error {
	// Create a dummy data object for validation
	data := TemplateData{
		Message:                "Test message",
		DisplayName:            "Test User",
		Avatar:                 "https://example.com/avatar.png",
		Date:                   time.Now().Format("January 2, 2006 at 3:04 PM"),
		ConversationID:         1,
		ConversationNumber:     "CONV-1",
		ConversationStatus:     "open",
		ConversationDepartment: "Support",
		ConversationPriority:   "medium",
	}

	_, err := RenderTemplate(templateHTML, data)
	return err
}
