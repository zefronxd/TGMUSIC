/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import (
	"github.com/zefronxd/TGMUSIC/src/core"
	td "github.com/AshokShau/gotdbot"
)

// btn creates a callback-data inline button.
func btn(text, data string) td.InlineKeyboardButton {
	return td.InlineKeyboardButton{
		Text: text,
		Type: &td.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
	}
}

// PlayerKeyboard returns the premium 4-row inline keyboard for the player panel.
//
//	Row 1: ⏮ Previous │ ⏸/▶ Pause·Resume │ ⏭ Skip
//	Row 2: 🔀 Shuffle  │ 🔁 Loop           │ 📜 Queue
//	Row 3: 🎧 Lyrics   │ 📥 Download       │ 📺 Video
//	Row 4: ❤️ Favorite  │ ⚙ Settings       │ ✕ Close
//
// Buttons update automatically based on isPaused.
func PlayerKeyboard(isPaused bool) *td.ReplyMarkupInlineKeyboard {
	pauseResumeBtn := btn("⏸ Pause", core.CBPlayerPauseResume)
	if isPaused {
		pauseResumeBtn = btn("▶ Resume", core.CBPlayerPauseResume)
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			// Row 1 – transport controls
			{btn("⏮", core.CBPlayerPrev), pauseResumeBtn, btn("⏭ Skip", core.CBPlayerSkip)},
			// Row 2 – queue/mode controls
			{btn("🔀 Shuffle", core.CBPlayerShuffle), btn("🔁 Loop", core.CBPlayerLoop), btn("📜 Queue", core.CBPlayerQueue)},
			// Row 3 – content features
			{btn("🎧 Lyrics", core.CBPlayerLyrics), btn("📥 Download", core.CBPlayerDownload), btn("📺 Video", core.CBPlayerVideo)},
			// Row 4 – extras
			{btn("❤️ Fav", core.CBPlayerFav), btn("⚙ Settings", core.CBPlayerSettings), btn("✕ Close", core.CBPlayerClose)},
		},
	}
}
