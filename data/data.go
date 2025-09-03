package data

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type VocabWord struct {
	ID                 int
	KoreanWord         string
	KoreanPhrase       *string
	EnglishTranslation *string
	ImagePrompt        *string
	ImageURL           *string
}

type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database and initializes the schema.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

// initSchema creates the vocab_words table with indexes.
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vocab_words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		korean_word TEXT NOT NULL,
		korean_phrase TEXT,
		english_translation TEXT,
		image_prompt TEXT,
		image_url TEXT
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_vocab_korean_word ON vocab_words(korean_word);
	CREATE INDEX IF NOT EXISTS idx_vocab_english_translation ON vocab_words(english_translation);
	`

	_, err := s.db.Exec(schema)
	return err
}

// AddWord inserts a new VocabWord into the database.
func (s *Store) AddWord(word VocabWord) error {
	query := `
	INSERT INTO vocab_words (korean_word, korean_phrase, english_translation, image_prompt, image_url)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(korean_word) DO NOTHING;
	`

	_, err := s.db.Exec(query, word.KoreanWord, word.KoreanPhrase, word.EnglishTranslation, word.ImagePrompt, word.ImageURL)
	return err
}

// GetAll retrieves all vocab words.
func (s *Store) GetAll() ([]VocabWord, error) {
	rows, err := s.db.Query(`SELECT id, korean_word, korean_phrase, english_translation, image_prompt, image_url FROM vocab_words`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []VocabWord
	for rows.Next() {
		var w VocabWord
		err := rows.Scan(&w.ID, &w.KoreanWord, &w.KoreanPhrase, &w.EnglishTranslation, &w.ImagePrompt, &w.ImageURL)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}

	return words, nil
}

// FindByKoreanWord looks up a word by its Korean form.
func (s *Store) FindByKoreanWord(word string) (*VocabWord, error) {
	row := s.db.QueryRow(`SELECT id, korean_word, korean_phrase, english_translation, image_prompt, image_url FROM vocab_words WHERE korean_word = ?`, word)

	var w VocabWord
	err := row.Scan(&w.ID, &w.KoreanWord, &w.KoreanPhrase, &w.EnglishTranslation, &w.ImagePrompt, &w.ImageURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &w, nil
}
