# ⚡ razify

<p align="center">
  <b>The Configuration Integrity Engine for Environment Variables</b>
  <br />
  Type-safe validation, secret leak scanning, and template synchronization for <code>.env</code> files.
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/razify"><img src="https://img.shields.io/npm/v/razify.svg?style=flat-square&color=6C63FF" alt="npm version" /></a>
  <a href="https://www.npmjs.com/package/razify"><img src="https://img.shields.io/npm/dm/razify.svg?style=flat-square&color=blue" alt="downloads" /></a>
  <a href="https://github.com/Hossiy21/razify/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green.svg?style=flat-square" alt="license" /></a>
</p>

---

## 🚀 Quickstart

No setup required. Run Razify instantly in any Node.js, Next.js, Vite, React, or TypeScript project:

```bash
# Run unified integrity check & secret leak scan
npx razify check

# Sync missing keys from .env.example to .env
npx razify fix

# Scan for exposed API keys and credentials
npx razify scan .env
```

---

## 🛠 Project Integration

Add Razify to your `package.json` to prevent broken builds or accidental secret leaks in CI/CD pipelines:

```bash
npm install -D razify
```

```json
{
  "scripts": {
    "config:check": "razify check",
    "prebuild": "razify check --quiet",
    "precommit": "razify check"
  }
}
```

---

## 🏷 Schema Validation Annotations

Annotate your `.env.example` file with inline comment tags to enforce type safety and constraints across your team:

```env
# Database connection URL @type(url) @required
DATABASE_URL=postgres://user:pass@localhost:5432/dbname

# App port number @type(port) @range(1000,65535)
PORT=3000

# Target deployment environment @enum(development,staging,production)
NODE_ENV=development

# Enable experimental features @type(bool)
ENABLE_FEATURE_FLAGS=false
```

---

## ✨ Features

- **⚡ Sub-10ms Performance**: Powered by a fast, native binary engine running in the background.
- **🔒 Secret Leak Detection**: Scans for AWS, Stripe, GitHub, database URLs, and API key exposures.
- **📋 Rich Type Enforcement**: Supports `@type(url|email|int|bool|ip|uuid|port)`, `@enum(...)`, `@range(...)`, and `@requires(...)`.
- **🌐 100% Offline & Private**: Zero external cloud calls. Your environment variables never leave your machine.
- **📦 Zero-Install**: Automatic binary caching for Windows, macOS, and Linux.

---

## 📚 Documentation & Ecosystem

- **GitHub Repository**: [Hossiy21/razify](https://github.com/Hossiy21/razify)
- **VS Code Extension**: [Razify VS Code Extension](https://marketplace.visualstudio.com/items?itemName=hossiy21.vscode-razify)

---

<p align="center">License: MIT • Built for developers by <a href="https://github.com/Hossiy21">Hossiy21</a></p>
