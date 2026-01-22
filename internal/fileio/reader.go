package fileio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EmailEntry struct {
	Email string `json:"email"`
}

func ReadOriginalFormat(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	estimatedCapacity := stat.Size() / 45
	if estimatedCapacity < 100 {
		estimatedCapacity = 100
	}
	if estimatedCapacity > 10_000_000 {
		estimatedCapacity = 10_000_000
	}

	emails := make([]string, 0, estimatedCapacity)

	decoder := json.NewDecoder(bufio.NewReaderSize(file, 1024*1024))

	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON: %w", err)
	}
	if token != json.Delim('{') {
		return nil, fmt.Errorf("expected object start, got %v", token)
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read token: %w", err)
		}

		if key, ok := token.(string); ok && key == "emails" {
			token, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("failed to read array start: %w", err)
			}
			if token != json.Delim('[') {
				return nil, fmt.Errorf("expected array start, got %v", token)
			}

			for decoder.More() {
				var entry EmailEntry
				if err := decoder.Decode(&entry); err != nil {
					return nil, fmt.Errorf("failed to decode email entry: %w", err)
				}
				if entry.Email != "" {
					emails = append(emails, entry.Email)
				}
			}

			if _, err := decoder.Token(); err != nil {
				return nil, fmt.Errorf("failed to read array end: %w", err)
			}
			break
		}
	}

	return emails, nil
}

func ReadSimpleFormat(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	var emails []string
	decoder := json.NewDecoder(bufio.NewReaderSize(file, 1024*1024))
	if err := decoder.Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode emails: %w", err)
	}

	return emails, nil
}

func FindInputFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)

	return files, nil
}

func ExtractFileNumber(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(name, "_")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "1"
}

func OutputFilesExist(outputDir, fileNum string) bool {
	validOutputFile := filepath.Join(outputDir, fmt.Sprintf("valid_emails_%s.json", fileNum))
	invalidOutputFile := filepath.Join(outputDir, fmt.Sprintf("invalid_emails_%s.json", fileNum))

	_, validErr := os.Stat(validOutputFile)
	_, invalidErr := os.Stat(invalidOutputFile)

	return validErr == nil && invalidErr == nil
}
