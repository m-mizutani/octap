# octap
[![Test](https://github.com/m-mizutani/octap/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/test.yml) [![Lint](https://github.com/m-mizutani/octap/actions/workflows/lint.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/lint.yml) [![Security](https://github.com/m-mizutani/octap/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/gosec.yml) [![Trivy](https://github.com/m-mizutani/octap/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/trivy.yml) [![CodeQL](https://github.com/m-mizutani/octap/actions/workflows/codeql.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/codeql.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/m-mizutani/octap)](https://goreportcard.com/report/github.com/m-mizutani/octap)

CLI GitHub Actions notifier - Monitor and notify when GitHub Actions workflows complete.

![octap example](docs/images/example.png)

## Features

- 🔄 **Real-time monitoring** of GitHub Actions workflows
- 🎯 **Commit-specific tracking** - Monitor workflows for specific commits
- 🔔 **Sound notifications** - Different sounds for individual and completion events
- 📊 **Live CUI display** - See workflow status in real-time
- ⏱️ **Configurable polling** - Adjust check intervals
- 🔐 **Secure authentication** - GitHub OAuth Device Flow (no token management needed)
- ⚙️ **Customizable hooks** - Configure custom actions via YAML config file
- 🎵 **Custom sounds** - Use your own sound files for different event types
- 💬 **Slack notifications** - Send workflow status to Slack channels
- 🔧 **Command execution** - Run custom scripts on workflow events
- 🚀 **Smart initial check** - Handles already-completed workflows gracefully

## What's New

### Latest Updates
- **Smart Completion Events**: When all workflows are already completed on initial check, only `complete_success` or `complete_failure` sounds play
- **Parallel Hook Execution**: Multiple hooks execute concurrently with proper synchronization
- **Enhanced Debug Logging**: Detailed logging for configuration loading and hook execution (use `--debug` flag)
- **Distinct Completion Sounds**: Different sounds for completion events vs individual workflow events

## Installation

### Using Go

```bash
go install github.com/m-mizutani/octap@latest
```

### From source

```bash
git clone https://github.com/m-mizutani/octap.git
cd octap
go build -o octap .
```

## Usage

### Basic usage

Monitor GitHub Actions for the current commit in the current directory:

```bash
octap
```

This will:
- Automatically detect the current Git repository and commit SHA
- Display real-time workflow status updates
- Play sound notifications when workflows complete
- Show detailed URLs for failed workflows

### Example Output

```
📋 Workflow Status:
──────────────────────────────────────────────────
✅ build                [success]
❌ test                 [failure] 🔗 https://github.com/user/repo/actions/runs/123456789
🔄 lint                 [in_progress]
⏳ deploy               [queued]
──────────────────────────────────────────────────
🔄 2/4 completed [15:47:30]

⏱️  Next check in: 5s
```

### Monitor specific commit

```bash
octap -c abc123def
```

### Adjust polling interval

```bash
# Default is 5 seconds
octap

# Check every 30 seconds
octap -i 30s

# Check every 2 minutes
octap -i 2m
```

### Disable sound notifications

```bash
octap --silent
```

### Verbose logging

```bash
# Show more detailed information
octap --verbose

# Show debug information including API calls
octap --debug
```

### Typical Workflow

1. **Push commits to GitHub**:
   ```bash
   git push origin feature-branch
   ```

2. **Monitor the workflows**:
   ```bash
   octap
   ```

3. **octap will**:
   - Authenticate with GitHub (first time only)
   - Monitor all workflows for your current commit
   - Show real-time updates as workflows progress
   - Play sounds when workflows complete (success/failure)
   - Display URLs for failed workflows so you can quickly investigate
   - Exit automatically when all workflows complete

## Authentication

octap uses GitHub OAuth Device Flow for authentication. On first run:

1. You'll receive a code to copy
2. Visit the provided GitHub URL  
3. Paste the code and authorize the app
4. octap will automatically complete the authentication

The token is stored locally at `~/.config/octap/token.json`.

### First-time Authentication Example

```
🔐 GitHub Device Flow Authentication
────────────────────────────────────
1. Copy this code: ABCD-1234
2. Visit: https://github.com/login/device
3. Paste the code and authorize the app

⏳ Waiting for authorization...
✅ Authentication successful!
```

### Using Your Own GitHub OAuth App

By default, octap uses a built-in OAuth Client ID for convenience. For production use or if you encounter rate limiting, you can create and use your own GitHub OAuth App:

1. **Create a GitHub OAuth App**:
   - Go to GitHub Settings → Developer settings → OAuth Apps
   - Click "New OAuth App"
   - Fill in the details:
     - Application name: Your app name (e.g., "My octap")
     - Homepage URL: Any valid URL (e.g., "https://github.com/yourusername/octap")
     - Authorization callback URL: `http://localhost` (not used but required)
   - Click "Register application"

2. **Use your Client ID**:
   ```bash
   # Set via environment variable
   export OCTAP_GITHUB_OAUTH_CLIENT_ID=your_client_id_here
   octap
   
   # Or pass directly via flag
   octap --github-oauth-client-id=your_client_id_here
   ```

**Note**: The Client Secret is not needed for Device Flow authentication.

## Configuration

### Command-line flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-c, --commit` | Specify commit SHA to monitor | Current HEAD | `octap -c abc123def` |
| `-i, --interval` | Polling interval | 5s | `octap -i 30s` |
| `--config` | Path to configuration file | `~/.config/octap/config.yml` | `octap --config ./my-config.yml` |
| `--silent` | Disable sound notifications | false | `octap --silent` |
| `--verbose` | Enable verbose logging | false | `octap --verbose` |
| `--debug` | Enable debug logging | false | `octap --debug` |
| `--github-oauth-client-id` | GitHub OAuth App Client ID | Built-in ID | `octap --github-oauth-client-id=Ov23...` |

**Environment Variables**:
- `OCTAP_GITHUB_OAUTH_CLIENT_ID`: Sets the GitHub OAuth App Client ID

### Configuration File

octap supports a YAML configuration file for customizing sound notifications. By default, it looks for `~/.config/octap/config.yml`.

The configuration system provides:
- **Four distinct event types** for granular control
- **OS-specific default sounds** that work out of the box
- **Custom sound file support** for personalization
- **Parallel action execution** for multiple hooks

#### Generate Configuration Template

```bash
# Generate default config file
octap config init

# Generate config at specific location
octap config init --output ./my-config.yml

# Force overwrite existing config
octap config init --force
```

#### Configuration Example

The generated template includes OS-specific default sound files:

**macOS:**
```yaml
hooks:
  check_success:
    - type: sound
      path: /System/Library/Sounds/Glass.aiff
  
  check_failure:
    - type: sound
      path: /System/Library/Sounds/Basso.aiff
  
  complete_success:
    - type: sound
      path: /System/Library/Sounds/Ping.aiff
  
  complete_failure:
    - type: sound
      path: /System/Library/Sounds/Funk.aiff
```

**Linux:**
```yaml
hooks:
  check_success:
    - type: sound
      path: /usr/share/sounds/freedesktop/stereo/complete.oga
  
  check_failure:
    - type: sound
      path: /usr/share/sounds/freedesktop/stereo/dialog-error.oga
  
  complete_success:
    - type: sound
      path: /usr/share/sounds/freedesktop/stereo/bell.oga
  
  complete_failure:
    - type: sound
      path: /usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga
```

**Windows:**
```yaml
hooks:
  check_success:
    - type: sound
      path: C:\Windows\Media\chimes.wav
  
  check_failure:
    - type: sound
      path: C:\Windows\Media\chord.wav
  
  complete_success:
    - type: sound
      path: C:\Windows\Media\ding.wav
  
  complete_failure:
    - type: sound
      path: C:\Windows\Media\Windows Critical Stop.wav
```

#### Hook Events

| Event | Description | When Triggered |
|-------|-------------|----------------|
| `check_success` | Individual workflow success | When a workflow completes successfully during monitoring |
| `check_failure` | Individual workflow failure | When a workflow fails during monitoring |
| `complete_success` | All workflows successful | When all workflows complete successfully (including initial check) |
| `complete_failure` | One or more workflows failed | When monitoring ends with failures (including initial check) |

**Note**: When all workflows are already completed on the initial check, only `complete_success` or `complete_failure` events are triggered, not individual `check_*` events.

#### Action Types

##### `sound` Action
Plays a sound file when the event occurs.

**Configuration**:
- `path`: Path to sound file (supports `~` for home directory)

##### `slack` Action
Sends a notification to Slack via Incoming Webhook.

**Configuration**:
- `webhook_url`: Slack Incoming Webhook URL (supports environment variables like `${SLACK_WEBHOOK_URL}`)
- `message`: Message template with support for template variables
- `color` (optional): Message color (`good`, `warning`, `danger`, or hex color code)
- `username` (optional): Override webhook's default username (only works if webhook allows customization)
- `icon_emoji` (optional): Override webhook's default icon (only works if webhook allows customization)

**Example**:
```yaml
hooks:
  check_failure:
    - type: slack
      webhook_url: ${SLACK_WEBHOOK_URL}
      message: "❌ Workflow {{.Workflow}} failed in {{.Repository}}"
      color: danger
  
  complete_success:
    - type: slack
      webhook_url: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX
      message: "✅ All workflows completed successfully for {{.Repository}}"
      color: good
```

##### `command` Action
Executes an arbitrary command with workflow information available as environment variables.

**Configuration**:
- `command`: Command to execute (supports `~` for home directory)
- `args` (optional): Array of command arguments (supports environment variable expansion)
- `timeout` (optional): Command execution timeout (default: 30s)
- `env` (optional): Additional environment variables to set

**Example**:
```yaml
hooks:
  check_failure:
    - type: command
      command: /usr/local/bin/notify
      args:
        - --title
        - "Build Failed"
        - --message
        - "$OCTAP_WORKFLOW failed"
      timeout: 10s
  
  complete_success:
    - type: command
      command: ~/scripts/deploy.sh
      args:
        - production
        - "$OCTAP_RUN_ID"
      env:
        - DEPLOY_ENV=production
```

#### Template Variables (Slack)

The following variables are available in Slack message templates:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Repository}}` | Repository name (owner/repo) | `m-mizutani/octap` |
| `{{.Workflow}}` | Workflow name | `CI Build` |
| `{{.RunID}}` | GitHub Actions run ID | `123456789` |
| `{{.EventType}}` | Hook event type | `check_success` |
| `{{.RunURL}}` | Direct link to the workflow run | `https://github.com/...` |
| `{{.Timestamp}}` | Current timestamp | `2024-01-01 12:00:00` |

