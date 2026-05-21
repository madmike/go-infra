package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestConfig is a test struct for loader tests.
type TestConfig struct {
	AppName    string        `env:"APP_NAME"`
	Port       int           `env:"PORT"`
	Debug      bool          `env:"DEBUG"`
	Timeout    time.Duration `env:"TIMEOUT"`
	Database   DBConfig      `env:"DB_"`
	LogLevel   string        // Uses field name by default
	SkipField  string        `env:"-"`
}

type DBConfig struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT"`
}

// TestLoaderLoadFromEnv loads config from environment variables.
func TestLoaderLoadFromEnv(t *testing.T) {
	oldAppName := os.Getenv("TEST_APP_NAME")
	oldPort := os.Getenv("TEST_PORT")
	defer func() {
		if oldAppName != "" {
			os.Setenv("TEST_APP_NAME", oldAppName)
		} else {
			os.Unsetenv("TEST_APP_NAME")
		}
		if oldPort != "" {
			os.Setenv("TEST_PORT", oldPort)
		} else {
			os.Unsetenv("TEST_PORT")
		}
	}()

	os.Setenv("TEST_APP_NAME", "myapp")
	os.Setenv("TEST_PORT", "8080")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "myapp", cfg.AppName)
	require.Equal(t, 8080, cfg.Port)
}

// TestLoaderNestedStruct loads nested struct fields.
func TestLoaderNestedStruct(t *testing.T) {
	oldHost := os.Getenv("TEST_DB_HOST")
	oldPort := os.Getenv("TEST_DB_PORT")
	defer func() {
		if oldHost != "" {
			os.Setenv("TEST_DB_HOST", oldHost)
		} else {
			os.Unsetenv("TEST_DB_HOST")
		}
		if oldPort != "" {
			os.Setenv("TEST_DB_PORT", oldPort)
		} else {
			os.Unsetenv("TEST_DB_PORT")
		}
	}()

	os.Setenv("TEST_DB_HOST", "localhost")
	os.Setenv("TEST_DB_PORT", "5432")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Database.Host)
	require.Equal(t, 5432, cfg.Database.Port)
}

// TestLoaderDefaultFieldName uses field name when no env tag.
func TestLoaderDefaultFieldName(t *testing.T) {
	oldLogLevel := os.Getenv("TEST_LOGLEVEL")
	defer func() {
		if oldLogLevel != "" {
			os.Setenv("TEST_LOGLEVEL", oldLogLevel)
		} else {
			os.Unsetenv("TEST_LOGLEVEL")
		}
	}()

	os.Setenv("TEST_LOGLEVEL", "debug")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "debug", cfg.LogLevel)
}

// TestLoaderSkippedFields with "-" tag are skipped.
func TestLoaderSkippedFields(t *testing.T) {
	oldSkip := os.Getenv("TEST_SKIPFIELD")
	defer func() {
		if oldSkip != "" {
			os.Setenv("TEST_SKIPFIELD", oldSkip)
		} else {
			os.Unsetenv("TEST_SKIPFIELD")
		}
	}()

	os.Setenv("TEST_SKIPFIELD", "should-be-ignored")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "", cfg.SkipField)
}

// TestLoaderBooleanConversion parses boolean values.
func TestLoaderBooleanConversion(t *testing.T) {
	oldDebug := os.Getenv("TEST_DEBUG")
	defer func() {
		if oldDebug != "" {
			os.Setenv("TEST_DEBUG", oldDebug)
		} else {
			os.Unsetenv("TEST_DEBUG")
		}
	}()

	os.Setenv("TEST_DEBUG", "true")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.True(t, cfg.Debug)
}

// TestLoaderIntegerConversion parses integer values.
func TestLoaderIntegerConversion(t *testing.T) {
	oldPort := os.Getenv("TEST_PORT")
	defer func() {
		if oldPort != "" {
			os.Setenv("TEST_PORT", oldPort)
		} else {
			os.Unsetenv("TEST_PORT")
		}
	}()

	os.Setenv("TEST_PORT", "9000")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, 9000, cfg.Port)
}

