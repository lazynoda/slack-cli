# slack-cli

A lightweight, profile-based command-line interface for the Slack Web API, written in Go. Built for developers, devops workflows, and terminal power users.

`slack-cli` makes it easy to read conversations, manage threads, send messages, search workspaces, and perform Slack Web API operations directly from your terminal. It supports standard API tokens (`xoxb-`/`xoxp-`) as well as cookie-backed client sessions (`xoxc-` client tokens + `d` session cookies).

Inspired by the design pattern of [`spogo`](https://github.com/openclaw/spogo), it uses a clean structure featuring a thin CLI entrypoint, `kong` for robust command-line argument parsing, and internal packages for configuration, application context, and API wrapper clients.

---

## Features

- 👤 **Profile-based Configuration**: Easily switch between workspaces or configurations using named profiles (stored in `~/.config/slack-cli/config.json` or OS equivalent).
- 🔑 **Multiple Auth Schemes**: Supports standard bot/user tokens as well as cookie-backed workspace sessions.
- 🌐 **Browser Cookie & Token Import**: Automatically import your active `d` session cookie and client token (`xoxc-`) from Chromium and Safari/Firefox browsers (Chrome, Brave, Arc, Microsoft Edge, Vivaldi, Opera, Safari, Firefox).
- 💬 **Conversation Management**: List channels, view message histories, fetch threads, join/leave/create/rename channels, set topic/purpose, open DMs/MPDMs, mark messages as read, and retrieve unread summaries.
- ✉️ **Messaging Ergonomics**: Send messages, post ephemeral alerts, update/delete messages, and generate message permalinks.
- 🔍 **Workspace Search**: Find messages across the entire workspace using the built-in search.
- 🛠️ **Raw API Access**: Call any Web API endpoint using the generic `api call` command with custom HTTP verbs and parameters.
- ⚙️ **Flexible Output Modes**: Outputs formatted for humans, clean `--plain` text for shell scripting, or structured `--json` (perfect for piping to `jq`).
- ⏳ **Rate-limit Resilience**: Built-in rate-limiting logic that automatically retries requests using Slack's `Retry-After` headers.

---

## Installation

To build `slack-cli` from source:

```bash
git clone https://github.com/lazynoda/slack-cli.git
cd slack-cli
go build ./cmd/slack-cli
```

Move the compiled binary to your system path (e.g., `/usr/local/bin`) to run it globally.

---

## Getting Started

### 1. Configure Authentication

`slack-cli` provides three main ways to authenticate:

#### Option A: Token Authentication (Recommended for Bots)
Set a Slack API token (`xoxb-...` or `xoxp-...`) for the default profile:
```bash
./slack-cli auth set-token xoxb-your-bot-token
```

#### Option B: Automatic Browser Import (Best for User accounts)
Import your active session cookie (`d`) and client token (`xoxc-...`) directly from your local browser (Chrome, Brave, Arc, Firefox, Safari, Edge, etc.):
```bash
# Import from Google Chrome's default profile
./slack-cli auth import-cookie --browser chrome

# Import from Chrome's specific profile directory (e.g. "Profile 6")
./slack-cli auth import-cookie --browser chrome --browser-profile "Profile 6"

# Import from Arc browser
./slack-cli auth import-cookie --browser arc
```
*Note: Under the hood, this command reads the browser's cookies and decrypts/parses your client configurations from Chrome's LevelDB Local Storage (`localConfig_v2` key).*

#### Option C: Environment Variables (Best for CI/CD)
Override any stored configurations dynamically using env variables:
```bash
export SLACK_CLI_TOKEN="xoxb-..."
export SLACK_CLI_COOKIE="d=..." # Optional cookie header
```

### 2. Verify Auth Status
Check which credentials are currently active:
```bash
./slack-cli auth status
./slack-cli auth test
```

---

## Usage Examples

### Reading & Navigation
```bash
# List all channels & DMs in human output
./slack-cli conversations list --limit 10

# List users in the workspace
./slack-cli users list --limit 20

# Resolve display name or email to a Slack ID
./slack-cli resolve user @alice
./slack-cli resolve conversation #general

# Get unread message counts for a channel
./slack-cli conversations unread #general
```

### Messaging
```bash
# Send a message to a public channel
./slack-cli chat send #general "Deployment successful! 🚀"

# Open a DM and send a direct message
./slack-cli chat dm @alice "Hey Alice, let's catch up!"

# Open a group DM (MPDM) and send a message
./slack-cli chat dm @alice @bob "Meeting starts in 5 minutes."

# Reply in a threaded conversation
./slack-cli chat send #general "Thread reply" --thread-ts 1746592769.123456

# Delete a message
./slack-cli chat delete #general 1746592769.123456
```

### Searching
```bash
# Search for messages containing a query
./slack-cli search messages "production outage" --sort timestamp
```

### Raw API Calls
If a command isn't built into the CLI yet, you can call any Slack Web API method directly:
```bash
./slack-cli api call conversations.info --http GET --param channel=C0123456789
```

---

## Scripting and Automation

`slack-cli` is optimized for script integrations. You can toggle output modes to fit your automation pipelines:

### JSON Output
Use `--json` to retrieve structured JSON payloads, perfect for scripting with `jq`:
```bash
# Fetch the email of a user
./slack-cli --json users info @alice | jq -r '.user.profile.email'
```

### Plain Output
Use `--plain` to output simple, tab-delimited values (useful for traditional bash/awk workflows):
```bash
# Extract the channel IDs of all public channels
./slack-cli --plain conversations list | awk '{print $1}'
```

---

## Configuration File

Your configurations, profiles, and credentials are saved locally. By default, the configuration file is stored in:

* **macOS**: `~/Library/Application Support/slack-cli/config.json`
* **Linux**: `~/.config/slack-cli/config.json`
* **Windows**: `%APPDATA%\slack-cli\config.json`

You can manually edit the config file or override it with the `--config` flag:
```json
{
  "profiles": {
    "default": {
      "token": "xoxb-default-token",
      "cookie": "d=default-cookie-value"
    },
    "work": {
      "token": "xoxc-work-token",
      "cookie": "d=work-cookie-value"
    }
  }
}
```

Switch between profiles easily:
```bash
./slack-cli --profile work conversations list
```

---

## Contributing and Verification

Ensure code changes are properly formatted and tested before contributing:

```bash
# Format codebase
gofmt -w ./...

# Run test suite
go test ./...

# Verify coverage is above 90%
./scripts/check-coverage.sh
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
