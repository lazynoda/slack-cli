# Agent Notes

This repo contains a Go CLI for talking to Slack through the Slack Web API.

## Purpose

The project is intentionally focused on everyday Slack conversation workflows:

- authenticate with a token and optional cookie
- resolve users and conversations from friendly names
- read conversations and threads
- open DMs and MPDMs
- send, update, delete, and permalink messages
- search messages

Do not expand the tool toward admin, app-management, profile-management, or calling features unless explicitly requested.

## Current Design

- `cmd/slack-cli/main.go`: thin process entrypoint
- `internal/cli/cli.go`: `kong` command tree and output shaping
- `internal/app/context.go`: runtime wiring, token/cookie lookup, name resolution
- `internal/slack/client.go`: Slack Web API HTTP client and typed method wrappers
- `internal/config/config.go`: local profile config
- `internal/output/output.go`: human/plain/json output helpers

## Auth Model

Requests may require both a bearer token and a cookie-backed session.

Supported env vars:

- `SLACK_CLI_TOKEN`
- `SLACK_TOKEN`
- `SLACK_CLI_COOKIE`
- `SLACK_COOKIE`

The client currently sends:

- `Authorization: Bearer <token>`
- `Cookie: <cookie>` when configured

When debugging auth issues, check `slack-cli auth status` first.

## Name Resolution

Commands should prefer the centralized resolver logic in `internal/app/context.go`.

Supported friendly references today:

- users: `@alice`, `alice`, display name, real name, raw `U...`
- conversations: `#general`, `general`, raw `C...` / `G...` / `D...`

If you add commands that take users or conversations, route them through the existing resolver methods instead of duplicating lookup logic.

## Output Conventions

Commands support:

- default human output
- `--plain`
- `--json`

When adding commands, preserve this pattern. Prefer typed output in the CLI layer rather than making the HTTP client care about presentation.

## Scope Guardrails

Good additions:

- conversation reading and messaging ergonomics
- better thread workflows
- better unread/inbox-style workflows
- small resolver improvements

Avoid by default:

- Slack admin APIs
- app installation or app config flows
- profile editing
- calls/huddles/media features
- unrelated automation outside conversation use

## Verification

Typical verification commands:

```bash
gofmt -w internal/app/context.go internal/cli/cli.go internal/slack/client.go
go build ./cmd/slack-cli
go test ./...
go run ./cmd/slack-cli --help
```

If you change command wiring, also run targeted help commands for the affected namespace.
