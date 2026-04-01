// Package config handles loading and validating structql configuration.
package config

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ScalarConfig defines how a custom GraphQL scalar maps to a Go type.
type ScalarConfig struct {
	Type   string `yaml:"type"`
	Import string `yaml:"import,omitempty"`
}

// Config represents the structql configuration file.
type Config struct {
	Schema       []string                `yaml:"schema"`
	Queries      string                  `yaml:"queries"`
	Output       string                  `yaml:"output"`
	Package      string                  `yaml:"package"`
	Scalars      map[string]ScalarConfig `yaml:"scalars,omitempty"`
	TypeMappings map[string]string       `yaml:"type_mappings,omitempty"`
	Prefix       string                  `yaml:"prefix,omitempty"`

	// basedir is the directory containing the config file, used for resolving relative paths.
	basedir string
}

// Load reads and parses a config file, resolving paths relative to the config file location.
func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.basedir = filepath.Dir(absPath)

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.resolvePaths()

	return &cfg, nil
}

// DefaultConfig returns a config with sensible defaults for scaffolding.
func DefaultConfig() *Config {
	return &Config{
		Schema:  []string{"schema.graphql"},
		Queries: "queries/",
		Output:  "generated/",
		Package: "generated",
		Scalars: map[string]ScalarConfig{},
	}
}

// Marshal serializes the config to YAML bytes.
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

func (c *Config) validate() error {
	if len(c.Schema) == 0 {
		return fmt.Errorf("schema: at least one schema file or pattern is required")
	}
	if c.Queries == "" {
		return fmt.Errorf("queries: query directory is required")
	}
	if c.Output == "" {
		return fmt.Errorf("output: output directory is required")
	}
	if c.Package == "" {
		c.Package = filepath.Base(c.Output)
	}
	if !token.IsIdentifier(c.Package) {
		return fmt.Errorf("package: %q is not a valid Go identifier", c.Package)
	}
	for name, sc := range c.Scalars {
		if sc.Type == "" {
			return fmt.Errorf("scalars.%s: type is required", name)
		}
	}
	return nil
}

func (c *Config) resolvePaths() {
	for i, s := range c.Schema {
		if !filepath.IsAbs(s) {
			c.Schema[i] = filepath.Join(c.basedir, s)
		}
	}
	if !filepath.IsAbs(c.Queries) {
		c.Queries = filepath.Join(c.basedir, c.Queries)
	}
	if !filepath.IsAbs(c.Output) {
		c.Output = filepath.Join(c.basedir, c.Output)
	}
}
