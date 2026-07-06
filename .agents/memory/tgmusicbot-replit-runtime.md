---
name: Running TgMusicBot on Replit
description: How to actually build and run the Go music bot in Replit's dev environment, despite docs saying native libs are unavailable.
---

The project's replit.md claims `libtdjson.so.1.8.65` and `ntgcalls` native libs are "not available in Replit's default environment." That's outdated — both can be fetched and the bot runs fine here.

Steps that work:
1. `go run github.com/AshokShau/gotdbot/scripts/tools` downloads `libtdjson.so.1.8.65` into the project root.
2. `go run setup_ntgcalls.go` downloads the ntgcalls static libs/headers into `src/vc/`.
3. `CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .` builds the binary (CGo link step is slow — run in background with a log file rather than a foreground command with a short timeout).
4. At runtime, `libtdjson.so` dynamically needs `libssl.so.3` and `libstdc++.so.6`, which aren't present in the base Nix env. Install `openssl` via system dependencies, and `libgcc` (nix package name `libgcc`, NOT `stdenv.cc.cc.lib` or `gcc-unwrapped.lib` which don't exist in rippkgs) for libstdc++.
5. `libgcc`'s libstdc++.so.6 isn't on the default loader path — the workflow run command must set `LD_LIBRARY_PATH` to the gcc lib output dir (find via `gcc -print-file-name=libstdc++.so.6`) before invoking `./main`.
6. Song downloads shell out to a `yt-dlp` binary on PATH — install it via system dependencies (nix package name `yt-dlp`), not pip. Missing it fails downloads with "executable file not found in $PATH".

**Why:** avoids re-diagnosing the same "cannot open shared object file"/"executable not found" errors each session and re-guessing correct Nix package names.

**How to apply:** when asked to run/build this bot, follow the above sequence directly instead of assuming it's impossible in Replit.
