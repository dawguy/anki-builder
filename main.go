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

	"github.com/davecgh/go-spew/spew"
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
	alreadySeenPhrase := make(map[string]struct{})
	fmt.Println("New words found in CSV:")
	for _, w := range newWords {
		if _, seen := alreadySeen[w.KoreanWord]; seen {
			continue
		}
		alreadySeen[w.KoreanWord] = struct{}{}
		if w.KoreanPhrase != nil {
			alreadySeenPhrase[*w.KoreanPhrase] = struct{}{}
		}
		language := "Korean"
		if len(os.Args) >= 2 {
			language = os.Args[1]
		}
		fmt.Printf("%s | %s\n", w.KoreanWord, ptrOrEmpty(w.KoreanPhrase))
		enrichedWord, err := aiClient.EnrichWord(context.Background(), language, w)
		if err != nil {
			log.Printf("OpenAI enrichment failed for %s: %v", w.KoreanWord, err)
			continue
		}
		fmt.Println(spew.Sdump(enrichedWord))
		fmt.Println("=============")
		w.KoreanWordDictionaryForm = enrichedWord.DictionaryFormWord
		w.KoreanShortExample = &enrichedWord.ShortExamplePhrase
		w.EnglishTranslationShort = &enrichedWord.EnglishTranslationShort
		w.EnglishTranslationLong = &enrichedWord.EnglishTranslationLong
		w.EnglishAlternateDefintions = &enrichedWord.EnglishAlternateDefintions
		w.ImagePrompt = &enrichedWord.ImagePrompt
		w.ImageURL = &enrichedWord.ImageURL

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
		if w.EnglishTranslationShort != nil {
			filename := sanitizeFilename(*w.EnglishTranslationShort)
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
