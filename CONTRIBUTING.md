# Contributing to git-wt

Thank you for considering a contribution. This guide covers the development
workflow, testing expectations, and pull request process.

## Development Setup

### Prerequisites

- **Go 1.24+** -- see [go.dev/dl](https://go.dev/dl/) for installation
- **git** -- any recent version
- **golangci-lint** (optional) -- used for linting
  ([installation](https://golangci-lint.run/welcome/install/))

### Clone the repository

```sh
git clone https://github.com/yasomaru/git-wt.git
cd git-wt
```

### Building

```sh
make build
```

The compiled binary is written to the project root as `git-wt`.

### Testing

```sh
make test              # run unit tests
make test-race         # run tests with the race detector
make test-cover        # run tests and generate a coverage report
make test-integration  # run integration tests
make lint              # run golangci-lint
```

## Pull Request Process

1. **Fork** the repository on GitHub.
2. **Create a feature branch** from `main`:
   ```sh
   git checkout -b feature/my-feature
   ```
3. **Make your changes.** Keep each commit focused on a single logical change.
4. **Run the test suite and linter** before pushing:
   ```sh
   make test && make lint
   ```
5. **Push** your branch and **open a pull request** against `main`.
6. Describe what your change does and why. Link to any related issues.

A maintainer will review your pull request and may request changes before
merging.

## Code Style

- Follow standard Go conventions (`gofmt`, `goimports`).
- Keep exported identifiers well-documented with GoDoc comments.
- Add tests for new functionality. Aim to cover both success and error paths.
- Run `make lint` to catch common issues before submitting.

## Reporting Issues

Open an issue on the GitHub issue tracker. Include:

- The version of `git-wt` (`git wt version`).
- Your operating system and Go version.
- Steps to reproduce the problem.
- Expected and actual behavior.
