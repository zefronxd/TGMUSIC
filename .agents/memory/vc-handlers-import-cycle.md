---
name: vc/handlers import cycle
description: how to trigger handlers-package logic from within vc package code without an import cycle
---

`src/handlers` imports `src/vc` (to enqueue/play tracks from commands), so `src/vc` can never import `src/handlers` back directly — that's a compile-time import cycle.

**Why:** Some vc-side lifecycle events (e.g. queue becoming empty) need to trigger handlers-level logic (e.g. posting a message with buttons, building keyboards from `src/core`). Since the dependency direction is fixed (handlers → vc), the callback must flow the other way at runtime.

**How to apply:** Declare a package-level exported func variable in `vc` (e.g. `var RecommendHook func(bot *td.Client, chatID int64, lastTrack *utils.CachedTrack)`), leave it `nil` by default, and call it defensively (`if RecommendHook != nil { go RecommendHook(...) }`) at the point in `vc` where the event happens. Then in `src/handlers`, assign the real implementation to that hook from an `init()` function in the relevant handler file. This keeps the import graph one-directional while still letting vc-side events drive handlers-side behavior.
