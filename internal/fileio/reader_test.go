package fileio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFileNumber(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "standard format with underscore",
			filename: "emails_001.json",
			expected: "001",
		},
		{
			name:     "multiple underscores",
			filename: "valid_emails_42.json",
			expected: "42",
		},
		{
			name:     "no underscore",
			filename: "emails.json",
			expected: "emails",
		},
		{
			name:     "numbered only",
			filename: "123.json",
			expected: "123",
		},
		{
			name:     "full path with underscore",
			filename: "/path/to/emails_5.json",
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFileNumber(tt.filename)
			if result != tt.expected {
				t.Errorf("ExtractFileNumber(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestReadOriginalFormat(t *testing.T) {
	t.Run("reads valid original format", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "emails.json")

		content := `{"emails":[{"email":"test1@example.com"},{"email":"test2@example.com"}]}`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		emails, err := ReadOriginalFormat(inputFile)
		if err != nil {
			t.Fatalf("ReadOriginalFormat() error = %v", err)
		}

		if len(emails) != 2 {
			t.Fatalf("ReadOriginalFormat() returned %d emails, want 2", len(emails))
		}
		if emails[0] != "test1@example.com" {
			t.Errorf("emails[0] = %q, want %q", emails[0], "test1@example.com")
		}
		if emails[1] != "test2@example.com" {
			t.Errorf("emails[1] = %q, want %q", emails[1], "test2@example.com")
		}
	})

	t.Run("handles empty emails array", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "empty.json")

		content := `{"emails":[]}`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		emails, err := ReadOriginalFormat(inputFile)
		if err != nil {
			t.Fatalf("ReadOriginalFormat() error = %v", err)
		}

		if len(emails) != 0 {
			t.Errorf("ReadOriginalFormat() returned %d emails, want 0", len(emails))
		}
	})

	t.Run("skips empty email entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "with_empty.json")

		content := `{"emails":[{"email":"valid@example.com"},{"email":""},{"email":"another@example.com"}]}`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		emails, err := ReadOriginalFormat(inputFile)
		if err != nil {
			t.Fatalf("ReadOriginalFormat() error = %v", err)
		}

		if len(emails) != 2 {
			t.Errorf("ReadOriginalFormat() returned %d emails, want 2", len(emails))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := ReadOriginalFormat("/nonexistent/file.json")
		if err == nil {
			t.Error("ReadOriginalFormat() expected error for missing file")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(inputFile, []byte("not json"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := ReadOriginalFormat(inputFile)
		if err == nil {
			t.Error("ReadOriginalFormat() expected error for invalid JSON")
		}
	})

	t.Run("returns error when not starting with object", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "array.json")

		if err := os.WriteFile(inputFile, []byte(`["email@example.com"]`), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := ReadOriginalFormat(inputFile)
		if err == nil {
			t.Error("ReadOriginalFormat() expected error when JSON is array instead of object")
		}
	})
}

func TestReadSimpleFormat(t *testing.T) {
	t.Run("reads valid simple format", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "simple.json")

		content := `["test1@example.com","test2@example.com","test3@example.com"]`
		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		emails, err := ReadSimpleFormat(inputFile)
		if err != nil {
			t.Fatalf("ReadSimpleFormat() error = %v", err)
		}

		if len(emails) != 3 {
			t.Fatalf("ReadSimpleFormat() returned %d emails, want 3", len(emails))
		}
		if emails[0] != "test1@example.com" {
			t.Errorf("emails[0] = %q, want %q", emails[0], "test1@example.com")
		}
	})

	t.Run("handles empty array", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "empty.json")

		if err := os.WriteFile(inputFile, []byte(`[]`), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		emails, err := ReadSimpleFormat(inputFile)
		if err != nil {
			t.Fatalf("ReadSimpleFormat() error = %v", err)
		}

		if len(emails) != 0 {
			t.Errorf("ReadSimpleFormat() returned %d emails, want 0", len(emails))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := ReadSimpleFormat("/nonexistent/file.json")
		if err == nil {
			t.Error("ReadSimpleFormat() expected error for missing file")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(inputFile, []byte("not json"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := ReadSimpleFormat(inputFile)
		if err == nil {
			t.Error("ReadSimpleFormat() expected error for invalid JSON")
		}
	})
}

func TestFindInputFiles(t *testing.T) {
	t.Run("finds JSON files in directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := []string{"emails_1.json", "emails_2.json", "emails_10.json"}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("[]"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		result, err := FindInputFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindInputFiles() error = %v", err)
		}

		if len(result) != 3 {
			t.Errorf("FindInputFiles() returned %d files, want 3", len(result))
		}
	})

	t.Run("ignores non-JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "emails.json"), []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("csv"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := FindInputFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindInputFiles() error = %v", err)
		}

		if len(result) != 1 {
			t.Errorf("FindInputFiles() returned %d files, want 1", len(result))
		}
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(tmpDir, "emails.json"), []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		subDir := filepath.Join(tmpDir, "subdir.json")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		result, err := FindInputFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindInputFiles() error = %v", err)
		}

		if len(result) != 1 {
			t.Errorf("FindInputFiles() returned %d files, want 1", len(result))
		}
	})

	t.Run("returns sorted file list", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := []string{"z_emails.json", "a_emails.json", "m_emails.json"}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("[]"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		result, err := FindInputFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindInputFiles() error = %v", err)
		}

		expected := []string{
			filepath.Join(tmpDir, "a_emails.json"),
			filepath.Join(tmpDir, "m_emails.json"),
			filepath.Join(tmpDir, "z_emails.json"),
		}

		for i, f := range result {
			if f != expected[i] {
				t.Errorf("result[%d] = %q, want %q", i, f, expected[i])
			}
		}
	})

	t.Run("returns empty for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := FindInputFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindInputFiles() error = %v", err)
		}

		if len(result) != 0 {
			t.Errorf("FindInputFiles() returned %d files, want 0", len(result))
		}
	})

	t.Run("returns error for nonexistent directory", func(t *testing.T) {
		_, err := FindInputFiles("/nonexistent/directory")
		if err == nil {
			t.Error("FindInputFiles() expected error for nonexistent directory")
		}
	})
}

