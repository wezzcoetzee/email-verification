package fileio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"email-verification/internal/verifier"
)

func WriteEmails(filename string, emails []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024)
	defer writer.Flush()

	dataJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	writer.Write(dataJSON)
	writer.WriteString("\n")

	return nil
}

func WriteValidEmails(filename string, emails []string) error {
	return WriteEmails(filename, emails)
}

func WriteInvalidEmails(filename string, emails []verifier.InvalidEmail, stats *verifier.Stats) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024)
	defer writer.Flush()

	writer.WriteString("{\n")
	writer.WriteString("  \"invalid_emails\": ")

	dataJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	writer.Write(dataJSON)
	writer.WriteString(",\n")

	fmt.Fprintf(writer, "  \"checked_at\": %q,\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "  \"total_checked\": %d,\n", stats.TotalChecked)
	fmt.Fprintf(writer, "  \"total_valid\": %d,\n", stats.TotalValid)
	fmt.Fprintf(writer, "  \"total_invalid\": %d,\n", stats.TotalInvalid)
	fmt.Fprintf(writer, "  \"processing_time_seconds\": %.2f\n", time.Since(stats.StartTime).Seconds())
	writer.WriteString("}\n")

	return nil
}
