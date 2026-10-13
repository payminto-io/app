# Security policy

This policy covers the Go backend and workers, merchant dashboard, hosted checkout, landing site, embeddable widget, MCP server, Solidity contracts, and Chainlink CRE workflows in this repository.
The [backend security checklist](backend/SECURITY.md) documents backend controls and deployment checks.
It does not establish that a deployment is safe for real funds.

## Report a vulnerability

Report vulnerabilities privately before disclosing exploit details publicly.
Open the repository's [Security page](https://github.com/payminto-io/app/security) and use **Report a vulnerability** if GitHub private reporting is available.
If that option is unavailable, use [repository Issues](https://github.com/payminto-io/app/issues) only to ask the maintainers for a private reporting channel, without describing the vulnerability or including reproduction steps.
Send the report after the maintainers provide that channel.
Do not put security reports, exploit code, credentials, or customer information in public issues, pull requests, or comments.

Include:

- The affected component, commit or release, and relevant configuration with secrets removed.
- The expected behavior, observed behavior, and likely impact, especially unauthorized access, ledger changes, or movement of funds.
- Minimal reproduction steps using disposable test data, accounts, and wallets.
- Relevant sanitized logs, request IDs, screenshots, and a suggested fix if available.

Remove API keys, passwords, JWTs, webhook secrets, private keys, seed phrases, vault contents, customer data, and sensitive financial information from all attachments.
Use test tokens and synthetic balances to demonstrate financial impact.
If a secret was exposed, revoke or rotate it through the responsible operator; do not send the secret as evidence.

## Authorized testing

Test only systems you own or have explicit authorization to test.
Use an isolated local environment or an authorized disposable testnet deployment with dedicated test databases, wallets, and credentials.
Do not reset shared databases, access other tenants' data, disrupt services, or move real funds to prove a finding.
Stop testing if you encounter real customer data or funds and report the minimal evidence privately.
