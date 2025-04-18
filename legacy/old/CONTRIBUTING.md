# Contributing to Sirius App Agent

We love your input! We want to make contributing to Sirius App Agent as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## We Develop with GitHub

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## We Use [Github Flow](https://guides.github.com/introduction/flow/index.html)

Pull requests are the best way to propose changes to the codebase. We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Development Process

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `go test ./...`
4. Build: `go build ./cmd/app-agent`

## Code Style

- Follow standard Go formatting guidelines
- Use `gofmt` to format your code
- Follow the project's established patterns
- Write descriptive commit messages

## Testing

- Write unit tests for new code
- Ensure all tests pass before submitting PR
- Include integration tests where appropriate
- Document test cases and expectations

## License

By contributing, you agree that your contributions will be licensed under its MIT License.
