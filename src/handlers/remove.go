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

	td "github.com/AshokShau/gotdbot"
)

// removeHandler handles the /remove command.
func removeHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return nil
	}

	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) == 0 {
		_, _ = m.ReplyText(c, "📋 <b>Queue Empty</b>\n\nThere are no tracks in the queue.", replyOpts)
		return nil
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "🗑 <b>Remove Track</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"<b>Usage:</b> <code>/remove [track number]</code>\n\n"+
			"▸ <code>/remove 1</code> — Remove track #1\n"+
			"▸ <code>/remove 2</code> — Remove track #2", replyOpts)
		return nil
	}

	trackNum, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ <b>Invalid Number</b>\n\nPlease provide a valid track number.", replyOpts)
		return nil
	}

	if trackNum <= 0 || trackNum > len(queue) {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Out of Range</b>\n\nChoose a number between <code>1</code> and <code>%d</code>.", len(queue)), replyOpts)
		return nil
	}

	cache.ChatCache.RemoveTrack(chatID, trackNum)
	_, err = m.ReplyText(c, fmt.Sprintf("🗑 <b>Track Removed</b>  ·  <code>#%d</code>\n\n👤 <i>Removed by %s</i>", trackNum, firstName(c, m)), replyOpts)
	return err
}
