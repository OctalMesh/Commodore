<!--suppress HtmlDeprecatedAttribute -->
<h1 align="center">Security Policy</h1>

<div align="center">
  <h6>
    <a rel="noopener noreferrer" href="README.md">Readme</a>
    ·
    <a rel="noopener noreferrer" href="CODE_OF_CONDUCT.md">Code of Conduct</a>
    ·
    <a rel="noopener noreferrer" href="CONTRIBUTING.md">Contributing</a>
    ·
    <a rel="noopener noreferrer" href="SUPPORT.md">Support</a>
    ·
    <a rel="noopener noreferrer" href="LICENSE.md">License</a>
  </h6>
</div>

<h1></h1>

> [!CAUTION]
> **Do not open a public issue, discussion, or pull request to report a security
> vulnerability.** Use the private channels listed below.

## Reporting a Vulnerability

| Channel                 | Details                                                                |
|-------------------------|------------------------------------------------------------------------|
| **Email** *(preferred)* | [security@octalmesh.com](mailto:security@octalmesh.com)                |
| **Direct contact**      | Reach a core [maintainer](https://github.com/nykonhrytsyshyn) directly |

### What to include

The more context you provide, the faster and more accurately we can triage.

- **Type of issue** - e.g. command injection, path traversal, privilege
  escalation
- **Affected surface** - command, package, role, or integration path
- **Location** - repository, branch, commit, or direct URL if public
- **Environment** - OS, shell, configuration assumptions
- **Reproduction steps** - step-by-step instructions
- **Proof-of-concept** - exploit code or demonstration, if available
- **Impact** - expected vs. actual behavior, realistic attack scenarios

Incomplete reports are still welcome, but detailed reports allow faster and more
accurate triage.

### Security reports are accepted in:

- English
- Ukrainian
- Russian

## Scope

This policy covers the following surfaces:

| In Scope                                         | Out of Scope                              |
|--------------------------------------------------|-------------------------------------------|
| The standalone `commodore` CLI                   | Third-party tools (Git, Go, Docker, Tilt) |
| Public SDK in `pkg/sdk`                          | Dependencies (Cobra, Bubble Tea, etc.)    |
| Built-in role implementations and command wiring | Forks or heavily modified codebases       |
| Repository-owned scripts                         |                                           |

> [!NOTE]
> Issues in third-party dependencies should also be reported to their respective
> maintainers when applicable.

## Response Timeline

| Stage                  | Target                         |
|------------------------|--------------------------------|
| **Acknowledgement**    | Within 48 hours                |
| **Initial assessment** | Within 5 business days         |
| **Fix & disclosure**   | As soon as reasonably possible |

Timelines may vary depending on severity and complexity.

<h1></h1>

<h6 align="center">
Security research helps keep Commodore reliable and boring - exactly how
security should be
</h6>
