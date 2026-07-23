<div align="center">
  <img src="icon.png" alt="Razify Logo" width="128" />
  <h1>⚡ Razify VS Code Extension</h1>
</div>

> **Real-time Configuration Integrity & Secret Guard for Visual Studio Code & Antigravity IDE.**

Bring sub-10ms environment variable validation, schema hover tooltips, secret leak scanning, and auto-fixes directly into your code editor.

---

## ✨ Features

- 🔴 **Inline Red & Yellow Diagnostics**: Real-time error squigglies inside `.env` files for missing template variables, type mismatches, and placeholder values.
- 🔒 **Secret Leak Prevention**: Instant visual highlights when API keys, passwords, or private keys are accidentally pasted into `.env`.
- 💡 **Schema Hover Tooltips**: Hover over any environment variable to view schema rules (`@type(...)`, `@enum(...)`, `@requires(...)`) defined in `.env.example`.
- ⚡ **Auto-Completion**: Autocomplete missing environment keys directly from `.env.example`.
- 🛡️ **Check on Save**: Automatically runs background checks whenever a `.env` file is saved.

---

## ⚙️ Extension Settings

- `razify.enable`: Enable or disable Razify background diagnostics (Default: `true`).
- `razify.checkOnSave`: Automatically run Razify checks on file save (Default: `true`).
- `razify.executablePath`: Path to the Razify executable binary (Default: `"razify"` or `"npx razify"`).

---

## 🎮 Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`)

* `Razify: Check Environment Integrity`
* `Razify: Fix & Sync Missing Keys`
* `Razify: Initialize .env.example Template`
* `Razify: Install Git Pre-Commit Guard`

---

For complete documentation and source code, visit [github.com/Hossiy21/razify](https://github.com/Hossiy21/razify).