#### Environment Variables (Command)

The following environment variables are set when executing commands:

| Variable | Description | Example |
|----------|-------------|---------|
| `OCTAP_EVENT_TYPE` | Hook event type | `check_failure` |
| `OCTAP_REPOSITORY` | Repository name | `m-mizutani/octap` |
| `OCTAP_WORKFLOW` | Workflow name | `CI Build` |
| `OCTAP_RUN_ID` | GitHub Actions run ID | `123456789` |
| `OCTAP_RUN_URL` | Direct link to the workflow run | `https://github.com/...` |

**Supported Sound Formats by Platform**:
| Platform | Supported Formats | Notes |
|----------|-------------------|-------|
| **macOS** | .aiff, .mp3, .wav, .m4a | Uses `afplay` command |
| **Linux** | .oga, .wav, .mp3, .ogg | Uses `paplay` (PulseAudio) or `aplay` (ALSA) as fallback |
| **Windows** | .wav | Uses PowerShell's `Media.SoundPlayer` |

### Sound Notifications

When no configuration file is provided, octap plays different system sounds based on workflow results:

- **Success**: Glass sound (macOS), complete sound (Linux), chimes (Windows)
- **Failure**: Basso sound (macOS), error sound (Linux), chord (Windows)
- **Final Summary**: Plays appropriate sound based on overall result

