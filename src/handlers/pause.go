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

	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

func pauseHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	if _, err := vc.Calls.Pause(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Pause Failed</b>\n\n<code>%s</code>", err.Error()), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	_, err := m.ReplyText(c, fmt.Sprintf("⏸ <b>Playback Paused</b>\n\n👤 <i>Paused by %s</i>", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML"})
	return err
}

func resumeHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if chatID > 0 {
		_, _ = m.ReplyText(c, "⚠️ <b>Groups Only</b>\n\nThis command can only be used in a supergroup.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	if _, err := vc.Calls.Resume(chatID); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Resume Failed</b>\n\n<code>%s</code>", err.Error()), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return nil
	}

	_, err := m.ReplyText(c, fmt.Sprintf("▶️ <b>Playback Resumed</b>\n\n👤 <i>Resumed by %s</i>", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML"})
	return err
}
