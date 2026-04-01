// structql generates type-safe Go code for hasura/go-graphql-client from GraphQL schema and query files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nseba/structql/internal/codegen"
	"github.com/nseba/structql/internal/config"
	"github.com/nseba/structql/internal/output"
	"github.com/nseba/structql/internal/query"
	"github.com/nseba/structql/internal/schema"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("structql %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `structql - Generate Go code for hasura/go-graphql-client

Usage:
  structql <command> [flags]

Commands:
  generate    Generate Go code from GraphQL schema and queries
  init        Create a default structql.yaml configuration file
  version     Print the version
  help        Show this help message

Flags (generate):
  --config    Path to config file (default: structql.yaml)
`)
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	configPath := fs.String("config", "structql.yaml", "path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	sch, err := schema.Load(cfg.Schema)
	if err != nil {
		return fmt.Errorf("loading schema: %w", err)
	}

	ops, err := query.Load(cfg.Queries, sch)
	if err != nil {
		return fmt.Errorf("loading queries: %w", err)
	}

	gen := codegen.New(cfg, sch)
	files, err := gen.Generate(ops)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	mgr := output.NewManager(cfg.Output)
	if err := mgr.Write(files); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("structql: generated %d files in %s\n", len(files), cfg.Output)
	return nil
}

func runInit() error {
	path := "structql.yaml"
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	cfg := config.DefaultConfig()
	data, err := cfg.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := []byte("# structql configuration\n# See https://github.com/nseba/structql for documentation\n\n")
	absPath, _ := filepath.Abs(path)
	if err := os.WriteFile(absPath, append(header, data...), 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Created %s\n", path)
	return nil
}
