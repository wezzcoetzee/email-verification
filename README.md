# Email Verification

A high-performance Go-based email verification system that validates email addresses using the [AfterShip email-verifier](https://github.com/AfterShip/email-verifier) library. Designed to handle **millions of emails** efficiently.

## Features

- ✅ **High Performance** - Concurrent worker pool for parallel processing
- ✅ **Scalable** - Handles 1M+ emails with configurable workers
- ✅ **Batch Processing** - Automatically processes multiple input files
- ✅ **Convert Mode** - Split large files into manageable chunks (100k records per file)
- ✅ **Progress Tracking** - Real-time progress, rate, and ETA
- ✅ Syntax validation
- ✅ MX record checking
- ✅ SMTP verification (optional)
- ✅ Disposable email detection
- ✅ Domain typo suggestions
- ✅ Rate limiting to avoid blocks

## Prerequisites

- [Go](https://golang.org/dl/) 1.21 or higher
- Make (optional, for using Makefile commands)

## Installation

```bash
cd email-verification

# Download dependencies
make deps
# or:
go mod tidy
```

## Quick Start

### Step 1: Convert your input data

If you have a large input file in the original format (`{"emails": [{"email": "..."}]}`), first convert it to the new format:

```bash
# Convert data/data.json into multiple files in input/ folder
go run main.go -convert -input=data/data.json

# This creates:
#   input/input_data_1.json (up to 100,000 emails)
#   input/input_data_2.json (up to 100,000 emails)
#   ...
```

### Step 2: Run verification

```bash
# Process all files in input/ folder
go run main.go

# Results are saved to output/ folder:
#   output/valid_emails_1.json
#   output/invalid_emails_1.json
#   output/valid_emails_2.json
#   output/invalid_emails_2.json
#   ...
```

## Configuration

The application can be configured via environment variables, a `.env` file, or command line flags.

**Priority order:** Command line flags > Environment variables > `.env` file > Defaults

### Environment Variables

Copy `env.example` to `.env` and adjust as needed:

```bash
cp env.example .env
```

| Variable              | Default         | Description                                      |
| --------------------- | --------------- | ------------------------------------------------ |
| `INPUT_FILE`          | `data/data.json`| Input file for convert mode                      |
| `INPUT_DIR`           | `input`         | Directory containing input files for verification|
| `OUTPUT_DIR`          | `output`        | Directory for output files                       |
| `CONVERT_ONLY`        | `false`         | Run in convert mode (no verification)            |
| `MAX_RECORDS_PER_FILE`| `100000`        | Max emails per file in convert mode              |
| `WORKERS`             | `2x CPU cores`  | Number of concurrent workers                     |
| `BATCH_SIZE`          | `1000`          | Progress report frequency                        |
| `RATE_LIMIT`          | `10ms`          | Rate limit between verifications per worker      |
| `ENABLE_SMTP`         | `true`          | Enable SMTP verification                         |
| `VERBOSE`             | `false`         | Enable verbose logging                           |

### Example `.env` file

```bash
# High performance settings
WORKERS=150
BATCH_SIZE=5000
RATE_LIMIT=1ms
ENABLE_SMTP=true

# Or conservative settings
WORKERS=8
RATE_LIMIT=100ms
ENABLE_SMTP=true

# Convert mode settings
MAX_RECORDS_PER_FILE=100000
```

## Usage

### Command Line Options

```bash
./email-verification [options]

Options:
  -input string       Input JSON file for convert mode (default "data/data.json")
  -input-dir string   Input directory for verification mode (default "input")
  -output-dir string  Output directory for results (default "output")
  -convert            Convert input file to multiple smaller files (no verification)
  -max-records int    Maximum records per file in convert mode (default: 100000)
  -workers int        Number of concurrent workers (default: 2x CPU cores)
  -batch int          Batch size for progress reporting (default: 1000)
  -rate duration      Rate limit between verifications per worker (default: 10ms)
  -smtp               Enable SMTP verification (may be blocked by ISP)
  -verbose            Enable verbose logging (logs each email result)
```

### Convert Mode

Convert a large input file into multiple smaller files:

```bash
# Convert with default settings (100k records per file)
go run main.go -convert -input=data/data.json

# Convert with custom max records
go run main.go -convert -input=data/data.json -max-records=50000

# Output files are created in input/ directory:
#   input/input_data_1.json
#   input/input_data_2.json
#   ...
```

### Verification Mode

Process all input files and generate results:

```bash
# Process all files in input/ directory
go run main.go

# Use custom directories
go run main.go -input-dir=my_input -output-dir=my_output

# High performance mode
go run main.go -workers=150 -rate=1ms

# With SMTP verification
go run main.go -smtp

# Verbose mode
go run main.go -verbose
```

### Using Make (Recommended)

```bash
# Show all available commands
make help

# Run with default settings
make run

# Run at maximum speed (32 workers, no rate limiting)
make run-fast

# Run with SMTP verification
make run-smtp

# Run with verbose logging
make run-verbose

# Build optimized binary
make build

# Clean up
make clean
```

### Performance Tuning

For **1 million emails**, recommended settings:

```bash
# Fast mode (syntax + MX only, ~1000 emails/sec)
go run main.go -workers=32 -rate=0

# Balanced mode (with rate limiting to avoid blocks)
go run main.go -workers=16 -rate=10ms

# With SMTP verification (slower, ~50-100 emails/sec)
go run main.go -workers=8 -rate=100ms -smtp
```

| Mode     | Workers | Rate Limit | Estimated Speed | Use Case            |
| -------- | ------- | ---------- | --------------- | ------------------- |
| Fast     | 32      | 0          | ~1000/sec       | Syntax + MX only    |
| Balanced | 16      | 10ms       | ~500/sec        | Production use      |
| Safe     | 8       | 50ms       | ~150/sec        | Avoid rate limiting |
| SMTP     | 8       | 100ms      | ~50/sec         | Full verification   |

## Input/Output Formats

### Original Input Format (for convert mode)

The original input file format (`data/data.json`):

```json
{
  "emails": [
    {"email": "user1@example.com"},
    {"email": "user2@gmail.com"},
    {"email": "invalid-email"},
    {"email": "test@nonexistent-domain.com"}
  ]
}
```

### Converted Input Format (for verification)

After conversion, input files use a simple array format (`input/input_data_1.json`):

```json
["user1@example.com", "user2@gmail.com", "user3@example.com"]
```

### Output Format

**Valid emails (`output/valid_emails_1.json`):**

```json
["user1@example.com", "user2@gmail.com"]
```

**Invalid emails (`output/invalid_emails_1.json`):**

```json
{
  "invalid_emails": [
    {
      "email": "invalid-email",
      "reason": "invalid email syntax"
    },
    {
      "email": "test@gmai.com",
      "reason": "possible typo, did you mean: gmail.com"
    }
  ],
  "checked_at": "2025-12-30T10:16:40Z",
  "total_checked": 100000,
  "total_valid": 85000,
  "total_invalid": 15000,
  "processing_time_seconds": 100.5
}
```

## Console Output

### Convert Mode

```
2025/12/30 10:00:00 🔄 Convert mode: Reading input file data/data.json
2025/12/30 10:00:05 📂 Loaded 500000 emails from input file
2025/12/30 10:00:05 📁 Splitting into 5 files (max 100000 records per file)
2025/12/30 10:00:06 ✅ Written 100000 emails to input/input_data_1.json
2025/12/30 10:00:07 ✅ Written 100000 emails to input/input_data_2.json
2025/12/30 10:00:08 ✅ Written 100000 emails to input/input_data_3.json
2025/12/30 10:00:09 ✅ Written 100000 emails to input/input_data_4.json
2025/12/30 10:00:10 ✅ Written 100000 emails to input/input_data_5.json

═══════════════════════════════════════════════════════
📊 CONVERSION COMPLETE
   Total emails: 500000
   Files created: 5
   Output directory: input
═══════════════════════════════════════════════════════
```

### Verification Mode

```
2025/12/30 10:00:00 📁 Found 5 input file(s) in input

📧 Processing file: input/input_data_1.json
2025/12/30 10:00:00 📂 Loaded 100000 emails from input/input_data_1.json
2025/12/30 10:00:00 ⚙️  Configuration: 16 workers, batch size 1000, rate limit 10ms, SMTP: true
2025/12/30 10:00:05 📈 Progress: 5000/100000 (5.0%) | Rate: 1000.0/s | ETA: 1m35s | Valid: 4750 | Invalid: 250
...
2025/12/30 10:01:40 ✅ File complete: 100000 checked, 85000 valid, 15000 invalid (1000.00/s)
   Valid: output/valid_emails_1.json
   Invalid: output/invalid_emails_1.json

📧 Processing file: input/input_data_2.json
...

═══════════════════════════════════════════════════════
📊 ALL FILES VERIFICATION COMPLETE
   Total emails checked: 500000
   Total valid emails: 425000
   Total invalid emails: 75000
   Total time elapsed: 8m20s
   Overall processing rate: 1000.00 emails/second
   Results saved to: output
═══════════════════════════════════════════════════════
```

## Validation Checks

| Check          | Description                                  | Requires SMTP |
| -------------- | -------------------------------------------- | ------------- |
| Syntax         | Validates email format                       | No            |
| MX Records     | Checks if domain has mail exchange records   | No            |
| Disposable     | Detects temporary/disposable email providers | No            |
| Typo Detection | Suggests corrections for common domain typos | No            |
| SMTP           | Verifies mailbox exists                      | Yes           |
| Deliverability | Checks if email can receive messages         | Yes           |

## Project Structure

```
email-verification/
├── main.go             # Main application logic
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
├── Makefile            # Build and run commands
├── README.md           # This file
├── env.example         # Example environment configuration
├── .env                # Your local configuration (create from env.example)
├── data/               # Original data directory
│   └── data.json           # Original input file (for convert mode)
├── input/              # Input directory (created by convert mode)
│   ├── input_data_1.json   # Input file 1 (up to 100k emails)
│   ├── input_data_2.json   # Input file 2 (up to 100k emails)
│   └── ...
└── output/             # Output directory (created by verification)
    ├── valid_emails_1.json     # Valid emails from input_data_1
    ├── invalid_emails_1.json   # Invalid emails from input_data_1
    ├── valid_emails_2.json     # Valid emails from input_data_2
    ├── invalid_emails_2.json   # Invalid emails from input_data_2
    └── ...
```

## Memory Usage

The application is optimized for large datasets:

- **Batch file processing** - Process one file at a time to limit memory usage
- **Buffered I/O** - 1MB buffers for efficient disk access
- **Pre-allocated slices** - Reduces GC pressure
- **Worker pool** - Fixed number of goroutines
- **100k records per file** - Manageable chunk sizes

For 100,000 emails per file, expect ~100-200MB RAM usage per file being processed.

## Troubleshooting

### SMTP Verification Hangs or Times Out

Most ISPs block port 25. Options:

- Set `ENABLE_SMTP=false` in `.env` or use `-smtp=false` flag
- Use a VPS where port 25 is open
- Use a SOCKS5 proxy

### Rate Limiting / Connection Refused

If you're getting many errors:

- Increase `-rate` value (e.g., `-rate=100ms`)
- Decrease `-workers` count
- Some mail servers block bulk verification

### No Input Files Found

Make sure to run convert mode first to create input files:

```bash
go run main.go -convert -input=data/data.json
```

### Out of Memory

If processing very large files:

- Reduce `MAX_RECORDS_PER_FILE` to create smaller chunks
- Reduce workers to limit concurrent memory usage
