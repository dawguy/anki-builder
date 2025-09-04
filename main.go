package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"anki-builder/ai"
	"anki-builder/data"
)

func main() {
	store, err := data.Open("./vocab.db")
	if err != nil {
		log.Fatal(err)
	}

	newWords, err := store.ParseCSVNewWords("vocab.csv")
	if err != nil {
		log.Fatal("CSV parse failed:", err)
	}

	aiClient := ai.NewClient()

	alreadySeen := make(map[string]struct{})
	fmt.Println("New words found in CSV:")
	for _, w := range newWords {
		if _, seen := alreadySeen[w.KoreanWord]; seen {
			continue
		}
		alreadySeen[w.KoreanWord] = struct{}{}
		fmt.Printf("%s | %s\n", w.KoreanWord, ptrOrEmpty(w.KoreanPhrase))
		eng, imgPrompt, imageUrl, err := aiClient.EnrichWord(context.Background(), w)
		if err != nil {
			log.Printf("OpenAI enrichment failed for %s: %v", w.KoreanWord, err)
			continue
		}
		w.EnglishTranslation = &eng
		w.ImagePrompt = &imgPrompt
		w.ImageURL = &imageUrl
		fmt.Printf("Word: %s\nEnglish: %s\nImage Prompt: %s\nImage URL: %s\n\n",
			w.KoreanWord, eng, imgPrompt, imageUrl)

		// Save enriched word into DB
		if err := store.AddWord(w); err != nil {
			log.Printf("Failed to save %s: %v", w.KoreanWord, err)
			continue
		}
		savedWord, err := store.FindByKoreanWord(w.KoreanWord)
		if err != nil {
			log.Printf("Failed to retrieve from store %s: %v", w.KoreanWord, err)
			continue
		}
		if w.EnglishTranslation != nil {
			filename := sanitizeFilename(*w.EnglishTranslation)
			err = os.Rename(fmt.Sprintf("raw_images/%s.png", filename), fmt.Sprintf("raw_images/%s.png", strconv.Itoa(savedWord.ID)))
			if err != nil {
				log.Printf("Failed to rename file %s to %s", fmt.Sprintf("raw_images/%s.png", filename), fmt.Sprintf("raw_images/%s.png", strconv.Itoa(savedWord.ID)))
				continue
			}
		}
	}
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var invalidChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// sanitizeFilename replaces characters that aren't allowed in filenames
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = invalidChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_") // remove leading/trailing underscores
	return s
}
