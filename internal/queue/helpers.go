/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

// Package queue provides a modular, high-performance queue management system
// for the Telegram music bot. Adding new queue features requires only one new
// file inside this package — no modifications elsewhere.
package queue

import (
	"fmt"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/utils"
)

// PageSize is the number of "up next" tracks shown per page.
const PageSize = 10

// trunc truncates s to max runes, appending "…" when shortened.
func trunc(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// platformEmoji maps a platform name to a compact display emoji.
func platformEmoji(platform string) string {
	switch strings.ToLower(platform) {
	case strings.ToLower(utils.YouTube):
		return "▶️"
	case strings.ToLower(utils.Spotify):
		return "🟢"
	case strings.ToLower(utils.JioSaavn):
		return "🎶"
	case strings.ToLower(utils.Apple):
		return "🍎"
	case strings.ToLower(utils.SoundCloud):
		return "☁️"
	case strings.ToLower(utils.Telegram):
		return "✈️"
	case strings.ToLower(utils.Deezer):
		return "🎼"
	case strings.ToLower(utils.Tidal):
		return "🌊"
	case strings.ToLower(utils.DirectLink):
		return "🔗"
	default:
		return "🎵"
	}
}

// queueHash returns a lightweight fingerprint of the current queue state
// used to detect changes and invalidate the page cache.
func queueHash(tracks []*utils.CachedTrack) string {
	n := len(tracks)
	if n == 0 {
		return "0:"
	}
	last := tracks[n-1].TrackID
	return fmt.Sprintf("%d:%s:%s", n, tracks[0].TrackID, last)
}

// progressBar builds a Unicode progress bar of the given width.
func progressBar(elapsed, total, width int) string {
	if total <= 0 || width <= 0 {
		return strings.Repeat("▱", width)
	}
	filled := elapsed * width / total
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", width-filled)
}

// totalDuration sums all track durations in seconds.
func totalDuration(tracks []*utils.CachedTrack) int {
	total := 0
	for _, t := range tracks {
		total += t.Duration
	}
	return total
}

// uniqueRequesters counts distinct requester names in the track list.
func uniqueRequesters(tracks []*utils.CachedTrack) int {
	seen := make(map[string]struct{}, len(tracks))
	for _, t := range tracks {
		seen[t.User] = struct{}{}
	}
	return len(seen)
}

// fmtHumanDur converts seconds to a brief human-readable form, e.g. "1h 34m".
func fmtHumanDur(secs int) string {
	if secs <= 0 {
		return "0m"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	case m > 0 && s > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
