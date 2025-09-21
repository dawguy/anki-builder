package data

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type VocabWord struct {
	ID int
	// Word being translated
	KoreanWord string
	// Dictionary form
	KoreanWordDictionaryForm string
	// Mined phrase or passage
	KoreanPhrase *string
	// Shorter example phrase
	KoreanShortExample *string

	// Ideally one word
	EnglishTranslationShort *string
	// Short description of the translation
	EnglishTranslationLong *string
	// Alternate definitions
	EnglishAlternateDefintions *string

	// Importance level <High / Medium / Low> expected values
	WordImportanceLevel *string

	// What prompt was used to generate the image
	ImagePrompt *string
	// URL image was saved at
	ImageURL *string
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
		korean_word_dictionary_form TEXT,
		korean_phrase TEXT,
		korean_short_example TEXT,
		english_translation_short TEXT,
		english_translation_long TEXT,
		english_alternate_definitions TEXT,
		word_importance_level TEXT,
		image_prompt TEXT,
		image_url TEXT
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_vocab_korean_word ON vocab_words(korean_word);
	CREATE INDEX IF NOT EXISTS idx_vocab_english_translation_short ON vocab_words(english_translation_short);
	CREATE INDEX IF NOT EXISTS idx_vocab_importance_level ON vocab_words(word_importance_level);
	`

	_, err := s.db.Exec(schema)
	return err
}

// AddWord inserts a new VocabWord into the database.
func (s *Store) AddWord(word VocabWord) error {
	query := `
	INSERT INTO vocab_words (
		korean_word,
		korean_word_dictionary_form,
		korean_phrase,
		korean_short_example,
		english_translation_short,
		english_translation_long,
		english_alternate_definitions,
		word_importance_level,
		image_prompt,
		image_url
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(korean_word) DO NOTHING;
	`

	_, err := s.db.Exec(query,
		word.KoreanWord,
		word.KoreanWordDictionaryForm,
		word.KoreanPhrase,
		word.KoreanShortExample,
		word.EnglishTranslationShort,
		word.EnglishTranslationLong,
		word.EnglishAlternateDefintions,
		word.WordImportanceLevel,
		word.ImagePrompt,
		word.ImageURL,
	)
	return err
}

// GetAll retrieves all vocab words.
func (s *Store) GetAll() ([]VocabWord, error) {
	rows, err := s.db.Query(`
	SELECT
		id,
		korean_word,
		korean_word_dictionary_form,
		korean_phrase,
		korean_short_example,
		english_translation_short,
		english_translation_long,
		english_alternate_definitions,
		word_importance_level,
		image_prompt,
		image_url
	FROM vocab_words
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []VocabWord
	for rows.Next() {
		var w VocabWord
		err := rows.Scan(
			&w.ID,
			&w.KoreanWord,
			&w.KoreanWordDictionaryForm,
			&w.KoreanPhrase,
			&w.KoreanShortExample,
			&w.EnglishTranslationShort,
			&w.EnglishTranslationLong,
			&w.EnglishAlternateDefintions,
			&w.WordImportanceLevel,
			&w.ImagePrompt,
			&w.ImageURL,
		)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}

	return words, nil
}

// FindByKoreanWord looks up a word by its Korean form.
func (s *Store) FindByKoreanWord(word string) (*VocabWord, error) {
	row := s.db.QueryRow(`
	SELECT
		id,
		korean_word,
		korean_word_dictionary_form,
		korean_phrase,
		korean_short_example,
		english_translation_short,
		english_translation_long,
		english_alternate_definitions,
		word_importance_level,
		image_prompt,
		image_url
	FROM vocab_words
	WHERE korean_word = ?
	`, word)

	var w VocabWord
	err := row.Scan(
		&w.ID,
		&w.KoreanWord,
		&w.KoreanWordDictionaryForm,
		&w.KoreanPhrase,
		&w.KoreanShortExample,
		&w.EnglishTranslationShort,
		&w.EnglishTranslationLong,
		&w.EnglishAlternateDefintions,
		&w.WordImportanceLevel,
		&w.ImagePrompt,
		&w.ImageURL,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &w, nil
}
