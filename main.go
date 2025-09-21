package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

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

	// worker pool setup
	const workerCount = 4
	jobs := make(chan data.VocabWord)
	var wg sync.WaitGroup

	// start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for w := range jobs {
				processWord(w, aiClient, store)
			}
		}()
	}

	// send jobs
	for _, w := range newWords {
		if _, seen := alreadySeen[w.KoreanWord]; seen {
			continue
		}
		alreadySeen[w.KoreanWord] = struct{}{}
		if w.KoreanPhrase != nil {
			alreadySeenPhrase[*w.KoreanPhrase] = struct{}{}
		}
		jobs <- w
	}

	close(jobs)
	wg.Wait()
}

func processWord(w data.VocabWord, aiClient *ai.Client, store *data.Store) {
	language := "Korean"
	if len(os.Args) >= 2 {
		language = os.Args[1]
	}

	fmt.Printf("%s | %s\n", w.KoreanWord, ptrOrEmpty(w.KoreanPhrase))

	enrichedWord, err := aiClient.EnrichWord(context.Background(), language, w)
	if err != nil {
		log.Printf("OpenAI enrichment failed for %s: %v", w.KoreanWord, err)
		return
	}

	fmt.Println(spew.Sdump(enrichedWord))
	fmt.Println("=============")

	w.KoreanWordDictionaryForm = enrichedWord.DictionaryFormWord
	w.KoreanShortExample = &enrichedWord.ShortExamplePhrase
	w.EnglishTranslationShort = &enrichedWord.EnglishTranslationShort
	w.EnglishTranslationLong = &enrichedWord.EnglishTranslationLong
	w.EnglishAlternateDefintions = &enrichedWord.EnglishAlternateDefintions
	w.WordImportanceLevel = &enrichedWord.WordImportanceLevel
	w.ImagePrompt = &enrichedWord.ImagePrompt
	w.ImageURL = &enrichedWord.ImageURL

	// Save enriched word into DB
	if err := store.AddWord(w); err != nil {
		log.Printf("Failed to save %s: %v", w.KoreanWord, err)
		return
	}
	savedWord, err := store.FindByKoreanWord(w.KoreanWord)
	if err != nil {
		log.Printf("Failed to retrieve from store %s: %v", w.KoreanWord, err)
		return
	}
	if w.EnglishTranslationShort != nil {
		filename := sanitizeFilename(*w.EnglishTranslationShort)
		err = os.Rename(
			fmt.Sprintf("raw_images/%s.png", filename),
			fmt.Sprintf("raw_images/%d.png", savedWord.ID),
		)
		if err != nil {
			log.Printf("Failed to rename file %s to %s",
				fmt.Sprintf("raw_images/%s.png", filename),
				fmt.Sprintf("raw_images/%d.png", savedWord.ID))
			return
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

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = invalidChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_") // remove leading/trailing underscores
	return s
}
