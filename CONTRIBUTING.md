# Contributing to ZK-Sequencer

Thank you for your interest in contributing to the ZK-Sequencer project! We welcome contributions from the community and are excited to work with you.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Branching Strategy](#branching-strategy)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Development Setup](#development-setup)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/nowa-zk.git
   cd nowa-zk
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/tannetwork/nowa-zk.git
   ```

## Development Workflow

### Prerequisites

Ensure you have the following installed:
- **Go 1.21+** for sequencer and prover
- **Foundry** (forge, cast, anvil) for smart contracts
- **Git** for version control
- **Docker** (optional) for testing

### Setting Up Your Development Environment

#### Smart Contracts (Foundry)
```bash
cd contracts
forge install
forge build
forge test
```

#### Sequencer (Go)
```bash
cd sequencer
go mod download
go build ./...
go test ./...
```

#### Prover (Go + Gnark)
```bash
cd prover
go mod download
go build ./...
go test ./...
```

## Branching Strategy

We use a milestone-based branching strategy:

### Branch Naming Convention

- **Features**: `feat/issue-XX-description` or `feat/description`
- **Bug fixes**: `fix/issue-XX-description` or `fix/description`
- **Tests**: `test/issue-XX-description` or `test/description`
- **Documentation**: `docs/description`
- **Milestone branches**: `milestone-X.X/description`

### Examples
```
feat/issue-16-iverifier
fix/issue-25-gas-optimization
test/issue-19-fuzzing
docs/update-readme
```

## Commit Message Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format
```
<type>: <subject>

[optional body]

[optional footer]
```

### Types
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

### Examples
```bash
# Feature with issue reference
git commit -m "feat: implement IVerifier interface (#16)"

# Bug fix with detailed description
git commit -m "fix: resolve gas estimation error in BatchRegistry

- Update gas calculation logic
- Add safety checks for overflow
- Update tests

Fixes #42"

# Multiline commit
git commit -m "refactor: migrate from Hardhat to Foundry (#15)" \
-m "- Remove Hardhat config and package files" \
-m "- Configure Foundry project (foundry.toml)" \
-m "- Update README and CI workflows for Foundry" \
-m "" \
-m "Fixes #15"
```

### Closing Issues

Use keywords in your commit message or PR description to automatically close issues:
- `Fixes #123`
- `Closes #123`
- `Resolves #123`

## Pull Request Process

### Before Submitting

1. **Update your branch** with the latest main:
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-feature-branch
   git rebase main
   ```

2. **Run all tests**:
   ```bash
   # Contracts
   cd contracts && forge test
   
   # Sequencer
   cd sequencer && go test ./...
   
   # Prover
   cd prover && go test ./...
   ```

3. **Check formatting**:
   ```bash
   # Contracts
   cd contracts && forge fmt --check
   
   # Go code
   gofmt -s -w .
   go vet ./...
   ```

4. **Update documentation** if needed

### Submitting a Pull Request

1. **Push your branch** to your fork:
   ```bash
   git push origin your-feature-branch
   ```

2. **Create a Pull Request** on GitHub with:
   - Clear title following commit message format
   - Description of changes
   - Reference to related issues
   - Screenshots (if UI changes)
   - Testing steps

3. **PR Template** (use this format):
   ```markdown
   ## Description
   Brief description of changes
   
   ## Related Issues
   Fixes #XX
   
   ## Changes Made
   - Change 1
   - Change 2
   
   ## Testing
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] Manual testing completed
   
   ## Checklist
   - [ ] Code follows project style guidelines
   - [ ] Self-review completed
   - [ ] Comments added for complex logic
   - [ ] Documentation updated
   - [ ] No new warnings introduced
   ```

4. **Address review feedback** promptly

### PR Review Process

- All PRs require at least one approval
- CI must pass before merging
- Maintainers may request changes
- Be responsive to feedback
- Keep PRs focused and reasonably sized

## Testing Guidelines

### Smart Contracts

```bash
cd contracts

# Run all tests
forge test

# Run with verbosity
forge test -vvv

# Run specific test
forge test --match-test testFunctionName

# Run with gas report
forge test --gas-report

# Check coverage
forge coverage
```

### Go Services

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...

# Verbose output
go test -v ./...
```

### Writing Tests

- Write tests for all new features
- Maintain or improve code coverage
- Use fuzzing for Solidity contracts
- Test edge cases and error conditions
- Include integration tests where appropriate

## Documentation

### Code Documentation

- **Solidity**: Use NatSpec comments
  ```solidity
  /// @notice Registers a new batch with proof
  /// @param batchId The unique identifier for the batch
  /// @param proof The zero-knowledge proof
  /// @return success Whether registration succeeded
  function registerBatch(uint256 batchId, bytes calldata proof) 
      external 
      returns (bool success);
  ```

- **Go**: Use godoc conventions
  ```go
  // ProcessBatch processes a batch of transactions and returns the state root.
  // It validates all transactions before processing and returns an error if any
  // transaction is invalid.
  func ProcessBatch(txs []Transaction) (common.Hash, error) {
      // implementation
  }
  ```

### Project Documentation

- Update README.md for user-facing changes
- Update ROADMAP.md for milestone changes
- Add architecture docs in `docs/` for major features
- Include examples in documentation

## Development Tips

### Running Local Development Environment

```bash
# Start local Foundry node (Anvil)
cd contracts
anvil

# Deploy contracts locally
forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --broadcast

# Run sequencer locally
cd sequencer
go run cmd/sequencer/main.go --config config.local.yaml
```

### Debugging

- Use `forge test -vvvv` for detailed Solidity traces
- Use `console.log` in Foundry tests
- Use Go debugger (dlv) for Go services
- Check GitHub Actions logs for CI failures

## Community

### Getting Help

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Pull Requests**: For code contributions

### Communication Guidelines

- Be respectful and constructive
- Search existing issues before creating new ones
- Provide detailed information in bug reports
- Stay on topic in discussions

## Recognition

Contributors will be recognized in:
- Project README (for significant contributions)
- Release notes
- GitHub contributor statistics

## Questions?

If you have questions about contributing, please:
1. Check existing documentation
2. Search closed issues and PRs
3. Open a GitHub Discussion
4. Reach out to maintainers

Thank you for contributing to ZK-Sequencer! 🚀
