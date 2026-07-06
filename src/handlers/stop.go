/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"fmt"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// stopHandler handles the /stop command.
func stopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return nil
	}

	_ = vc.Calls.Stop(chatID, false)
	_, _ = m.ReplyText(c, fmt.Sprintf("⏹ <b>Stream Ended</b>\n\n👤 <i>Stopped by %s</i>", firstName(c, m)), replyOpts)
	return nil
}
