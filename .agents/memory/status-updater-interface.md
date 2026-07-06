---
name: Status engine Updater interface pattern
description: How the status engine integrates with existing handler code without changing public signatures
---

The `status.Updater` interface has exactly one method matching `(*td.Message).EditText`:
```go
EditText(c *td.Client, text string, opts *td.EditTextMessageOpts) (*td.Message, error)
```
Both `*td.Message` (gotdbot) and `*status.Status` satisfy it structurally — no explicit declaration needed.

**Why:** Allows handler internal functions (`handleSingleTrack`, `handleMedia`, etc.) to accept `status.Updater` instead of `*td.Message`, so animated status messages are drop-in compatible. Public command handlers (`playHandler`, etc.) never change.

**How to apply:** When adding a new handler that uses a loading message: send via `status.New(c, m, status.TypeXxx)`, pass the returned `*Status` to internal helpers that accept `status.Updater`. All `updater.EditText(...)` calls work unchanged.
