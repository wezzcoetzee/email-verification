package verifier

import (
	"testing"

	emailverifier "github.com/AfterShip/email-verifier"
)

func TestEvaluateResult(t *testing.T) {
	tests := []struct {
		name           string
		result         *emailverifier.Result
		expectedValid  bool
		expectedReason string
	}{
		{
			name: "valid email",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				Reachable:    "unknown",
			},
			expectedValid:  true,
			expectedReason: "",
		},
		{
			name: "invalid syntax",
			result: &emailverifier.Result{
				Syntax: emailverifier.Syntax{Valid: false},
			},
			expectedValid:  false,
			expectedReason: "invalid email syntax",
		},
		{
			name: "disposable email",
			result: &emailverifier.Result{
				Syntax:     emailverifier.Syntax{Valid: true},
				Disposable: true,
			},
			expectedValid:  false,
			expectedReason: "disposable email address",
		},
		{
			name: "typo suggestion",
			result: &emailverifier.Result{
				Syntax:     emailverifier.Syntax{Valid: true},
				Disposable: false,
				Suggestion: "gmail.com",
			},
			expectedValid:  false,
			expectedReason: "possible typo, did you mean: gmail.com",
		},
		{
			name: "no MX records",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: false,
			},
			expectedValid:  false,
			expectedReason: "domain has no MX records",
		},
		{
			name: "SMTP host does not exist",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				SMTP:         &emailverifier.SMTP{HostExists: false},
			},
			expectedValid:  false,
			expectedReason: "SMTP host does not exist",
		},
		{
			name: "email not deliverable",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				SMTP:         &emailverifier.SMTP{HostExists: true, Deliverable: false},
			},
			expectedValid:  false,
			expectedReason: "email is not deliverable",
		},
		{
			name: "mailbox disabled",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				SMTP:         &emailverifier.SMTP{HostExists: true, Deliverable: true, Disabled: true},
			},
			expectedValid:  false,
			expectedReason: "mailbox is disabled",
		},
		{
			name: "email not reachable",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				Reachable:    "no",
			},
			expectedValid:  false,
			expectedReason: "email is not reachable",
		},
		{
			name: "valid with SMTP check passed",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				Disposable:   false,
				HasMxRecords: true,
				SMTP:         &emailverifier.SMTP{HostExists: true, Deliverable: true, Disabled: false},
				Reachable:    "yes",
			},
			expectedValid:  true,
			expectedReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, reason := evaluateResult(tt.result)
			if valid != tt.expectedValid {
				t.Errorf("evaluateResult() valid = %v, want %v", valid, tt.expectedValid)
			}
			if reason != tt.expectedReason {
				t.Errorf("evaluateResult() reason = %q, want %q", reason, tt.expectedReason)
			}
		})
	}
}

func TestResult(t *testing.T) {
	r := Result{
		Email:   "test@example.com",
		IsValid: true,
		Reason:  "",
	}

	if r.Email != "test@example.com" {
		t.Errorf("Result.Email = %q, want %q", r.Email, "test@example.com")
	}
	if !r.IsValid {
		t.Error("Result.IsValid = false, want true")
	}
}

func TestInvalidEmail(t *testing.T) {
	ie := InvalidEmail{
		Email:  "bad@invalid.com",
		Reason: "invalid syntax",
	}

	if ie.Email != "bad@invalid.com" {
		t.Errorf("InvalidEmail.Email = %q, want %q", ie.Email, "bad@invalid.com")
	}
	if ie.Reason != "invalid syntax" {
		t.Errorf("InvalidEmail.Reason = %q, want %q", ie.Reason, "invalid syntax")
	}
}

func TestWorkerConfig(t *testing.T) {
	cfg := WorkerConfig{
		Workers:    10,
		BatchSize:  1000,
		RateLimit:  0,
		EnableSMTP: true,
		Verbose:    false,
	}

	if cfg.Workers != 10 {
		t.Errorf("WorkerConfig.Workers = %d, want %d", cfg.Workers, 10)
	}
	if cfg.BatchSize != 1000 {
		t.Errorf("WorkerConfig.BatchSize = %d, want %d", cfg.BatchSize, 1000)
	}
}

func TestStats(t *testing.T) {
	stats := Stats{}

	if stats.TotalChecked != 0 {
		t.Errorf("Stats.TotalChecked = %d, want 0", stats.TotalChecked)
	}
	if stats.TotalValid != 0 {
		t.Errorf("Stats.TotalValid = %d, want 0", stats.TotalValid)
	}
	if stats.TotalInvalid != 0 {
		t.Errorf("Stats.TotalInvalid = %d, want 0", stats.TotalInvalid)
	}
}

func TestJob(t *testing.T) {
	job := Job{
		Index: 5,
		Email: "test@example.com",
	}

	if job.Index != 5 {
		t.Errorf("Job.Index = %d, want %d", job.Index, 5)
	}
	if job.Email != "test@example.com" {
		t.Errorf("Job.Email = %q, want %q", job.Email, "test@example.com")
	}
}
