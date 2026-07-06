/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

// Package playerui provides the Now Playing Engine for Zefron Music.
// It renders a premium interactive player panel with live progress updates,
// per-chat state management, and a full suite of playback control callbacks.
//
// Adding a new player feature requires only one new file in this package — no
// changes to handlers, vc, or any other package are required.
package playerui

import (
	"github.com/zefronxd/TGMUSIC/src/utils"
	"fmt"
)

// qualityLabel returns a human-readable audio quality string for a platform.
func qualityLabel(platform string) string {
	switch platform {
	case utils.Spotify:
		return "Premium · 320 kbps"
	case utils.Apple:
		return "Lossless · ALAC"
	case utils.Tidal:
		return "HiFi · FLAC"
	case utils.JioSaavn:
		return "HD · 320 kbps"
	case utils.Deezer:
		return "HQ · 320 kbps"
	case utils.YouTube:
		return "HD · 256 kbps"
	case utils.SoundCloud:
		return "HQ · 128 kbps"
	case utils.Telegram:
		return "Local File"
	default:
		return "High Quality"
	}
}

// loopLabel returns a short display label for the loop count.
func loopLabel(count int) string {
	switch count {
	case 0:
		return "Off"
	case 1:
		return "1×"
	case 3:
		return "3×"
	case 5:
		return "5×"
	default:
		return fmt.Sprintf("%d×", count)
	}
}

// nextLoopCount advances the loop count through the cycle: 0 → 1 → 3 → 5 → 0.
func nextLoopCount(current int) int {
	switch current {
	case 0:
		return 1
	case 1:
		return 3
	case 3:
		return 5
	default:
		return 0
	}
}
