# File Operations

Shared library for safe file operations used by detection modules.

## Purpose

- Safe file reading with size limits and timeouts
- Hash calculation (SHA256, SHA1, MD5, SHA512)
- File existence and permission checks
- Protection against large files and slow I/O

## Key Files

- `read.go` - Safe file reading utilities
- `hash.go` - Hash calculation functions
- `exists.go` - File existence checks

