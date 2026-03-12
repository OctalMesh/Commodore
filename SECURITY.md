<!--suppress HtmlDeprecatedAttribute -->
<h1 align="center">Security Policy</h1>

## Reporting a Vulnerability

If you discover a security vulnerability, **do not open a public issue,
discussion, or pull request**.
Instead, report it privately:

- Email: <security@octalmesh.com> *(preferred)*
- Or contact a core [maintainer](https://github.com/nykonhrytsyshyn) directly

### Please include as much of the following information as possible:

- Type of issue
- Affected command, package, role, or integration path
- Location of the issue
  *(repository, branch, commit, or direct URL if public)*
- Configuration or environment assumptions
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code *(if available)*
- Expected and actual behavior
- Potential impact and realistic attack scenarios

Incomplete reports are still welcome, but detailed reports allow faster and more
accurate triage.

### Security reports are accepted in:

- English
- Ukrainian
- Russian

## Scope

This policy applies to:

- The standalone `commodore` CLI
- The public SDK in `pkg/sdk`
- Built-in role implementations and command wiring
- Repository-owned scripts

Issues in third-party tools such as Git, Go, Docker, Tilt, Cobra, Bubble Tea,
or other dependencies should also be reported to their respective maintainers
when applicable.

## Response Timeline

We aim to follow this process:

- **Acknowledgement:** within 48 hours
- **Initial assessment:** within 5 business days
- **Fix & disclosure:** as soon as reasonably possible

Timelines may vary depending on severity and complexity.

#

<h6 align="center">
Security research helps keep Commodore reliable and boring - exactly how
security should be
</h6>
