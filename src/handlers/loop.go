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

func loopHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "🔁 <b>Loop Control</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"<b>Usage:</b> <code>/loop [count]</code>\n\n"+
			"▸ <code>0</code> — Disable looping\n"+
			"▸ <code>1–10</code> — Set repeat count", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	argsInt, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ <b>Invalid Value</b>\n\nPlease provide a number between <code>0</code> and <code>10</code>.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	if argsInt < 0 || argsInt > 10 {
		_, err = m.ReplyText(c, "❌ <b>Out of Range</b>\n\nLoop count must be between <code>0</code> and <code>10</code>.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	cache.ChatCache.SetLoopCount(chatID, argsInt)

	var action string
	if argsInt == 0 {
		action = "🔁 <b>Loop Disabled</b>"
	} else {
		action = fmt.Sprintf("🔁 <b>Loop Set</b>  ·  <code>%d</code> repeat(s)", argsInt)
	}

	_, err = m.ReplyText(c, fmt.Sprintf("%s\n\n👤 <i>Changed by %s</i>", action, firstName(c, m)), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}