All platforms support sound notifications with their native audio systems.

### Workflow Status Icons

| Icon | Status | Description |
|------|--------|-------------|
| ⏳ | queued | Workflow is waiting to start |
| 🔄 | in_progress | Workflow is currently running |
| ✅ | success | Workflow completed successfully |
| ❌ | failure | Workflow failed (includes URL for investigation) |
| ⚪ | cancelled | Workflow was cancelled |
| ⏭️ | skipped | Workflow was skipped |

## Requirements

- **Git repository**: Must be run inside a Git repository with GitHub remote
- **Pushed commits**: The commit you want to monitor must be pushed to GitHub
- **Internet connection**: Required for GitHub API access

## Troubleshooting

### Common Issues

**"Current commit has not been pushed to GitHub"**
```bash
git push origin your-branch
```

**"failed to get repository info"**
- Ensure you're in a Git repository with a GitHub remote
- Check that the remote URL is accessible

**"No saved token found, starting authentication"**
- This is normal on first run, follow the authentication flow


### Supported Platforms

- **macOS**: Full support with system sounds using `afplay`
- **Linux**: Full support with system sounds (requires `paplay` or `aplay`)
- **Windows**: Full support with system sounds using PowerShell

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Masayuki Mizutani ([@m-mizutani](https://github.com/m-mizutani))