// TestLoaderMissingFileSkipped missing config file is not an error.
func TestLoaderMissingFileSkipped(t *testing.T) {
	cfg := &TestConfig{
		AppName: "default",
		Port:    8080,
	}

	loader := NewLoader("/nonexistent/config.yaml", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	// Config should still have defaults
	require.Equal(t, "default", cfg.AppName)
}

// TestLoaderInvalidFileFormat rejects unsupported formats.
func TestLoaderInvalidFileFormat(t *testing.T) {
	tempFile := "/tmp/config.txt"
	os.WriteFile(tempFile, []byte("invalid"), 0644)
	defer os.Remove(tempFile)

	cfg := &TestConfig{}
	loader := NewLoader(tempFile, "TEST_")

	err := loader.Load(cfg)
	require.Error(t, err)
}

// TestLoaderNonPointerConfig rejects non-pointer config.
func TestLoaderNonPointerConfig(t *testing.T) {
	loader := NewLoader("", "TEST_")

	cfg := TestConfig{}
	err := loader.Load(cfg)
	require.Error(t, err)
}

// TestNewLoader creates loader with correct fields.
func TestNewLoader(t *testing.T) {
	loader := NewLoader("/etc/config.yaml", "APP_")

	require.NotNil(t, loader)
	require.Equal(t, "/etc/config.yaml", loader.configPath)
	require.Equal(t, "APP_", loader.envPrefix)
}

// TestLoaderEmptyPrefix uses no prefix.
func TestLoaderEmptyPrefix(t *testing.T) {
	oldName := os.Getenv("APPNAME")
	defer func() {
		if oldName != "" {
			os.Setenv("APPNAME", oldName)
		} else {
			os.Unsetenv("APPNAME")
		}
	}()

	os.Setenv("APPNAME", "testapp")

	cfg := &TestConfig{}
	loader := NewLoader("", "")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "testapp", cfg.AppName)
}

// TestLoaderEnvPrefixCombined uses prefix + field name.
func TestLoaderEnvPrefixCombined(t *testing.T) {
	oldName := os.Getenv("SERVICE_APP_NAME")
	defer func() {
		if oldName != "" {
			os.Setenv("SERVICE_APP_NAME", oldName)
		} else {
			os.Unsetenv("SERVICE_APP_NAME")
		}
	}()

	os.Setenv("SERVICE_APP_NAME", "myservice")

	cfg := &TestConfig{}
	loader := NewLoader("", "SERVICE_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "myservice", cfg.AppName)
}

// TestLoaderPartialConfig loads only specified fields.
func TestLoaderPartialConfig(t *testing.T) {
	oldPort := os.Getenv("TEST_PORT")
	defer func() {
		if oldPort != "" {
			os.Setenv("TEST_PORT", oldPort)
		} else {
			os.Unsetenv("TEST_PORT")
		}
	}()

	os.Setenv("TEST_PORT", "3000")

	cfg := &TestConfig{
		AppName: "preset",
	}

	loader := NewLoader("", "TEST_")
	err := loader.Load(cfg)

	require.NoError(t, err)
	require.Equal(t, "preset", cfg.AppName)
	require.Equal(t, 3000, cfg.Port)
}

// TestLoaderInvalidIntegerValue rejects non-integer for int field.
func TestLoaderInvalidIntegerValue(t *testing.T) {
	oldPort := os.Getenv("TEST_PORT")
	defer func() {
		if oldPort != "" {
			os.Setenv("TEST_PORT", oldPort)
		} else {
			os.Unsetenv("TEST_PORT")
		}
	}()

	os.Setenv("TEST_PORT", "not-a-number")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.Error(t, err)
}

// TestLoaderMultipleFields loads multiple fields.
func TestLoaderMultipleFields(t *testing.T) {
	oldName := os.Getenv("TEST_APP_NAME")
	oldPort := os.Getenv("TEST_PORT")
	oldDebug := os.Getenv("TEST_DEBUG")
	defer func() {
		os.Unsetenv("TEST_APP_NAME")
		os.Unsetenv("TEST_PORT")
		os.Unsetenv("TEST_DEBUG")
		if oldName != "" {
			os.Setenv("TEST_APP_NAME", oldName)
		}
		if oldPort != "" {
			os.Setenv("TEST_PORT", oldPort)
		}
		if oldDebug != "" {
			os.Setenv("TEST_DEBUG", oldDebug)
		}
	}()

	os.Setenv("TEST_APP_NAME", "multitest")
	os.Setenv("TEST_PORT", "4000")
	os.Setenv("TEST_DEBUG", "false")

	cfg := &TestConfig{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "multitest", cfg.AppName)
	require.Equal(t, 4000, cfg.Port)
	require.False(t, cfg.Debug)
}

// TestLoaderUnexportedFieldSkipped ignores unexported fields.
type TestConfigWithPrivate struct {
	Public  string `env:"PUBLIC"`
	private string `env:"PRIVATE"`
}

func TestLoaderUnexportedFieldSkipped(t *testing.T) {
	oldPublic := os.Getenv("TEST_PUBLIC")
	oldPrivate := os.Getenv("TEST_PRIVATE")
	defer func() {
		os.Unsetenv("TEST_PUBLIC")
		os.Unsetenv("TEST_PRIVATE")
		if oldPublic != "" {
			os.Setenv("TEST_PUBLIC", oldPublic)
		}
		if oldPrivate != "" {
			os.Setenv("TEST_PRIVATE", oldPrivate)
		}
	}()

	os.Setenv("TEST_PUBLIC", "visible")
	os.Setenv("TEST_PRIVATE", "hidden")

	cfg := &TestConfigWithPrivate{}
	loader := NewLoader("", "TEST_")

	err := loader.Load(cfg)
	require.NoError(t, err)
	require.Equal(t, "visible", cfg.Public)
	require.Equal(t, "", cfg.private) // Private field unchanged
}
