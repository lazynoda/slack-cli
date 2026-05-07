# slack-cli

`slack-cli` is a small Go CLI for the Slack Web API, shaped after the same style [`spogo`](https://github.com/openclaw/spogo) uses: a thin `cmd/<name>` entrypoint, `kong` for commands and flags, and internal packages for config, app context, and API clients.

It uses Slack bearer tokens with the official Web API at `https://slack.com/api/*`.

## Features

- Profile-based token storage in `os.UserConfigDir()/slack-cli/config.json`
- Env var override via `SLACK_CLI_TOKEN` or `SLACK_TOKEN`
- Optional cookie header support via `SLACK_CLI_COOKIE` or `SLACK_COOKIE`
- `auth status`, `auth set-token`, `auth clear`, `auth test`
- Raw `api call` for any Slack Web API method
- `users list|info|presence`
- `conversations list|info|history|replies|members|open|join|leave|create|rename|topic|purpose|archive|unarchive|mark|unread`
- `chat send|ephemeral|update|delete|permalink`
- `search messages`
- Human, JSON, and plain output modes
- Basic rate-limit retry using `Retry-After`

## Build

```bash
go build ./cmd/slack-cli
```

## Quick Start

Create a Slack app and install it to a workspace, then store a bot or user token with the scopes you need.

```bash
./slack-cli auth set-token xoxb-...
export SLACK_COOKIE='d=...'
./slack-cli auth test
./slack-cli users list --limit 10
./slack-cli conversations list --limit 20
./slack-cli conversations history #general --limit 10
./slack-cli conversations replies #general 1746592769.123456
./slack-cli conversations open @alice
./slack-cli conversations open @alice @bob
./slack-cli chat send #general "hello from slack-cli"
./slack-cli chat dm @alice "hello in DM"
./slack-cli chat dm @alice @bob "hello group DM"
./slack-cli chat permalink #general 1746592769.123456
./slack-cli users presence @alice
./slack-cli resolve user @alice
./slack-cli resolve conversation #general
./slack-cli search messages "deployment failed"
./slack-cli api call conversations.info --http GET --param channel=C0123456789
```

On macOS, the default config path is typically:

```bash
~/Library/Application\ Support/slack-cli/config.json
```

## Usage

```bash
slack-cli [global flags] <command> [args]
```

Global flags:

- `--config <path>` config file path
- `--profile <name>` config profile name
- `--timeout <dur>` request timeout (default `10s`)
- `--json` structured JSON output
- `--plain` script-friendly plain output

Commands:

- `auth status|set-token|clear|test`
- `api call`
- `resolve user|conversation`
- `users list|info|presence`
- `conversations list|info|history|replies|members|open|join|leave|create|rename|topic|purpose|archive|unarchive|mark|unread`
- `chat send|dm|ephemeral|update|delete|permalink`
- `search messages`

## Tokens and Scopes

The CLI sends tokens in the `Authorization: Bearer <token>` header, which matches Slack's current Web API guidance.

Typical scopes you may need:

- `users:read` for `users list|info|presence`
- `conversations:read`, plus channel/private-group history scopes depending on token/workspace setup, for `conversations list|info|history|replies|members|unread`
- `conversations:write`-style scopes where applicable for `conversations create|rename|topic|purpose|archive|unarchive|mark|join|leave|open`
- `chat:write` for `chat send|dm|ephemeral|update|delete`
- search scopes supported by your token for `search messages`

Exact scope requirements depend on the token type and the methods you call.

## Notes

- `users list`, `conversations list`, `conversations history`, `conversations replies`, and `conversations members` support `--all` and follow `response_metadata.next_cursor`.
- Most commands accept raw Slack IDs or friendly references like `#general`, `general`, and `@alice`.
- `conversations unread` derives unread messages from `conversations.info` -> `last_read` plus `conversations.history`; it depends on Slack returning `last_read` for the active token and conversation.
- `api call` is intentionally thin and prints JSON so you can reach any Slack Web API method before adding a typed command.
- If both config and env vars are present, env vars win.
- If Slack requires a cookie-backed session for your token, set `SLACK_COOKIE` to the raw `Cookie` header value and the CLI will send it with each request.

## Inspiration

This layout intentionally follows the same broad pattern as `spogo`:

- `cmd/slack-cli/main.go` keeps startup logic thin
- `internal/cli` defines the command tree
- `internal/app` owns runtime wiring
- `internal/config` handles local config
- `internal/slack` wraps remote API calls
