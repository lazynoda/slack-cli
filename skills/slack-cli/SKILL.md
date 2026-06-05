---
name: slack-cli
description: Use the slack-cli command-line tool to interact with Slack workspaces. Trigger this skill whenever the user wants to list, send, update, delete, search, or read Slack messages, channels, threads, DMs, group DMs (MPDMs), or user presence information, or if they need to manage Slack CLI configurations, authenticate, or call any Slack Web API endpoint via the slack-cli tool.
---

# Slack CLI Skill

Use this skill to control and interact with Slack workspaces using the custom Go `slack-cli` tool.

---

## Authentication & Setup

Before invoking any commands that make API calls, verify the configuration state.

### Check Stored Auth
```bash
./slack-cli auth status
```

### Run Auth Test
```bash
./slack-cli auth test
```

### Store Token / Cookie manually
If you have a token or cookie, save it to the active profile (defaults to the `default` profile):
```bash
# Set Bearer token (xoxb- or xoxp-)
./slack-cli auth set-token xoxb-your-token-value

# Set session cookie (d=xoxd-...)
./slack-cli auth set-cookie "d=your-cookie-value"
```

### Import from Browser
You can extract session cookies and client tokens (`xoxc-...`) directly from local browser databases:
```bash
# Import from Chrome's default profile
./slack-cli auth import-cookie --browser chrome

# Import from Chrome's specific profile
./slack-cli auth import-cookie --browser chrome --browser-profile "Profile 6"

# Import from Brave, Arc, Edge, Firefox, Safari, etc.
./slack-cli auth import-cookie --browser brave
./slack-cli auth import-cookie --browser arc
```

---

## Core Command Reference

### Global Flags
These flags apply to all commands and must precede the subcommand:
* `--profile <name>`: Switch profiles (default: `default`).
* `--json`: Format the output as JSON for scripting.
* `--plain`: Output space/tab-delimited lines.
* `--timeout <duration>`: Request timeout (default: `10s`).

---

### 1. Resolving Names to Slack IDs
You can refer to users and channels by friendly names (e.g., `@alice`, `#general`), but if you need to resolve them explicitly:

```bash
# Resolve user details
./slack-cli resolve user @alice

# Resolve channel details
./slack-cli resolve conversation #general
```

---

### 2. User Information
```bash
# List workspace users (limit response for speed)
./slack-cli users list --limit 10

# List ALL workspace users (paginates automatically)
./slack-cli users list --all

# Show user info
./slack-cli users info @alice

# Check if a user is active/online
./slack-cli users presence @alice
```

---

### 3. Conversation & Channel Management
```bash
# List public/private channels and DMs
./slack-cli conversations list --limit 10

# Create a public or private channel
./slack-cli conversations create my-new-channel
./slack-cli conversations create private-channel --private

# Archive / Unarchive a channel
./slack-cli conversations archive #my-new-channel
./slack-cli conversations unarchive #my-new-channel

# Join / Leave a public channel
./slack-cli conversations join #general
./slack-cli conversations leave #general

# Set topic / purpose
./slack-cli conversations topic #general "Discussing general topics"
./slack-cli conversations purpose #general "General communication channel"
```

---

### 4. Reading Messages
```bash
# Read conversation history (newest first)
./slack-cli conversations history #general --limit 20

# Read conversation history within a specific time range
./slack-cli conversations history #general --oldest "1717580000" --latest "1717590000"

# Fetch replies in a threaded conversation
./slack-cli conversations replies #general 1746592769.123456

# Read unread messages in a channel
# (Only works if last_read cursor is returned by Slack for the active credentials)
./slack-cli conversations unread #general

# Get unread message counts across all channels & DMs in a single request
# (Requires user session cookie/token authentication)
./slack-cli api call users.counts
```

---

### 5. Messaging & Chat
```bash
# Send a message to a channel
./slack-cli chat send #general "Hello world!"

# Send a threaded reply
./slack-cli chat send #general "Thread reply content" --thread-ts 1746592769.123456

# Send a DM (Direct Message)
./slack-cli chat dm @alice "Hi Alice!"

# Send a group DM (MPDM)
./slack-cli chat dm @alice @bob "Hey team!"

# Send an ephemeral message (only visible to targeted user)
./slack-cli chat ephemeral #general @alice "Private alert for Alice"

# Update an existing message
./slack-cli chat update #general 1746592769.123456 "Updated message text"

# Delete a message
./slack-cli chat delete #general 1746592769.123456

# Get a permalink for a message
./slack-cli chat permalink #general 1746592769.123456
```

---

### 6. Workspace Search
Search messages using Slack's workspace search syntax:
```bash
./slack-cli search messages "outage" --count 10
./slack-cli search messages "from:@alice database"
```

---

### 7. Generic API Call (Escape Hatch)
Call any Slack Web API method directly:
```bash
./slack-cli api call users.profile.get --http GET --param user=U12345
```

---

## Programming & Scripting Recipes

To parse output programmatically, use `--json` combined with `jq`:

### Get User Email by Handle
```bash
./slack-cli --json users info @alice | jq -r '.user.profile.email'
```

### Get ID of a Specific Channel
```bash
./slack-cli --json conversations list | jq -r '.channels[] | select(.name=="general") | .id'
```

### Fetch All Channels/DMs with Unread Messages (Single Command)
```bash
# Get channels and private groups with active unread counts
./slack-cli --json api call users.counts | jq '[.channels[], .groups[]] | .[] | select(.unread_count > 0) | {name, unread_count}'
```

---

## Troubleshooting

* **"no Slack token configured..."**:
  You must set a token or import cookies/tokens from a browser. Execute `auth set-token` or `auth import-cookie`.
* **Database Lock during browser imports**:
  `slack-cli` automatically copies browser databases to a temp directory before reading them to avoid locking conflicts if Chrome is open. If you still encounter locks, close the target browser and retry.
* **Rate Limits (HTTP 429)**:
  `slack-cli` has built-in retry-after backoff. If you are doing bulk lookups, request only the necessary pages and use pagination cursors (`--cursor`) instead of fetching all pages.
