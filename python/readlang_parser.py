import csv
import argparse
import re
import string

def clean_text(text):
    """Remove [[...]] markup and all punctuation from text."""
    # Remove [[...]] first
    text = re.sub(r'\[\[(.*?)\]\]', r'\1', text)
    # Remove all punctuation
    translator = str.maketrans('', '', string.punctuation)
    text = text.translate(translator)
    return text

def read_first_file(file1_path):
    """Read the first CSV (comma-delimited) and return a set of keys (punctuation removed)."""
    keys = set()
    with open(file1_path, 'r', encoding='utf-8') as f:
        reader = csv.reader(f)
        for row in reader:
            if row:
                key = row[0].strip()
                key = key.translate(str.maketrans('', '', string.punctuation))
                keys.add(key)
    return keys

def process_second_file(file2_path, existing_keys):
    """Read the second CSV, clean it, deduplicate, and skip keys already in first file."""
    seen = set()
    rows = []

    with open(file2_path, 'r', encoding='utf-8') as f:
        reader = csv.reader(f, delimiter=';')
        for row in reader:
            if not row:
                continue
            key = clean_text(row[0].strip())
            if key in seen or key in existing_keys:
                continue
            seen.add(key)
            value = clean_text(row[1].strip()) if len(row) > 1 else ""
            rows.append([key, value])

    return rows

def main():
    parser = argparse.ArgumentParser(description="Create DIFF.csv with cleaned, deduplicated rows from second CSV, removing punctuation.")
    parser.add_argument("file1", help="First CSV file (comma-delimited)")
    parser.add_argument("file2", help="Second CSV file (; delimited with [[...]] markup)")
    args = parser.parse_args()

    existing_keys = read_first_file(args.file1)
    processed_rows = process_second_file(args.file2, existing_keys)

    # Write output as comma-delimited, matching format of first CSV
    with open("DIFF.csv", "w", encoding='utf-8', newline='') as f:
        writer = csv.writer(f)
        for row in processed_rows:
            writer.writerow([row[0],row[1]])

    print(f"Processed {len(processed_rows)} unique new rows. Output written to DIFF.csv")

if __name__ == "__main__":
    main()

