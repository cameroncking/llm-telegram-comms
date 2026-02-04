# llm-telegram-comms

A Telegram bot that passes user messages to a backend command via stdin and returns the command's output as a reply.

> Note: This is not designed for high-performance, high-security, or enterprise production use. It's intended for power users who want a conveniant way to map telegram chats to custom backends.

While this tool was designed to be flexible and work with different backends, it was built with [llm-sandbox](https://github.com/cameroncking/llm-sandbox) in mind. Since llm-sandbox itself uses Docker, it is non-trivial to providing a working Dockerfile and docker-compose.yaml that will work in all environments.  The specifics of how to do this get into implementation details that the author of this project could not know in advance.

## Installation

```bash
go build -o llm-telegram-comms .
```

## Usage

```bash
# Run in foreground
./llm-telegram-comms -c config.json

# Run as daemon
./llm-telegram-comms -d -c config.json

# Run as daemon with PID file
./llm-telegram-comms -d -p /var/run/bot.pid -c config.json

# Run as daemon with PID file and log file
./llm-telegram-comms -d -p /var/run/bot.pid -l /var/log/bot.log -c config.json

# Restart an existing daemon
./llm-telegram-comms -d -r -p /var/run/bot.pid -c config.json
```

## Command Line Options

| Option | Description |
|--------|-------------|
| `-c, --config FILE` | Path to config file (required) |
| `-d, --daemon` | Fork the process in the background |
| `-p, --pid FILE` | Write PID to file (requires `-d`) |
| `-l, --log FILE` | Write logs to file instead of stdout |
| `-r, --restart` | Kill existing process from PID file before starting (requires `-p`) |

## Daemon Mode Examples

### Start a new daemon

```bash
$ ./llm-telegram-comms -d -p /tmp/bot.pid -c config.json
Started daemon with PID 12345
```

### Attempt to start when already running (fails)

```bash
$ ./llm-telegram-comms -d -p /tmp/bot.pid -c config.json
Error: process already running with PID 12345
$ echo $?
1
```

### Restart an existing daemon

```bash
$ ./llm-telegram-comms -d -r -p /tmp/bot.pid -c config.json
Killed existing process 12345
Started daemon with PID 12346
```

### PID file points to a different process

If the PID file exists but references a process with a different name (e.g., leftover from a previous program), the bot will start normally and overwrite the PID file.

## Configuration

Create a `config.json` file (see `config.example.json`):

```json
{
  "telegram_token": "YOUR_BOT_TOKEN_FROM_BOTFATHER",
  "backend_command": "./my-script.sh",
  "working_directory": "/path/to/workdir",
  "user_allowlist_required": false,
  "user_allowlist": [123456789],
  "group_allowlist_required": false,
  "group_allowlist": [-100123456789],
  "environment": {
    "API_KEY": "secret"
  },
  "drop_environment": false,
  "enable_attachments": true,
  "attachment_path": "/path/to/attachments",
  "attachment_method": "xml",
  "attachment_path_chat_prefix": "/path/to/attachments/"
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `telegram_token` | string | **required** | Bot token from BotFather |
| `backend_command` | string | **required** | Command to execute for each message |
| `working_directory` | string | `""` | Working directory for the command (empty = inherit from process) |
| `user_allowlist_required` | bool | `false` | When true, only users in `user_allowlist` can use the bot |
| `user_allowlist` | int64[] | `[]` | List of allowed user IDs |
| `group_allowlist_required` | bool | `false` | When true, only groups in `group_allowlist` can use the bot |
| `group_allowlist` | int64[] | `[]` | List of allowed group/supergroup IDs |
| `environment` | map | `{}` | Environment variables to set for the backend command |
| `drop_environment` | bool | `false` | When true, only variables in `environment` are passed (no inherited env) |
| `enable_attachments` | bool | `false` | When true, save received attachments to disk |
| `attachment_path` | string | `""` | Directory to save attachments (empty = current working directory) |
| `attachment_method` | string | `"xml"` | How to pass attachments to backend: `xml`, `plaintext`, or `datasette_llm` |
| `attachment_path_chat_prefix` | string | `""` | Path prefix prepended to filenames in all attachment methods |
| `attachment_path_chatid_suffix` | bool | `false` | When true, append chat ID to attachment_path (e.g., `/userdata/123456`) |
| `aggressive_shell_escape` | bool | `true` | When true, only allow safe characters in filenames passed to shell |
| `telegram_chat_type_env` | string | `""` | Environment variable name to pass chat type (`user` or `group`) to backend |
| `telegram_chat_id_env` | string | `""` | Environment variable name to pass chat ID to backend |

## How It Works

1. User sends a message to the bot (DM or in a group)
2. Bot checks if the user/group is allowed (if allowlists are enabled)
3. If attachments are enabled, any images/audio/video are saved to the attachment path
4. The message text (or caption for media messages) is passed as stdin to the backend command
5. The command's stdout is sent back as a reply to the user
6. In group chats, the bot replies to the original message; in private chats, it sends a regular message

## Attachments

When `enable_attachments` is true, the bot saves received files with a timestamp prefix:

```
IMG_9182.png → 2026-02-04T14-30-25Z_IMG_9182.png
```

Supported attachment types:
- Photos
- Documents
- Audio
- Video
- Voice messages
- Video notes
- Stickers

### Attachment Methods

The `attachment_method` config option controls how attachment information is passed to the backend command:

#### xml (default)

Prepends XML tags to the message stdin:

```
<attachment>2026-02-04T14-30-25Z_IMG_9812.png</attachment>
<attachment>2026-02-04T14-30-25Z_VID_9813.mov</attachment>
Process and summarize the contents of this media.
```

#### plaintext

Prepends plaintext attachment lines to the message stdin:

```
Attachment: 2026-02-04T14-30-25Z_IMG_9812.png
Attachment: 2026-02-04T14-30-25Z_VID_9813.mov
Process and summarize the contents of this media.
```

#### datasette_llm

Appends `-a FILENAME` arguments to the backend command for each attachment. This is designed for use with [datasette-llm](https://github.com/simonw/llm).

With config:
```json
{
  "backend_command": "llm -m gpt-4",
  "enable_attachments": true,
  "attachment_method": "datasette_llm",
  "attachment_path_chat_prefix": "/data/attachments/"
}
```

The resulting command becomes:
```bash
llm -m gpt-4 -a '/data/attachments/2026-02-04T14-30-25Z_IMG_9812.png' -a '/data/attachments/2026-02-04T14-30-25Z_VID_9813.mov'
```

### Path Prefix

The `attachment_path_chat_prefix` option (default: empty) specifies a path prefix to prepend to filenames in all attachment methods. This is useful when the attachment directory differs from the working directory or when the backend expects a specific path format.

### Per-Chat Attachment Directories

Set `attachment_path_chatid_suffix` to `true` to automatically create per-chat subdirectories:

```json
{
  "attachment_path": "/data/attachments",
  "attachment_path_chatid_suffix": true
}
```

With chat ID `123456789`, files will be saved to `/data/attachments/123456789/`. The directory is created automatically if it doesn't exist.

### Shell Escaping

When using the `datasette_llm` attachment method, filenames are passed as shell arguments. The `aggressive_shell_escape` option (default: `true`) controls how filenames are sanitized:

**Aggressive mode (default):** Only allows safe characters: `a-z A-Z 0-9 . _ - /`. All other characters are replaced with `_`.

```
"my file.png"        → "my_file.png"
"$(whoami).png"      → "__whoami_.png"
"file's name.png"    → "file_s_name.png"
```

**Less-aggressive mode:** Preserves all characters using single-quote escaping. This maintains the original filename.

```json
{
  "aggressive_shell_escape": false
}
```

> **Note:** Setting `aggressive_shell_escape` to `false` is rarely needed. There may be some unique edge cases where files need to preserve their unusual names, and this decision is up to the sysadmin. However, keeping aggressive shell escape enabled (the default) will be the right choice for almost every case.

### Chat Context Environment Variables

You can pass chat context to the backend command via environment variables:

```json
{
  "telegram_chat_type_env": "CHAT_TYPE",
  "telegram_chat_id_env": "CHAT_ID"
}
```

When configured, the backend command receives:
- `CHAT_TYPE`: Either `user` (for private/DM chats) or `group` (for group/supergroup chats)
- `CHAT_ID`: The numeric Telegram chat ID

This is useful for:
- Per-user or per-group conversation isolation
- Logging and analytics
- Custom routing logic in your backend

Example backend script using these variables:
```bash
#!/bin/bash
# Route to different databases based on chat
if [ "$CHAT_TYPE" = "user" ]; then
  llm -d "/data/users/$CHAT_ID.db" "$@"
else
  llm -d "/data/groups/$CHAT_ID.db" "$@"
fi
```

## Integration with llm-sandbox

[llm-sandbox](https://github.com/cameroncking/llm-sandbox) is a Docker-based sandbox for running LLM commands securely. It pairs well with llm-telegram-comms for a complete Telegram-to-LLM pipeline.

### Sample Configuration

```json
{
  "telegram_token": "YOUR_BOT_TOKEN",
  "backend_command": "./llm-sandbox -m openrouter/openrouter/auto -d /userdata/log.db -c",
  "working_directory": "/path/to/llm-sandbox",
  "user_allowlist_required": true,
  "user_allowlist": [YOUR_TELEGRAM_USER_ID],
  "group_allowlist_required": true,
  "group_allowlist": [],
  "environment": {
    "SANDBOX_USERDATA": "/path/to/userdata",
    "SANDBOX_SYSDATA": "/path/to/sysdata"
  },
  "drop_environment": true,
  "enable_attachments": true,
  "attachment_path": "/path/to/userdata/attachments",
  "attachment_method": "datasette_llm",
  "attachment_path_chat_prefix": "/userdata/attachments/",
  "telegram_chat_id_env": "SANDBOX_USERDATA_SUFFIX"
}
```

### How it works

1. **backend_command**: Runs `llm-sandbox` with:
   - `-m openrouter/openrouter/auto`: Uses OpenRouter's auto model selection
   - `-d /userdata/log.db`: Stores conversation history in a SQLite database
   - `-c`: Continues the conversation (maintains context across messages)

2. **environment**: Sets the sandbox data directories:
   - `SANDBOX_USERDATA`: Writable directory for logs, attachments, and outputs
   - `SANDBOX_SYSDATA`: Read-only directory for system prompts and configuration

3. **telegram_chat_id_env**: Sets `SANDBOX_USERDATA_SUFFIX` to the chat ID, which llm-sandbox uses to create per-user/per-group subdirectories for conversation isolation

4. **attachments**: When using `datasette_llm` method:
   - Files are saved to the host at `attachment_path`
   - The `attachment_path_chat_prefix` maps to the container's view (`/userdata/attachments/`)
   - llm-sandbox receives `-a /userdata/attachments/timestamp_filename.jpg` arguments

### Directory structure

```
/path/to/llm-sandbox/          # working_directory
├── llm-sandbox                 # The llm-sandbox script
├── .env                        # API keys (OPENROUTER_KEY, etc.)
└── ...

/path/to/userdata/              # SANDBOX_USERDATA
├── log.db                      # Conversation history
└── attachments/                # attachment_path
    ├── 2026-02-04T14-30-25Z_photo.jpg
    └── ...

/path/to/sysdata/               # SANDBOX_SYSDATA (optional)
├── prompt.txt                  # System prompts
└── ...
```

### Multi-user setup

With `telegram_chat_id_env` set to `SANDBOX_USERDATA_SUFFIX`, llm-sandbox automatically creates per-user subdirectories:

```
/path/to/userdata/
├── 123456789/              # User A's data
│   ├── log.db
│   └── attachments/
├── 987654321/              # User B's data
│   ├── log.db
│   └── attachments/
└── -100555555555/          # Group chat data
    ├── log.db
    └── attachments/
```

Each user or group gets isolated conversation history and attachments automatically.

## Getting User/Group IDs

To get your user ID or group ID, temporarily disable the allowlists and check the logs when you send a message. The bot logs user ID and chat ID for each message.

## Testing

```bash
go test ./...
```

## License

ISC License

Copyright (c) Cameron King

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.
