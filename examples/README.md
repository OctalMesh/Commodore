<!--suppress HtmlDeprecatedAttribute, HtmlUnknownTarget -->
<div align="center">
  <h1>Commodore Examples</h1>
  <p>Reference configurations and SDK usage for <strong>Commodore</strong></p>
  <h6>
    <a rel="noopener noreferrer" href="../README.md"><- Back to main README</a>
    &nbsp;·&nbsp;
    <a rel="noopener noreferrer" href="../CONTRIBUTING.md">Contributing</a>
    &nbsp;·&nbsp;
    <a rel="noopener noreferrer" href="../LICENSE.md">License</a>
  </h6>
</div>

<div align="center">
  <h2>What's inside</h2>
</div>

```
examples/
├── configuration/
│   ├── .commodore.squadron.yaml <- Full squadron config with every field
│   └── .commodore.unit.yaml     <- Full unit config with every field
└── sdk/
    └── main.go                  <- Custom binary built with the Go SDK
```

> [!NOTE]
> These are reference templates - not a runnable project on their own. Copy the
> relevant files into your project and adapt them to your needs.

<div align="center">
  <h2>Hierarchy Quick Reference</h2>
</div>

Commodore models infrastructure as a naval fleet. Two config roles exist:

```
root-division         (squadron)  <- .commodore.yaml
├── sqd-frontend      (squadron)  <- resolves to ./modules/frontend/.commodore.yaml
│   └── app-desktop   (unit)      <- resolves to ./apps/desktop/.commodore.yaml
├── svc-auth          (unit)      <- resolves to ./services/auth/.commodore.yaml
└── svc-notification  (unit)      <- resolves to ./services/notification/.commodore.yaml
```

Each node declares its own role, reactor, environments, and maneuvers. The root
squadron stitches them together via `manifest`.

<div align="center">
  <h2>Configuration Examples</h2>
</div>

Ready-to-copy YAML templates covering the full configuration surface.

### `.commodore.squadron.yaml`

Defines a **squadron** - a recursive orchestrator that manages subordinates.

Key concepts demonstrated:

| Feature                                    | Where                    |
|--------------------------------------------|--------------------------|
| `role: squadron` declaration               | Top of file              |
| Tilt reactor with multi-env blueprints     | `reactor.blueprints`     |
| Environment variable cascading             | `reactor.environments`   |
| Manifest with tag-based `after:` selectors | `manifest[].after`       |
| Health checks per subordinate              | `manifest[].healthcheck` |
| Custom maneuver (`deploy`)                 | `maneuvers`              |

```bash
# Copy to your project root and rename
cp examples/configuration/.commodore.squadron.yaml ./.commodore.yaml
```

### `.commodore.unit.yaml`

Defines a **unit** - an atomic execution leaf for a single service.

Key concepts demonstrated:

| Feature                                           | Where                  |
|---------------------------------------------------|------------------------|
| `role: unit` declaration                          | Top of file            |
| Tilt blueprint + native `action:` blueprint       | `reactor.blueprints`   |
| Per-unit environment variables and `.env` loading | `reactor.environments` |
| Standalone-first unit overrides                   | `reactor.standalone`   |
| Multiple maneuvers (`deploy`, `lint`, `test`)     | `maneuvers`            |

```bash
# Copy next to your service and rename
cp examples/configuration/.commodore.unit.yaml ./services/my-service/.commodore.yaml
```

> [!IMPORTANT]
> The `id:` in a unit config **must match** the `id:` declared for that
> subordinate in the parent squadron's `manifest`.

> [!NOTE]
> Standalone mode applies when a unit is started directly from its own
> directory. Resolution order is: `reactor.standalone` first, then fallback to
> `reactor` defaults.

<div align="center">
  <h2>Common Patterns</h2>
</div>

### Start the full stack for `dev`

```bash
commodore up --env dev
```

### Start only backend services

```bash
commodore up --tags backend
```

### Run a maneuver on a specific unit

```bash
commodore signal svc-auth test
commodore signal svc-auth lint
```

### Start a sub-squadron and all its subordinates for `staging`

```bash
commodore signal sqd-frontend up --env staging
```

### Deep delegation - target any node without `cd`

```bash
# From the root: tell sqd-frontend to tell app-web to run lint
commodore signal sqd-frontend signal app-web lint
```

### Validate everything before `up`

```bash
commodore doctor --env staging
```

### Inspect the discovered hierarchy

```bash
commodore tree
```

<div align="center">
  <h2>Config Resolution</h2>
</div>

Commodore resolves subordinate configs by looking for files in the subordinate's
directory, following this order:

1. `<path>/.commodore.yaml`
2. `<path>/.commodore.yml`
3. `<path>/.commodore`

> [!TIP]
> You can point `path:` directly to a specific YAML file if you don't want to
> use the default naming convention.

<div align="center">
  <!--
  =====================
         FOOTER
  =====================
  -->
  <h1></h1>
  <br />
  <!-- OctalMesh Logo -->
  <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com">
    <picture>
      <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/logo/svg/octal_mesh_center.svg" />
      <img alt="OctalMesh" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/logo/svg/octal_mesh_center_white.svg" height="48" />
    </picture>
  </a>
  <br /><br />
  <!-- Socials -->
  <div>
    <!-- Telegram Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/telegram">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/telegram.svg" />
        <img alt="Telegram" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/telegram_white.svg" width="48" />
      </picture>
    </a>
    &nbsp;
    <!-- YouTube Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/youtube">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/youtube.svg" />
        <img alt="YouTube" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/youtube_white.svg" width="48" />
      </picture>
    </a>
    &nbsp;
    <!-- TikTok Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/tiktok">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/tiktok.svg" />
        <img alt="TikTok" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/tiktok_white.svg" width="48" />
      </picture>
    </a>
    &nbsp;
    <!-- Instagram Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/instagram">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/instagram.svg" />
        <img alt="Instagram" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/instagram_white.svg" width="48" />
      </picture>
    </a>
    &nbsp;
    <!-- X Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/x">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/x.svg" />
        <img alt="X" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/x_white.svg" width="48" />
      </picture>
    </a>
    &nbsp;
    <!-- Reddit Badge -->
    <a rel="noopener noreferrer" target="_blank" href="https://octalmesh.com/reddit">
      <picture>
        <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/reddit.svg" />
        <img alt="Reddit" src="https://raw.githubusercontent.com/OctalMesh/OctalDesign/release/assets/icon/svg/reddit_white.svg" width="48" />
      </picture>
    </a>
  </div>
</div>
<h6>
  <div align="center">
    • • •
    <br /><br />
    This project is licensed under the <a rel="noopener noreferrer" href="../LICENSE.md">MIT License</a>
    <br /><br />
  </div>
  <div align="justify">
    <ul>
      <li>Feel free to use this project for any purpose, including commercial applications.</li>
      <li>You are permitted to modify, distribute, and include this project in any form, as long as the original copyright notice is retained.</li>
      <li>If you share or publish modified versions, attribution to the original <a rel="noopener noreferrer" href="https://github.com/OctalMesh/Commodore">GitHub repository</a> is appreciated.</li>
      <li>This software is provided "as is", without any warranties or guarantees, as detailed in the license terms.</li>
    </ul>
  </div>
</h6>
