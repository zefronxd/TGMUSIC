/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package status

import "fmt"

const separator = "━━━━━━━━━━━━━━━━━━━━━━"

// dots maps a frame index (0–3) to the trailing punctuation shown after the
// status label.  Frame 0 is the clean initial render; frames 1–3 cycle dots.
var dots = [4]string{"", ".", "..", "..."}

// renderLoading returns the HTML for a loading state at the given dot frame.
//
//	━━━━━━━━━━━━━━━━━━━━━━
//	🎵 Zefron Music
//
//	🔍 Searching Music...
//
//	Please wait…
//	━━━━━━━━━━━━━━━━━━━━━━
func renderLoading(d statusDef, dotFrame int) string {
	frame := dotFrame % len(dots)
	return fmt.Sprintf(
		"%s\n🎵 <b>Zefron Music</b>\n\n%s <b>%s</b>%s\n\n<i>%s</i>\n%s",
		separator,
		d.Emoji, d.Label, dots[frame],
		d.Hint,
		separator,
	)
}
