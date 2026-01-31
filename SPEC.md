# SRS Tool Specification

## Overview

This specification defines the architecture and design decisions for the CLI SRS (Spaced Repetition System) tool, establishing guardrails to prevent future yak-shaving.

## 1. Card File Format

**Location**: `cards/*.md` (one file per card)

**Structure**: Markdown with YAML frontmatter containing:
- `id`: UUID v4 (unique identifier, never changes)
- `created`: ISO 8601 timestamp
- `tags`: Optional array of strings

**Content**: Markdown body (question/answer, cloze, or other formats)

**Example**:
```markdown
---
id: 550e8400-e29b-41d4-a716-446655440000
created: 2024-01-15T10:30:00Z
tags: [go, programming, basics]
---

# What is a goroutine?

A goroutine is a lightweight thread managed by the Go runtime.
```

## 2. Command List (MVP)

Four core commands:

- `srs add [file]` - Create a new card from markdown file or interactive prompt
- `srs review` - Start review session showing due cards
- `srs list [flags]` - List cards (with filters: --due, --tag, --all)
- `srs stats` - Show statistics (total cards, due count, retention rate)

## 3. Database Schema

**SQLite database**: `srs.db` in project root

**Tables**:

- `cards` - Scheduling state (links to markdown files via UUID)
  - `id` TEXT PRIMARY KEY (UUID from frontmatter)
  - `ease_factor` REAL (default 2.5, SM-2 parameter)
  - `interval_days` INTEGER (current interval in days)
  - `repetitions` INTEGER (number of successful reviews)
  - `next_review` TEXT (ISO 8601 timestamp)
  - `last_review` TEXT (ISO 8601 timestamp, nullable)
  - `created_at` TEXT (ISO 8601 timestamp)

- `reviews` - Review history (optional, for analytics)
  - `id` INTEGER PRIMARY KEY AUTOINCREMENT
  - `card_id` TEXT REFERENCES cards(id)
  - `rating` INTEGER (0=Again, 1=Hard, 2=Good, 3=Easy)
  - `reviewed_at` TEXT (ISO 8601 timestamp)

## 4. Scheduling Rules (SM-2 Algorithm)

**Initial State** (new card):
- `ease_factor`: 2.5
- `interval_days`: 0
- `repetitions`: 0
- `next_review`: current time (immediately due)

**After Review** (based on rating):

- **Again (0)**: Reset card
  - `repetitions` → 0
  - `interval_days` → 0
  - `next_review` → now (show again immediately)
  - `ease_factor` → max(1.3, ease_factor - 0.2)

- **Hard (1)**: Slight penalty
  - `ease_factor` → max(1.3, ease_factor - 0.15)
  - If `repetitions` = 0: `interval_days` → 1
  - Else: `interval_days` → max(1, interval_days * 1.2)
  - `repetitions` → repetitions + 1

- **Good (2)**: Normal progression
  - If `repetitions` = 0: `interval_days` → 1
  - Else if `repetitions` = 1: `interval_days` → 6
  - Else: `interval_days` → interval_days * ease_factor
  - `repetitions` → repetitions + 1
  - `ease_factor` unchanged

- **Easy (3)**: Bonus
  - If `repetitions` = 0: `interval_days` → 4
  - Else: `interval_days` → interval_days * ease_factor * 1.3
  - `repetitions` → repetitions + 1
  - `ease_factor` → min(2.5, ease_factor + 0.15)

**Due Cards**: Cards where `next_review <= current_time`

**Rounding**: All `interval_days` values rounded to nearest integer

## Implementation Notes

- Cards are source of truth (markdown files), database only tracks scheduling
- UUID in frontmatter ensures cards can be moved/renamed without breaking links
- Database initialized on first command run
- Card files parsed on-demand (no separate indexing step)
- Review session shows cards in random order (shuffled due cards)

## File Structure

```
srs/
├── SPEC.md          # This specification
├── go.mod
├── srs.db           # SQLite database (created at runtime)
└── cards/           # Card markdown files
    ├── card-1.md
    └── card-2.md
```

