/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import (
	"strings"

	"github.com/zefronxd/TGMUSIC/src/utils"
)

// Filter key constants.
const (
	FilterAll      = ""    // show every track (default)
	FilterYouTube  = "yt"  // YouTube tracks only
	FilterSpotify  = "sp"  // Spotify tracks only
	FilterTelegram = "tg"  // Telegram file tracks only
	FilterVideo    = "vid" // video tracks only
	FilterAudio    = "aud" // audio-only tracks
)

// ApplyFilter returns a new slice with the now-playing track (index 0) always
// preserved, followed by only the "up next" tracks that match filter.
// When filter is FilterAll (empty string) or the queue is empty, the original
// slice is returned unchanged.
func ApplyFilter(tracks []*utils.CachedTrack, filter string) []*utils.CachedTrack {
	if len(tracks) == 0 || filter == FilterAll {
		return tracks
	}
	// Always keep tracks[0] (now playing); filter only the up-next portion.
	out := make([]*utils.CachedTrack, 1, len(tracks))
	out[0] = tracks[0]
	for _, t := range tracks[1:] {
		if matchFilter(t, filter) {
			out = append(out, t)
		}
	}
	return out
}

// matchFilter reports whether a single track satisfies the filter key.
func matchFilter(t *utils.CachedTrack, filter string) bool {
	switch filter {
	case FilterYouTube:
		return strings.EqualFold(t.Platform, utils.YouTube)
	case FilterSpotify:
		return strings.EqualFold(t.Platform, utils.Spotify)
	case FilterTelegram:
		return strings.EqualFold(t.Platform, utils.Telegram)
	case FilterVideo:
		return t.IsVideo
	case FilterAudio:
		return !t.IsVideo
	}
	return true
}

// FilterLabel returns a human-readable label for a filter key.
func FilterLabel(filter string) string {
	switch filter {
	case FilterYouTube:
		return "▶️ YouTube"
	case FilterSpotify:
		return "🟢 Spotify"
	case FilterTelegram:
		return "✈️ Telegram"
	case FilterVideo:
		return "🎥 Video"
	case FilterAudio:
		return "🎵 Audio"
	default:
		return "📂 All"
	}
}
