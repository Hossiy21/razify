# How to distribute Razify via Homebrew

To enable `brew install razify`, you should create a personal Homebrew Tap.

### 1. Create a Tap Repository
Create a new GitHub repository named `homebrew-tap`.

### 2. Add the Formula
Create a file named `Formula/razify.rb` in that repository with the following content:

```ruby
class Razify < Formula
  desc "The missing CLI tool for .env file management"
  homepage "https://github.com/Hossiy21/razify"
  url "https://github.com/Hossiy21/razify/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256" # Run: curl -L [url] | shasum -a 256
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    system "#{bin}/razify", "version"
  end
end
```

### 3. Usage for Users
Once the tap is public, anyone can install Razify with:

```bash
brew tap Hossiy21/tap
brew install razify
```

---

## 🚀 Automation with GoReleaser (Recommended)

To automatically generate Homebrew formulas and Scoop manifests on every release, add a `.goreleaser.yaml` to your project:

```yaml
before:
  hooks:
    - go mod tidy
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
archives:
  - replacements:
      darwin: Darwin
      linux: Linux
      windows: Windows
      386: i386
      amd64: x86_64
checksum:
  name_template: 'checksums.txt'
snapshot:
  name_template: "{{ .Tag }}-next"
changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'

brews:
  - name: razify
    tap:
      owner: hossiy21
      name: homebrew-tap
    homepage: "https://github.com/hossiy21/razify"
    description: "The missing CLI tool for .env file management"

scoops:
  - name: razify
    tap:
      owner: hossiy21
      name: scoop-bucket
    homepage: "https://github.com/hossiy21/razify"
    description: "The missing CLI tool for .env file management"
```
