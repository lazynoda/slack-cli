# Workflow Notes

## Core Conversation Flows

These are the main workflows the repo currently supports.

## Inspect auth

```bash
./slack-cli auth status
./slack-cli auth test
```

Use this first when requests fail.

## Resolve names to IDs

```bash
./slack-cli resolve user @alice
./slack-cli resolve conversation #general
```

This is useful when debugging ambiguous names or scripting with raw IDs.

## Read a channel

```bash
./slack-cli conversations history #general --limit 20
./slack-cli conversations replies #general 1746592769.123456
./slack-cli conversations unread #general
```

## Start or continue a DM

```bash
./slack-cli conversations open @alice
./slack-cli conversations open @alice @bob
./slack-cli chat dm @alice "hello"
./slack-cli chat dm @alice @bob "hello"
```

## Send to a named conversation

```bash
./slack-cli chat send #general "hello team"
./slack-cli chat update #general 1746592769.123456 "updated text"
./slack-cli chat delete #general 1746592769.123456
./slack-cli chat permalink #general 1746592769.123456
```

## Search

```bash
./slack-cli search messages "deployment failed"
./slack-cli search messages "from:me"
./slack-cli search messages "in:#general"
```

Slack query syntax is often the fastest way to implement inbox-like or discovery workflows without adding more commands.

## Practical Guidance For Agents

- Prefer extending existing commands before creating new top-level namespaces
- Prefer friendly references in examples unless a raw ID is necessary
- If a command can reasonably accept a user or conversation name, hook it into the resolver layer
- If the user asks for “unread” behavior, first check whether it can be expressed through existing `conversations unread` or `search messages`
