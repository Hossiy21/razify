# Threat Model: Agent-Safe Sandbox (`razify run --sandbox`)

## 1. Overview and Intent

The `razify run --sandbox` command isolates command execution inside an unprivileged OCI container to reduce the risk of secret leakage, unwanted network egress, and unintended host filesystem modifications when executing untrusted or semi-trusted commands (such as those generated or driven by autonomous AI coding agents).

The objective is defense-in-depth: **mitigate accidental and unvetted exposure of sensitive configuration values without disrupting developer or agent workflows.**

> [!IMPORTANT]
> The sandbox provides operational containment and reduces exposure surfaces. It does **not** provide mathematical guarantees against determined, adversarial code executing with injected secrets. Security boundaries depend on underlying Linux container primitives, hypervisor VM isolation on non-Linux hosts, and Docker daemon integrity.

---

## 2. Assumptions

The security model of `razify run --sandbox` operates under the following explicit assumptions:

1. **Trusted Host Operator:** The user executing `razify` on the host machine is trusted, authorized to run containers, and not actively hostile to their own environment.
2. **Intact Container Runtime:** The host Docker engine / CLI is properly installed, authenticated, correctly configured, and uncompromised.
3. **Uncompromised Workspace Baseline:** The target project repository on the host is not already compromised prior to invocation (e.g., no pre-existing malicious symlinks pointing to sensitive host directories outside the workspace, no weaponized `.git/hooks`, and no compromised host binaries).
4. **Trusted or Pinned Base Images:** Container images specified via `--image` are either trusted official distribution images or pinned to immutable SHA256 digests (`image@sha256:...`).
5. **No Host Docker Socket Passthrough:** The host Docker daemon socket (`/var/run/docker.sock` or Windows named pipe `\\.\pipe\docker_engine`) is never mounted into the container.

---

## 3. What the Sandbox Mitigates

The sandbox is designed to address specific threat vectors common in AI-assisted development and automated script execution:

### 3.1 Unintentional Secret Leakage to LLMs and Agents
* **Threat:** Autonomous agents or build scripts routinely log environment variables, dump crash traces, serialize active processes, or inspect configuration files, exposing raw API keys and database credentials into agent context windows, cloud logging services, or telemetry.
* **Mitigation:** By default, Razify activates `--fake-secrets`, generating synthetic values that conform to schema definitions (types, enums, ranges, regexes) found in `.env.example`. Real production secrets are never injected unless explicitly requested via `--inject-secrets`.

### 3.2 Unvetted Network Exfiltration
* **Threat:** Hallucinating or malicious scripts, compromised dependencies (supply chain packages), or untrusted agent tools attempt outbound connections to external endpoints, telemetry servers, or command-and-control infrastructure to transmit discovered environment variables or source code.
* **Mitigation:** The default network posture is strictly `--network none`. Container network interfaces are restricted to loopback (`lo`), mitigating outbound TCP/UDP traffic and DNS requests.

### 3.3 Workspace and Host Filesystem Tampering
* **Threat:** Errant scripts or agent operations accidentally modify, overwrite, or delete source files, git histories, or host system paths.
* **Mitigation:** The host workspace is mounted into the container as read-only (`:ro`) by default. The container root filesystem is mounted read-only (`--read-only`), providing only ephemeral, memory-backed scratch space via `--tmpfs /tmp:rw,noexec,nosuid,size=64m`.

### 3.4 Direct `.env` File Inspection
* **Threat:** Even if environment variables are not injected into the process, processes running inside the workspace might read disk files such as `.env`, `.env.local`, or `.env.production`.
* **Mitigation:** Razify discovers all `.env` and `.env.*` files within the workspace and masks them by bind-mounting an empty temporary file over each one inside the container, mitigating direct disk reads of host secrets.

### 3.5 Privilege Escalation Within the Container
* **Threat:** Containerized processes exploit setuid binaries or Linux capabilities to gain elevated privileges or attempt container escape.
* **Mitigation:** All Linux capabilities are dropped (`--cap-drop=ALL`), privilege escalation is disabled (`--security-opt no-new-privileges`), and on native Linux hosts, execution is mapped to the invoking user's UID and GID (`--user $(id -u):$(id -g)`).

### 3.6 Process Table Secret Leakage
* **Threat:** Passing sensitive environment variables via command-line arguments (e.g., `docker run -e KEY=value`) exposes plaintext values in the host process table (`ps aux`, `/proc`), visible to other local processes and unprivileged users.
* **Mitigation:** When secrets must be injected, Razify uses Docker's `-e KEY` syntax (without inline `=value`), passing values directly through the Docker CLI process environment block rather than in `argv`.

---

## 4. What the Sandbox Does NOT Mitigate (Out of Scope)

The sandbox has defined boundaries and does not claim to eliminate all risks:

### 4.1 Process Reading Injected Secrets
* If the operator explicitly runs with `--inject-secrets`, the containerized process receives real values in its environment. Any code executing within that container can read and inspect those values directly.

