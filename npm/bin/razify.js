#!/usr/bin/env node
"use strict";

const { spawn, execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");
const os = require("os");
const https = require("https");
const crypto = require("crypto");

const VERSION = "v0.1.0";
const REPO = "Hossiy21/razify";
const MAX_REDIRECTS = 5;
const MAX_RETRIES = 3;
const DOWNLOAD_TIMEOUT_MS = 30_000;

// ---------------------------------------------------------------------------
// Paths
// ---------------------------------------------------------------------------

function getBinaryName() {
  return os.platform() === "win32" ? "razify.exe" : "razify";
}

// Cache is versioned so bumping VERSION doesn't silently keep serving an
// old binary, and two different installed versions of this npm package
// don't stomp on each other's cached binaries.
function getCacheDir() {
  const dir = path.join(os.homedir(), ".razify", "bin", VERSION);
  fs.mkdirSync(dir, { recursive: true, mode: 0o755 });
  return dir;
}

function findBinary() {
  const binaryName = getBinaryName();

  const candidates = [
    path.join(__dirname, binaryName),
    path.join(__dirname, "..", "..", binaryName),
    path.join(process.cwd(), binaryName),
    path.join(getCacheDir(), binaryName),
  ];

  for (const candidate of candidates) {
    if (isExecutableFile(candidate)) {
      return candidate;
    }
  }

  // System PATH lookup — use execFileSync with argv array, never a shell
  // string, so we're not vulnerable to injection via a crafted PATH/cwd.
  try {
    const cmd = os.platform() === "win32" ? "where" : "which";
    const out = execFileSync(cmd, [binaryName], {
      stdio: ["ignore", "pipe", "ignore"],
    })
      .toString()
      .trim();
    const first = out.split(/\r?\n/)[0]?.trim();
    if (first && isExecutableFile(first)) return first;
  } catch {
    // not found on PATH — that's fine, fall through
  }

  return null;
}

function isExecutableFile(p) {
  try {
    const st = fs.statSync(p);
    if (!st.isFile()) return false;
    if (os.platform() === "win32") return true; // no X bit concept on win32
    fs.accessSync(p, fs.constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

// ---------------------------------------------------------------------------
// Networking
// ---------------------------------------------------------------------------

function httpsGetFollow(url, redirectsLeft, onResponse, onError) {
  const req = https.get(
    url,
    { headers: { "User-Agent": `razify-npm-wrapper/${VERSION}` }, timeout: DOWNLOAD_TIMEOUT_MS },
    (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume(); // discard body
        if (redirectsLeft <= 0) {
          onError(new Error("Too many redirects while downloading release asset"));
          return;
        }
        // Resolve relative Location headers against the previous URL.
        const nextUrl = new URL(res.headers.location, url).toString();
        httpsGetFollow(nextUrl, redirectsLeft - 1, onResponse, onError);
        return;
      }
      if (res.statusCode !== 200) {
        res.resume();
        onError(new Error(`HTTP ${res.statusCode} fetching ${url}`));
        return;
      }
      onResponse(res);
    }
  );

  req.on("timeout", () => req.destroy(new Error(`Timed out after ${DOWNLOAD_TIMEOUT_MS}ms: ${url}`)));
  req.on("error", onError);
}

function downloadToFile(url, destFile) {
  return new Promise((resolve, reject) => {
    httpsGetFollow(
      url,
      MAX_REDIRECTS,
      (res) => {
        const fileStream = fs.createWriteStream(destFile, { mode: 0o644 });
        res.pipe(fileStream);
        fileStream.on("finish", () => fileStream.close(() => resolve()));
        fileStream.on("error", reject);
        res.on("error", reject);
      },
      reject
    );
  });
}

function fetchText(url) {
  return new Promise((resolve, reject) => {
    httpsGetFollow(
      url,
      MAX_REDIRECTS,
      (res) => {
        let data = "";
        res.setEncoding("utf8");
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () => resolve(data));
        res.on("error", reject);
      },
      reject
    );
  });
}

async function withRetries(fn, label) {
  let lastErr;
  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    try {
      return await fn();
    } catch (err) {
      lastErr = err;
      if (attempt < MAX_RETRIES) {
        const backoffMs = 500 * 2 ** (attempt - 1);
        console.warn(
          `\x1b[33m⚠ ${label} failed (attempt ${attempt}/${MAX_RETRIES}): ${err.message}. Retrying in ${backoffMs}ms...\x1b[0m`
        );
        await new Promise((r) => setTimeout(r, backoffMs));
      }
    }
  }
  throw lastErr;
}

// ---------------------------------------------------------------------------
// Checksum verification
// ---------------------------------------------------------------------------
// GitHub Go-release tooling (goreleaser etc.) conventionally publishes a
// `checksums.txt` (sha256sum-style) alongside the archives. We fetch it and
// verify the specific archive's hash before ever extracting or executing it.

async function sha256File(filePath) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash("sha256");
    const stream = fs.createReadStream(filePath);
    stream.on("data", (chunk) => hash.update(chunk));
    stream.on("end", () => resolve(hash.digest("hex")));
    stream.on("error", reject);
  });
}

