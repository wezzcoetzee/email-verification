package verifier

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

type InvalidEmail struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

type Stats struct {
	TotalChecked int64
	TotalValid   int64
	TotalInvalid int64
	StartTime    time.Time
}

type Job struct {
	Index int
	Email string
}

type Result struct {
	Email   string
	IsValid bool
	Reason  string
}

type WorkerConfig struct {
	Workers    int
	BatchSize  int
	RateLimit  time.Duration
	EnableSMTP bool
	Verbose    bool
}

func ProcessBatch(emails []string, cfg WorkerConfig, stats *Stats, totalEmails int) ([]string, []InvalidEmail) {
	jobs := make(chan Job, cfg.Workers*2)
	results := make(chan Result, cfg.Workers*2)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, cfg, &wg)
	}

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

			if checked%int64(cfg.BatchSize) == 0 || time.Since(lastReport) > 5*time.Second {
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

	for i, email := range emails {
		jobs <- Job{Index: i, Email: email}
	}
	close(jobs)

	wg.Wait()
	close(results)

	collectorWg.Wait()

	return validEmails, invalidEmails
}

func worker(id int, jobs <-chan Job, results chan<- Result, cfg WorkerConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest().
		EnableAutoUpdateDisposable()

	if cfg.EnableSMTP {
		verifier = verifier.EnableSMTPCheck()
	}

	for job := range jobs {
		result := verify(verifier, job.Email, cfg.Verbose)
		results <- result

		if cfg.RateLimit > 0 {
			time.Sleep(cfg.RateLimit)
		}
	}
}

func verify(verifier *emailverifier.Verifier, email string, verbose bool) Result {
	result, err := verifier.Verify(email)
	if err != nil {
		reason := fmt.Sprintf("verification error: %v", err)
		if verbose {
			log.Printf("  ❌ %s - %s", email, reason)
		}
		return Result{Email: email, IsValid: true, Reason: reason}
	}

	isValid, reason := evaluateResult(result)

	if verbose {
		if isValid {
			log.Printf("  ✅ %s", email)
		} else {
			log.Printf("  ❌ %s - %s", email, reason)
		}
	}

	return Result{Email: email, IsValid: isValid, Reason: reason}
}

func evaluateResult(result *emailverifier.Result) (bool, string) {
	if !result.Syntax.Valid {
		return false, "invalid email syntax"
	}

	if result.Disposable {
		return false, "disposable email address"
	}

	if result.Suggestion != "" {
		return false, fmt.Sprintf("possible typo, did you mean: %s", result.Suggestion)
	}

	if !result.HasMxRecords {
		return false, "domain has no MX records"
	}

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

	if result.Reachable == "no" {
		return false, "email is not reachable"
	}

	return true, ""
}
