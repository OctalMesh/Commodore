# Commodore Examples

This directory contains reference configurations for the **Commodore**
orchestration system. Commodore uses a naval-inspired hierarchy to manage
complex microservice ecosystems through recursive nodes.

## Hierarchy Overview

> [!IMPORTANT]
> Commodore by default looks for `.commodore[.yaml, .yml]` files in the project
> directory. You can define what configuration file to use with the `--config`
> flag when running the CLI.

1. **Squadron (`.commodore.squadron.yaml`)**: A recursive manager (node). It
   orchestrates other squadrons or units, manages environment cascading, and
   defines the boot order.
2. **Unit (`.commodore.unit.yaml`)**: An atomic execution unit (leaf). It
   defines how a single service is built, run, and maintained via its "Reactor".

## 🛰 Squadron Commander Reference

`.commodore.squadron.yaml` (template name in this `examples` folder)

| Field                       | Type       | Possible Values / Format                 | Description                                                             |
|-----------------------------|------------|------------------------------------------|-------------------------------------------------------------------------|
| **role**                    | `string`   | `squadron`                               | Defines the role of this config as a squadron commander.                |
| **id**                      | `string`   | Any unique string                        | Unique identifier (must be unique across the manifest).                 |
| **binary**                  | `string`   | Default: `commodore`                     | Optional: Custom binary for this squadron to execute.                   |
| **reactor**                 | `object`   | See below                                | Defines how this squadron manages its subordinates.                     |
| **reactor.provider**        | `string`   | `tilt`                                   | Specifies the reactor provider.                                         |
| **reactor.network**         | `string`   | Any string                               | Optional: Custom network for inter-unit communication.                  |
| **reactor.context**         | `string`   | Path (e.g., `./`)                        | Optional: Build context for the reactor.                                |
| **reactor.blueprints**      | `list`     | `env`, `path`                            | Mapping of environment names to specific configuration files.           |
| **reactor.environments**    | `list`     | `name`, `files`, `variables`             | Define environment variables accessible to all units and sub-squadrons. |
| **manifest**                | `list`     | List of subordinates                     | Defines the hierarchy. Each path leads to another config.               |
| **manifest[].id**           | `string`   | Any string                               | Unique identifier for the subordinate (squadron or unit).               |
| **manifest[].path**         | `string`   | Path                                     | Points to a directory or a specific `.yaml` file.                       |
| **manifest[].tags**         | `list`     | List of strings                          | Optional: Used for grouping and dependency management.                  |
| **manifest[].after**        | `list`     | `id` or `$tag-name`                      | Wait for specified IDs or all units with a certain tag to be healthy.   |
| **manifest[].healthcheck**  | `object`   | `test`, `interval`, `timeout`, `retries` | Define how to verify the subordinate's readiness.                       |
| **maneuvers**               | `list`     | List of actions                          | Executable actions that this squadron can perform by using the CLI.     |
| **maneuvers[].call**        | `string`   | Command name                             | The identifier used to trigger the maneuver (e.g., `lint`, `test`).     |
| **maneuvers[].description** | `string`   | Any string                               | Human-readable explanation of the maneuver.                             |
| **maneuvers[].action**      | `list`     | Array of strings                         | The actual command, script, or sequence of operations to execute.       |

## Unit Designation Reference

`.commodore.unit.yaml` (template name in this `examples` folder)

| Field                       | Type     | Possible Values / Format     | Description                                                          |
|-----------------------------|----------|------------------------------|----------------------------------------------------------------------|
| **role**                    | `string` | `unit`                       | Defines the role as a standalone execution unit.                     |
| **id**                      | `string` | Any unique string            | Unique identifier (must match the ID in the Squadron manifest).      |
| **reactor**                 | `object` | See below                    | Defines how this unit is built, run, and maintained.                 |
| **reactor.provider**        | `string` | `tilt`, `native`             | Specifies the reactor provider.                                      |
| **reactor.context**         | `string` | Path (e.g., `./`)            | Optional: Build context for the reactor.                             |
| **reactor.blueprints**      | `list`   | `env`, `path`, `action`      | Environment-specific logic. Can be a path or a direct command array. |
| **reactor.environments**    | `list`   | `name`, `files`, `variables` | Local environment variables and `.env` file overrides.               |
| **maneuvers**               | `list`   | List of actions              | Executable actions that this unit can perform by using the CLI.      |
| **maneuvers[].call**        | `string` | Command name                 | The identifier used to trigger the maneuver (e.g., `lint`, `test`).  |
| **maneuvers[].description** | `string` | Any string                   | Human-readable explanation of the maneuver.                          |
| **maneuvers[].action**      | `list`   | Array of strings             | The actual command, script, or sequence of operations to execute.    |

## Variable Cascading Logic

Commodore employs a **Top-Down Inheritance** model for environment variables:
1. **Fleet Level**: Global defaults.
2. **Squadron Level**: Mid-level overrides (applied to all subordinates).
3. **Unit Level**: Final local overrides (highest priority).

## Startup Sequence

The boot order is determined by the `after` field in the Squadron manifest.
- Use **Direct ID** (`svc-auth`) for specific dependencies.
- Use **Tag Selectors** (`$tag-backend`) to wait for an entire group of units to
  become healthy before proceeding.

## CLI Usage

Commodore generates a dynamic CLI based on your hierarchy. You can execute
commands at any level, and the system will handle the orchestration according to
the defined structure and dependencies.

### Base Commands

| Command      | Description                                                                       |
|--------------|-----------------------------------------------------------------------------------|
| `up`         | Start the current squadron or unit and all its subordinates in the correct order. |
| `down`       | Stop all reactors and cleanup resources within the squadron.                      |
| `doctor`     | Run diagnostics to ensure requirements are met and configurations are valid.      |
| `status`     | Show the health status of all units and squadrons in the current context.         |
| `tree`       | Show the discovered subordinate structure as a tree view.                         |
| `completion` | Generate the autocompletion script for the specified shell                        |
| `modules`    | Manage git submodules (status, init, update, sync)                                |
| `signal`     | Send a command or maneuver to a specific subordinate.                             |

### Flags & Environments

| Flag                    | Description                                                                                               |
|-------------------------|-----------------------------------------------------------------------------------------------------------|
| `-h`, `--help`          | Show help information for current command.                                                                |
| `-e`,`--env <name>`     | Specify environment (e.g., `dev`, `staging`). Should match one of the defined environments in the config. |
| `-c`, `--config <path>` | Path to a custom configuration file.                                                                      |
| `-t`, `--tags <tags>`   | Filter operations to specific groups (e.g., `commodore up -t backend`).                                   |

### Recursive Navigation

Access subordinates directly by their IDs in the command path. This allows you
to run maneuvers or commands on specific units or squadrons without affecting
the entire hierarchy.

```bash
# General pattern: [binary] signal [subordinate-id] [command/maneuver]

# Execute a maneuver on a direct Unit
commodore signal svc-auth test

# Delegate a command to a sub-Squadron
commodore signal sqd-frontend up

# Chain of Command (Deep Delegation)
# This signals 'sqd-frontend' to then signal its own 'app-web' subordinate to
# run the 'lint' maneuver defined in its config.
commodore signal sqd-frontend signal app-web lint
```
