# Fibo Planner

Fibo Planner is a lightweight, real-time planning poker app for agile teams. Open a room, share the link, and estimate stories with Fibonacci cards until the team agrees on effort.

No accounts, no database, no extra services. A single Go binary (or Docker image) serves the UI and keeps everyone in sync over WebSockets.

## Key features

### Rooms and lobby

- **Create a room in one click.** Optional display name (for example, “Sprint 42 backlog”) plus a random 6-digit ID and a shareable URL (`/123456`).
- **Live lobby.** The home page lists every open room with current user counts. Counts update while the page is open.
- **Named join.** Each person enters a display name before voting. The name is remembered in the browser for that room.
- **Idle cleanup.** Empty rooms are removed after 30 minutes so the lobby does not fill with abandoned sessions.

### Planning poker

- **Fibonacci cards:** 1, 2, 3, 5, 8, 13, 20, plus a blank card to clear your vote.
- **Hidden votes by default.** Other people’s points stay masked (`???`) until every voter has picked a card, so nobody anchors on the first vote.
- **Always show votes.** Flip a room-wide switch when you want scores visible as they come in.
- **Observer mode.** Join as a facilitator or stakeholder without voting. Observers are listed separately and do not block reveal.
- **Topic title.** Set the story or ticket the room is estimating; it updates live for everyone.
- **Preloaded topic queue.** Paste a backlog (one title per line, up to 200). Load Next Topic advances the queue, sets the heading, and clears votes for the next round.

### Consensus and results

Results appear only after every voter has voted:

- **Tally table** with point value, count, and percentage. The leading value is highlighted.
- **Agreed points** when the room meets both consensus rules; otherwise `N/A`.
- **Agreement status** showing whether the leading vote meets the percentage threshold and whether the spread is within the allowed range.

You can tune what “agreement” means:

| Control        | Meaning                                                                                                 | Range   |
| -------------- | ------------------------------------------------------------------------------------------------------- | ------- |
| **Percentage** | Share of voters that must land on the same value                                                        | 50–100% |
| **Max spread** | Allowed distance on the Fibonacci scale between the lowest and highest vote (3 and 5 → 1; 1 and 20 → 6) | 0–6     |

Team maturity presets apply both knobs at once:

- **Full** — 100% agreement, 0 spread (everyone on the same card)
- **Good** — 80% agreement, spread of 1
- **Relaxed** — 50% agreement, spread of 3

### Live, self-contained app

- Instant updates for votes, joins, role changes, and lobby counts (WebSocket + HTMX).
- One process on port `8080`. HTML is embedded in the binary; the Docker image is built `FROM scratch`.
- MIT licensed.

## Run it

```bash
go run ./app
```

Or with Docker:

```bash
docker run -p 8080:8080 siakhooi/fibo-planner
```

Then open [http://localhost:8080](http://localhost:8080).

With [just](https://github.com/casey/just): `just run` or `just docker-run`.

## Typical session

1. Create a room from the home page and share the URL.
2. Teammates join with their names.
3. Set a topic (or load the next preloaded title).
4. Everyone votes. Votes stay hidden until the last voter submits (unless always-show is on).
5. Read the tally, agreed points, and agreement status. Adjust consensus rules if the team wants a looser or tighter bar.
6. Clear votes or load the next topic and repeat.

## Reference

### Deliverables

- https://hub.docker.com/r/siakhooi/fibo-planner

### Quality

- https://sonarcloud.io/project/overview?id=siakhooi_fibo-planner
- https://qlty.sh/gh/siakhooi/projects/fibo-planner

## Badges

![GitHub](https://img.shields.io/github/license/siakhooi/fibo-planner?logo=github)
![GitHub last commit](https://img.shields.io/github/last-commit/siakhooi/fibo-planner?logo=github)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/siakhooi/fibo-planner?logo=github)
![GitHub issues](https://img.shields.io/github/issues/siakhooi/fibo-planner?logo=github)
![GitHub closed issues](https://img.shields.io/github/issues-closed/siakhooi/fibo-planner?logo=github)
![GitHub pull requests](https://img.shields.io/github/issues-pr-raw/siakhooi/fibo-planner?logo=github)
![GitHub closed pull requests](https://img.shields.io/github/issues-pr-closed-raw/siakhooi/fibo-planner?logo=github)
![GitHub top language](https://img.shields.io/github/languages/top/siakhooi/fibo-planner?logo=github)
![GitHub language count](https://img.shields.io/github/languages/count/siakhooi/fibo-planner?logo=github)
![GitHub repo size](https://img.shields.io/github/repo-size/siakhooi/fibo-planner?logo=github)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/siakhooi/fibo-planner?logo=github)
![Workflow](https://img.shields.io/badge/Workflow-github-purple)
![workflow](https://github.com/siakhooi/fibo-planner/actions/workflows/build.yaml/badge.svg)
![workflow](https://github.com/siakhooi/fibo-planner/actions/workflows/release.yaml/badge.svg)

![Release](https://img.shields.io/badge/Release-github-purple)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/siakhooi/fibo-planner?label=GPR%20release&logo=github)
![GitHub all releases](https://img.shields.io/github/downloads/siakhooi/fibo-planner/total?color=33cb56&logo=github)
![GitHub Release Date](https://img.shields.io/github/release-date/siakhooi/fibo-planner?logo=github)

![Quality-Qlty](https://img.shields.io/badge/Quality-Qlty-purple)
[![Maintainability](https://qlty.sh/gh/siakhooi/projects/fibo-planner/maintainability.svg)](https://qlty.sh/gh/siakhooi/projects/fibo-planner)
[![Code Coverage](https://qlty.sh/gh/siakhooi/projects/fibo-planner/coverage.svg)](https://qlty.sh/gh/siakhooi/projects/fibo-planner)

![Quality-Sonar](https://img.shields.io/badge/Quality-SonarCloud-purple)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=bugs)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=coverage)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)
![Sonar Violations (short format)](https://img.shields.io/sonar/violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (short format)](https://img.shields.io/sonar/blocker_violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (short format)](https://img.shields.io/sonar/critical_violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (short format)](https://img.shields.io/sonar/major_violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (short format)](https://img.shields.io/sonar/minor_violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (short format)](https://img.shields.io/sonar/info_violations/siakhooi_fibo-planner?server=https%3A%2F%2Fsonarcloud.io)
![Sonar Violations (long format)](https://img.shields.io/sonar/violations/siakhooi_fibo-planner?format=long&server=http%3A%2F%2Fsonarcloud.io)

[![Wise](https://img.shields.io/badge/Funding-Wise-33cb56.svg?logo=wise)](https://wise.com/pay/me/siakn3)
![visitors](https://hit-tztugwlsja-uc.a.run.app/?outputtype=badge&counter=ghmd-fibo-planner)
