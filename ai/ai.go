package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"anki-builder/data"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/option"
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

	1. Please provide a phrase that could be used to generate an image which will help the learner recall this word when they see it on a flashcard study app.`, englishWord)
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

	img, err := c.GenerateImage(ctx, imageDescription)
	if err != nil {
		return englishWord, imageDescription, "", nil
	}
	
	return englishWord, imageDescription, img, nil
}

// GenerateImage generates a 512x512 image for a given prompt and returns the URL.
func (c *Client) GenerateImage(ctx context.Context, prompt string) (string, error) {
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

	return resp.Data[0].URL, nil
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

