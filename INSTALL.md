# Install Fibo Planner

Fibo Planner is a single binary. After any method below, start it and open [http://localhost:8080](http://localhost:8080).

```bash
fibo-planner
```

The process listens on `:8080`. Flags, environment variables, Docker, and Kubernetes are in the [README](README.md#run-it).

| Method                            | Where it runs                                              |
| --------------------------------- | ---------------------------------------------------------- |
| [Homebrew](#homebrew)             | macOS and Linux                                            |
| [Scoop](#scoop)                   | Windows (x86_64)                                           |
| [APT](#apt-debian-and-ubuntu)     | Debian and Ubuntu (amd64, arm64)                           |
| [RPM](#rpm-yum-and-dnf)           | yum and dnf (x86_64, aarch64)                              |
| [GitHub release](#github-release) | Linux, macOS, and Windows archives, plus `.deb` and `.rpm` |
| [go install](#go-install)         | Any machine with the Go toolchain                          |
| [Docker](#docker)                 | Any machine with Docker                                    |

## Homebrew

Tap and formula: [siakhooi/homebrew-tap](https://github.com/siakhooi/homebrew-tap).

```bash
brew tap siakhooi/tap
brew install fibo-planner
```

## Scoop

Bucket: [siakhooi/scoop-bucket](https://github.com/siakhooi/scoop-bucket). The manifest installs the Windows x86_64 build.

```bash
scoop bucket add siakhooi https://github.com/siakhooi/scoop-bucket.git
scoop install fibo-planner
```

## APT (Debian and Ubuntu)

Repository: [siakhooi/apt-linux](https://github.com/siakhooi/apt-linux). Site: [siakhooi.github.io/apt-linux](https://siakhooi.github.io/apt-linux/).

One-time setup, then install. The package name is `fibo-planner` and the binary is `/usr/bin/fibo-planner`.

```bash
sudo curl -L https://siakhooi.github.io/apt-linux/siakhooi-apt-linux.list | sudo tee /etc/apt/sources.list.d/siakhooi-apt-linux.list > /dev/null
sudo curl -L https://siakhooi.github.io/apt-linux/siakhooi-apt-linux.gpg | sudo tee /usr/share/keyrings/siakhooi-apt-linux.gpg > /dev/null
sudo apt update
sudo apt install fibo-planner
```

As root, the same files can be written directly:

```bash
curl -L https://siakhooi.github.io/apt-linux/siakhooi-apt-linux.list -o /etc/apt/sources.list.d/siakhooi-apt-linux.list
curl -L https://siakhooi.github.io/apt-linux/siakhooi-apt-linux.gpg -o /usr/share/keyrings/siakhooi-apt-linux.gpg
apt update
apt install fibo-planner
```

## RPM (yum and dnf)

Repository: [siakhooi/rpms](https://github.com/siakhooi/rpms). Site: [siakhooi.github.io/rpms](https://siakhooi.github.io/rpms/).

One-time setup writes `/etc/yum.repos.d/siakhooi-rpms.repo`. The package name is `fibo-planner`.

```bash
sudo curl -L https://siakhooi.github.io/rpms/siakhooi-rpms.repo | sudo tee /etc/yum.repos.d/siakhooi-rpms.repo > /dev/null
sudo yum install fibo-planner
```

On Fedora and other dnf systems, use the same repo file:

```bash
sudo dnf install fibo-planner
```

As root:

```bash
curl -L https://siakhooi.github.io/rpms/siakhooi-rpms.repo -o /etc/yum.repos.d/siakhooi-rpms.repo
yum install fibo-planner
```

## GitHub release

Download a build from the [latest release](https://github.com/siakhooi/fibo-planner/releases/latest). Each release also publishes `checksums.txt`.

Archive names follow `fibo-planner_<version>_<OS>_<arch>`:

| OS      | x86_64                                        | arm64                                        |
| ------- | --------------------------------------------- | -------------------------------------------- |
| Linux   | `fibo-planner_<version>_Linux_x86_64.tar.gz`  | `fibo-planner_<version>_Linux_arm64.tar.gz`  |
| macOS   | `fibo-planner_<version>_Darwin_x86_64.tar.gz` | `fibo-planner_<version>_Darwin_arm64.tar.gz` |
| Windows | `fibo-planner_<version>_Windows_x86_64.zip`   | `fibo-planner_<version>_Windows_arm64.zip`   |

The same release includes `.deb` (`amd64`, `arm64`) and `.rpm` (`x86_64`, `aarch64`) if you want a package file without adding a repository:

```bash
sudo apt install ./fibo-planner_<version>_amd64.deb
sudo dnf install ./fibo-planner-<version>-1.x86_64.rpm
```

Linux x86_64 example for [v0.7.1](https://github.com/siakhooi/fibo-planner/releases/tag/v0.7.1). Replace the version and asset name for a newer release.

```bash
curl -LO https://github.com/siakhooi/fibo-planner/releases/download/v0.7.1/fibo-planner_0.7.1_Linux_x86_64.tar.gz
curl -LO https://github.com/siakhooi/fibo-planner/releases/download/v0.7.1/checksums.txt
grep fibo-planner_0.7.1_Linux_x86_64.tar.gz checksums.txt | sha256sum -c -
tar -xzf fibo-planner_0.7.1_Linux_x86_64.tar.gz
./fibo-planner
```

On macOS, `shasum -a 256 -c` checks the same `checksums.txt` line.

## go install

The module is `github.com/siakhooi/fibo-planner` and the main package is `./app`. `go.mod` requires Go 1.27.1 or newer.

```bash
go install github.com/siakhooi/fibo-planner/app@latest
```

Pin a release tag the same way, for example `go install github.com/siakhooi/fibo-planner/app@v0.7.1`.

`go install` names the binary after the package directory, so the command on `PATH` is `app`. It is installed to `$(go env GOBIN)` when that is set, otherwise `$(go env GOPATH)/bin`. Rename it to match the release binary:

```bash
bindir="$(go env GOBIN)"
if [ -z "$bindir" ]; then
  bindir="$(go env GOPATH)/bin"
fi
mv "$bindir/app" "$bindir/fibo-planner"
```

A `go install` build reports `Version: 0.0.0` from `--version`. Release archives, packages, and Docker images embed the version through GoReleaser.

## Docker

```bash
docker run -p 8080:8080 siakhooi/fibo-planner
```

Image: [hub.docker.com/r/siakhooi/fibo-planner](https://hub.docker.com/r/siakhooi/fibo-planner). Listen address, lobby list, and custom HTML mounts are in the [README](README.md#run-it).
