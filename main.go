package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Config holds the application configuration
type Config struct {
	InputFile         string
	InputDir          string
	OutputDir         string
	Workers           int
	BatchSize         int
	RateLimit         time.Duration
	EnableSMTP        bool
	Verbose           bool
	ConvertOnly       bool
	MaxRecordsPerFile int
}

// InvalidEmail represents an email that failed verification
type InvalidEmail struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

// ValidEmail represents an email that passed verification
type ValidEmail struct {
	Email string `json:"email"`
}

// Stats tracks verification statistics
type Stats struct {
	TotalChecked int64
	TotalValid   int64
	TotalInvalid int64
	StartTime    time.Time
}

// EmailJob represents a job for the worker pool
type EmailJob struct {
	Index int
	Email string
}

// EmailResult represents the result of email verification
type EmailResult struct {
	Email   string
	IsValid bool
	Reason  string
}

const (
	inputDir                 = "input"
	outputDir                = "output"
	defaultMaxRecordsPerFile = 100000
)

func main() {
	// Load .env file if it exists
	loadEnvFile(".env")

	config := parseConfig()

	// Ensure directories exist
	if err := os.MkdirAll(config.InputDir, 0755); err != nil {
		log.Fatalf("Error creating input directory: %v", err)
	}
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	// Convert-only mode: split input file into multiple files
	if config.ConvertOnly {
		runConvertMode(config)
		return
	}

	// Verification mode: process all input files
	runVerificationMode(config)
}

// runConvertMode converts a single input file into multiple smaller files
func runConvertMode(config Config) {
	log.Printf("🔄 Convert mode: Reading input file %s", config.InputFile)

	// Read emails from input file
	emails, err := readEmailsFromOriginalFormat(config.InputFile)
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	totalEmails := len(emails)
	log.Printf("📂 Loaded %d emails from input file", totalEmails)

	// Calculate number of files needed
	numFiles := (totalEmails + config.MaxRecordsPerFile - 1) / config.MaxRecordsPerFile
	log.Printf("📁 Splitting into %d files (max %d records per file)", numFiles, config.MaxRecordsPerFile)

	// Write emails to multiple files
	for i := 0; i < numFiles; i++ {
		start := i * config.MaxRecordsPerFile
		end := start + config.MaxRecordsPerFile
		if end > totalEmails {
			end = totalEmails
		}

		chunk := emails[start:end]
		filename := filepath.Join(config.InputDir, fmt.Sprintf("input_data_%d.json", i+1))

		if err := writeEmailsToFile(filename, chunk); err != nil {
			log.Fatalf("Error writing file %s: %v", filename, err)
		}

		log.Printf("✅ Written %d emails to %s", len(chunk), filename)
	}

	log.Println("\n═══════════════════════════════════════════════════════")
	log.Printf("📊 CONVERSION COMPLETE")
	log.Printf("   Total emails: %d", totalEmails)
	log.Printf("   Files created: %d", numFiles)
	log.Printf("   Output directory: %s", config.InputDir)
	log.Println("═══════════════════════════════════════════════════════")
}

