# Contributing to structql

Thank you for your interest in contributing to structql!

## Getting Started

1. Fork and clone the repository
2. Install Go 1.22 or later
3. Run `make test` to verify everything works

## Development Workflow

1. Create a feature branch from `main`
2. Make your changes
3. Add or update tests (especially golden files for codegen changes)
4. Run `make all` to verify lint, vet, tests, and build pass
5. Submit a pull request

## Golden File Tests

Code generation correctness is verified via golden file tests in `testdata/golden/`. If you change codegen behavior:

1. Run `make test-update` to regenerate golden files
2. Review the diff carefully to confirm the changes are correct
3. Commit the updated golden files with your changes

## Code Style

- Follow standard Go conventions
- Run `gofmt` and `go vet` before committing
- Keep functions focused and well-documented

## Reporting Issues

Please include:

- Go version (`go version`)
- structql version (`structql version`)
- Minimal reproduction case (schema + query files + config)
- Expected vs actual output

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
