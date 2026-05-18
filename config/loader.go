package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading configuration from multiple sources with environment variable substitution
type Loader struct {
	configPath string
	envPrefix  string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string, envPrefix string) *Loader {
	return &Loader{
		configPath: configPath,
		envPrefix:  envPrefix,
	}
}

// Load configuration from file and environment variables
// The config parameter should be a pointer to a struct
func (l *Loader) Load(config any) error {
	// Load from file if path is provided
	if l.configPath != "" {
		if err := l.loadFromFile(config); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	if err := l.loadFromEnv(config); err != nil {
		return fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Substitute environment variables in config values (${VAR_NAME} patterns)
	if err := l.substituteEnvVars(config); err != nil {
		return fmt.Errorf("failed to substitute environment variables: %w", err)
	}

	// Set defaults if the config implements SetDefaults()
	if defaultSetter, ok := config.(interface{ SetDefaults() }); ok {
		defaultSetter.SetDefaults()
	}

	return nil
}

// loadFromFile loads configuration from a YAML or JSON file
func (l *Loader) loadFromFile(config any) error {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, skip file loading
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the file content before parsing
	expandedData := l.expandEnvVar(string(data))

	ext := strings.ToLower(filepath.Ext(l.configPath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal([]byte(expandedData), config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal([]byte(expandedData), config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables using struct tags
func (l *Loader) loadFromEnv(config any) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct")
	}

	return l.loadStructFromEnv(v.Elem(), l.envPrefix)
}

// loadStructFromEnv recursively loads struct fields from environment variables
func (l *Loader) loadStructFromEnv(v reflect.Value, prefix string) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the env tag or use field name
		envKey := fieldType.Tag.Get("env")
		if envKey == "" {
			envKey = strings.ToUpper(fieldType.Name)
		}
		if envKey == "-" {
			continue
		}

		fullKey := prefix + envKey

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := l.loadStructFromEnv(field, fullKey+"_"); err != nil {
				return err
			}
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(fullKey)
		if envValue == "" {
			continue
		}

		// Set the field value
		if err := l.setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, fullKey, err)
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value from a string
func (l *Loader) setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration specially
		if field.Type().String() == "time.Duration" {
			// Try parsing as duration string (e.g., "5s", "10m")
			// For simplicity, we'll just parse as int64 nanoseconds
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// substituteEnvVars recursively substitutes ${VAR_NAME} patterns in all string fields
func (l *Loader) substituteEnvVars(config any) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	l.substituteValue(v.Elem())
	return nil
}

// substituteValue recursively substitutes environment variables in a reflect.Value
func (l *Loader) substituteValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(l.expandEnvVar(v.String()))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			l.substituteValue(v.Field(i))
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			if val.Kind() == reflect.String {
				v.SetMapIndex(key, reflect.ValueOf(l.expandEnvVar(val.String())))
			} else {
				l.substituteValue(val)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			l.substituteValue(v.Index(i))
		}
	case reflect.Ptr:
		if !v.IsNil() {
			l.substituteValue(v.Elem())
		}
	case reflect.Interface:
		if !v.IsNil() {
			l.substituteValue(v.Elem())
		}
	}
}

// expandEnvVar expands environment variable references in a string
// Supports ${VAR_NAME}, ${VAR_NAME:-default}, and $VAR_NAME patterns
func (l *Loader) expandEnvVar(s string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Z_][A-Z0-9_]*)`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name and optional default
		var varName, defaultValue string

		if strings.HasPrefix(match, "${") {
			// Handle ${VAR_NAME} or ${VAR_NAME:-default}
			content := match[2 : len(match)-1]
			if strings.Contains(content, ":-") {
				parts := strings.SplitN(content, ":-", 2)
				varName = parts[0]
				defaultValue = parts[1]
			} else {
				varName = content
			}
		} else {
			// Handle $VAR_NAME
			varName = match[1:]
		}

		// Get value from environment
		if value := os.Getenv(varName); value != "" {
			return value
		}

		// Return default if provided
		if defaultValue != "" {
			return defaultValue
		}

		// Return original if not found and no default
		return match
	})
}

// LoadFromFile is a convenience function to load config from a file
func LoadFromFile(path string, config any) error {
	loader := NewLoader(path, "")
	return loader.Load(config)
}

// LoadFromEnv is a convenience function to load config from environment variables only
func LoadFromEnv(envPrefix string, config any) error {
	loader := NewLoader("", envPrefix)
	return loader.Load(config)
}

// LoadWithDefaults loads config from file and env vars with a given prefix
func LoadWithDefaults(configPath string, envPrefix string, config any) error {
	loader := NewLoader(configPath, envPrefix)
	return loader.Load(config)
}
