package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/iesreza/homa-backend/apps/models"
)

// TranslationItem represents a single item to translate
type TranslationItem struct {
	ID      uint   `json:"id"`
	Content string `json:"content"`
}

// TranslatedItem represents a translated item
type TranslatedItem struct {
	ID         uint   `json:"id"`
	Original   string `json:"original"`
	Translated string `json:"translated"`
}

// TranslateText translates a single text from one language to another
func TranslateText(text, fromLang, toLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	if fromLang == toLang {
		return text, nil
	}

	client := GetClient()
	if client == nil {
		return "", fmt.Errorf("AI client not initialized")
	}

	prompt := fmt.Sprintf(`Translate this customer service chat message from %s to %s.

Rules:
- Translate naturally, like a native speaker would say it - NOT word-by-word
- Keep the same tone (formal/informal, friendly/professional)
- Adapt idioms and expressions to sound natural in %s
- Preserve emojis, formatting, and line breaks
- If informal (like "u" for "you", "pls" for "please"), keep it informal
- Only output the translation, nothing else

Text: %s`, getLanguageName(fromLang), getLanguageName(toLang), getLanguageName(toLang), text)

	messages := []ChatMessage{
		{Role: "system", Content: "You are a native bilingual translator specializing in customer service conversations. Your translations sound completely natural - as if originally written in the target language. Never translate word-by-word. Adapt expressions to how native speakers actually communicate."},
		{Role: "user", Content: prompt},
	}

	resp, err := client.ChatCompletion(messages, 2000, 0.3)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no translation response")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// TranslateBatch translates multiple texts at once for efficiency
func TranslateBatch(items []TranslationItem, fromLang, toLang string) ([]TranslatedItem, error) {
	if len(items) == 0 {
		return []TranslatedItem{}, nil
	}

	if fromLang == toLang {
		result := make([]TranslatedItem, len(items))
		for i, item := range items {
			result[i] = TranslatedItem{
				ID:         item.ID,
				Original:   item.Content,
				Translated: item.Content,
			}
		}
		return result, nil
	}

	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("AI client not initialized")
	}

	// Build batch translation prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Translate these customer service chat messages from %s to %s.\n\n", getLanguageName(fromLang), getLanguageName(toLang)))
	sb.WriteString("Rules:\n")
	sb.WriteString("- Translate naturally like a native speaker - NOT word-by-word\n")
	sb.WriteString("- Keep the same tone (formal/informal)\n")
	sb.WriteString("- Adapt idioms and expressions to sound natural\n")
	sb.WriteString("- Preserve emojis and formatting\n")
	sb.WriteString("- Return JSON array with 'id' and 'translated' fields\n\n")
	sb.WriteString("Messages:\n")

	for _, item := range items {
		sb.WriteString(fmt.Sprintf("ID %d: %s\n", item.ID, item.Content))
	}

	sb.WriteString("\nRespond with ONLY valid JSON array, no markdown.")

	messages := []ChatMessage{
		{Role: "system", Content: "You are a native bilingual translator for customer service chats. Translations must sound completely natural - as if originally written in the target language. Never translate word-by-word. Always respond with valid JSON."},
		{Role: "user", Content: sb.String()},
	}

	resp, err := client.ChatCompletion(messages, 4000, 0.3)
	if err != nil {
		// Fallback to individual translations
		log.Warning("Batch translation failed, falling back to individual: %v", err)
		return translateIndividually(items, fromLang, toLang)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no translation response")
	}

	// Parse JSON response
	responseText := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Remove potential markdown code blocks
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var translatedItems []struct {
		ID         uint   `json:"id"`
		Translated string `json:"translated"`
	}

	if err := json.Unmarshal([]byte(responseText), &translatedItems); err != nil {
		log.Warning("Failed to parse batch translation response, falling back to individual: %v", err)
		return translateIndividually(items, fromLang, toLang)
	}

	// Map results back to original items
	translationMap := make(map[uint]string)
	for _, t := range translatedItems {
		translationMap[t.ID] = t.Translated
	}

	result := make([]TranslatedItem, len(items))
	for i, item := range items {
		translated := translationMap[item.ID]
		if translated == "" {
			translated = item.Content // Fallback to original if not found
		}
		result[i] = TranslatedItem{
			ID:         item.ID,
			Original:   item.Content,
			Translated: translated,
		}
	}

	return result, nil
}

// translateIndividually translates items one by one as fallback
func translateIndividually(items []TranslationItem, fromLang, toLang string) ([]TranslatedItem, error) {
	result := make([]TranslatedItem, len(items))
	for i, item := range items {
		translated, err := TranslateText(item.Content, fromLang, toLang)
		if err != nil {
			log.Warning("Individual translation failed for ID %d: %v", item.ID, err)
			translated = item.Content // Use original on failure
		}
		result[i] = TranslatedItem{
			ID:         item.ID,
			Original:   item.Content,
			Translated: translated,
		}
	}
	return result, nil
}

