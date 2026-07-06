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

func muteHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if args := Args(m); args != "" {
		return td.EndGroups
	}

	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	if _, err := vc.Calls.Mute(chatID); err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("❌ <b>Mute Failed</b>\n\n<code>%s</code>", err.Error()), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	_, err := m.ReplyText(c, fmt.Sprintf("🔇 <b>Muted</b>\n\n👤 <i>Muted by %s</i>", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML"})
	return err
}

func unmuteHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if args := Args(m); args != "" {
		return td.EndGroups
	}

	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	if _, err := vc.Calls.Unmute(chatID); err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("❌ <b>Unmute Failed</b>\n\n<code>%s</code>", err.Error()), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	_, err := m.ReplyText(c, fmt.Sprintf("🔊 <b>Unmuted</b>\n\n👤 <i>Unmuted by %s</i>", firstName(c, m)), &td.SendTextMessageOpts{ReplyMarkup: core.ControlButtons("unmute"), ParseMode: "HTML"})
	return err
}
