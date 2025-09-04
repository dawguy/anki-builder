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

// EnrichWord calls OpenAI to get translation + image prompt.
func (c *Client) EnrichWord(ctx context.Context, word data.VocabWord) (string, string, string, error) {
	prompt := fmt.Sprintf(`You are helping a Korean learner.
Word: %s
Phrase (if given): %s

1. Provide a concise English translation of the word. Ideally your reply would be one word long, but feel free to use multiple if it would help.`, word.KoreanWord, ptrOrEmpty(word.KoreanPhrase))

	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", "", "", err
	}

	text := resp.Choices[0].Message.Content
	lines := splitTwoLines(text)
	if len(lines) == 0 {
		return "", "", "", err
	}
	englishWord := lines[0]

	imagePrompt := fmt.Sprintf(`English word: %s

	1. Please provide a phrase that could be used to generate an image which will help the learner recall this word when they see it on a flashcard study app. Do not include any text other than the phrase which will be used to generate an image.`, englishWord)
	resp, err = c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(imagePrompt),
		},
	})
	if err != nil {
		return "", "", "", err
	}
	text = resp.Choices[0].Message.Content
	lines = splitTwoLines(text)
	if len(lines) == 0 {
		return englishWord, "", "", err
	}
	imageDescription := lines[0]

	img, err := c.GenerateImage(ctx, imageDescription, englishWord)
	if err != nil {
		return englishWord, imageDescription, "", nil
	}
	
	return englishWord, imageDescription, img, nil
}

// GenerateImage generates a 512x512 image for a given prompt and returns the URL.
func (c *Client) GenerateImage(ctx context.Context, prompt string, word string) (string, error) {
	resp, err := c.api.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt: prompt,
		Size:   "512x512",
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
	filename := sanitizeFilename(word) + ".png"
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
