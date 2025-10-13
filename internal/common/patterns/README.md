# Pattern Matching

Regular expression matching with timeout protection.

## Purpose

- Regex matching with ReDoS protection
- Timeout handling (5s max per match)
- Pattern compilation caching
- Extract matched text and line numbers

## Key Files

- `match.go` - Regex matching with timeout
- `cache.go` - Compiled pattern cache (optional)