// runVerificationMode processes all input files and outputs results
func runVerificationMode(config Config) {
	// Find all input files
	inputFiles, err := findInputFiles(config.InputDir)
	if err != nil {
		log.Fatalf("Error finding input files: %v", err)
	}

	if len(inputFiles) == 0 {
		log.Fatalf("No input files found in %s. Use -convert to create input files first.", config.InputDir)
	}

	// Count how many files will be skipped
	skippedCount := 0
	for _, inputFile := range inputFiles {
		baseName := filepath.Base(inputFile)
		fileNum := extractFileNumber(baseName)
		if outputFilesExist(config.OutputDir, fileNum) {
			skippedCount++
		}
	}

	log.Printf("📁 Found %d input file(s) in %s", len(inputFiles), config.InputDir)
	if skippedCount > 0 {
		log.Printf("⏭️  %d file(s) will be skipped (output already exists)", skippedCount)
	}

	// Global stats
	globalStats := &Stats{
		StartTime: time.Now(),
	}

	// Process each input file
	filesProcessed := 0
	for _, inputFile := range inputFiles {
		baseName := filepath.Base(inputFile)
		fileNum := extractFileNumber(baseName)
		if !outputFilesExist(config.OutputDir, fileNum) {
			processInputFile(inputFile, config, globalStats)
			filesProcessed++
		} else {
			log.Printf("⏭️  Skipping %s - output files already exist", baseName)
		}
	}

	// Print global summary
	elapsed := time.Since(globalStats.StartTime)
	var emailsPerSecond float64
	if elapsed.Seconds() > 0 && globalStats.TotalChecked > 0 {
		emailsPerSecond = float64(globalStats.TotalChecked) / elapsed.Seconds()
	}

	log.Println("\n═══════════════════════════════════════════════════════")
	log.Printf("📊 ALL FILES VERIFICATION COMPLETE")
	log.Printf("   Files processed: %d", filesProcessed)
	log.Printf("   Files skipped: %d", skippedCount)
	log.Printf("   Total emails checked: %d", globalStats.TotalChecked)
	log.Printf("   Total valid emails: %d", globalStats.TotalValid)
	log.Printf("   Total invalid emails: %d", globalStats.TotalInvalid)
	log.Printf("   Total time elapsed: %v", elapsed.Round(time.Second))
	if emailsPerSecond > 0 {
		log.Printf("   Overall processing rate: %.2f emails/second", emailsPerSecond)
	}
	log.Printf("   Results saved to: %s", config.OutputDir)
	log.Println("═══════════════════════════════════════════════════════")
}

// outputFilesExist checks if output files already exist for a given file number
func outputFilesExist(outputDir, fileNum string) bool {
	validOutputFile := filepath.Join(outputDir, fmt.Sprintf("valid_emails_%s.json", fileNum))
	invalidOutputFile := filepath.Join(outputDir, fmt.Sprintf("invalid_emails_%s.json", fileNum))

	// Check if both output files exist
	_, validErr := os.Stat(validOutputFile)
	_, invalidErr := os.Stat(invalidOutputFile)

	return validErr == nil && invalidErr == nil
}

// processInputFile processes a single input file and writes results
func processInputFile(inputFile string, config Config, globalStats *Stats) {
	// Extract file number from input filename
	baseName := filepath.Base(inputFile)
	fileNum := extractFileNumber(baseName)

	log.Printf("\n📧 Processing file: %s", inputFile)

	// Read emails from input file
	emails, err := readEmailsSimpleFormat(inputFile)
	if err != nil {
		log.Printf("Error reading input file %s: %v", inputFile, err)
		return
	}

	totalEmails := len(emails)
	log.Printf("📂 Loaded %d emails from %s", totalEmails, inputFile)
	log.Printf("⚙️  Configuration: %d workers, batch size %d, rate limit %v, SMTP: %v",
		config.Workers, config.BatchSize, config.RateLimit, config.EnableSMTP)

	// Initialize stats for this file
	fileStats := &Stats{
		StartTime: time.Now(),
	}

	// Process emails
	validEmails, invalidEmails := processEmailsBatch(emails, config, fileStats, totalEmails)

	// Generate output filenames
	validOutputFile := filepath.Join(config.OutputDir, fmt.Sprintf("valid_emails_%s.json", fileNum))
	invalidOutputFile := filepath.Join(config.OutputDir, fmt.Sprintf("invalid_emails_%s.json", fileNum))

	// Write results
	if err := writeValidEmailsToFile(validOutputFile, validEmails, fileStats); err != nil {
		log.Printf("Error writing valid emails file: %v", err)
	}
	if err := writeInvalidEmailsToFile(invalidOutputFile, invalidEmails, fileStats); err != nil {
		log.Printf("Error writing invalid emails file: %v", err)
	}

	// Update global stats
	atomic.AddInt64(&globalStats.TotalChecked, fileStats.TotalChecked)
	atomic.AddInt64(&globalStats.TotalValid, fileStats.TotalValid)
	atomic.AddInt64(&globalStats.TotalInvalid, fileStats.TotalInvalid)

	// Print file summary
	elapsed := time.Since(fileStats.StartTime)
	emailsPerSecond := float64(fileStats.TotalChecked) / elapsed.Seconds()

	log.Printf("✅ File complete: %d checked, %d valid, %d invalid (%.2f/s)",
		fileStats.TotalChecked, fileStats.TotalValid, fileStats.TotalInvalid, emailsPerSecond)
	log.Printf("   Valid: %s", validOutputFile)
	log.Printf("   Invalid: %s", invalidOutputFile)
}

