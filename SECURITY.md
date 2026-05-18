# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | ✅        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues by emailing the maintainers directly or using [GitHub private vulnerability reporting](https://github.com/mahdi13830510/open-orch/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested mitigations

You will receive a response within 72 hours. If the issue is confirmed, a patch will be released as quickly as possible.

## Security Considerations

- **ORCH_SECRET_KEY**: AES-256 key protecting integration credentials at rest. Rotate if compromised; all stored secrets must be re-entered.
- **ORCH_GITHUB_WEBHOOK_SECRET**: HMAC key verifying GitHub webhook payloads. Set to a strong random value.
- **Database**: The PostgreSQL instance should not be exposed publicly. Use network isolation (e.g., `open_orch_edge` Docker network).
- **Docker socket**: The orchestrator requires access to the Docker socket. Run on a dedicated host or use a socket proxy.
