package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

// Config holds the application configuration
type Config struct {
	InputFile  string
	OutputFile string
	Workers    int
	BatchSize  int
	RateLimit  time.Duration
	EnableSMTP bool
	Verbose    bool
}

// InvalidEmail represents an email that failed verification
type InvalidEmail struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
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

const dataDir = "data"

func main() {
	// Load .env file if it exists
	loadEnvFile(".env")

	config := parseConfig()

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Error creating data directory: %v", err)
	}

	// Read emails from input file
	emails, err := readEmailsStreaming(config.InputFile)
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	totalEmails := len(emails)
	log.Printf("📧 Starting email verification for %d emails...", totalEmails)
	log.Printf("⚙️  Configuration: %d workers, batch size %d, rate limit %v, SMTP: %v",
		config.Workers, config.BatchSize, config.RateLimit, config.EnableSMTP)

	// Initialize stats
	stats := &Stats{
		StartTime: time.Now(),
	}

	// Process emails concurrently
	invalidEmails := processEmails(emails, config, stats)

	// Write results
	if err := writeResultsStreaming(config.OutputFile, invalidEmails, stats); err != nil {
		log.Fatalf("Error writing output file: %v", err)
	}

	// Print summary
	elapsed := time.Since(stats.StartTime)
	emailsPerSecond := float64(stats.TotalChecked) / elapsed.Seconds()

	log.Println("\n═══════════════════════════════════════════════════════")
	log.Printf("📊 VERIFICATION COMPLETE")
	log.Printf("   Total emails checked: %d", stats.TotalChecked)
	log.Printf("   Valid emails: %d", stats.TotalValid)
	log.Printf("   Invalid emails: %d", stats.TotalInvalid)
	log.Printf("   Time elapsed: %v", elapsed.Round(time.Second))
	log.Printf("   Processing rate: %.2f emails/second", emailsPerSecond)
	log.Printf("   Results saved to: %s", config.OutputFile)
	log.Println("═══════════════════════════════════════════════════════")
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
	defaultInputFile := getEnvString("INPUT_FILE", dataDir+"/data.json")
	defaultOutputFile := getEnvString("OUTPUT_FILE", dataDir+"/invalid_emails.json")

	config := Config{}

	// Command line flags (override environment variables)
	flag.StringVar(&config.InputFile, "input", defaultInputFile, "Input JSON file with emails")
	flag.StringVar(&config.OutputFile, "output", defaultOutputFile, "Output JSON file for invalid emails")
	flag.IntVar(&config.Workers, "workers", defaultWorkers, "Number of concurrent workers")
	flag.IntVar(&config.BatchSize, "batch", defaultBatchSize, "Batch size for progress reporting")
	flag.DurationVar(&config.RateLimit, "rate", defaultRateLimit, "Rate limit between verifications per worker")
	flag.BoolVar(&config.EnableSMTP, "smtp", defaultEnableSMTP, "Enable SMTP verification (disable with -smtp=false if blocked by ISP)")
	flag.BoolVar(&config.Verbose, "verbose", defaultVerbose, "Enable verbose logging")

	flag.Parse()

	// Override with positional arguments for backwards compatibility
	args := flag.Args()
	if len(args) > 0 {
		config.InputFile = args[0]
	}
	if len(args) > 1 {
		config.OutputFile = args[1]
	}

	return config
}

func processEmails(emails []string, config Config, stats *Stats) []InvalidEmail {
	totalEmails := len(emails)

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
	var invalidEmails []InvalidEmail
	var invalidMu sync.Mutex
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)

	go func() {
		defer collectorWg.Done()
		lastReport := time.Now()

		for result := range results {
			if result.IsValid {
				atomic.AddInt64(&stats.TotalValid, 1)
			} else {
				atomic.AddInt64(&stats.TotalInvalid, 1)
				invalidMu.Lock()
				invalidEmails = append(invalidEmails, InvalidEmail{
					Email:  result.Email,
					Reason: result.Reason,
				})
				invalidMu.Unlock()
			}

			checked := atomic.AddInt64(&stats.TotalChecked, 1)

			// Progress reporting every batch or every 5 seconds
			if checked%int64(config.BatchSize) == 0 || time.Since(lastReport) > 5*time.Second {
				elapsed := time.Since(stats.StartTime)
				rate := float64(checked) / elapsed.Seconds()
				remaining := totalEmails - int(checked)
				eta := time.Duration(float64(remaining)/rate) * time.Second

				log.Printf("📈 Progress: %d/%d (%.1f%%) | Rate: %.1f/s | ETA: %v | Invalid: %d",
					checked, totalEmails,
					float64(checked)/float64(totalEmails)*100,
					rate,
					eta.Round(time.Second),
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

	return invalidEmails
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
		return EmailResult{Email: email, IsValid: false, Reason: reason}
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

// readEmailsStreaming reads emails from JSON file using streaming for memory efficiency
func readEmailsStreaming(filename string) ([]string, error) {
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

	// Estimate capacity: assume average email is ~30 bytes + JSON overhead
	estimatedCapacity := stat.Size() / 35
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

			// Read each email
			for decoder.More() {
				var email string
				if err := decoder.Decode(&email); err != nil {
					return nil, fmt.Errorf("failed to decode email: %w", err)
				}
				emails = append(emails, email)
			}

			// Read array end
			if _, err := decoder.Token(); err != nil {
				return nil, fmt.Errorf("failed to read array end: %w", err)
			}
			break
		}
	}

	log.Printf("📂 Loaded %d emails from %s", len(emails), filename)
	return emails, nil
}

// writeResultsStreaming writes results using streaming for memory efficiency
func writeResultsStreaming(filename string, invalidEmails []InvalidEmail, stats *Stats) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 1024*1024) // 1MB buffer
	defer writer.Flush()

	// Write header
	writer.WriteString("{\n")
	writer.WriteString("  \"invalid_emails\": [\n")

	// Write each invalid email
	for i, email := range invalidEmails {
		emailJSON, err := json.Marshal(email)
		if err != nil {
			return fmt.Errorf("failed to marshal email: %w", err)
		}

		writer.WriteString("    ")
		writer.Write(emailJSON)
		if i < len(invalidEmails)-1 {
			writer.WriteString(",")
		}
		writer.WriteString("\n")
	}

	// Write footer with stats
	writer.WriteString("  ],\n")
	fmt.Fprintf(writer, "  \"checked_at\": %q,\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "  \"total_checked\": %d,\n", stats.TotalChecked)
	fmt.Fprintf(writer, "  \"total_valid\": %d,\n", stats.TotalValid)
	fmt.Fprintf(writer, "  \"total_invalid\": %d,\n", stats.TotalInvalid)
	fmt.Fprintf(writer, "  \"processing_time_seconds\": %.2f\n", time.Since(stats.StartTime).Seconds())
	writer.WriteString("}\n")

	return nil
}