// extractFileNumber extracts the file number from filename like "input_data_1.json"
func extractFileNumber(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Try to extract number after last underscore
	parts := strings.Split(name, "_")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "1"
}

// findInputFiles finds all JSON files in the input directory
func findInputFiles(dir string) ([]string, error) {
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

	// Sort files to process in order
	sort.Strings(files)

	return files, nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// .env file is optional, don't error if it doesn't exist
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already set (command line/environment takes precedence)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// getEnvString returns environment variable or default value
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		value = strings.ToLower(value)
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseConfig() Config {
	// Default values from environment variables
	defaultWorkers := getEnvInt("WORKERS", runtime.NumCPU()*2)
	defaultBatchSize := getEnvInt("BATCH_SIZE", 1000)
	defaultRateLimit := getEnvDuration("RATE_LIMIT", 10*time.Millisecond)
	defaultEnableSMTP := getEnvBool("ENABLE_SMTP", true)
	defaultVerbose := getEnvBool("VERBOSE", false)
	defaultConvertOnly := getEnvBool("CONVERT_ONLY", false)
	defaultMaxRecordsPerFile := getEnvInt("MAX_RECORDS_PER_FILE", defaultMaxRecordsPerFile)
	defaultInputFile := getEnvString("INPUT_FILE", "data/data.json")
	defaultInputDir := getEnvString("INPUT_DIR", inputDir)
	defaultOutputDir := getEnvString("OUTPUT_DIR", outputDir)

	config := Config{}

	// Command line flags (override environment variables)
	flag.StringVar(&config.InputFile, "input", defaultInputFile, "Input JSON file for convert mode")
	flag.StringVar(&config.InputDir, "input-dir", defaultInputDir, "Input directory for verification mode")
	flag.StringVar(&config.OutputDir, "output-dir", defaultOutputDir, "Output directory for results")
	flag.IntVar(&config.Workers, "workers", defaultWorkers, "Number of concurrent workers")
	flag.IntVar(&config.BatchSize, "batch", defaultBatchSize, "Batch size for progress reporting")
	flag.DurationVar(&config.RateLimit, "rate", defaultRateLimit, "Rate limit between verifications per worker")
	flag.BoolVar(&config.EnableSMTP, "smtp", defaultEnableSMTP, "Enable SMTP verification (disable with -smtp=false if blocked by ISP)")
	flag.BoolVar(&config.Verbose, "verbose", defaultVerbose, "Enable verbose logging")
	flag.BoolVar(&config.ConvertOnly, "convert", defaultConvertOnly, "Convert input file to multiple smaller files (no verification)")
	flag.IntVar(&config.MaxRecordsPerFile, "max-records", defaultMaxRecordsPerFile, "Maximum records per file in convert mode")

	flag.Parse()

	return config
}

// processEmailsBatch processes emails and collects results in memory
func processEmailsBatch(emails []string, config Config, stats *Stats, totalEmails int) ([]string, []InvalidEmail) {
	// Create channels
	jobs := make(chan EmailJob, config.Workers*2)
	results := make(chan EmailResult, config.Workers*2)

	// Create worker pool
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, config, &wg)
	}

	// Start result collector
	var validEmails []string
	var invalidEmails []InvalidEmail
	var mu sync.Mutex
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)

	go func() {
		defer collectorWg.Done()
		lastReport := time.Now()

		for result := range results {
			mu.Lock()
			if result.IsValid {
				atomic.AddInt64(&stats.TotalValid, 1)
				validEmails = append(validEmails, result.Email)
			} else {
				atomic.AddInt64(&stats.TotalInvalid, 1)
				invalidEmails = append(invalidEmails, InvalidEmail{Email: result.Email, Reason: result.Reason})
			}
			mu.Unlock()

			checked := atomic.AddInt64(&stats.TotalChecked, 1)

			// Progress reporting every batch or every 5 seconds
			if checked%int64(config.BatchSize) == 0 || time.Since(lastReport) > 5*time.Second {
				elapsed := time.Since(stats.StartTime)
				rate := float64(checked) / elapsed.Seconds()
				remaining := totalEmails - int(checked)
				eta := time.Duration(float64(remaining)/rate) * time.Second

				log.Printf("📈 Progress: %d/%d (%.1f%%) | Rate: %.1f/s | ETA: %v | Valid: %d | Invalid: %d",
					checked, totalEmails,
					float64(checked)/float64(totalEmails)*100,
					rate,
					eta.Round(time.Second),
					atomic.LoadInt64(&stats.TotalValid),
					atomic.LoadInt64(&stats.TotalInvalid))
				lastReport = time.Now()
			}
		}
	}()

	// Send jobs to workers
	for i, email := range emails {
		jobs <- EmailJob{Index: i, Email: email}
	}
	close(jobs)

	// Wait for workers to finish
	wg.Wait()
	close(results)

	// Wait for collector to finish
	collectorWg.Wait()

	return validEmails, invalidEmails
}

