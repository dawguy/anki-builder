package main

import (
	"context"
	"fmt"
	"log"

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

	fmt.Println("New words found in CSV:")
	for _, w := range newWords {
		fmt.Printf("%s | %s\n", w.KoreanWord, ptrOrEmpty(w.KoreanPhrase))
		eng, imgPrompt, err := aiClient.EnrichWord(context.Background(), w)
		if err != nil {
			log.Printf("OpenAI enrichment failed for %s: %v", w.KoreanWord, err)
			continue
		}
		w.EnglishTranslation = &eng
		w.ImageURL = &imgPrompt // for now store prompt instead of URL
		fmt.Printf("Word: %s\nEnglish: %s\nImage Prompt: %s\n\n",
			w.KoreanWord, eng, imgPrompt)

		// Save enriched word into DB
//		if err := store.AddWord(w); err != nil {
//			log.Printf("Failed to save %s: %v", w.KoreanWord, err)
//		}
	}
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
