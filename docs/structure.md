# Code Structure

## Overview

The CLI is split into a small number of focused packages.

## `cmd/slack-cli`

Process startup only.

- creates the root command
- parses args with `kong`
- builds the application context
- runs the selected command

Keep this layer thin.

## `internal/cli`

Owns the user-facing command surface.

- command structs
- argument and flag definitions
- output formatting for human/plain/json modes
- orchestration between resolvers and client methods

If a change is about CLI UX, it probably belongs here.

## `internal/app`

Owns runtime context and shared behavior.

- config path loading
- token and cookie lookup from env/config
- Slack client creation
- user and conversation resolution

If multiple commands need the same operational logic, prefer putting it here rather than duplicating it in `internal/cli`.

## `internal/slack`

Owns Slack HTTP requests and typed responses.

- HTTP transport
- headers
- rate-limit retry handling
- typed wrappers for Slack Web API methods

Keep presentation logic out of this package.

## `internal/config`

Owns profile-based config persistence.

Current stored data is intentionally small. Avoid adding lots of state unless there is a clear need.

## `internal/output`

Small helpers for:

- JSON output
- plain line output
- tabular human/plain output

## Editing Guidance

- If you add a new Slack method, start in `internal/slack/client.go`
- If you add a command, wire it in `internal/cli/cli.go`
- If the command needs name resolution, use `app.Context` methods
- If the command needs new shared resolution behavior, extend `internal/app/context.go`
