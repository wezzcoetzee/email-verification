# Email Checker

A high-performance Go-based email verification system that validates email addresses using the [AfterShip email-verifier](https://github.com/AfterShip/email-verifier) library. Designed to handle **millions of emails** efficiently.

## Features

- ✅ **High Performance** - Concurrent worker pool for parallel processing
- ✅ **Scalable** - Handles 1M+ emails with configurable workers
- ✅ **Memory Efficient** - Streaming JSON read/write
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
cd email-checker

# Download dependencies
make deps
# or:
go mod tidy
```

## Quick Start

```bash
# 1. Copy env.example to .env and configure (optional)
cp env.example .env

# 2. Add your emails to data/data.json

# 3. Run the checker
make run

# 4. Check data/invalid_emails.json for results
```

## Configuration

The application can be configured via environment variables, a `.env` file, or command line flags.

**Priority order:** Command line flags > Environment variables > `.env` file > Defaults

### Environment Variables

Copy `env.example` to `.env` and adjust as needed:

```bash
cp env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `INPUT_FILE` | `data/data.json` | Input JSON file with emails |
| `OUTPUT_FILE` | `data/invalid_emails.json` | Output JSON file for invalid emails |
| `WORKERS` | `2x CPU cores` | Number of concurrent workers |
| `BATCH_SIZE` | `1000` | Progress report frequency |
| `RATE_LIMIT` | `10ms` | Rate limit between verifications per worker |
| `ENABLE_SMTP` | `true` | Enable SMTP verification |
| `VERBOSE` | `false` | Enable verbose logging |

### Example `.env` file

```bash
# High performance settings
WORKERS=32
BATCH_SIZE=5000
RATE_LIMIT=0
ENABLE_SMTP=false

# Or conservative settings
WORKERS=8
RATE_LIMIT=100ms
ENABLE_SMTP=true
```

## Usage

### Command Line Options

```bash
./email-checker [options]

Options:
  -input string     Input JSON file with emails (default "data/data.json")
  -output string    Output JSON file for invalid emails (default "data/invalid_emails.json")
  -workers int      Number of concurrent workers (default: 2x CPU cores)
  -batch int        Batch size for progress reporting (default: 1000)
  -rate duration    Rate limit between verifications per worker (default: 10ms)
  -smtp             Enable SMTP verification (may be blocked by ISP)
  -verbose          Enable verbose logging (logs each email result)
```

### Using Make (Recommended)

```bash
# Show all available commands
make help

# Run with default settings (16 workers, 10ms rate limit)
make run

# Run at maximum speed (32 workers, no rate limiting)
make run-fast

# Run with SMTP verification
make run-smtp

# Run with verbose logging (shows each email result)
make run-verbose

# Build optimized binary
make build

# Run the compiled binary
make run-build

# Clean up
make clean
```

### Using Go Directly

```bash
# Default settings
go run main.go

# Custom input/output files
go run main.go -input=data/my_emails.json -output=data/results.json

# High performance mode (32 workers, no rate limiting)
go run main.go -workers=32 -rate=0

# With SMTP verification
go run main.go -smtp

# Verbose mode
go run main.go -verbose
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

| Mode | Workers | Rate Limit | Estimated Speed | Use Case |
|------|---------|------------|-----------------|----------|
| Fast | 32 | 0 | ~1000/sec | Syntax + MX only |
| Balanced | 16 | 10ms | ~500/sec | Production use |
| Safe | 8 | 50ms | ~150/sec | Avoid rate limiting |
| SMTP | 8 | 100ms | ~50/sec | Full verification |

## Input Format

Create a `data/data.json` file with an array of emails:

```json
{
  "emails": [
    "user1@example.com",
    "user2@gmail.com",
    "invalid-email",
    "test@nonexistent-domain.com"
  ]
}
```

## Output

### Console Progress

```
2025/12/30 10:00:00 📧 Starting email verification for 1000000 emails...
2025/12/30 10:00:00 ⚙️  Configuration: 16 workers, batch size 1000, rate limit 10ms
2025/12/30 10:00:00 📂 Loaded 1000000 emails from data/data.json
2025/12/30 10:00:05 📈 Progress: 5000/1000000 (0.5%) | Rate: 1000.0/s | ETA: 16m35s | Invalid: 250
2025/12/30 10:00:10 📈 Progress: 10000/1000000 (1.0%) | Rate: 1000.0/s | ETA: 16m30s | Invalid: 502
...
2025/12/30 10:16:40 ═══════════════════════════════════════════════════════
2025/12/30 10:16:40 📊 VERIFICATION COMPLETE
2025/12/30 10:16:40    Total emails checked: 1000000
2025/12/30 10:16:40    Valid emails: 850000
2025/12/30 10:16:40    Invalid emails: 150000
2025/12/30 10:16:40    Time elapsed: 16m40s
2025/12/30 10:16:40    Processing rate: 1000.00 emails/second
2025/12/30 10:16:40    Results saved to: data/invalid_emails.json
2025/12/30 10:16:40 ═══════════════════════════════════════════════════════
```

### JSON Output (`data/invalid_emails.json`)

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
  "total_checked": 1000000,
  "total_valid": 850000,
  "total_invalid": 150000,
  "processing_time_seconds": 1000.50
}
```

## Validation Checks

| Check | Description | Requires SMTP |
|-------|-------------|---------------|
| Syntax | Validates email format | No |
| MX Records | Checks if domain has mail exchange records | No |
| Disposable | Detects temporary/disposable email providers | No |
| Typo Detection | Suggests corrections for common domain typos | No |
| SMTP | Verifies mailbox exists | Yes |
| Deliverability | Checks if email can receive messages | Yes |

## Project Structure

```
email-checker/
├── main.go             # Main application logic
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
├── Makefile            # Build and run commands
├── README.md           # This file
├── env.example         # Example environment configuration
├── .env                # Your local configuration (create from env.example)
└── data/               # Data directory for input/output
    ├── data.json           # Input file (emails to verify)
    └── invalid_emails.json # Output file (generated)
```

## Memory Usage

The application is optimized for large datasets:

- **Streaming JSON parsing** - Doesn't load entire file into memory at once
- **Buffered I/O** - 1MB buffers for efficient disk access
- **Pre-allocated slices** - Reduces GC pressure
- **Worker pool** - Fixed number of goroutines

For 1 million emails, expect ~200-500MB RAM usage depending on email lengths.

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

### Out of Memory

For very large datasets (10M+):
- Process in batches by splitting input file
- Reduce workers to limit concurrent memory usage

## License

MIT
