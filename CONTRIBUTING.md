# Contributing to Capture

Thank you for your interest in contributing to Capture! Whether you're fixing bugs, adding features, or improving documentation, your efforts are appreciated.

## Getting Started

- **Star the Repository**: If you like the project, give it a star to show your support.
- **Join the Community**: Connect with us on [Discord](https://discord.com/invite/NAb6H3UTjK) for discussions and support.
- **Explore Issues**: Check out [open issues](https://github.com/bluewave-labs/capture/issues) and look for those labeled `good first issue` if you're new.

## Reporting Bugs

1. **Search Existing Issues**: Before reporting, see if the issue already exists.
2. **Open a New Issue**: If not found, create a new issue with:
   - A clear title and description.
   - Steps to reproduce the problem.
   - Expected vs. actual behavior.
   - Relevant logs or screenshots.

## Suggesting Features

1. **Check for Similar Requests**: Look through existing issues to avoid duplicates.
2. **Create a Feature Request**: If unique, open a new issue detailing:
   - The problem you're addressing.
   - Proposed solution or feature.
   - Any alternatives considered.

## Architecture Overview

Read a detailed structure of Capture if you would like to deep dive into the architecture.

- [Architecture Overview](docs/README.md#high-level-overview)
- [Capture DeepWiki Page](https://deepwiki.com/bluewave-labs/capture)

## Development Setup

1. **Clone the Repository**:

   ```bash
   git clone https://github.com/bluewave-labs/capture.git
   cd capture

2. **Install Dependencies**:

   ```bash
    go mod tidy
    ```

3. **Run the Application**:

    ```bash
    go run cmd/capture/main.go
    ```

## Coding Guidelines

- **Code Style**: Follow Go conventions and project-specific guidelines.
- **Linting**: Run linters to catch issues early.
- **Documentation**: Update or add documentation as needed.
- **Tests**: Write tests for new features and bug fixes.

## Submitting Changes

1. **Create a Branch**: Use a descriptive name for your branch.

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**: Ensure code is clean and well-documented.
3. **Test Your Changes**: Verify that everything works as expected.
4. **Commit and Push**:

   ```bash
   git add .
   git commit -m "Brief description of changes"
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request**: Go to the repository and open a PR against the develop branch.

Your contributions make Capture better for everyone. We appreciate your time and effort!
