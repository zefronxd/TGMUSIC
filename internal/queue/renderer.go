/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import (
	"fmt"
	"html"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/utils"
)

// RenderPage builds the full HTML text for a single page of the queue viewer.
//
// Parameters:
//   - chatTitle  – display name of the Telegram group.
//   - queue      – the full sorted+filtered queue snapshot (index 0 = now playing).
//   - page       – current 1-based page number.
//   - playedSecs – elapsed playback time in seconds (0 when unavailable).
//   - sortBy     – active sort key (used in the footer status line).
//   - filterBy   – active filter key (used in the footer status line).
//   - locked     – whether the queue is locked for normal users.
func RenderPage(
	chatTitle string,
	queue []*utils.CachedTrack,
	page int,
	playedSecs int,
	sortBy, filterBy string,
	locked bool,
) string {
	var b strings.Builder

	// ── Header ──────────────────────────────────────────────────────────────
	lockIcon := ""
	if locked {
		lockIcon = " 🔒"
	}
	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("🎵 <b>Queue</b>  ·  %s%s\n", html.EscapeString(trunc(chatTitle, 30)), lockIcon))
	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(queue) == 0 {
		b.WriteString("📋 <i>Queue is empty.  Use /play to add a track.</i>\n")
		return b.String()
	}

	// ── Now Playing ─────────────────────────────────────────────────────────
	current := queue[0]
	dur := current.Duration
	elapsed := playedSecs
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > dur {
		elapsed = dur
	}

	titleText := html.EscapeString(trunc(current.Name, 50))
	if current.URL != "" {
		titleText = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(current.URL), titleText)
	}

	bar := ""
	if dur > 0 {
		bar = fmt.Sprintf("\n  <code>%s</code>  %s  <code>%s</code>",
			utils.SecToMin(elapsed),
			progressBar(elapsed, dur, 12),
			utils.SecToMin(dur),
		)
	}

	mediaType := "🎵"
	if current.IsVideo {
		mediaType = "🎬"
	}

	b.WriteString(fmt.Sprintf("🎧 <b>Now Playing</b>  %s\n", platformEmoji(current.Platform)))
	b.WriteString(fmt.Sprintf("▸ <b>%s</b>%s\n", titleText, bar))
	if current.Channel != "" {
		b.WriteString(fmt.Sprintf("  👤 %s\n", html.EscapeString(trunc(current.Channel, 30))))
	}
	b.WriteString(fmt.Sprintf("  %s Requested by: <i>%s</i>\n", mediaType, html.EscapeString(trunc(current.User, 25))))

	// ── Up Next ─────────────────────────────────────────────────────────────
	upNext := PageSlice(queue, page)
	totalPages := PageCount(len(queue))
	offset := QueueOffset(page)

	if len(upNext) > 0 {
		b.WriteString(fmt.Sprintf("\n━━━━━━━━━━━━━━━━━━━━━━\n📜 <b>Up Next</b>  ·  <code>%d</code> track(s)\n", len(queue)-1))
		b.WriteString("─────────────────────\n")

		for i, song := range upNext {
			pos := offset + i
			name := html.EscapeString(trunc(song.Name, 38))
			dur := utils.SecToMin(song.Duration)
			src := platformEmoji(song.Platform)
			b.WriteString(fmt.Sprintf(
				"<code>%2d.</code> %s\n     %s  <code>%s</code>  ·  <i>%s</i>\n",
				pos, name, src, dur, html.EscapeString(trunc(song.User, 18)),
			))
		}
	}

	// ── Stats footer ────────────────────────────────────────────────────────
	allQueued := len(queue) - 1
	remDur := totalDuration(queue[1:])
	requesters := uniqueRequesters(queue)

	b.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf(
		"📊 <b>%d</b> songs  ·  <b>%s</b> remaining  ·  <b>%d</b> requester(s)\n",
		allQueued, fmtHumanDur(remDur), requesters,
	))

	// Footer status line
	pageInfo := ""
	if totalPages > 1 {
		pageInfo = fmt.Sprintf("  🔢 Page <b>%d</b> / <b>%d</b>", page, totalPages)
	}
	sortInfo := ""
	if sortBy != SortDefault {
		sortInfo = fmt.Sprintf("  ·  %s", SortLabel(sortBy))
	}
	filterInfo := ""
	if filterBy != FilterAll {
		filterInfo = fmt.Sprintf("  ·  %s", FilterLabel(filterBy))
	}
	if pageInfo != "" || sortInfo != "" || filterInfo != "" {
		b.WriteString(pageInfo + sortInfo + filterInfo + "\n")
	}
	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━")

	return b.String()
}

// RenderDetail returns the HTML detail view for a single track.
// idx is the 1-based position in the "up next" list (raw queue index = idx).
func RenderDetail(queue []*utils.CachedTrack, idx int) string {
	if idx < 1 || idx >= len(queue) {
		return "❌ <b>Track not found.</b>"
	}

	t := queue[idx]
	var b strings.Builder

	titleText := html.EscapeString(trunc(t.Name, 50))
	if t.URL != "" {
		titleText = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(t.URL), titleText)
	}

	mediaType := "🎵 Audio"
	if t.IsVideo {
		mediaType = "🎬 Video"
	}

	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	b.WriteString(fmt.Sprintf("🎵 <b>Track #%d Details</b>\n", idx))
	b.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	b.WriteString(fmt.Sprintf("▸ <b>%s</b>\n\n", titleText))
	if t.Channel != "" {
		b.WriteString(fmt.Sprintf("👤 <b>Artist:</b> %s\n", html.EscapeString(trunc(t.Channel, 40))))
	}
	b.WriteString(fmt.Sprintf("⏱ <b>Duration:</b> <code>%s</code>\n", utils.SecToMin(t.Duration)))
	b.WriteString(fmt.Sprintf("%s <b>Type:</b> %s\n", platformEmoji(t.Platform), mediaType))
	b.WriteString(fmt.Sprintf("🌐 <b>Source:</b> %s\n", html.EscapeString(t.Platform)))
	if t.Views != "" {
		b.WriteString(fmt.Sprintf("👁 <b>Views:</b> %s\n", html.EscapeString(t.Views)))
	}
	b.WriteString(fmt.Sprintf("🙋 <b>Requested by:</b> <i>%s</i>\n", html.EscapeString(t.User)))

	b.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━")
	return b.String()
}

// RenderStatsAlert returns a compact one-line stats string for a cb.Answer alert.
func RenderStatsAlert(queue []*utils.CachedTrack, page, totalPages int) string {
	if len(queue) == 0 {
		return "Queue is empty."
	}
	queued := len(queue) - 1
	remDur := totalDuration(queue[1:])
	requesters := uniqueRequesters(queue)
	return fmt.Sprintf(
		"Page %d/%d  •  %d queued  •  %s remaining  •  %d requester(s)",
		page, totalPages, queued, fmtHumanDur(remDur), requesters,
	)
}
