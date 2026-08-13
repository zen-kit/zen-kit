# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

The visual layer zen-review and zen-octo paint diffs with, at `zen-kit/zen-kit` (`origin`). Three packages and a demo:

```
theme/        palettes. Every style reads from one.
syntax/       Chroma tokens, not rendered text.
paint/        the diff-line painter. Pure functions.
cmd/kitdemo/  paints a canned diff to stdout and exits.
```

`theme` and `syntax` started as copies of zen-octo's. zen-octo still runs its own
copies and swaps them out under ZNO-43, on its own schedule. Two copies coexist
until then, and a change here does not reach zen-octo until it does.

**This module holds no behaviour.** No model, no state, no layout, no keys. Keys
are a convention written down in the consumers' `CLAUDE.md` files, not shared
code, so the two tools feel the same without either being hostage to the other's
release cycle.

Pre-1.0 while both consumers are ours. A breaking change is a minor bump and two
import fixes, not a deprecation cycle.

**`main` is the product branch.** Feature work flows ticket → branch → PR on `origin`.

Two things skip the PR and commit straight to `main`:

- Genuinely trivial tweaks. A typo, a one-liner.
- **Doc-only changes with no code.** Markdown, comments, `CLAUDE.md`, rules files. A PR for prose is ceremony.

A tracked pre-push hook rejects pushes to `main`, so an agent commits these and Drew pushes them. Don't reach for `--no-verify`.

Anything published under Drew's name (PR bodies, issues, README) must be shown to him word-for-word before pushing. His voice: terse, considerate, stoic, no strong adverbs, no em-dashes.

## Conventions

@.claude/rules/code-quality.md

That file holds only the Go specifics. The principles and voice rules are global and load automatically; don't copy them in here, that only creates drift.

## Commands

```sh
make all              # lint (gofmt + mod-tidy + golangci-lint) + test + build
make test             # go test -race -coverprofile ./...
make lint             # includes gofmt check and go.mod tidiness
make fmt-fix          # gofmt -w .
make golden           # regenerate paint/testdata
go run ./cmd/kitdemo  # look at it
go test ./paint/ -run TestName   # single test
```

Run checks directly, never through a pipe that swallows exit codes. `make lint | tail` reports success on failure.

**Judge a rendering change in a terminal, not in a diff.** `go run ./cmd/kitdemo` is the proof; a golden file only holds it still. The demo covers a hunk header, all three line kinds, a tab-indented line, a clipped row and a `Fill` row, so a theme change shows everything it broke in one screen.

### Lint version pin

CI pins golangci-lint to match the local brew version (`.github/workflows/ci.yml`). Keep the pin current with the local version, or CI and local runs stop agreeing.

### Git hooks

`.githooks/pre-push` is tracked and rejects pushes to `main`. `git config core.hooksPath .githooks` wires it up; the SessionStart hook does this on every session so a fresh clone is covered. Untracked `.git/hooks/` files don't survive a clone, which is why the hook lives here instead.

## Charm module paths

The Charm v2 line lives under `charm.land/*`, not `github.com/charmbracelet/*`. `github.com/charmbracelet/lipgloss/v2` does not resolve. Version numbers are the same across both paths.

## Project Management

Work is tracked in Linear: Praxis Labs workspace, **Zen Review** team (key `ZNR`, tickets `ZNR-###`), reached through the `linear-zen-review` MCP server declared in `.mcp.json`. zen-kit has no team of its own: it buys nothing while zen-review is its only consumer. Address projects and statuses **by name, never a UUID**; ids don't survive workspace moves.

The bucket names are shared with other teams, so `save_issue` resolving a bare project name can land on another team's copy and fail the call. Pass the Zen Review project id in that one argument when it does.

### Projects

Five long-running buckets, plus the current epic. Every ticket belongs to exactly one:

- **Polish & Bugs**: bugs and rough edges in surfaces that already ship.
- **Feature Backlog**: net-new capabilities. Ideas live here until promoted.
- **Performance and Code-Quality**: improves the code, no user-visible change.
- **Website**: the public site, its copy, its SEO.
- **Release & Distribution**: how the binary gets from `main` to a user.
- **Zen Review v0.1**: the current epic, milestones M0 through M8. zen-kit work files here while zen-review v0.1 is what it serves.

### Tickets

- Every ticket gets the team, exactly one project, a priority, and a status. No orphans.
- Create tickets as we go; never dump a full backlog up front.
- PR-sized scoping: 1 ticket = 1 branch = 1 PR as the rule of thumb. A ticket spanning both repos gets one PR in each.
- Keep descriptions lean: clear title, short goal and scope. No boilerplate acceptance criteria.
- Use Linear's generated branch name (`gitBranchName` from the MCP), never an invented one.
- Reference the ticket id in commits and the PR title/body so Linear auto-links.
- Status ladder: agent drives Backlog → Todo → In Progress. The GitHub integration owns In Review and Done; never write those by hand.

### Shipping

Feature-complete work ships via the global `ship-feature` skill: `make all` green, push, draft PR, Copilot + `/code-review`, triage with no tech debt, push then mark ready as separate actions. Manual invocation only.

**There is no copy of it in this repo.** Drew's global skills are a symlink into drucial-dots and load in every repo, so a copy here only shadows the real one and drifts behind it, which is what the copy this repo used to carry did. Edit the skill at its source.

### Specs

`docs/superpowers/specs/` holds the design docs that shaped a milestone. `docs/` otherwise describes only what is true today. Durable context lives in Linear project descriptions and tickets.

## Rendering traps

Each of these looks like working code and produces a broken frame. They are why `paint` exists rather than each tool rolling its own.

- **Every styled cell ends in a full SGR reset**, which clears the background along with the foreground. A row background has to be set per cell; wrapping a joined row paints only the first one, and the tint stops at the first token.
- **A row with a background has to be padded to the full width.** Otherwise the tint ends where the code does and the block reads as ragged. A row with no background needs no padding, which is the only reason a context line is cheaper.
- **`Style.Width` wraps before it clips.** Truncating to a column width means clipping explicitly first, or one long line of code becomes two rows.
- **Soft wrap and a line-number gutter cannot both be on.** One long line folds onto a second row, and every line under it is then one further out of step with the number beside it. Clip instead, and only ever measure at a width where something overflows.
- **A lexer carries state across lines.** Highlighting line by line comes apart on the first multi-line string. Tokenise the whole file, and tokenise the two sides of a diff separately, or the lexer reads a file holding both halves of every change. `syntax.Lines` takes a whole body for this reason; splitting a diff into two bodies is the caller's job.
- **A raw tab is a variable number of cells.** One anywhere in a line puts every column after it out of step with the line above. `paint` expands them.
- **Chroma's terminal formatter is unusable here.** It renders its own escapes, resets included. `syntax` returns tokens so the caller keeps control of the row.
- **A Chroma style carries a background.** Taking it paints over the terminal's, which is what keeps a transparent one transparent. Read the foreground only.