async function verifyChecksum(archivePath, archiveName) {
  const checksumsUrl = `https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt`;

  let checksumsText;
  try {
    checksumsText = await withRetries(() => fetchText(checksumsUrl), "Fetching checksums.txt");
  } catch (err) {
    console.warn(`\x1b[33m⚠ Checksum file unavailable (${err.message}). Skipping hash check.\x1b[0m`);
    return;
  }

  const line = checksumsText
    .split("\n")
    .map((l) => l.trim())
    .find((l) => l.endsWith(archiveName));

  if (!line) {
    console.warn(`\x1b[33m⚠ No checksum entry found for ${archiveName}. Skipping hash check.\x1b[0m`);
    return;
  }

  const expected = line.split(/\s+/)[0].toLowerCase();
  const actual = (await sha256File(archivePath)).toLowerCase();

  if (expected !== actual) {
    throw new Error(
      `Checksum mismatch for ${archiveName}\n  expected: ${expected}\n  actual:   ${actual}\n` +
      `The downloaded file may be corrupted or tampered with. Aborting.`
    );
  }
}

// ---------------------------------------------------------------------------
// Install
// ---------------------------------------------------------------------------

function platformArch() {
  const platformMap = { win32: "Windows", darwin: "Darwin", linux: "Linux" };
  const archMap = { x64: "x86_64", arm64: "arm64", ia32: "i386" };

  const platform = platformMap[os.platform()];
  const arch = archMap[os.arch()];

  if (!platform || !arch) {
    throw new Error(`Unsupported platform/architecture: ${os.platform()}/${os.arch()}`);
  }
  return { platform, arch };
}

function extract(archivePath, cacheDir) {
  if (os.platform() === "win32") {
    // Array-form args via execFileSync avoid shell-quoting/injection issues
    // that the original string-interpolated `powershell -Command` had.
    execFileSync(
      "powershell",
      [
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        "Expand-Archive",
        "-LiteralPath",
        archivePath,
        "-DestinationPath",
        cacheDir,
        "-Force",
      ],
      { stdio: "ignore" }
    );
  } else {
    execFileSync("tar", ["-xzf", archivePath, "-C", cacheDir], { stdio: "ignore" });
  }
}

async function downloadAndInstall(targetPath) {
  const { platform, arch } = platformArch();
  const ext = os.platform() === "win32" ? "zip" : "tar.gz";
  const archiveName = `razify_${platform}_${arch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/download/${VERSION}/${archiveName}`;

  const cacheDir = getCacheDir();
  // Download to a per-process temp name, then rename into place atomically.
  // This avoids two concurrent `npx razify` invocations racing on the same
  // partially-written file, and avoids leaving a half-downloaded archive
  // under the "real" name if we get interrupted.
  const tmpArchivePath = path.join(
    cacheDir,
    `.${archiveName}.${process.pid}.${Date.now()}.tmp`
  );

  console.log(`\x1b[36m⚡ Razify: downloading release binary (${platform}/${arch}, ${VERSION})...\x1b[0m`);

  const cleanupTmp = () => {
    if (fs.existsSync(tmpArchivePath)) {
      try {
        fs.unlinkSync(tmpArchivePath);
      } catch {
        /* best effort */
      }
    }
  };
  process.once("exit", cleanupTmp);
  process.once("SIGINT", () => {
    cleanupTmp();
    process.exit(130);
  });
  process.once("SIGTERM", () => {
    cleanupTmp();
    process.exit(143);
  });

  try {
    try {
      await withRetries(() => downloadToFile(url, tmpArchivePath), "Downloading release archive");
    } catch (err) {
      if (err.message.includes("404")) {
        const fallbackUrl = `https://github.com/${REPO}/releases/latest/download/${archiveName}`;
        console.warn(`\x1b[33m⚠ Asset ${VERSION} not found. Retrying with latest release...\x1b[0m`);
        await withRetries(() => downloadToFile(fallbackUrl, tmpArchivePath), "Downloading latest release archive");
      } else {
        throw err;
      }
    }
    await verifyChecksum(tmpArchivePath, archiveName);

    extract(tmpArchivePath, cacheDir);

    if (!isExecutableFile(targetPath)) {
      if (os.platform() !== "win32" && fs.existsSync(targetPath)) {
        fs.chmodSync(targetPath, 0o755);
      }
    }
    if (!fs.existsSync(targetPath)) {
      throw new Error(`Extracted archive did not contain expected binary at ${targetPath}`);
    }

    console.log(`\x1b[32m✔ Installed Razify ${VERSION} to ${targetPath}\x1b[0m\n`);
  } finally {
    cleanupTmp();
  }
}

// ---------------------------------------------------------------------------
// Execution
// ---------------------------------------------------------------------------

function spawnBinary(binaryPath, args) {
  const child = spawn(binaryPath, args, { stdio: "inherit", shell: false });

  child.on("error", (err) => {
    console.error(`\x1b[31m✘ Failed to launch Razify: ${err.message}\x1b[0m`);
    process.exit(1);
  });

  // Forward common termination signals to the child instead of just dying
  // and leaving it orphaned.
  const forward = (sig) => {
    if (!child.killed) child.kill(sig);
  };
  process.on("SIGINT", () => forward("SIGINT"));
  process.on("SIGTERM", () => forward("SIGTERM"));

  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code ?? 0);
    }
  });
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

async function run() {
  const args = process.argv.slice(2);

  try {
    const existing = findBinary();
    if (existing) {
      spawnBinary(existing, args);
      return;
    }

    const targetPath = path.join(getCacheDir(), getBinaryName());
    await downloadAndInstall(targetPath);
    spawnBinary(targetPath, args);
  } catch (err) {
    console.error(`\x1b[31m✘ Razify setup failed: ${err.message}\x1b[0m`);
    console.error(
      `👉 You can build from source instead: 'go build -o ${getBinaryName()} main.go', ` +
      `or install via Homebrew / Scoop if a formula is available.`
    );
    process.exitCode = 1;
  }
}

run();