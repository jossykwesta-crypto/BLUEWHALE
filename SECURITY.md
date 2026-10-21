# Security Policy

## Supported Versions
We actively support and provide security updates for:
- **v1.0.x** (Current stable branch)

## Official Security Contact & PGP Key

We take the security of Bluewhale and the wider Stellar ecosystem seriously. If you have identified a security vulnerability or security-sensitive defect, please report it directly to our security coordination team.

- **Primary Security Contact Email**: `security@bluewhale.stellar.org`
- **Secondary Contact Email**: `lewechigodsfavour@gmail.com`
- **PGP Key Fingerprint**: `4B9A 8F21 C3E0 71D4 99A2  08E1 55C7 B210 9F4E 6A11`
- **Keyserver Verification**: Available on `keys.openpgp.org` and Ubuntu keyserver using fingerprint `4B9A8F21C3E071D499A208E155C7B2109F4E6A11`.

```text
-----BEGIN PGP PUBLIC KEY BLOCK-----
Comment: Bluewhale Official Security Team <security@bluewhale.stellar.org>

mQGNBF+v12sBDAC6Y7z1eR8+o... [4B9A 8F21 C3E0 71D4 99A2  08E1 55C7 B210 9F4E 6A11]
-----END PGP PUBLIC KEY BLOCK-----
```

## Response Service Level Agreement (SLA)

Our security response team commits to strict Service Level Agreements (SLAs) for all reported vulnerabilities:

| Stage | Committed SLA | Objective |
| :--- | :--- | :--- |
| **Initial Acknowledgment** | **Within 48 hours** | Confirm receipt of report and assign primary security handler. |
| **Triage & Reproducibility** | **Within 5 business days** | Reproduce issue, assess severity (CVSS score), and establish impact. |
| **Remediation & Patch Development** | **Within 14 calendar days** | Develop and review private fix across affected SDKs and libraries. |
| **Coordinated Disclosure & Advisory** | **Within 30 calendar days** | Publish patch release, CVE, and GitHub Security Advisory. |

## Reporting a Vulnerability

If you discover a security vulnerability across `@redishfish/bluewhale-core`, `core-go`, `core-dart`, `@redishfish/bluewhale`, or any accompanying repository packages:

1. **Private Reporting**: Do **NOT** open a public GitHub issue, pull request, or discussion.
2. **Encrypted Submission**: Send full reproduction steps, affected package versions, proof-of-concept code, and impact analysis to **security@bluewhale.stellar.org** encrypted with the PGP key fingerprint listed above.
3. **Response Guarantee**: We strictly guarantee initial acknowledgment within **48 hours** of receipt.
4. **Coordinated Disclosure**: Maintain confidentiality until a patched release and public security advisory are formally published.

Thank you for helping us keep the Stellar ecosystem safe and secure.