// writeValidEmailsToFile writes valid emails to a JSON file
func writeValidEmailsToFile(filename string, emails []string, stats *Stats) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024) // 1MB buffer
	defer writer.Flush()

	// Write the array directly
	dataJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	writer.Write(dataJSON)
	writer.WriteString("\n")

	return nil
}

// writeInvalidEmailsToFile writes invalid emails to a JSON file
func writeInvalidEmailsToFile(filename string, emails []InvalidEmail, stats *Stats) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024) // 1MB buffer
	defer writer.Flush()

	// Write header
	writer.WriteString("{\n")
	writer.WriteString("  \"invalid_emails\": ")

	// Marshal and write the array
	dataJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	writer.Write(dataJSON)
	writer.WriteString(",\n")

	// Write stats
	fmt.Fprintf(writer, "  \"checked_at\": %q,\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "  \"total_checked\": %d,\n", stats.TotalChecked)
	fmt.Fprintf(writer, "  \"total_valid\": %d,\n", stats.TotalValid)
	fmt.Fprintf(writer, "  \"total_invalid\": %d,\n", stats.TotalInvalid)
	fmt.Fprintf(writer, "  \"processing_time_seconds\": %.2f\n", time.Since(stats.StartTime).Seconds())
	writer.WriteString("}\n")

	return nil
}

// writeEmailsToFile writes emails to a JSON file in simple array format
func writeEmailsToFile(filename string, emails []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024) // 1MB buffer
	defer writer.Flush()

	// Write as simple array: ["email1", "email2", ...]
	dataJSON, err := json.Marshal(emails)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	writer.Write(dataJSON)
	writer.WriteString("\n")

	return nil
}

func worker(id int, jobs <-chan EmailJob, results chan<- EmailResult, config Config, wg *sync.WaitGroup) {
	defer wg.Done()

	// Each worker gets its own verifier instance
	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest().
		EnableAutoUpdateDisposable()

	if config.EnableSMTP {
		verifier = verifier.EnableSMTPCheck()
	}

	for job := range jobs {
		result := verifyEmail(verifier, job.Email, config.Verbose)
		results <- result

		// Rate limiting per worker
		if config.RateLimit > 0 {
			time.Sleep(config.RateLimit)
		}
	}
}

