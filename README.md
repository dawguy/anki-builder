# ANKI BUILDER

Project goals:

* Have a hosted solution which allows me to add new cards and manage my decks from anywhere.
* Auto-sync upon update

Nice to haves:
* Auto-image generation
** Thinking of using my spare desktop for this

---

# Usage

Create vocab.csv file in root directory
WARN: vocab.csv should contain all words and never be deleted.

From root directory

`go run main.go` - This will generate a file called vocab.db and download images from open ai

Once all images have been created + database has been updated

`cd python`
`uv run main.py`

This will create a file called `korean_vocab.apkg`

Import this file into Anki and it will just work! New cards should be handled as expected without duplicates on repeated downloads.