// GetOrCreateTranslations fetches existing translations or creates new ones
func GetOrCreateTranslations(conversationID uint, messageIDs []uint, fromLang, toLang, translationType string) ([]models.TranslationResponse, error) {
	if len(messageIDs) == 0 {
		return []models.TranslationResponse{}, nil
	}

	// Fetch existing translations
	var existingTranslations []models.ConversationMessageTranslation
	db.Where("conversation_id = ? AND message_id IN ? AND to_lang = ? AND type = ?",
		conversationID, messageIDs, toLang, translationType).
		Find(&existingTranslations)

	existingMap := make(map[uint]models.ConversationMessageTranslation)
	for _, t := range existingTranslations {
		existingMap[t.MessageID] = t
	}

	// Find messages that need translation
	var missingIDs []uint
	for _, id := range messageIDs {
		if _, exists := existingMap[id]; !exists {
			missingIDs = append(missingIDs, id)
		}
	}

	// Fetch messages that need translation
	if len(missingIDs) > 0 {
		var messages []models.Message
		db.Where("id IN ?", missingIDs).Find(&messages)

		// Prepare items for batch translation
		items := make([]TranslationItem, 0, len(messages))
		for _, msg := range messages {
			if msg.Body != "" {
				items = append(items, TranslationItem{
					ID:      msg.ID,
					Content: msg.Body,
				})
			}
		}

		// Translate batch
		if len(items) > 0 {
			translated, err := TranslateBatch(items, fromLang, toLang)
			if err != nil {
				log.Error("Batch translation failed: %v", err)
			} else {
				// Save translations to database
				for _, t := range translated {
					translation := models.ConversationMessageTranslation{
						ConversationID: conversationID,
						MessageID:      t.ID,
						FromLang:       fromLang,
						ToLang:         toLang,
						Content:        t.Translated,
						Type:           translationType,
					}
					if err := db.Create(&translation).Error; err != nil {
						log.Warning("Failed to save translation for message %d: %v", t.ID, err)
					} else {
						existingMap[t.ID] = translation
					}
				}
			}
		}
	}

	// Fetch original messages for the response
	var messages []models.Message
	db.Where("id IN ?", messageIDs).Find(&messages)

	messageMap := make(map[uint]models.Message)
	for _, msg := range messages {
		messageMap[msg.ID] = msg
	}

	// Build response
	result := make([]models.TranslationResponse, 0, len(messageIDs))
	for _, id := range messageIDs {
		msg := messageMap[id]
		originalContent := msg.Body

		translation, exists := existingMap[id]
		if exists {
			result = append(result, models.TranslationResponse{
				MessageID:         id,
				OriginalContent:   originalContent,
				TranslatedContent: translation.Content,
				FromLang:          fromLang,
				ToLang:            toLang,
				Type:              translationType,
				IsTranslated:      true,
			})
		} else {
			result = append(result, models.TranslationResponse{
				MessageID:         id,
				OriginalContent:   originalContent,
				TranslatedContent: originalContent, // Use original if translation failed
				FromLang:          fromLang,
				ToLang:            toLang,
				Type:              translationType,
				IsTranslated:      false,
			})
		}
	}

	return result, nil
}

// TranslateOutgoingMessage translates an outgoing message from agent to customer language
func TranslateOutgoingMessage(conversationID, messageID uint, content, fromLang, toLang string) (string, error) {
	if fromLang == toLang || content == "" {
		return content, nil
	}

	// Check if translation already exists
	var existing models.ConversationMessageTranslation
	if err := db.Where("conversation_id = ? AND message_id = ? AND to_lang = ? AND type = ?",
		conversationID, messageID, toLang, models.TranslationTypeOutgoing).First(&existing).Error; err == nil {
		return existing.Content, nil
	}

	// Translate the message
	translated, err := TranslateText(content, fromLang, toLang)
	if err != nil {
		return "", err
	}

	// Save translation
	translation := models.ConversationMessageTranslation{
		ConversationID: conversationID,
		MessageID:      messageID,
		FromLang:       fromLang,
		ToLang:         toLang,
		Content:        translated,
		Type:           models.TranslationTypeOutgoing,
	}
	db.Create(&translation)

	return translated, nil
}

