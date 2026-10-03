# Fibo Planner

Fibo Planner is a lightweight, real-time planning poker app for agile teams. Open a room, share the link, and estimate stories on a modified Fibonacci scale until the team agrees on effort.

No accounts, no database, no extra services. A single Go binary (or Docker image) serves the UI and keeps everyone in sync over WebSockets.

## Key features

- **One-click rooms.** Optional display name, a random 6-digit ID, and a shareable URL. Empty rooms are removed after 30 minutes.
- **Named join.** Each person enters a display name before voting. The name stays in the browser for that room and is not written to the process access log.
- **Modified Fibonacci scale:** 1, 2, 3, 5, 8, 13, 20, plus a blank card. Votes stay hidden until everyone has voted, unless the room turns on always-show.
- **Observer mode.** Sit out of voting. Observers do not block reveal.
- **Topic and queue.** Set the story title, or paste a backlog and load the next topic.
- **Consensus.** After everyone votes, the room shows a tally, agreed points, and whether the leading vote meets the percentage and spread rules.
- **Anyone in the room** can change the topic, clear votes, and adjust consensus rules. There is no facilitator role.

Voting, consensus, and a full session walkthrough are in the [user guide](USER_GUIDE.md).

## Try it

A public sample is hosted on [Google Cloud Run](https://cloud.google.com/run):

**[https://fibo-planner-189393002108.asia-southeast1.run.app/](https://fibo-planner-189393002108.asia-southeast1.run.app/)**

Create a room, share the link, and run a planning session there. Empty rooms are still removed after 30 minutes of idle time.

That sample is built from [fibo-planner-on-gcp](https://github.com/siakhooi/fibo-planner-on-gcp). Use it as a starting point if you want to host your own copy.

## Run it

Install a release binary with Homebrew, Scoop, APT, RPM, a GitHub release archive, or `go install`. Steps, flags, Docker, and Kubernetes are in [INSTALL.md](INSTALL.md).

From a checkout:

```bash
go run ./app
```

Or with Docker:

```bash
docker run -p 8080:8080 siakhooi/fibo-planner
```

Then open [http://localhost:8080](http://localhost:8080).

## Reference

### Deliverables

- https://fibo-planner-189393002108.asia-southeast1.run.app/
- https://github.com/siakhooi/fibo-planner-on-gcp
- https://hub.docker.com/r/siakhooi/fibo-planner

### Quality

- https://sonarcloud.io/project/overview?id=siakhooi_fibo-planner
- https://qlty.sh/gh/siakhooi/projects/fibo-planner

## Badges

![GitHub](https://img.shields.io/github/license/siakhooi/fibo-planner?logo=github)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/siakhooi/fibo-planner?label=GPR%20release&logo=github)
![workflow](https://github.com/siakhooi/fibo-planner/actions/workflows/build.yaml/badge.svg)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=siakhooi_fibo-planner&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=siakhooi_fibo-planner)

[![Wise](https://img.shields.io/badge/Funding-Wise-33cb56.svg?logo=wise)](https://wise.com/pay/me/siakn3)
![visitors](https://hit-tztugwlsja-uc.a.run.app/?outputtype=badge&counter=ghmd-fibo-planner)
