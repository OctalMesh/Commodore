<!--suppress HtmlDeprecatedAttribute -->
<h1 align="center">Contributing to Commodore</h1>

<div align="center">
  <h6>
    <a rel="noopener noreferrer" href="README.md">Readme</a>
    ·
    <a rel="noopener noreferrer" href="CODE_OF_CONDUCT.md">Code of Conduct</a>
    ·
    <a rel="noopener noreferrer" href="SECURITY.md">Security Policy</a>
    ·
    <a rel="noopener noreferrer" href="SUPPORT.md">Support</a>
    ·
    <a rel="noopener noreferrer" href="LICENSE.md">License</a>
  </h6>
</div>

Thank you for your interest in **Commodore**!

Commodore is the shared CLI engine and SDK used across OctalMesh projects.
Contributions are welcome, especially fixes that improve reliability,
documentation, developer experience, and role-specific CLI behavior.

## Before You Start

> [!IMPORTANT]
> Open an issue before starting work on a large feature or a behavioral change.
> Small bug fixes, documentation updates, and focused refactors can be submitted
> directly as pull requests.

When reporting a problem or proposing a change, include:

- What you expected to happen
- What actually happened
- Steps to reproduce the issue
- Relevant OS and shell details
- Any `.commodore` configuration details that affect the behavior

## Pull Request Guidelines

Keep pull requests focused and easy to review:

- [ ] One logical change per pull request
- [ ] Existing project structure and naming conventions preserved
- [ ] Documentation updated when CLI behavior or public SDK usage changes
- [ ] Tests added or updated when the change affects verifiable behavior
- [ ] No unrelated formatting-only changes

## Development Notes

This repository contains both the standalone `commodore` binary and the public
Go SDK under `pkg/sdk`.

| Area                  | Notes                                                                            |
|-----------------------|----------------------------------------------------------------------------------|
| **CLI behavior**      | Driven by `.commodore` configuration files discovered at runtime                 |
| **Roles**             | Implemented separately - keep them consistent with each other                    |
| **Command semantics** | Consider both direct CLI usage and SDK consumers before changing                 |
| **Public SDK**        | `pkg/sdk` is a stable public API, breaking changes require deliberate versioning |

> [!NOTE]
> Changes to command semantics or public SDK interfaces should be discussed in
> an issue first, as they may affect external consumers building on top of
> Commodore.

## Code of Conduct

By participating in this project, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing to this repository, you agree that your contributions will be
licensed under the [MIT License](LICENSE.md).

<h1></h1>

<h6 align="center">
Thanks for helping improve Commodore and the tooling built on top of it.
</h6>
