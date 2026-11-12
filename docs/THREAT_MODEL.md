# Threat Model: Agent-Safe Sandbox (`razify run --sandbox`)

## 1. Overview and Intent

The `razify run --sandbox` command isolates command execution inside an unprivileged OCI container to reduce the risk of secret leakage, unwanted network egress, and unintended host filesystem modifications when executing untrusted or semi-trusted commands (such as those generated or driven by autonomous AI coding agents).

The objective is defense-in-depth: **mitigate accidental and unvetted exposure of sensitive configuration values without disrupting developer or agent workflows.**

> [!IMPORTANT]
> The sandbox provides operational containment and reduces exposure surfaces. It does **not** provide cryptographic certainty against determined, adversarial code executing with injected secrets. Isolation boundaries depend on underlying Linux container primitives, hypervisor VM isolation on non-Linux hosts, and Docker daemon integrity.

---

## 2. Assumptions & Host Boundary

### 2.1 Operational Assumptions
The threat model of `razify run --sandbox` operates under the following explicit assumptions:

1. **Trusted Host Operator:** The user executing `razify` on the host machine is trusted, authorized to run containers, and not actively hostile to their own environment.
2. **Intact Container Runtime:** The host Docker engine / CLI is properly installed, authenticated, correctly configured, and uncompromised.
3. **Uncompromised Workspace Baseline:** The target project repository on the host is not already compromised prior to invocation (e.g., no pre-existing malicious symlinks pointing to sensitive host directories outside the workspace, no weaponized `.git/hooks`, and no compromised host binaries).
4. **Trusted or Pinned Base Images:** Container images specified via `--image` are either trusted official distribution images or pinned to immutable SHA256 digests (`image@sha256:...`).
5. **No Host Docker Socket Passthrough:** The host Docker daemon socket (`/var/run/docker.sock` or Windows named pipe `\\.\pipe\docker_engine`) is never mounted into the container.

