# ⚡ Razify (`npx razify`)

> **The Configuration Integrity Engine for Environment Variables.**

Run Razify instantly in any JavaScript, TypeScript, Next.js, Vite, or Node.js project without installing system binaries.

[![npm version](https://img.shields.io/npm/v/razify.svg?color=6C63FF&style=flat-square)](https://www.npmjs.com/package/razify)
[![License](https://img.shields.io/badge/license-MIT-brightgreen?style=flat-square)](https://github.com/Hossiy21/razify/blob/main/LICENSE)

---

## 🚀 Quick Usage

### Zero-Install Execution (`npx`)
```bash
npx razify check
```

### Run on specific environment files
```bash
npx razify check .env.staging .env.example
```

### Automatically sync missing keys
```bash
npx razify fix
```

### Run secret leak scanning
```bash
npx razify scan .env
```

---

## 🛠️ Add to `package.json` Scripts

Add Razify integrity checks directly to your project lifecycle scripts:

```json
{
  "scripts": {
    "config:check": "razify check",
    "prebuild": "razify check --quiet",
    "test:env": "razify check .env.test .env.example"
  },
  "devDependencies": {
    "razify": "^1.0.0"
  }
}
```

---

## 💎 Features
- ⚡ **Sub-10ms performance**: Native compiled binary engine running in background.
- 🔒 **Secret Leak Protection**: Detects exposed API keys, passwords, and connection strings.
- 📋 **Type Schema Engine**: Enforces `@type(...)`, `@enum(...)`, `@range(...)`, and `@requires(...)`.
- 🌐 **Zero Cloud Dependency**: 100% local, offline-first, and private.

For complete documentation, VS Code extension, and source code, visit [github.com/Hossiy21/razify](https://github.com/Hossiy21/razify).