func verifyEmail(verifier *emailverifier.Verifier, email string, verbose bool) EmailResult {
	result, err := verifier.Verify(email)
	if err != nil {
		reason := fmt.Sprintf("verification error: %v", err)
		if verbose {
			log.Printf("  ❌ %s - %s", email, reason)
		}
		// Marking these as valid because it means the server responded and bounced us for spamming, which means it should be valid
		return EmailResult{Email: email, IsValid: true, Reason: reason}
	}

	isValid, reason := evaluateResult(result)

	if verbose {
		if isValid {
			log.Printf("  ✅ %s", email)
		} else {
			log.Printf("  ❌ %s - %s", email, reason)
		}
	}

	return EmailResult{Email: email, IsValid: isValid, Reason: reason}
}

// evaluateResult checks the verification result and returns validity status and reason
func evaluateResult(result *emailverifier.Result) (bool, string) {
	// Check syntax first
	if !result.Syntax.Valid {
		return false, "invalid email syntax"
	}

	// Check if it's a disposable email
	if result.Disposable {
		return false, "disposable email address"
	}

	// Check domain suggestion (typo detection)
	if result.Suggestion != "" {
		return false, fmt.Sprintf("possible typo, did you mean: %s", result.Suggestion)
	}

	// Check if MX records exist
	if !result.HasMxRecords {
		return false, "domain has no MX records"
	}

	// Check SMTP result if available
	if result.SMTP != nil {
		if !result.SMTP.HostExists {
			return false, "SMTP host does not exist"
		}
		if !result.SMTP.Deliverable {
			return false, "email is not deliverable"
		}
		if result.SMTP.Disabled {
			return false, "mailbox is disabled"
		}
	}

	// Check reachability
	if result.Reachable == "no" {
		return false, "email is not reachable"
	}

	return true, ""
}

// EmailEntry represents an email entry in the original input JSON format
type EmailEntry struct {
	Email string `json:"email"`
}

// readEmailsFromOriginalFormat reads emails from the original format: {"emails": [{"email": "..."}]}
func readEmailsFromOriginalFormat(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	// Get file size for pre-allocation estimate
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Estimate capacity: assume average email entry is ~40 bytes + JSON overhead
	estimatedCapacity := stat.Size() / 45
	if estimatedCapacity < 100 {
		estimatedCapacity = 100
	}
	if estimatedCapacity > 10_000_000 {
		estimatedCapacity = 10_000_000
	}

	emails := make([]string, 0, estimatedCapacity)

	decoder := json.NewDecoder(bufio.NewReaderSize(file, 1024*1024)) // 1MB buffer

	// Read opening brace
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON: %w", err)
	}
	if token != json.Delim('{') {
		return nil, fmt.Errorf("expected object start, got %v", token)
	}

	// Read until we find "emails" key
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read token: %w", err)
		}

		if key, ok := token.(string); ok && key == "emails" {
			// Read the array
			token, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("failed to read array start: %w", err)
			}
			if token != json.Delim('[') {
				return nil, fmt.Errorf("expected array start, got %v", token)
			}

			// Read each email entry object
			for decoder.More() {
				var entry EmailEntry
				if err := decoder.Decode(&entry); err != nil {
					return nil, fmt.Errorf("failed to decode email entry: %w", err)
				}
				if entry.Email != "" {
					emails = append(emails, entry.Email)
				}
			}

			// Read array end
			if _, err := decoder.Token(); err != nil {
				return nil, fmt.Errorf("failed to read array end: %w", err)
			}
			break
		}
	}

	return emails, nil
}

// readEmailsSimpleFormat reads emails from simple array format: ["email1", "email2", ...]
func readEmailsSimpleFormat(filename string) ([]string, error) {
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
