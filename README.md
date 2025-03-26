# Claude Squad

> A terminal-based session manager for Claude Code and other CLI AI assistants

Claude Squad is a TUI (terminal user interface) application that helps you manage multiple Claude Code sessions in separate workspaces, allowing you to work on different tasks simultaneously without conflicts.

![Claude Squad Screenshot](assets/screenshot.png)

### Features

- Manage multiple Claude Code sessions:
  - Create and isolate sessions using git worktrees
  - Preview session content in real-time
  - Pause/resume sessions with automatic commit of changes
- Navigate easily between sessions
- Monitor session status (Running, Ready, Loading, Paused)
- Support for various CLI AI tools (Claude Code, Aider, etc.)

### Installation

The easiest way to install `claude-squad` is by running the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/stmg-ai/claude-squad/main/install.sh | bash
```

This will install the `claude-squad` binary to `~/.local/bin` and add it to your PATH.

Alternatively, you can install `claude-squad` by building from source or installing a pre-built binary (see project repository for details).

### Prerequisites

- [tmux](https://github.com/tmux/tmux/wiki/Installing)
- [git](https://git-scm.com/downloads)

<br />

### Usage

```
Usage:
  claude-squad [flags]
  claude-squad [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  debug       Print debug information like config paths
  help        Help about any command

Flags:
  -y, --autoyes          [experimental] If enabled, all instances will automatically accept prompts
  -h, --help             help for claude-squad
  -p, --program string   Program to run in new instances (e.g. 'aider --model ollama_chat/gemma3:1b')
      --reset            Reset all stored instances
```

Run the application with:

```bash
claude-squad
```

To use a specific AI assistant program:

```bash
claude-squad -p "aider --model ollama_chat/gemma3:1b"
```

<br />

### Menu Options

The menu at the bottom of the screen shows available commands:

#### Instance Management
- `n` - Create a new session
- `d` - Kill (delete) the selected session

#### Actions
- `↑/j`, `↓/k` - Navigate between sessions
- `⏎/o` - Attach to the selected session
- `s` - Submit/commit changes to git
- `p` - Pause session (preserves branch, removes worktree)
- `r` - Resume paused session

#### System
- `tab` - Switch preview tab
- `q` - Quit the application

<br />

### Session States

- **Running** - Claude is actively working
- **Ready** - Claude is waiting for input
- **Loading** - Session is starting up
- **Paused** - Session is paused (worktree removed, branch preserved)

## How It Works

Claude Squad uses:
1. **tmux** to create isolated terminal sessions for each Claude instance
2. **git worktrees** to isolate codebases so each session works on its own branch
3. A simple TUI interface for easy navigation and management

When you create a new session:
1. A new git branch is created for your session
2. A git worktree is created from that branch
3. A tmux session is launched with your chosen AI assistant tool (Claude Code by default)

When you pause a session:
1. Changes are committed to the branch
2. The tmux session is closed
3. The worktree is removed (but the branch is preserved)
4. Branch name is copied to clipboard for reference

When you resume a session:
1. The worktree is recreated from the preserved branch
2. A new tmux session is launched with your AI assistant
3. You can continue from where you left off

## License

[AGPL-3.0](LICENSE.md)
