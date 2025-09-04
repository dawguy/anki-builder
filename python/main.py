import sqlite3
import genanki
import requests
import os
from PIL import Image
from io import BytesIO

# --- CONFIG ---
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB_PATH = os.path.join(BASE_DIR, "vocab.db")
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
        {"name": "KoreanPhrase"},
        {"name": "EnglishTranslation"},
        {"name": "ImageUrl"},
    ],
    templates=[
        {
            "name": "Card 1",
            "qfmt": "{{KoreanWord}}<br><br>{{KoreanPhrase}}",
            "afmt": "{{EnglishTranslation}}<br><br>{{ImageUrl}}",
        },
    ],
)

# Connect DB
conn = sqlite3.connect(DB_PATH)
cursor = conn.cursor()

cursor.execute("SELECT id, korean_word, korean_phrase, english_translation, image_url FROM vocab_words")
rows = cursor.fetchall()

media_files = []

for row in rows:
    word_id, korean_word, korean_phrase, english_translation, image_url = row

    img_tag = ""
    if image_url:
        try:
            ext = os.path.splitext(image_url)[-1] or ".jpg"
            filename = f"img_{word_id}{ext}"
            filepath = os.path.join(MEDIA_DIR, filename)

            if not os.path.exists(filepath):
                print(f"Getting image {image_url}")
                r = requests.get(image_url, timeout=10)
                if r.status_code == 200:
                    img = Image.open(BytesIO(r.content))
                    img = img.resize(IMAGE_SIZE, Image.LANCZOS)
                    img.save(filepath)

            img_tag = f"<img src='{filename}'>"
            media_files.append(filepath)

        except Exception as e:
            print(f"Could not process {image_url}: {e}")

    # Use DB id as guid so we never add a duplicate note
    note = genanki.Note(
        model=model,
        fields=[korean_word or "", korean_phrase or "", english_translation or "", img_tag],
        guid=genanki.guid_for("anki-builder-" + str(word_id))
    )
    deck.add_note(note)

conn.close()

# Package with media
package = genanki.Package(deck)
package.media_files = media_files
package.write_to_file(OUTPUT_FILE)

print(f"Deck exported to {OUTPUT_FILE}")

