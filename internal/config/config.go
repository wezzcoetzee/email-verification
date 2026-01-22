package config

import (
	"bufio"
	"flag"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultInputDir          = "input"
	DefaultOutputDir         = "output"
	DefaultMaxRecordsPerFile = 100000
)

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

func LoadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		value = strings.ToLower(value)
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func Parse() Config {
	defaultWorkers := getEnvInt("WORKERS", runtime.NumCPU()*2)
	defaultBatchSize := getEnvInt("BATCH_SIZE", 1000)
	defaultRateLimit := getEnvDuration("RATE_LIMIT", 10*time.Millisecond)
	defaultEnableSMTP := getEnvBool("ENABLE_SMTP", true)
	defaultVerbose := getEnvBool("VERBOSE", false)
	defaultConvertOnly := getEnvBool("CONVERT_ONLY", false)
	defaultMaxRecordsPerFile := getEnvInt("MAX_RECORDS_PER_FILE", DefaultMaxRecordsPerFile)
	defaultInputFile := getEnvString("INPUT_FILE", "data/data.json")
	defaultInputDir := getEnvString("INPUT_DIR", DefaultInputDir)
	defaultOutputDir := getEnvString("OUTPUT_DIR", DefaultOutputDir)

	cfg := Config{}

	flag.StringVar(&cfg.InputFile, "input", defaultInputFile, "Input JSON file for convert mode")
	flag.StringVar(&cfg.InputDir, "input-dir", defaultInputDir, "Input directory for verification mode")
	flag.StringVar(&cfg.OutputDir, "output-dir", defaultOutputDir, "Output directory for results")
	flag.IntVar(&cfg.Workers, "workers", defaultWorkers, "Number of concurrent workers")
	flag.IntVar(&cfg.BatchSize, "batch", defaultBatchSize, "Batch size for progress reporting")
	flag.DurationVar(&cfg.RateLimit, "rate", defaultRateLimit, "Rate limit between verifications per worker")
	flag.BoolVar(&cfg.EnableSMTP, "smtp", defaultEnableSMTP, "Enable SMTP verification (disable with -smtp=false if blocked by ISP)")
	flag.BoolVar(&cfg.Verbose, "verbose", defaultVerbose, "Enable verbose logging")
	flag.BoolVar(&cfg.ConvertOnly, "convert", defaultConvertOnly, "Convert input file to multiple smaller files (no verification)")
	flag.IntVar(&cfg.MaxRecordsPerFile, "max-records", defaultMaxRecordsPerFile, "Maximum records per file in convert mode")

	flag.Parse()

	return cfg
}
