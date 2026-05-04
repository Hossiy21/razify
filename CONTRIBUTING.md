# Contributing to Razify

First off, thank you for considering contributing to Razify! It's people like you who make this tool better for everyone.

## 🚀 Quick Start

1. **Fork the Repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/razify.git
   cd razify
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Build and run**:
   ```bash
   go build .
   ./razify --help
   ```

## 🧪 Testing

Before submitting a PR, please ensure all tests pass:
```bash
go test ./...
```

## 📬 Submitting a Pull Request

1. **Create a new branch** for your feature or bugfix:
   ```bash
   git checkout -b feature/amazing-new-feature
   ```
2. **Commit your changes** with descriptive commit messages.
3. **Push to the branch**:
   ```bash
   git push origin feature/amazing-new-feature
   ```
4. **Open a Pull Request** against the `master` branch.

## 📝 Coding Standards

- Keep functions small and focused.
- Ensure all new logic is covered by unit tests.
- Run `go fmt` before committing.

Thank you for helping make Razify the best tool for environment management!