func TestOutputFilesExist(t *testing.T) {
	t.Run("returns true when both files exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		validFile := filepath.Join(tmpDir, "valid_emails_1.json")
		invalidFile := filepath.Join(tmpDir, "invalid_emails_1.json")

		if err := os.WriteFile(validFile, []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create valid file: %v", err)
		}
		if err := os.WriteFile(invalidFile, []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create invalid file: %v", err)
		}

		if !OutputFilesExist(tmpDir, "1") {
			t.Error("OutputFilesExist() = false, want true")
		}
	})

	t.Run("returns false when valid file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		invalidFile := filepath.Join(tmpDir, "invalid_emails_1.json")
		if err := os.WriteFile(invalidFile, []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create invalid file: %v", err)
		}

		if OutputFilesExist(tmpDir, "1") {
			t.Error("OutputFilesExist() = true, want false")
		}
	})

	t.Run("returns false when invalid file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		validFile := filepath.Join(tmpDir, "valid_emails_1.json")
		if err := os.WriteFile(validFile, []byte("[]"), 0644); err != nil {
			t.Fatalf("failed to create valid file: %v", err)
		}

		if OutputFilesExist(tmpDir, "1") {
			t.Error("OutputFilesExist() = true, want false")
		}
	})

	t.Run("returns false when both files missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		if OutputFilesExist(tmpDir, "1") {
			t.Error("OutputFilesExist() = true, want false")
		}
	})
}

func TestEmailEntry(t *testing.T) {
	entry := EmailEntry{Email: "test@example.com"}
	if entry.Email != "test@example.com" {
		t.Errorf("EmailEntry.Email = %q, want %q", entry.Email, "test@example.com")
	}
}
