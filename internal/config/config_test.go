package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns env value when set",
			envKey:       "TEST_STRING_VAR",
			envValue:     "custom_value",
			defaultValue: "default",
			expected:     "custom_value",
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_STRING_UNSET",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvString(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "returns env value when valid int",
			envKey:       "TEST_INT_VAR",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_INT_UNSET",
			envValue:     "",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "returns default when env invalid",
			envKey:       "TEST_INT_INVALID",
			envValue:     "not_a_number",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvInt(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvInt() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "returns true for 'true'",
			envKey:       "TEST_BOOL_TRUE",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns true for 'TRUE' (case insensitive)",
			envKey:       "TEST_BOOL_TRUE_UPPER",
			envValue:     "TRUE",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns true for '1'",
			envKey:       "TEST_BOOL_ONE",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns true for 'yes'",
			envKey:       "TEST_BOOL_YES",
			envValue:     "yes",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns true for 'YES' (case insensitive)",
			envKey:       "TEST_BOOL_YES_UPPER",
			envValue:     "YES",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "returns false for 'false'",
			envKey:       "TEST_BOOL_FALSE",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "returns false for '0'",
			envKey:       "TEST_BOOL_ZERO",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_BOOL_UNSET",
			envValue:     "",
			defaultValue: true,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvBool(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "returns env value when valid duration",
			envKey:       "TEST_DURATION_VAR",
			envValue:     "100ms",
			defaultValue: 10 * time.Millisecond,
			expected:     100 * time.Millisecond,
		},
		{
			name:         "returns env value for seconds",
			envKey:       "TEST_DURATION_SEC",
			envValue:     "5s",
			defaultValue: 10 * time.Millisecond,
			expected:     5 * time.Second,
		},
		{
			name:         "returns default when env not set",
			envKey:       "TEST_DURATION_UNSET",
			envValue:     "",
			defaultValue: 10 * time.Millisecond,
			expected:     10 * time.Millisecond,
		},
		{
			name:         "returns default when env invalid",
			envKey:       "TEST_DURATION_INVALID",
			envValue:     "not_a_duration",
			defaultValue: 10 * time.Millisecond,
			expected:     10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.envKey)
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvDuration(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	t.Run("loads env file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `# Comment line
TEST_LOAD_STRING=hello
TEST_LOAD_INT=42

TEST_LOAD_BOOL=true
`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test env file: %v", err)
		}

		os.Unsetenv("TEST_LOAD_STRING")
		os.Unsetenv("TEST_LOAD_INT")
		os.Unsetenv("TEST_LOAD_BOOL")
		defer func() {
			os.Unsetenv("TEST_LOAD_STRING")
			os.Unsetenv("TEST_LOAD_INT")
			os.Unsetenv("TEST_LOAD_BOOL")
		}()

		LoadEnvFile(envFile)

		if got := os.Getenv("TEST_LOAD_STRING"); got != "hello" {
			t.Errorf("TEST_LOAD_STRING = %q, want %q", got, "hello")
		}
		if got := os.Getenv("TEST_LOAD_INT"); got != "42" {
			t.Errorf("TEST_LOAD_INT = %q, want %q", got, "42")
		}
		if got := os.Getenv("TEST_LOAD_BOOL"); got != "true" {
			t.Errorf("TEST_LOAD_BOOL = %q, want %q", got, "true")
		}
	})

	t.Run("does not override existing env vars", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `TEST_EXISTING=from_file`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test env file: %v", err)
		}

		os.Setenv("TEST_EXISTING", "from_env")
		defer os.Unsetenv("TEST_EXISTING")

		LoadEnvFile(envFile)

		if got := os.Getenv("TEST_EXISTING"); got != "from_env" {
			t.Errorf("TEST_EXISTING = %q, want %q (should not override)", got, "from_env")
		}
	})

	t.Run("handles missing file gracefully", func(t *testing.T) {
		LoadEnvFile("/nonexistent/path/.env")
	})

	t.Run("ignores malformed lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")

		content := `VALID_KEY=valid_value
malformed_line_without_equals
ANOTHER_VALID=another_value
`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test env file: %v", err)
		}

		os.Unsetenv("VALID_KEY")
		os.Unsetenv("ANOTHER_VALID")
		defer func() {
			os.Unsetenv("VALID_KEY")
			os.Unsetenv("ANOTHER_VALID")
		}()

		LoadEnvFile(envFile)

		if got := os.Getenv("VALID_KEY"); got != "valid_value" {
			t.Errorf("VALID_KEY = %q, want %q", got, "valid_value")
		}
		if got := os.Getenv("ANOTHER_VALID"); got != "another_value" {
			t.Errorf("ANOTHER_VALID = %q, want %q", got, "another_value")
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	if DefaultInputDir != "input" {
		t.Errorf("DefaultInputDir = %q, want %q", DefaultInputDir, "input")
	}
	if DefaultOutputDir != "output" {
		t.Errorf("DefaultOutputDir = %q, want %q", DefaultOutputDir, "output")
	}
	if DefaultMaxRecordsPerFile != 100000 {
		t.Errorf("DefaultMaxRecordsPerFile = %d, want %d", DefaultMaxRecordsPerFile, 100000)
	}
}
