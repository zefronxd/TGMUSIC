/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import (
	"fmt"
	"html"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/utils"
)

// RenderPlayer builds the full premium HTML now-playing panel message.
// elapsedSecs is the current playback position in seconds.
// All user-supplied strings are HTML-escaped internally.
func RenderPlayer(track *utils.CachedTrack, chatID int64, elapsedSecs int) string {
	if track == nil {
		return "⚠️ <b>No Active Stream</b>"
	}

	name := html.EscapeString(track.Name)
	user := html.EscapeString(track.User)

	title := name
	if track.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(track.URL), name)
	}

	total := track.Duration
	elapsed := elapsedSecs
	if elapsed < 0 {
		elapsed = 0
	}
	if total > 0 && elapsed > total {
		elapsed = total
	}

	remaining := total - elapsed
	if remaining < 0 {
		remaining = 0
	}

	bar := ProgressBar(elapsed, total)
	loopCount := cache.ChatCache.GetLoopCount(chatID)
	queueLen := cache.ChatCache.GetQueueLength(chatID)

	mediaType := "🎵 Audio"
	if track.IsVideo {
		mediaType = "🎬 Video"
	}

	var sb strings.Builder

	// ── Header ───────────────────────────────────────────────────
	sb.WriteString("🎵 <b>Now Playing</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// ── Track info ────────────────────────────────────────────────
	fmt.Fprintf(&sb, "🎶 <b>%s</b>\n", title)
	if track.Channel != "" {
		fmt.Fprintf(&sb, "🎙 <i>%s</i>\n", html.EscapeString(track.Channel))
	}

	// ── Progress ─────────────────────────────────────────────────
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprintf(&sb, "⏱ <code>%s / %s</code>\n", FormatTime(elapsed), FormatTime(total))
	fmt.Fprintf(&sb, "<code>%s</code>\n", bar)

	if total > 0 {
		fmt.Fprintf(&sb, "⏳ <code>-%s remaining</code>\n", FormatTime(remaining))
	}

	// ── Metadata ─────────────────────────────────────────────────
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprintf(&sb, "🎧 <b>%s</b>\n", qualityLabel(track.Platform))

	if track.Platform != "" {
		fmt.Fprintf(&sb, "%s <b>%s</b>\n",
			utils.PlatformEmoji(track.Platform),
			utils.PlatformLabel(track.Platform))
	}

	fmt.Fprintf(&sb, "📺 <b>%s</b>\n", mediaType)

	if track.Views != "" && track.Views != "0" {
		fmt.Fprintf(&sb, "👁 <i>%s views</i>\n", html.EscapeString(track.Views))
	}

	fmt.Fprintf(&sb, "👤 <b>Requested by</b> <i>%s</i>\n", user)

	if queueLen > 0 {
		fmt.Fprintf(&sb, "📜 <b>Position</b> <code>#1 of %d</code>\n", queueLen)
	}

	fmt.Fprintf(&sb, "🔁 <b>Loop</b>  ·  <code>%s</code>\n", loopLabel(loopCount))
	sb.WriteString("🔊 <b>Volume</b>  ·  <code>100%</code>\n")

	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")

	return sb.String()
}

// RenderStopped returns the panel text shown after playback has ended.
func RenderStopped(stoppedBy string) string {
	actor := html.EscapeString(stoppedBy)
	var sb strings.Builder
	sb.WriteString("⏹ <b>Stream Ended</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString("The voice chat session has finished.\n\n")
	sb.WriteString("<i>Use /play to start a new stream.</i>")
	if actor != "" {
		fmt.Fprintf(&sb, "\n\n👤 <i>Stopped by %s</i>", actor)
	}
	return sb.String()
}

// RenderPaused returns the panel text shown while playback is paused.
func RenderPaused(track *utils.CachedTrack, chatID int64, elapsedSecs int, pausedBy string) string {
	base := RenderPlayer(track, chatID, elapsedSecs)
	// Replace the header to show "Paused" state.
	base = strings.Replace(base, "🎵 <b>Now Playing</b>", "⏸ <b>Paused</b>", 1)
	if pausedBy != "" {
		base += fmt.Sprintf("\n\n👤 <i>Paused by %s</i>", html.EscapeString(pausedBy))
	}
	return base
}
