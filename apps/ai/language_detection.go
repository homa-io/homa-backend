package ai

import (
	"strings"
	"sync"

	"github.com/pemistahl/lingua-go"
)

var (
	detector     lingua.LanguageDetector
	detectorOnce sync.Once
)

// getDetector returns a singleton language detector instance
func getDetector() lingua.LanguageDetector {
	detectorOnce.Do(func() {
		// Build detector with common languages for better performance
		detector = lingua.NewLanguageDetectorBuilder().
			FromLanguages(
				lingua.English,
				lingua.Persian,
				lingua.Arabic,
				lingua.Spanish,
				lingua.French,
				lingua.German,
				lingua.Chinese,
				lingua.Japanese,
				lingua.Korean,
				lingua.Russian,
				lingua.Portuguese,
				lingua.Turkish,
				lingua.Italian,
				lingua.Dutch,
				lingua.Polish,
				lingua.Ukrainian,
				lingua.Vietnamese,
				lingua.Thai,
				lingua.Indonesian,
				lingua.Malay,
				lingua.Hindi,
				lingua.Bengali,
				lingua.Tamil,
				lingua.Telugu,
				lingua.Urdu,
				lingua.Hebrew,
				lingua.Greek,
				lingua.Czech,
				lingua.Swedish,
				lingua.Danish,
				lingua.Finnish,
				lingua.Hungarian,
				lingua.Romanian,
			).
			WithMinimumRelativeDistance(0.25).
			Build()
	})
	return detector
}

// languageCodeMap maps lingua language codes to ISO 639-1 codes
var languageCodeMap = map[lingua.Language]string{
	lingua.English:    "en",
	lingua.Persian:    "fa",
	lingua.Arabic:     "ar",
	lingua.Spanish:    "es",
	lingua.French:     "fr",
	lingua.German:     "de",
	lingua.Chinese:    "zh",
	lingua.Japanese:   "ja",
	lingua.Korean:     "ko",
	lingua.Russian:    "ru",
	lingua.Portuguese: "pt",
	lingua.Turkish:    "tr",
	lingua.Italian:    "it",
	lingua.Dutch:      "nl",
	lingua.Polish:     "pl",
	lingua.Ukrainian:  "uk",
	lingua.Vietnamese: "vi",
	lingua.Thai:       "th",
	lingua.Indonesian: "id",
	lingua.Malay:      "ms",
	lingua.Hindi:      "hi",
	lingua.Bengali:    "bn",
	lingua.Tamil:      "ta",
	lingua.Telugu:     "te",
	lingua.Urdu:       "ur",
	lingua.Hebrew:     "he",
	lingua.Greek:      "el",
	lingua.Czech:      "cs",
	lingua.Swedish:    "sv",
	lingua.Danish:     "da",
	lingua.Finnish:    "fi",
	lingua.Hungarian:  "hu",
	lingua.Romanian:   "ro",
}

// DetectLanguage detects the language of the given text
// Returns ISO 639-1 language code (e.g., "en", "fa", "es")
// Returns empty string if detection fails or text is too short
func DetectLanguage(text string) string {
	if text == "" {
		return ""
	}

	// Clean and trim the text
	text = strings.TrimSpace(text)

	// Need at least a few characters for reliable detection
	if len(text) < 3 {
		return ""
	}

	detector := getDetector()
	language, exists := detector.DetectLanguageOf(text)

	if !exists {
		return ""
	}

	// Convert to ISO 639-1 code
	if code, ok := languageCodeMap[language]; ok {
		return code
	}

	return ""
}

// BackfillMessageLanguages detects and updates language for messages that don't have it set
func BackfillMessageLanguages(conversationID uint) error {
	// Import is avoided here - we'll call this from the controller
	return nil
}

// DetectLanguageWithConfidence returns the detected language and confidence score
func DetectLanguageWithConfidence(text string) (string, float64) {
	if text == "" {
		return "", 0
	}

	text = strings.TrimSpace(text)
	if len(text) < 3 {
		return "", 0
	}

	detector := getDetector()
	confidenceValues := detector.ComputeLanguageConfidenceValues(text)

	if len(confidenceValues) == 0 {
		return "", 0
	}

	// Get the highest confidence language
	topResult := confidenceValues[0]
	if code, ok := languageCodeMap[topResult.Language()]; ok {
		return code, topResult.Value()
	}

	return "", 0
}
