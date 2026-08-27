# Contributing to raind

Thanks for taking a look. raind is a small Go program, and contributions that fit its size are welcome. Bug reports, feature ideas, and pull requests all help.

## Getting started

You need Go 1.21 or newer.

Build and run:

```bash
go build -o raind .
./raind
```

The hotkeys and flags are documented in the README, so `./raind --help` plus the key list there covers most of what you need to try a change.

The project has one runtime dependency, tcell, and raind should stay a single static binary with no CGO. New dependencies need a good reason and a conversation first, usually in the issue or the PR.

## Making changes

Keep each pull request focused on one thing. A PR that fixes a bug and rewrites the README at the same time is harder to review, so split those apart. Small PRs get reviewed faster.

For anything more than a one line fix, opening an issue first is a good idea. It gives the maintainers a place to say yes, no, or steer the approach before you put work in.

## Commit messages

The repo already uses conventional prefixes, so keep that going:

```text
feat: add city skyline to snow mode
fix: clamp meteor speed on small terminals
docs: list the new --city flag
test: cover FrameDelay speed scaling
```

Write the subject in the imperative, keep it under about 70 characters, and save details for the body. One commit per self contained change. Avoid bundling several unrelated edits into one commit, it makes a bad history to search later.

## Style

Format with gofmt and run `go vet ./...` before pushing. The codebase is small, so match the style of the file you are editing rather than introducing a new pattern.

Comments should explain why, not restate what the code says. A comment like "increment the frame counter" earns its keep only when the line is actually confusing.

## Testing

There are no tests today, which is a known gap. If your change touches one of the pure functions (mode parsing, frame delay, speed scaling), add a table test alongside it. Run the suite with `go test ./...` before opening the PR.

## Pull requests

Open PRs against `main`. In the description, say what the change does and why, and mention anything you did not test. Screenshots or recordings of terminal output are welcome and often the fastest way to show the result.

Maintainers review in their own time. If a PR sits for a week with no reply, a polite nudge in a comment is fine.