### 2.2 Host Agent Boundary: The Agent Itself Is Not Sandboxed
> [!WARNING]
> **The AI coding agent running on the host is NOT sandboxed by this command.**
> 
> If an AI coding agent (e.g., Cursor, Claude Code, Antigravity, Aider) executes directly on the host machine, the agent process inherits the host user's file permissions. Such an agent can directly inspect any file on the host filesystem, including `.env` files located in the repository root.
> 
> `razify run --sandbox` strictly isolates the **sub-commands invoked by or for the agent** (e.g., `go test`, `npm run build`, `pytest`), ensuring that those child processes do not view or exfiltrate secrets.
> 
> **Mitigations for Host-Level Agent Exposure:**
> 1. **Phase 4 Agent-Guard / MCP Server (`razify mcp` / `razify guard --agent`):** Provides mediated environment access via the Model Context Protocol, exposing only schema metadata rather than raw secret values to AI models (partial mitigation).
> 2. **Best-Practice Secret Placement:** Keep real production secrets outside the repository workspace entirely (e.g., in the host user's native credential store, 1Password CLI, `pass`, or in parent directories outside the git root) rather than storing unmasked credentials in local `.env` files.

---

## 3. What the Sandbox Mitigates

The sandbox is designed to address specific threat vectors common in AI-assisted development and automated script execution:

### 3.1 Unintentional Secret Leakage to LLMs and Agents
* **Threat:** Autonomous agents or build scripts routinely log environment variables, dump crash traces, serialize active processes, or inspect configuration files, exposing raw API keys and database credentials into agent context windows, cloud logging services, or telemetry.
* **Mitigation:** By default, Razify activates `--fake-secrets`, generating synthetic values conforming to schema definitions (types, enums, ranges, regexes) found in `.env.example`. Real production secrets are never injected unless explicitly requested via `--inject-secrets`. Synthetic values utilize an explicit, non-secret prefix (`razify_fake_<type>_<hash>`) so they are immediately identifiable during debugging and do not trigger secret scanners (such as Razify's scanner, Trufflehog, or GitGuardian).

### 3.2 Unvetted Network Exfiltration
* **Threat:** Hallucinating or malicious scripts, compromised dependencies (supply chain packages), or untrusted agent tools attempt outbound connections to external endpoints, telemetry servers, or command-and-control infrastructure to transmit discovered environment variables or source code.
* **Mitigation:** The default network posture is strictly `--network none`. Container network interfaces are restricted to loopback (`lo`), mitigating outbound TCP/UDP traffic and DNS requests.

### 3.3 Workspace and Host Filesystem Tampering
* **Threat:** Errant scripts or agent operations accidentally modify, overwrite, or delete source files, git histories, or host system paths.
* **Mitigation:** The host workspace is mounted into the container as read-only (`:ro`) by default. The container root filesystem is mounted read-only (`--read-only`), providing ephemeral scratch space via memory-backed tmpfs.

### 3.4 Direct `.env` File Inspection
* **Threat:** Even if environment variables are not injected into the process, processes running inside the workspace might read disk files such as `.env`, `.env.local`, or `.env.production`.
* **Mitigation:** Razify discovers all `.env` and `.env.*` files within the workspace and masks them by bind-mounting an empty temporary file over each one inside the container, mitigating direct disk reads of host secrets.

### 3.5 Privilege Escalation Within the Container
* **Threat:** Containerized processes exploit setuid binaries or Linux capabilities to gain elevated privileges or attempt container escape.
* **Mitigation:** All Linux capabilities are dropped (`--cap-drop=ALL`), privilege escalation is disabled (`--security-opt no-new-privileges`), and on native Linux hosts, execution is mapped to the invoking user's UID and GID (`--user $(id -u):$(id -g)`).

### 3.6 Host Process Table Exposure
* **Threat:** Passing sensitive environment variables via command-line arguments (e.g., `docker run -e KEY=value`) exposes plaintext values in the host process table (`ps aux`, `/proc`), visible to other local processes and unprivileged users.
* **Mitigation:** When secrets must be injected, Razify uses Docker's `-e KEY` syntax (without inline `=value`), passing values directly through the Docker CLI process environment block rather than in command-line `argv`.
* **Accuracy Caveat:** While `-e KEY` avoids exposing values in process command line arguments (`/proc/<pid>/cmdline`), on Linux systems the Docker CLI process environment block itself remains readable via `/proc/<pid>/environ` by the same user or root on the host. This mitigates world-readable process table exposure on multi-user systems but does not isolate values from other processes running under the same host UID.

---

## 4. Toolchain Usability & Writable Cache Architecture

### 4.1 The Usability Challenge
Enforcing a completely read-only container rootfs (`--read-only`), a `noexec` temporary filesystem (`--tmpfs /tmp:noexec`), and a read-only workspace (`:ro`) causes standard modern toolchains to fail:
* `go test` and `go build`: Compile temporary test binaries into `$TMPDIR` and invoke them immediately (requiring `exec` permissions on `/tmp`), while caching compilation artifacts in `$GOCACHE` and downloading modules to `$GOMODCACHE`.
* `npm` / `yarn` / `pnpm`: Require a writable `$HOME/.npm` cache and a local or symlinked `node_modules` directory to run build scripts and tests.
* `python` / `pytest` / `pip`: Require a writable `$HOME/.cache/pip` directory, local bytecode compilation cache (`__pycache__`), and writable virtual environments.

### 4.2 Writable Cache Design
To maintain containment while allowing builds and tests to function, Razify configures an **exec-capable, isolated cache architecture**:
1. **Ephemeral Exec Scratch:** `/tmp` is mounted as an exec-capable tmpfs:
   ```text
   --tmpfs /tmp:rw,exec,nosuid,size=512m
   ```
2. **Dedicated Toolchain Caches:** Toolchain cache paths are mapped either to an isolated memory-backed tmpfs or to a dedicated named Docker volume (`razify-cache-<project>-<profile>`), keeping cache writes off the host workspace:
   * **Go:** `GOCACHE=/cache/go/build`, `GOMODCACHE=/cache/go/mod`
   * **Node:** `npm_config_cache=/cache/npm`
   * **Python:** `PYTHONPYCACHEPREFIX=/cache/python/pycache`, `PIP_CACHE_DIR=/cache/python/pip`
3. **Dedicated User Home:** `$HOME` is mapped to an isolated writable tmpfs directory (`/home/razify`) so tools can write configuration and lockfiles without touching host directories.

### 4.3 Project-Type Profiles
Razify inspects project manifests to automatically select appropriate base images and cache configurations:

| Profile | Trigger Manifest | Default Image | Cache Mounts & Environment Overrides |
|---|---|---|---|
| **`go`** | `go.mod` | `golang:1.25-alpine` | `GOCACHE=/cache/go/build`<br>`GOMODCACHE=/cache/go/mod`<br>`/tmp:rw,exec,nosuid` |
| **`node`** | `package.json` | `node:22-alpine` | `npm_config_cache=/cache/npm`<br>`npm_config_update_notifier=false` |
| **`python`** | `requirements.txt`, `pyproject.toml` | `python:3.12-alpine` | `PYTHONPYCACHEPREFIX=/cache/python/pycache`<br>`PIP_CACHE_DIR=/cache/python/pip` |
| **`generic`** | Fallback / None | `alpine:latest` | `/tmp:rw,exec,nosuid,size=256m` |

### 4.4 Dependency Isolation & The Warm-Cache Workflow
Under the default `--network none` posture, package managers cannot download dependencies at runtime.
* **Vendoring:** For Go projects, vendored dependencies (`go mod vendor`) work immediately without network access.
* **Warm-Cache Workflow:** For projects requiring external packages (Node, Python, unvendored Go):
  1. **Pre-fetch step (Warm phase):** The developer or agent fetches dependencies on the host or inside a designated setup phase (e.g., `npm ci`, `go mod download`).
  2. **Execution phase (Contained phase):** Razify executes the test or build command under `--network none`, mounting the pre-populated cache volume or local dependency directory.

---

## 5. Secret Selection and Classification Engine

### 5.1 Scoped Injection
To reduce exposure surfaces, Razify never injects the full host environment block:
* Only variables explicitly declared as keys in `.env.example` (or the project schema) are candidates for injection. Host environment variables unrelated to the project schema are ignored.
* Operators may further restrict injection using `--only KEY1,KEY2,...`, ensuring only the minimal set of required variables is passed.

### 5.2 Secret Classification Heuristics
Razify inspects every declared variable to classify it as either a **Secret** or **Non-Secret Config**:
1. **Schema Annotations:** Explicit tags in `.env.example` (e.g. `# @type secret`, `# @type password`) trigger secret classification.
2. **Provider Signatures:** Known token formats (AWS keys, Stripe keys, GitHub tokens, Slack webhooks, PEM blocks) detected via Razify's regex scanner.
3. **Entropy Analysis:** High-entropy string values (Shannon entropy threshold exceeding default scanner levels).
4. **Name Pattern Heuristics:** Variable names matching sensitive naming conventions (case-insensitive substring or regex matches for `*SECRET*`, `*KEY*`, `*TOKEN*`, `*PASSWORD*`, `*PASSWD*`, `*AUTH*`, `*CREDENTIAL*`, `*PRIVATE*`, `*APIKEY*`).

### 5.3 Synthetic Value Generation
Variables classified as **Non-Secret Config** (e.g., `PORT=8080`, `LOG_LEVEL=info`, `NODE_ENV=test`, `HOST=localhost`) retain their specified or default values.

Variables classified as **Secrets** default to generated synthetic mock values:
```text
razify_fake_<type_or_key>_<hash>
```
* Example: `razify_fake_stripe_live_8f3a1c9e`, `razify_fake_api_key_4b2e6d1a`.
* **Scanner Safety:** These synthetic values use an explicit `razify_fake_` prefix to ensure they do not trigger secret scanners (including Razify's own scanner and external CI scanners) while allowing applications to parse and validate format-specific fields.

---

## 6. Guardrails and Strict Rejections

### 6.1 Prohibited Configurations (Hard Rejections)
Razify enforces strict CLI guardrails. Specifying any of the following configurations results in an immediate fatal exit (code 2) before any container is created:

| Prohibited Configuration | Reason for Hard Rejection |
|---|---|
| `--privileged` | Disables container isolation and exposes all host devices to the container. |
| Docker socket mount (`/var/run/docker.sock`, Windows pipe `docker_engine`) | Grants complete control over the host Docker daemon, resulting in trivial container breakout. |
| `--security-opt seccomp=unconfined` | Disables kernel syscall filtering. |
| `--pid=host` | Exposes host process tree, allowing the container to signal or inspect host processes. |
| `--network host` | Attaches container directly to host network interfaces, bypassing network isolation. |

### 6.2 Dangerous Configurations (Loud Warnings)
If an operator specifies overrides that weaken isolation, Razify outputs a prominent visual warning to `stderr`:
* `--network bridge` (or any non-`none` network): Warns that network egress is active and data exfiltration mitigations are disabled.
* `--inject-secrets`: Warns that real credentials are being passed into container memory and may be logged or inspected.
* `--no-mask-env`: Warns that `.env` files are unmasked and readable by processes within the container.
* `--read-write`: Warns that the workspace filesystem is writable, introducing host-compromise vectors.

### 6.3 The `--read-write` Host-Compromise Vector
Allowing containerized processes to write to the project workspace (`--read-write`) introduces a delayed host-compromise vector:
* **The Vector:** An untrusted script, compromised dependency, or hallucinating agent running in the container cannot immediately break out to the host. However, it can modify files in the workspace that are subsequently executed on the host:
  * `.git/hooks/*` (e.g., `pre-commit`, `post-checkout` scripts that execute on the host during future git actions).
  * `package.json` scripts (`postinstall`, `prepare`, `test` hooks run during local developer builds).
  * `Makefiles`, shell scripts, task configurations.
  * CI workflow files (`.github/workflows/*.yml`) that execute in CI runners with privileged repo tokens.
* **Mitigations Enforced by Razify:**
  1. **`.git` Stays Read-Only:** Even when `--read-write` is specified, Razify automatically mounts `<workspace>/.git` as read-only (`:ro`), mitigating modification of `.git/hooks` and git metadata.
  2. **Prefer Output Mounts:** Rather than making the entire workspace writable, operators are encouraged to use `--output-dir <path>`, mounting only a dedicated build artifact directory as writable.

---

## 7. What the Sandbox Does NOT Mitigate (Out of Scope & Limitations)

The sandbox has defined boundaries and does not claim to eliminate all risks:

### 7.1 Process Reading Injected Secrets
* If the operator explicitly runs with `--inject-secrets`, the containerized process receives real values in its environment. Any code executing within that container can read and inspect those values directly.

### 7.2 Host-Level `docker inspect` Exposure
* Environment variable names and values passed to containers can be read via `docker inspect <container-id>` by any user or process with access to the Docker daemon. Docker retains environment variable declarations in container configuration metadata.

### 7.3 Advanced Output-Masking Bypasses
* Razify provides best-effort stream masking on stdout and stderr to redact known injected secret values. However, stream masking cannot guarantee redaction against:
  * Chunked output (e.g., printing a secret character-by-character or across multiple lines).
  * Encoded variants (hexadecimal, custom ciphers, non-standard base64, ROT13, or URL encoding variations).
  * Steganographic or mathematical transformations.
  * Side-channel leakage (timing channels, memory consumption patterns, error code encoding).

### 7.4 Compromised Docker Daemon or Host Kernel
* Container isolation relies on the host Linux kernel's namespaces, cgroups, and LSMs (AppArmor/SELinux). If the host kernel has unpatched privilege escalation vulnerabilities, or if the Docker daemon itself is compromised, container boundaries can be bypassed.

### 7.5 Docker Desktop VM Boundary Differences (Windows and macOS)
* On Windows and macOS, Docker operates inside a virtual machine (Hyper-V, WSL2, or Apple Virtualization Framework).
* `--user $(id -u):$(id -g)` UID mapping behaves differently across VM filesystem sharing layers (virtiofs, 9p, gRPC-FUSE) than native Linux cgroups. Files created on shared mounts may exhibit default VM permissions or root ownership within the VM.
* Host firewall rules and loopback behaviors differ between host and VM network stacks.

### 7.6 Outbound Traffic Under Explicit Network Access
* If the user explicitly specifies `--network bridge` (or any non-`none` network driver), network isolation is disabled. The container can make arbitrary outbound network requests, and network exfiltration mitigations no longer apply.

### 7.7 File Masking Gaps
Bind-mounting an empty file over `.env*` targets standard `.env`, `.env.local`, `.env.production`, etc. It does **not** cover:
* Files with alternative secret naming conventions (e.g., `.envrc`, `.npmrc`, `*.pem`, `credentials.json`, `id_rsa`, `.netrc`).
* Embedded credentials in `.git/config` (mitigated by mounting `.git` as `:ro`).
* Files created dynamically by processes *after* container launch.

### 7.8 Host Image Pulling Out of Scope for Container Network Isolation
* Docker image pulls (`docker pull`) are executed by the host Docker daemon over the host network before container launch. The `--network none` flag applies strictly to the container runtime environment, not to the host Docker daemon downloading images.

---

## 8. Default Posture, Defense-in-Depth Controls, and Verification Mapping

The table below details all default security controls enforced by `razify run --sandbox`, their purpose, override mechanisms, and the corresponding automated test that verifies enforcement:

| Control Dimension | Default Sandbox Value | Purpose / Mitigated Vector | Override Flag | Verification Test |
|---|---|---|---|---|
| **Secret Injection** | Synthetic values (`razify_fake_...`) | Reduces exposure of real credentials to agent contexts and logs | `--inject-secrets` | `TestSandboxArgv_FakeSecrets` |
| **Variable Scope** | `.env.example` keys only | Restricts environment injection to declared schema keys | `--only <keys>` | `TestSandboxArgv_ScopedVariables` |
| **Argument Passing** | `-e KEY` (via environment, not argv) | Mitigates host `ps` / process table argument inspection | N/A | `TestSandboxArgv_NoInlineValues` |
| **Network Egress** | `--network none` | Mitigates external network exfiltration and C2 traffic | `--network <mode>` | `TestSandboxArgv_NetworkNone`<br>`TestSandboxIntegration_NetworkNone` |
| **Workspace Mount** | Read-only (`:ro`) | Mitigates accidental or malicious workspace corruption | `--read-write` | `TestSandboxArgv_WorkspaceReadOnly`<br>`TestSandboxIntegration_WorkspaceReadOnly` |
| **`.git` Protection** | Always read-only (`:ro`) | Mitigates host-compromise via `.git/hooks` tampering | N/A | `TestSandboxArgv_GitReadOnly`<br>`TestSandboxIntegration_GitReadOnly` |
| **Root Filesystem** | `--read-only` | Mitigates container-level persistence and system binary tampering | `--writable-rootfs` | `TestSandboxArgv_ReadOnlyRootfs` |
| **Scratch Space** | `--tmpfs /tmp:rw,exec,nosuid,size=512m` | Provides exec-capable scratch space required for compilers and tests | `--tmpfs <spec>` | `TestSandboxArgv_TmpfsExec` |
| **Cache Mounts** | Dedicated tmpfs or volume per profile | Provides isolated, writable toolchain cache (Go, Node, Python) | `--cache-volume <name>` | `TestSandboxArgv_ProjectProfiles` |
| **`.env` File Masking** | Bind-mount empty file over `.env*` | Mitigates reading raw secrets from disk within workspace | `--no-mask-env` | `TestSandboxArgv_MaskEnv`<br>`TestSandboxIntegration_EnvFileMasking` |
| **Linux Capabilities** | `--cap-drop=ALL` | Drops all kernel capabilities to restrict privileged syscalls | `--cap-add <cap>` | `TestSandboxArgv_CapDropAll` |
| **Privilege Escalation**| `--security-opt no-new-privileges` | Blocks `setuid` / `setgid` privilege elevation | N/A | `TestSandboxArgv_NoNewPrivileges` |
| **Memory Limit** | `--memory 2g` | Mitigates denial-of-service via host memory exhaustion | `--memory <size>` | `TestSandboxArgv_ResourceLimits` |
| **CPU Limit** | `--cpus 2.0` | Mitigates denial-of-service via host CPU starvation | `--cpus <count>` | `TestSandboxArgv_ResourceLimits` |
| **PID Limit** | `--pids-limit 256` | Mitigates fork bombs within the container | `--pids-limit <num>` | `TestSandboxArgv_ResourceLimits` |
| **Init Process** | `--init` | Reaps zombie processes and ensures proper signal propagation | `--no-init` | `TestSandboxArgv_InitProcess`<br>`TestSandboxIntegration_SignalForwarding` |
| **Auto Cleanup** | `--rm` | Automatically cleans up container layers and metadata upon exit | `--no-rm` | `TestSandboxArgv_AutoRemove` |
| **User Identity** | Non-root `--user UID:GID` (Unix) | Restricts root-in-container privileges on native Linux | `--user <uid:gid>` | `TestSandboxArgv_UserMapping` |
| **Stream Masking** | Enabled (stdout / stderr) | Redacts known secret strings from console and terminal output | `--no-mask-output` | `TestSandboxOutputMasking` |
| **Guardrail Rejections**| Disallow privileged, docker.sock, etc. | Halts execution on unsafe container configurations | N/A | `TestSandboxArgv_Rejections` |
| **Guardrail Warnings** | Warn on bridge, inject-secrets, etc. | Alerts operator to relaxed security boundaries | N/A | `TestSandboxWarnings` |

---

## 9. Residual Risks

Operators should remain aware of residual risks inherent to containerized execution:

1. **Plaintext in Container Memory:** Even with output masking and read-only filesystems, any injected secret exists in plaintext within the container process memory space. Memory dumps or debugger attachments can extract these values.
2. **Best-Effort Output Redaction:** Redaction operates on known strings. It cannot detect derived secrets, partial tokens, or deliberate obfuscation techniques used by adversarial code.
3. **Hypervisor and Kernel Zero-Days:** Container escapes via Linux kernel bugs or Docker Desktop hypervisor vulnerabilities remain an inherent residual risk of shared-kernel virtualization.
4. **Denial of Service via Heavy Workloads:** While resource limits (`--memory`, `--cpus`, `--pids-limit`) constrain consumption, intensive container execution can still impact host performance.
5. **Untrusted Container Image Payloads:** An untrusted base image could contain pre-installed malicious software or backdoors that activate upon execution. Pinning image digests (`@sha256:...`) is strongly recommended.

---

## 10. Summary

Razify's sandbox establishes a controlled execution envelope tailored for AI agent safety:
- **Reduces secret exposure** by substituting synthetic values by default and restricting injection to `.env.example` keys.
- **Constrains blast radius** by isolating the network (`--network none`) and enforcing read-only workspace policies (`:ro`, with `.git:ro` enforcement).
- **Supports developer toolchains** through dedicated, exec-capable cache profiles (`go`, `node`, `python`).
- **Mitigates process inspection** by passing secrets via Docker CLI environment blocks rather than command-line arguments.

Operators must treat the sandbox as an operational defense-in-depth barrier rather than a substitute for least-privilege credential management.
