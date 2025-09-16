import sqlite3
import genanki
import os
from PIL import Image

# --- CONFIG ---
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(BASE_DIR, "vocab.db")
RAW_DIR = os.path.join(BASE_DIR, "raw_images")
MEDIA_DIR = os.path.join(BASE_DIR, "images")
OUTPUT_FILE = os.path.join(BASE_DIR, "korean_vocab.apkg")
DECK_NAME = "Korean Vocab"
IMAGE_SIZE = (360, 360)

os.makedirs(MEDIA_DIR, exist_ok=True)

# Create deck
deck = genanki.Deck(
    2059403110,
    DECK_NAME
)

# Define model
model = genanki.Model(
    1607391319,
    "Korean Vocab Model",
    fields=[
        {"name": "KoreanWord"},
        {"name": "KoreanDictionaryForm"},
        {"name": "KoreanPhrase"},
        {"name": "KoreanShortExample"},
        {"name": "EnglishShort"},
        {"name": "EnglishLong"},
        {"name": "EnglishAlternate"},
        {"name": "ImageUrl"},
    ],
    templates=[
        {
            "name": "Card 1",
            "qfmt": "<h2>{{KoreanDictionaryForm}}</h2><br>{{KoreanShortExample}}",
            "afmt": "{{EnglishLong}}<br><br>Alternates: {{EnglishAlternate}}<br><br>{{ImageUrl}}",
        },
    ],
)

# Connect to DB
conn = sqlite3.connect(DB_PATH)
cursor = conn.cursor()

cursor.execute("""
    SELECT id, korean_word, korean_word_dictionary_form, korean_phrase, 
           korean_short_example, english_translation_short, english_translation_long,
           english_alternate_definitions, image_url
    FROM vocab_words
""")
rows = cursor.fetchall()

media_files = []

for row in rows:
    (word_id, korean_word, korean_dict, korean_phrase, korean_short,
     english_short, english_long, english_alt, image_url) = row

    img_tag = ""
    raw_path = os.path.join(RAW_DIR, f"{word_id}.png")
    if os.path.exists(raw_path):
        try:
            img = Image.open(raw_path)
            img = img.resize(IMAGE_SIZE, Image.LANCZOS)

            filename = f"img_{word_id}.png"
            filepath = os.path.join(MEDIA_DIR, filename)
            img.save(filepath)

            img_tag = f"<img src='{filename}'>"
            media_files.append(filepath)

        except Exception as e:
            print(f"Could not process {raw_path}: {e}")
    else:
        print(f"No raw image for word_id {word_id}, skipping")

    # Use DB id as guid so we never add a duplicate note
    note = genanki.Note(
        model=model,
        fields=[
            korean_word or "",
            korean_dict or "",
            korean_phrase or "",
            korean_short or "",
            english_short or "",
            english_long or "",
            english_alt or "",
            img_tag
        ],
        guid=genanki.guid_for("anki-builder-" + str(word_id))
    )
    deck.add_note(note)

conn.close()

# Package with media
package = genanki.Package(deck)
package.media_files = media_files
package.write_to_file(OUTPUT_FILE)

print(f"Deck exported to {OUTPUT_FILE}")
