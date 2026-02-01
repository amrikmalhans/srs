# SRS

A CLI spaced repetition system (SRS) for managing flashcards. Cards are stored as markdown files with scheduling state tracked in SQLite.

## Installation

### Using Go (Recommended)

Install directly from GitHub:

```bash
go install github.com/amrikmalhans/srs@latest
```

Make sure `~/go/bin` is in your PATH. Add this to your `~/.zshrc` (or `~/.bashrc`):

```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell:
```bash
source ~/.zshrc  # or source ~/.bashrc
```

### Building from Source

If you prefer to build from source:

```bash
git clone https://github.com/amrikmalhans/srs.git
cd srs
go build -o srs .
```

## Quick Start

1. **Sync your cards** (if you have existing card files):
   ```bash
   srs sync
   ```

2. **Start reviewing**:
   ```bash
   srs review
   ```

3. **Or do a quick 2-minute sprint**:
   ```bash
   srs sprint
   ```

## Commands

- `srs add` - Create a new card
- `srs review` - Start a review session
- `srs sprint` - Quick 2-minute review session
- `srs list` - List all cards and statistics
- `srs find <query>` - Search cards by text
- `srs stats` - Show review statistics
- `srs sync` - Sync database from card files
- `srs reset` - Reset all review statistics (use `--force` to skip confirmation)

## Configuration

### Cards Directory

By default, cards are stored in a `cards/` directory relative to where you run the command. You can specify a custom cards directory using:

**Option 1: Command-line flag**
```bash
srs add --cards-dir ~/Desktop/cards
srs review --cards-dir ~/Desktop/cards
```

**Option 2: Environment variable**
```bash
export SRS_CARDS_DIR=~/Desktop/cards
srs add  # Uses Desktop/cards automatically
srs review  # Uses Desktop/cards automatically
```

The flag takes precedence over the environment variable, and both take precedence over the default `./cards` directory.

## Card Format

Cards are markdown files in the `cards/` directory with YAML frontmatter:

```markdown
---
id: 550e8400-e29b-41d4-a716-446655440000
created: 2024-01-15T10:30:00Z
tags: [french, vocabulary]
---

# Q

What does "bonjour" mean?

# A

Hello / Good day
```

## Review Flow

1. Press space or enter to reveal the answer
2. Grade your performance:
   - `1` = Again (didn't remember)
   - `2` = Hard (remembered with difficulty)
   - `3` = Good (remembered correctly)
   - `4` = Easy (remembered easily)

The system uses the SM-2 algorithm to schedule cards based on your performance.
