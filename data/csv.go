package data

import (
	"encoding/csv"
	"io"
	"os"
)

// ParseCSVNewWords reads a CSV file and returns new words not already in the DB.
func (s *Store) ParseCSVNewWords(path string) ([]VocabWord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	// Skip header
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	var newWords []VocabWord
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) < 1 {
			continue
		}

		word := record[0]
		var phrase *string
		if len(record) > 1 && record[1] != "" {
			phrase = &record[1]
		}

		// Check if word already exists in DB
		existing, err := s.FindByKoreanWord(word)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			continue // skip duplicates
		}

		// Append new word
		newWords = append(newWords, VocabWord{
			KoreanWord:   word,
			KoreanPhrase: phrase,
		})
	}

	return newWords, nil
}
