<!--suppress HtmlDeprecatedAttribute -->
<h1 align="center">Contributing to Commodore</h1>

Thank you for your interest in **Commodore**.

Commodore is the shared CLI engine and SDK used across OctalMesh projects.
Contributions are welcome, especially fixes that improve reliability,
documentation, developer experience, and role-specific CLI behavior.

## Before You Start

Please open an issue before starting work on a large feature or a behavioral
change. Small bug fixes, documentation updates, and focused refactors can be
submitted directly as pull requests.

When reporting a problem or proposing a change, include:

- What you expected to happen
- What actually happened
- Steps to reproduce the issue
- Relevant OS and shell details
- Any `.commodore` configuration details that affect the behavior

## Pull Request Guidelines

Please keep pull requests focused and easy to review.

- Make one logical change per pull request
- Preserve the existing project structure and naming conventions
- Update documentation when CLI behavior or public SDK usage changes
- Add or update tests when the change affects behavior that can be verified
  automatically
- Avoid unrelated formatting-only changes

## Development Notes

This repository contains both the standalone `commodore` binary and the public
Go SDK under `pkg/sdk`.

- The CLI behavior is driven by `.commodore` configuration files discovered at
  runtime
- Foreman, brigadier, and worker roles are implemented separately and should
  stay consistent
- Changes to command semantics should consider both direct CLI usage and SDK
  consumers

## Code of Conduct

By participating in this project, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

By contributing to this repository, you agree that your contributions will be
licensed under the [MIT License](LICENSE.md).

#

<h6 align="center">
Thanks for helping improve Commodore and the tooling built on top of it.
</h6>
