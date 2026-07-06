/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import "fmt"

const progressBarWidth = 18

// ProgressBar renders a Unicode progress bar of fixed width.
// elapsed and total are in seconds.  total ≤ 0 returns a full empty bar.
func ProgressBar(elapsed, total int) string {
	if total <= 0 {
		return buildBar(0, progressBarWidth)
	}
	ratio := float64(elapsed) / float64(total)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	filled := int(ratio * float64(progressBarWidth))
	return buildBar(filled, progressBarWidth)
}

func buildBar(filled, width int) string {
	buf := make([]rune, width)
	for i := range buf {
		if i < filled {
			buf[i] = '█'
		} else {
			buf[i] = '░'
		}
	}
	return string(buf)
}

// FormatTime formats seconds as MM:SS (or HH:MM:SS when ≥ 1 hour).
func FormatTime(secs int) string {
	if secs < 0 {
		secs = 0
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
