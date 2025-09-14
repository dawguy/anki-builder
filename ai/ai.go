package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"anki-builder/data"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
)

type Client struct {
	api *openai.Client
}

func NewClient() *Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY not set")
	}

	c := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &Client{
		api: &c,
	}
}

type EnrichedWord struct {
	OriginalWord   string
	OriginalPhrase string

	FullResponse string

	DictionaryFormWord         string
	ShortExamplePhrase         string
	EnglishTranslationShort    string
	EnglishTranslationLong     string
	EnglishAlternateDefintions string

	ImagePrompt string
	ImageURL    string
}

// EnrichWord calls OpenAI to get translation + image prompt.
func (c *Client) EnrichWord(ctx context.Context, languageName string, word data.VocabWord) (EnrichedWord, error) {
	if languageName == "" {
		languageName = "Korean"
	}
	prompt := fmt.Sprintf(`
You are helping a young adult learn %s as a foreign language.
Word we're trying to translate: %s
Phrase where the learner first encountered the word: %s

Please provide your response in the below format

"""
Original Word: <word being translated here>
Original Dictionary Form Of Word: <word being translated's base dictionary form as some langauges have different conjugations>
Original Phrase: <phrase being translated here>
English Translation Long: <please produce an English translation of the word in roughly 1 or 2 sentences>
English Translation Short: <please produce an English translation of the word based on the provided phrase in as few words as possible>
English Alternative Definitions: <some words can have multiple meanings. Please list any alternate meanings in a comma separate list here>
Short example using word: <please generate an example sentence using vocabulary an 7th grader would know in %s language. This example setence will be used on a flashcard so it MUST contain the original word, but the grammatical endings could be changed. The example should be entirely in the %s language.>
Image prompt: <please create a prompt based on the word which can be fed into an AI image generator at a later point in time>
"""
`, languageName, word.KoreanWord, ptrOrEmpty(word.KoreanPhrase), languageName, languageName)

	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return EnrichedWord{}, err
	}

	text := resp.Choices[0].Message.Content
	enrichedWord := ParseEnrichedWord(text)

	imgPath, err := c.GenerateImage(ctx, enrichedWord.ImagePrompt, enrichedWord.EnglishTranslationShort)
	if err != nil {
		return EnrichedWord{}, err
	}
	enrichedWord.ImageURL = imgPath

	return enrichedWord, nil
}

// GenerateImage generates a 512x512 image for a given prompt and returns the URL.
func (c *Client) GenerateImage(ctx context.Context, prompt string, idStr string) (string, error) {
	resp, err := c.api.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt: prompt,
		Size:   "1024x1024",
		N:      param.NewOpt[int64](1),
	})
	if err != nil {
		return "", err
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("no image returned")
	}

	imageURL := resp.Data[0].URL

	// 2. Create raw_images folder if it doesn't exist
	if err := os.MkdirAll("raw_images", os.ModePerm); err != nil {
		return "", err
	}

	// 3. Download the image
	respHTTP, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer respHTTP.Body.Close()

	// 4. Save file with sanitized name
	filename := sanitizeFilename(idStr) + ".png"
	localPath := filepath.Join("raw_images", filename)
	file, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, respHTTP.Body)
	if err != nil {
		return "", err
	}

	return localPath, nil
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func splitTwoLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

var invalidChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// sanitizeFilename replaces characters that aren't allowed in filenames
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = invalidChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_") // remove leading/trailing underscores
	return s
}

func ParseEnrichedWord(input string) EnrichedWord {
	// Clean up outer quotes if model includes """ ... """
	input = strings.Trim(input, "\"\n ")

	lines := strings.Split(input, "\n")
	word := EnrichedWord{
		FullResponse: input,
	}

	var currentKey string
	fieldMap := map[string]*string{
		"original word:":                    &word.OriginalWord,
		"original dictionary form of word:": &word.DictionaryFormWord,
		"original phrase:":                  &word.OriginalPhrase,
		"english translation long:":         &word.EnglishTranslationLong,
		"english translation short:":        &word.EnglishTranslationShort,
		"english alternative definitions:":  &word.EnglishAlternateDefintions,
		"short example using word:":         &word.ShortExamplePhrase,
		"image prompt:":                     &word.ImagePrompt,
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)

		// check if this line starts a new field
		foundKey := ""
		for key := range fieldMap {
			if strings.HasPrefix(lower, key) {
				foundKey = key
				break
			}
		}

		if foundKey != "" {
			// new field
			val := strings.TrimSpace(line[len(foundKey):])
			*fieldMap[foundKey] = val
			currentKey = foundKey
		} else if currentKey != "" {
			// continuation of previous field
			prev := *fieldMap[currentKey]
			if prev != "" {
				prev += " "
			}
			*fieldMap[currentKey] = prev + line
		}
	}

	return word
}
