#!/usr/bin/env bash
# Collect all Go files and copy them to clipboard with a custom header message

HEADER="Give me back only the part of the code that should be adjusted."

echo "Collecting .go files..."
{
  echo "$HEADER"
  echo
  find . -type f -name "*.go" \
    -not -path "./vendor/*" \
    -not -path "./testdata/*" \
    -not -name "*.pb.go" \
    -not -name "*_mock.go" \
    -not -name "*_gen.go" \
    -not -name "zz_generated*.go" \
  | sort | while read -r file; do
      echo "===== $file ====="
      cat "$file"
      echo -e "\n"
    done
} | wl-copy

echo "✅ All .go files (with header) copied to clipboard!"
