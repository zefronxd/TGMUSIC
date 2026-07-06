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
	"strconv"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// speedHandler handles the /speed command.
func speedHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return err
	}

	if playingSong := cache.ChatCache.GetPlayingTrack(chatID); playingSong == nil {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "🎚 <b>Playback Speed</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"<b>Usage:</b> <code>/speed [value]</code>\n\n"+
			"▸ Range: <code>0.5</code> – <code>4.0</code>\n"+
			"▸ Normal: <code>1.0</code>  ·  Fast: <code>2.0</code>", replyOpts)
		return nil
	}

	speed, err := strconv.ParseFloat(args, 64)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ <b>Invalid Value</b>\n\nProvide a number between <code>0.5</code> and <code>4.0</code>.", replyOpts)
		return nil
	}

	if err = vc.Calls.ChangeSpeed(c, chatID, speed); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Speed Change Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return nil
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("🎚 <b>Speed Set</b>  ·  <code>%.2fx</code>", speed), replyOpts)
	return nil
}
