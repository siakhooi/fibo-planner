# Install Fibo Planner

Fibo Planner is a single binary. After any method below, start it and open [http://localhost:8080](http://localhost:8080).

```bash
fibo-planner
```

The process listens on `:8080`. Flags, environment variables, Kubernetes, and custom HTML are [below](#configure). How a planning session works is in the [user guide](USER_GUIDE.md).

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

Image: [hub.docker.com/r/siakhooi/fibo-planner](https://hub.docker.com/r/siakhooi/fibo-planner). The image is a static binary `FROM scratch`, running as UID 65532. Listen address, lobby list, and custom HTML mounts are [below](#configure).

## Run from a checkout

```bash
go run ./app
```

Print the build version and exit (`0.0.0` / `unknown` unless the binary was built with GoReleaser or `just build`). `-v` is the short form:

```bash
go run ./app --version
```

`-h` or `--help` prints command help and exits.

With [just](https://github.com/casey/just): `just run` or `just docker-run`.

Pages are embedded in the binary. The default UI still loads HTMX from jsDelivr.

## Kubernetes

A single-replica sample is in [`deploy/fibo-planner.yaml`](deploy/fibo-planner.yaml). Room state stays in the process, so keep `replicas: 1`.

```bash
kubectl apply -f deploy/fibo-planner.yaml
kubectl port-forward svc/fibo-planner 8080:80
```

Then open [http://localhost:8080](http://localhost:8080). The environment variables in [Configure](#configure) can be set on the container. Leave `FIBO_PLANNER_ADDR` unset so the process listens on `:8080` inside the pod.

[`deploy/fibo-planner-custom-html.yaml`](deploy/fibo-planner-custom-html.yaml) is the same sample with a ConfigMap mounted at `/custom`. Apply that file instead of the plain one. The process reads those files only at startup, so restart the Deployment after changing the ConfigMap.

## Configure

Restart the process after changing any of these. When a flag and its environment variable are both set, the flag wins.

SIGINT and SIGTERM stop the HTTP server (header timeout 10s, idle timeout 60s). WebSocket sessions are not drained as part of that shutdown. The server pings idle sockets so proxies (including Cloud Run) are less likely to drop a quiet planning session.

### Listen address

By default the process listens on `:8080` (all interfaces, port 8080). Set `FIBO_PLANNER_ADDR` or pass `--addr` (`-a`) to bind somewhere else.

```bash
FIBO_PLANNER_ADDR=127.0.0.1:9090 go run ./app
```

```bash
go run ./app --addr 127.0.0.1:9090
```

```bash
docker run -p 9090:9090 -e FIBO_PLANNER_ADDR=:9090 siakhooi/fibo-planner
```

### WebSocket origins

Browsers must send a same-origin `Origin` header (the page host). If the public site origin differs from the process `Host` header (some reverse proxies), set `FIBO_PLANNER_WS_ORIGINS` or pass `--ws-origins` with a comma-separated list of allowed origins.

```bash
FIBO_PLANNER_WS_ORIGINS=https://planner.example.com go run ./app
```

```bash
go run ./app --ws-origins https://planner.example.com
```

### Lobby room list

By default the home page shows totals only (people in the lobby, number of rooms, people in all rooms). Set `FIBO_PLANNER_LOBBY_LIST_ROOMS=Y` or pass `--lobby-list-rooms` to also list every open room with a link and its user count. Any other env value (or unset) keeps the list hidden. When the flag is passed it wins, including `--lobby-list-rooms=false` while the env var is `Y`.

```bash
FIBO_PLANNER_LOBBY_LIST_ROOMS=Y go run ./app
```

```bash
go run ./app --lobby-list-rooms
```

```bash
docker run -p 8080:8080 -e FIBO_PLANNER_LOBBY_LIST_ROOMS=Y siakhooi/fibo-planner
```

### Custom HTML

Set `FIBO_PLANNER_CUSTOM_HTML_DIR` or pass `--custom-html-dir` to a directory of optional snippets. Each file that exists is applied at process start:

| File              | Insertion point                                                                |
| ----------------- | ------------------------------------------------------------------------------ |
| `head.html`       | last line of `<head>` on every full page, just before `</head>`               |
| `body-start.html` | first line of `<body>` on every full page, just after `<body>`                |
| `body-end.html`   | last line of `<body>` on every full page, just before `</body>`               |
| `disclaimer.html` | body copy of `/disclaimer` only, after the heading and before the site footer |
| `privacy.html`    | body copy of `/privacy` only, after the heading and before the site footer    |
| `terms.html`      | body copy of `/terms` only, after the heading and before the site footer      |
| `llms.txt`        | replaces the built-in body of `GET /llms.txt` (not inserted into HTML pages)  |

The legal files are HTML fragments (paragraphs, headings, links), not full pages. Title, crumb, `<h1>`, footer, `head.html`, `body-start.html`, and `body-end.html` stay in place.

`GET /llms.txt` returns a plain-text guide for agents (create a room, join over the WebSocket, vote, and read results). A `llms.txt` in the custom directory replaces that guide entirely, including when the file is empty.

Missing files are skipped. Snippets are inserted as-is (not escaped); only use a directory you control.

```bash
FIBO_PLANNER_CUSTOM_HTML_DIR=/path/to/custom-html go run ./app
```

```bash
go run ./app --custom-html-dir /path/to/custom-html
```

Docker (`FROM scratch`, UID 65532) can still read a mounted directory. The files must be readable by that user:

```bash
docker run -p 8080:8080 \
  -e FIBO_PLANNER_CUSTOM_HTML_DIR=/custom \
  -v /path/to/custom-html:/custom:ro \
  siakhooi/fibo-planner
```
