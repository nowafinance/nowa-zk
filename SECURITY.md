# Security Policy

## Overview

The ZK-Sequencer is a critical component of the Nowa-ZK network infrastructure. Security is our top priority, and we take all security vulnerabilities seriously. This document outlines our security policy and procedures.

## Supported Versions

We currently support the following versions with security updates:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| main    | :white_check_mark: | Active Development |
| 0.x.x   | :white_check_mark: | Pre-release |

**Note**: This project is currently in active development (pre-v1.0.0). Security patches will be applied to the main branch and the latest release.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability, please follow these steps:

### 1. Private Disclosure

Send details of the vulnerability to our security team:

- **Email**: [Your security email - add later]
- **Subject**: `[SECURITY] Brief description of vulnerability`

### 2. Information to Include

Please provide as much information as possible:

- **Type of vulnerability** (e.g., buffer overflow, SQL injection, cross-site scripting)
- **Component affected** (contracts, sequencer, prover)
- **Affected versions** or commit hash
- **Step-by-step instructions** to reproduce the issue
- **Proof of concept** or exploit code (if available)
- **Impact assessment** (what an attacker could achieve)
- **Suggested fix** (if you have one)
- **Your contact information** for follow-up questions

### Example Report Template

```
Subject: [SECURITY] Potential overflow in BatchRegistry contract

Component: Smart Contracts (contracts/src/BatchRegistry.sol)
Version: commit abc1234

Description:
The BatchRegistry.registerBatch() function may be vulnerable to integer 
overflow when processing batch IDs larger than uint128.

Steps to Reproduce:
1. Deploy BatchRegistry contract
2. Call registerBatch() with batchId = type(uint256).max
3. Observe overflow in internal accounting

Impact:
An attacker could potentially...

Proof of Concept:
[Code or detailed steps]

Suggested Fix:
Use SafeMath or Solidity 0.8.x checked arithmetic
```

## Response Timeline

- **Initial Response**: Within 48 hours of report
- **Triage**: Within 7 days
- **Fix Development**: Varies based on severity
- **Patch Release**: As soon as possible after fix validation

### Severity Levels

| Severity | Response Time | Description |
|----------|---------------|-------------|
| **Critical** | 24-48 hours | Actively exploitable, affects funds or network integrity |
| **High** | 3-7 days | Significant impact, difficult to exploit |
| **Medium** | 7-14 days | Moderate impact, requires specific conditions |
| **Low** | 14-30 days | Minor impact, theoretical or difficult to exploit |

## Security Update Process

1. **Acknowledgment**: We confirm receipt of the report
2. **Validation**: We reproduce and validate the vulnerability
3. **Fix Development**: We develop and test a fix
4. **Private Review**: Security researchers can review the fix
5. **Patch Release**: We release a security patch
6. **Public Disclosure**: We publish a security advisory after patch deployment

## Coordinated Disclosure

We follow a **90-day coordinated disclosure policy**:

- We aim to patch critical vulnerabilities within 90 days
- We will work with you to establish a disclosure timeline
- We appreciate researchers who allow us time to fix issues before public disclosure
- After a fix is deployed, we will publicly acknowledge your contribution (with your permission)

## Bug Bounty Program

**Status**: Coming soon

We are working on establishing a bug bounty program for security researchers. Details will be announced in the future.

## Security Best Practices

### For Contributors

If you're contributing code, please follow these security practices:

#### Smart Contracts
- Use latest stable Solidity version (0.8.x) for built-in overflow protection
- Follow [Consensys Smart Contract Best Practices](https://consensys.github.io/smart-contract-best-practices/)
- Include comprehensive tests, including edge cases
- Use formal verification where applicable
- Run static analysis tools (Slither, Mythril)
- Get security reviews for critical functions

#### Go Services (Sequencer/Prover)
- Validate all inputs
- Use safe integer arithmetic
- Handle errors explicitly
- Use context for timeouts and cancellation
- Implement rate limiting
- Sanitize user inputs
- Use secure random number generation
- Keep dependencies updated

#### General
- Never commit secrets, private keys, or sensitive data
- Use environment variables for configuration
- Follow the principle of least privilege
- Review and test all changes thoroughly
- Keep dependencies updated

## Security Audits

### Planned Audits

We plan to conduct professional security audits before major releases:

- **Pre-Mainnet Audit**: Before v1.0.0 release
- **Annual Audits**: After mainnet launch
- **Component-Specific Audits**: For major changes

### Past Audits

*No audits completed yet. This section will be updated as audits are conducted.*

## Known Issues

We maintain transparency about known security issues:

- Currently no known security vulnerabilities
- See [GitHub Security Advisories](https://github.com/nowafinance/nowa-zk/security/advisories) for historical issues

## Security Features

### Current Security Measures

- **Smart Contracts**: Using Solidity 0.8.x with checked arithmetic
- **CI/CD**: Automated testing on all pull requests
- **Code Review**: All code changes require review
- **Dependency Scanning**: Automated dependency vulnerability checks (planned)

### Planned Security Enhancements

- [ ] Static analysis integration (Slither, Mythril)
- [ ] Fuzzing framework integration
- [ ] Formal verification for critical contracts
- [ ] Automated dependency updates
- [ ] Security-focused CI pipeline
- [ ] Professional security audit

## Responsible Disclosure Examples

We appreciate responsible disclosure practices:

✅ **Good Example**:
- Privately report to security team
- Provide detailed reproduction steps
- Allow time for fix before disclosure
- Coordinate public disclosure

❌ **Bad Example**:
- Public disclosure without notification
- Exploiting vulnerability for personal gain
- Demanding payment for disclosure
- Threatening immediate public disclosure

## Hall of Fame

We recognize security researchers who responsibly disclose vulnerabilities:

*This section will list contributors who have helped improve our security (with their permission).*

## Resources

### Security Tools

- **Foundry**: Built-in fuzzing and testing
- **Slither**: Static analysis for Solidity
- **Mythril**: Security analysis tool
- **Echidna**: Smart contract fuzzer

### Educational Resources

- [Ethereum Smart Contract Security Best Practices](https://consensys.github.io/smart-contract-best-practices/)
- [SWC Registry](https://swcregistry.io/) - Smart Contract Weakness Classification
- [Secureum](https://secureum.substack.com/) - Security research
- [Trail of Bits Resources](https://github.com/trailofbits/publications)

## Contact

For security-related inquiries:

- **Security Reports**: [Security email - add later]
- **General Security Questions**: Open a GitHub Discussion
- **Non-Security Issues**: Use GitHub Issues

## Acknowledgments

We thank the security research community for helping keep ZK-Sequencer and the Nowa-ZK network secure.

---

**Last Updated**: November 2025