// GetOrCreateTranslationsPerMessage handles per-message language translation
// Each message can have a different source language, and all are translated to the agent's language
func GetOrCreateTranslationsPerMessage(conversationID uint, messageIDs []uint, messageMap map[uint]models.Message, agentLang string) ([]models.TranslationResponse, error) {
	if len(messageIDs) == 0 {
		return []models.TranslationResponse{}, nil
	}

	// Fetch existing translations (to agent language)
	var existingTranslations []models.ConversationMessageTranslation
	db.Where("conversation_id = ? AND message_id IN ? AND to_lang = ? AND type = ?",
		conversationID, messageIDs, agentLang, models.TranslationTypeIncoming).
		Find(&existingTranslations)

	existingMap := make(map[uint]models.ConversationMessageTranslation)
	for _, t := range existingTranslations {
		existingMap[t.MessageID] = t
	}

	// Group messages by source language for batch translation
	langGroups := make(map[string][]TranslationItem)
	var missingIDs []uint

	for _, msgID := range messageIDs {
		if _, exists := existingMap[msgID]; exists {
			continue // Already translated
		}
		msg, ok := messageMap[msgID]
		if !ok || msg.Body == "" || msg.Language == "" {
			continue
		}
		missingIDs = append(missingIDs, msgID)
		langGroups[msg.Language] = append(langGroups[msg.Language], TranslationItem{
			ID:      msg.ID,
			Content: msg.Body,
		})
	}

	// Translate each language group
	for fromLang, items := range langGroups {
		if fromLang == agentLang {
			continue // No translation needed for same language
		}

		translated, err := TranslateBatch(items, fromLang, agentLang)
		if err != nil {
			log.Error("Batch translation failed for %s->%s: %v", fromLang, agentLang, err)
			continue
		}

		// Save translations to database
		for _, t := range translated {
			translation := models.ConversationMessageTranslation{
				ConversationID: conversationID,
				MessageID:      t.ID,
				FromLang:       fromLang,
				ToLang:         agentLang,
				Content:        t.Translated,
				Type:           models.TranslationTypeIncoming,
			}
			if err := db.Create(&translation).Error; err != nil {
				log.Warning("Failed to save translation for message %d: %v", t.ID, err)
			} else {
				existingMap[t.ID] = translation
			}
		}
	}

	// Build response
	result := make([]models.TranslationResponse, 0, len(messageIDs))
	for _, id := range messageIDs {
		msg := messageMap[id]
		originalContent := msg.Body
		fromLang := msg.Language

		// Skip if same language as agent
		if fromLang == agentLang || fromLang == "" {
			continue
		}

		translation, exists := existingMap[id]
		if exists {
			result = append(result, models.TranslationResponse{
				MessageID:         id,
				OriginalContent:   originalContent,
				TranslatedContent: translation.Content,
				FromLang:          fromLang,
				ToLang:            agentLang,
				Type:              models.TranslationTypeIncoming,
				IsTranslated:      true,
			})
		} else {
			result = append(result, models.TranslationResponse{
				MessageID:         id,
				OriginalContent:   originalContent,
				TranslatedContent: originalContent, // Use original if translation failed
				FromLang:          fromLang,
				ToLang:            agentLang,
				Type:              models.TranslationTypeIncoming,
				IsTranslated:      false,
			})
		}
	}

	return result, nil
}

// getLanguageName returns the full language name for a language code
func getLanguageName(code string) string {
	languages := map[string]string{
		"en":  "English",
		"fa":  "Persian",
		"ar":  "Arabic",
		"es":  "Spanish",
		"fr":  "French",
		"de":  "German",
		"zh":  "Chinese",
		"ja":  "Japanese",
		"ko":  "Korean",
		"ru":  "Russian",
		"pt":  "Portuguese",
		"tr":  "Turkish",
		"it":  "Italian",
		"nl":  "Dutch",
		"pl":  "Polish",
		"uk":  "Ukrainian",
		"vi":  "Vietnamese",
		"th":  "Thai",
		"id":  "Indonesian",
		"ms":  "Malay",
		"hi":  "Hindi",
		"bn":  "Bengali",
		"ta":  "Tamil",
		"te":  "Telugu",
		"ur":  "Urdu",
		"he":  "Hebrew",
		"el":  "Greek",
		"cs":  "Czech",
		"sv":  "Swedish",
		"da":  "Danish",
		"fi":  "Finnish",
		"no":  "Norwegian",
		"hu":  "Hungarian",
		"ro":  "Romanian",
		"bg":  "Bulgarian",
		"hr":  "Croatian",
		"sk":  "Slovak",
		"sl":  "Slovenian",
		"sr":  "Serbian",
		"lt":  "Lithuanian",
		"lv":  "Latvian",
		"et":  "Estonian",
		"fil": "Filipino",
		"sw":  "Swahili",
		"af":  "Afrikaans",
	}

	if name, ok := languages[code]; ok {
		return name
	}
	return code
}