### 4.2 Host-Level `docker inspect` Exposure
* Environment variable names and values passed to containers can be read via `docker inspect <container-id>` by any user or process with access to the Docker daemon. Docker retains environment variable declarations in container configuration metadata.

### 4.3 Advanced Output-Masking Bypasses
* Razify provides best-effort stream masking on stdout and stderr to redact known injected secret values. However, stream masking cannot guarantee redaction against:
  * Chunked output (e.g., printing a secret character-by-character or across multiple lines).
  * Encoded variants (hexadecimal, custom ciphers, non-standard base64, ROT13, or URL encoding variations).
  * Steganographic or mathematical transformations.
  * Side-channel leakage (timing channels, memory consumption patterns, error code encoding).

### 4.4 Compromised Docker Daemon or Host Kernel
* Container isolation relies on the host Linux kernel's namespaces, cgroups, and LSMs (AppArmor/SELinux). If the host kernel has unpatched privilege escalation vulnerabilities, or if the Docker daemon itself is compromised, container boundaries can be bypassed.

### 4.5 Docker Desktop VM Boundary Differences (Windows and macOS)
* On Windows and macOS, Docker operates inside a virtual machine (Hyper-V, WSL2, or Apple Virtualization Framework).
* `--user $(id -u):$(id -g)` UID mapping behaves differently across VM filesystem sharing layers (virtiofs, 9p, gRPC-FUSE) than native Linux cgroups. Files created on shared mounts may exhibit default VM permissions or root ownership within the VM.
* Host firewall rules and loopback behaviors differ between host and VM network stacks.

### 4.6 Outbound Traffic When Network is Explicitly Enabled
* If the user explicitly specifies `--network bridge` (or any non-`none` network driver), network isolation is disabled. The container can make arbitrary outbound network requests, and network exfiltration mitigations no longer apply.

---

## 5. Default Posture and Defense-in-Depth Settings

The table below details the default security controls enforced by `razify run --sandbox`:

| Control Dimension | Default Sandbox Value | Purpose / Mitigated Vector | Override Flag |
|---|---|---|---|
| **Secret Injection** | Synthetic values (`--fake-secrets`) | Shields real credentials from agent contexts and process memory | `--inject-secrets` |
| **Argument Passing** | `-e KEY` (via environment, not argv) | Mitigates host `ps` / process table inspection | N/A |
| **Network Egress** | `--network none` | Mitigates external network exfiltration and C2 traffic | `--network <mode>` |
| **Workspace Mount** | Read-only (`:ro`) | Mitigates accidental or malicious workspace corruption | `--read-write` |
| **Root Filesystem** | `--read-only` | Mitigates container-level persistence and system binary tampering | `--writable-rootfs` |
| **Scratch Space** | `--tmpfs /tmp:rw,noexec,nosuid,size=64m` | Provides ephemeral storage without persistence or exec capability | `--tmpfs <spec>` |
| **`.env` File Masking** | Bind-mount empty file over `.env*` | Mitigates reading raw secrets from disk within workspace | `--no-mask-env` |
| **Linux Capabilities** | `--cap-drop=ALL` | Drops all kernel capabilities to restrict privileged syscalls | `--cap-add <cap>` |
| **Privilege Escalation** | `--security-opt no-new-privileges` | Blocks `setuid` / `setgid` privilege elevation | N/A |
| **User Identity** | Non-root `--user UID:GID` (Unix) | Mitigates root-in-container privileges on native Linux | `--user <uid:gid>` |
| **Stream Masking** | Enabled (stdout / stderr) | Redacts known secret strings from console and terminal output | `--no-mask-output` |

---

## 6. Residual Risks

Operators should remain aware of residual risks inherent to containerized execution:

1. **Plaintext in Container Memory:** Even with output masking and read-only filesystems, any injected secret exists in plaintext within the container process memory space. Memory dumps or debugger attachments can extract these values.
2. **Best-Effort Output Redaction:** Redaction operates on known strings. It cannot detect derived secrets, partial tokens, or deliberate obfuscation techniques used by adversarial code.
3. **Hypervisor and Kernel Zero-Days:** Container escapes via Linux kernel bugs or Docker Desktop hypervisor vulnerabilities remain an inherent residual risk of shared-kernel virtualization.
4. **Denial of Service / Host Resource Contention:** Without explicit resource limits (e.g., `--memory`, `--cpus`), a containerized build or test loop could consume substantial host memory or CPU cycles, leading to host degradation.
5. **Untrusted Container Image Payloads:** An untrusted base image could contain pre-installed malicious software or backdoors that activate upon execution. Pinning image digests (`@sha256:...`) is strongly recommended.

---

## 7. Summary

Razify's sandbox establishes a controlled execution envelope tailored for AI agent safety:
- **Reduces secret exposure** by substituting synthetic values by default.
- **Constrains blast radius** by isolating the network and enforcing read-only filesystem policies.
- **Shields secrets from host process inspection** by avoiding command-line argument passing.

Operators must treat the sandbox as an operational defense-in-depth barrier rather than a substitute for least-privilege credential management.
