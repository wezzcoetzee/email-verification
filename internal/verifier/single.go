package verifier

import (
	"fmt"
	"sync"

	emailverifier "github.com/AfterShip/email-verifier"
)

type VerifyOptions struct {
	EnableSMTP bool
}

type VerifyResult struct {
	Email  string `json:"email"`
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
}

func VerifySingle(email string, opts VerifyOptions) VerifyResult {
	v := emailverifier.NewVerifier().
		EnableDomainSuggest().
		EnableAutoUpdateDisposable()

	if opts.EnableSMTP {
		v = v.EnableSMTPCheck()
	}

	result, err := v.Verify(email)
	if err != nil {
		return VerifyResult{
			Email:  email,
			Valid:  true,
			Reason: fmt.Sprintf("verification error: %v", err),
		}
	}

	isValid, reason := evaluateResult(result)
	return VerifyResult{
		Email:  email,
		Valid:  isValid,
		Reason: reason,
	}
}

func VerifyMultiple(emails []string, opts VerifyOptions) []VerifyResult {
	if len(emails) == 0 {
		return []VerifyResult{}
	}

	if len(emails) <= 10 {
		return verifySequential(emails, opts)
	}

	return verifyWithWorkerPool(emails, opts)
}

func verifySequential(emails []string, opts VerifyOptions) []VerifyResult {
	results := make([]VerifyResult, len(emails))
	for i, email := range emails {
		results[i] = VerifySingle(email, opts)
	}
	return results
}

func verifyWithWorkerPool(emails []string, opts VerifyOptions) []VerifyResult {
	numWorkers := 10
	if len(emails) < numWorkers {
		numWorkers = len(emails)
	}

	type indexedResult struct {
		index  int
		result VerifyResult
	}

	jobs := make(chan int, len(emails))
	resultsChan := make(chan indexedResult, len(emails))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := emailverifier.NewVerifier().
				EnableDomainSuggest().
				EnableAutoUpdateDisposable()

			if opts.EnableSMTP {
				v = v.EnableSMTPCheck()
			}

			for idx := range jobs {
				email := emails[idx]
				result, err := v.Verify(email)

				var vr VerifyResult
				if err != nil {
					vr = VerifyResult{
						Email:  email,
						Valid:  true,
						Reason: fmt.Sprintf("verification error: %v", err),
					}
				} else {
					isValid, reason := evaluateResult(result)
					vr = VerifyResult{
						Email:  email,
						Valid:  isValid,
						Reason: reason,
					}
				}

				resultsChan <- indexedResult{index: idx, result: vr}
			}
		}()
	}

	for i := range emails {
		jobs <- i
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make([]VerifyResult, len(emails))
	for ir := range resultsChan {
		results[ir.index] = ir.result
	}

	return results
}
