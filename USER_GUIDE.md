# Fibo Planner user guide

How a planning session works. Install and server options are in [INSTALL.md](INSTALL.md).

## Rooms and lobby

- **Create a room in one click.** Optional display name (for example, “Sprint 42 backlog”) plus a random 6-digit ID and a shareable URL (`/123456`).
- **Live lobby.** The home page shows how many people are on the lobby, how many rooms are open, and how many people are in rooms. Counts update while the page is open. Listing each room with its user count is off by default; turn it on with `FIBO_PLANNER_LOBBY_LIST_ROOMS` or `--lobby-list-rooms` ([INSTALL.md](INSTALL.md#lobby-room-list)).
- **Named join.** Each person enters a display name before voting. The name is remembered in the browser for that room and sent on the WebSocket after connect, not as a `?name=` query string, so the process access log does not record it.
- **Idle cleanup.** Empty rooms are removed after 30 minutes so the lobby does not fill with abandoned sessions.

## Planning poker

- **Modified Fibonacci scale:** 1, 2, 3, 5, 8, 13, 20 (20 in place of the usual 21), plus a blank card to clear your vote.
- **Hidden votes by default.** Other people’s points stay masked (`???`) until every voter has picked a card, so nobody anchors on the first vote.
- **Always show votes.** Flip a room-wide switch when you want scores visible as they come in.
- **Observer mode.** Sit out of voting (for example as a stakeholder). Observers are listed separately and do not block reveal. This only changes whether you vote; it is not an admin role.
- **Topic title.** Set the story or ticket the room is estimating; it updates live for everyone. Set Topic changes the heading only and leaves the current votes in place.
- **Preloaded topic queue.** Paste a backlog (one title per line, up to 200). Load Next Topic advances the queue, sets the heading, and clears votes for the next round.

If the room socket drops, the room page shows a reconnecting banner and tries to reconnect.

## Consensus and results

Results appear only after every voter has voted:

- **Tally table** with point value, count, and percentage. The leading value is highlighted.
- **Agreed points** when the room meets both consensus rules; otherwise `N/A`.
- **Agreement status** showing whether the leading vote meets the percentage threshold and whether the spread is within the allowed range.

You can tune what “agreement” means:

| Control        | Meaning                                                                                                          | Range   |
| -------------- | ---------------------------------------------------------------------------------------------------------------- | ------- |
| **Percentage** | Share of voters that must land on the same value                                                                 | 50–100% |
| **Max spread** | Allowed distance on the modified Fibonacci scale between the lowest and highest vote (3 and 5 → 1; 1 and 20 → 6) | 0–6     |

Team maturity presets apply both knobs at once:

- **Full** — 100% agreement, 0 spread (everyone on the same card)
- **Good** — 80% agreement, spread of 1
- **Relaxed** — 50% agreement, spread of 3

## Who can administer a room

There are no accounts. **Anyone who has joined the room** can clear votes, set the topic and preloaded queue, toggle always-show votes, and change consensus rules. The server does not distinguish a facilitator from other occupants; the Administration panel is the same for every socket.

Limiting those controls to some members is a planned enhancement: [issue #60](https://github.com/siakhooi/fibo-planner/issues/60).

## Typical session

1. Create a room from the home page and share the URL.
2. Teammates join with their names.
3. Anyone in the room can set a topic. Set Topic changes the heading and leaves the current votes in place. Load Next Topic sets the heading from the preloaded queue and clears votes.
4. Everyone votes. Votes stay hidden until the last voter submits (unless always-show is on).
5. Read the tally, agreed points, and agreement status. Anyone can adjust consensus rules if the team wants a looser or tighter bar.
6. Clear votes to estimate the same topic again, or load the next topic (that also clears votes) and repeat.
