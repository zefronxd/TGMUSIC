/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/config"
	"fmt"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// activeVcHandler handles the /activevc command.
// It takes a telegram.NewMessage object as input.
// It returns an error if any.
func activeVcHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	activeChats := cache.ChatCache.GetActiveChats()
	if len(activeChats) == 0 {
		_, err := m.ReplyText(c, "🔴 <b>No Active Voice Chats</b>\n\nAll voice chats are currently idle.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎙 <b>Active Voice Chats</b>  ·  <code>%d</code>\n", len(activeChats)))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for _, chatID := range activeChats {
		queueLength := cache.ChatCache.GetQueueLength(chatID)
		currentSong := cache.ChatCache.GetPlayingTrack(chatID)

		var songInfo string
		if currentSong != nil {
			songInfo = fmt.Sprintf(
				"  🎶 <a href='%s'>%s</a>  <code>%ds</code>",
				currentSong.URL,
				currentSong.Name,
				currentSong.Duration,
			)
		} else {
			songInfo = "  🔇 <i>Idle</i>"
		}

		sb.WriteString(fmt.Sprintf(
			"▸ <code>%d</code>  ·  Queue: <code>%d</code>\n%s\n\n",
			chatID,
			queueLength,
			songInfo,
		))
	}

	text := sb.String()
	if len(text) > 4096 {
		text = fmt.Sprintf("🎙 <b>Active Voice Chats</b>  ·  <code>%d</code>", len(activeChats))
	}

	_, err := m.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
	if err != nil {
		return err
	}

	return nil
}

// Handles the /clearass command to remove all assistant assignments
func clearAssistantsHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	done, err := db.Instance.ClearAllAssistants()
	if err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Error</b>\n\nFailed to clear assistants: <code>%s</code>", err.Error()), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ <b>Assistants Cleared</b>\n\n▸ Removed from <code>%d</code> chats.", done), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}

// Handles the /leaveall command to leave all chats
func leaveAllHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	reply, err := m.ReplyText(c, "🚪 <b>Leaving all voice chats…</b>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	if err != nil {
		return err
	}

	leftCount, err := vc.Calls.LeaveAll()
	if err != nil {
		_, _ = reply.EditText(c, fmt.Sprintf("❌ <b>Error</b>\n\nFailed to leave all chats: <code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
		return err
	}

	_, err = reply.EditText(c, fmt.Sprintf("✅ <b>Done</b>\n\n▸ Left <code>%d</code> voice chats.", leftCount), &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

// Handles the /logger command to toggle logger status
func loggerHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if config.LoggerId == 0 {
		_, _ = m.ReplyText(c, "⚠️ <b>Logger Not Configured</b>\n\nSet <code>LOGGER_ID</code> in your environment first.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	loggerStatus := db.Instance.GetLoggerStatus()
	args := strings.ToLower(Args(m))
	statusStr := "disabled"
	if loggerStatus {
		statusStr = "enabled"
	}
	if len(args) == 0 {
		_, _ = m.ReplyText(c, fmt.Sprintf("📋 <b>Logger</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"▸ Status: <code>%s</code>\n\n"+
			"<b>Usage:</b> <code>/logger [enable|disable|on|off]</code>", statusStr), &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	switch args {
	case "enable", "on":
		_ = db.Instance.SetLoggerStatus(true)
		_, _ = m.ReplyText(c, "✅ <b>Logger Enabled</b>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	case "disable", "off":
		_ = db.Instance.SetLoggerStatus(false)
		_, _ = m.ReplyText(c, "🔴 <b>Logger Disabled</b>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	default:
		_, _ = m.ReplyText(c, "❌ <b>Invalid Argument</b>\n\nUse <code>enable</code>, <code>disable</code>, <code>on</code>, or <code>off</code>.", &td.SendTextMessageOpts{ParseMode: "HTML"})
	}

	return td.EndGroups
}
