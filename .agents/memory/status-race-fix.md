---
name: Status engine animation vs resolution race fix
description: How the status engine prevents a late animation tick from overwriting a resolved message
---

`stop()` (closing `done` channel) alone is insufficient to prevent a late animation tick from writing after the resolution edit. A goroutine can already be past the `select` statement when `done` is closed.

**Fix:** `Status` holds `mu sync.Mutex` + `resolved bool`. Both `animate()` and `EditText()` acquire `mu` before calling `s.msg.EditText(...)`. Inside the lock, `animate()` checks `resolved` first and exits if true. `EditText()` sets `resolved = true` before its edit, while holding the lock.

**Why:** Makes final edit and any late animation tick mutually exclusive at the Telegram API-call boundary. In all interleavings, the resolution text is always the last message edit.

**How to apply:** Any future code that edits the status message from outside the `Status` type must also acquire `mu` and check `resolved`, or route through `Status.EditText()`.
