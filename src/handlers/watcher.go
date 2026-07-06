/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"fmt"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func handleVoiceChatMessage(c *td.Client, update *td.UpdateNewMessage) error {
	m := update.Message
	chatID := m.ChatId

	if m.IsGroup() {
		text := fmt.Sprintf(
			"⚠️ <b>Supergroup Required</b>\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"Chat <code>%d</code> is a basic group — I need a <b>supergroup</b> to work.\n\n"+
				"▸ Convert this chat to a supergroup\n"+
				"▸ Add me as an admin\n\n"+
				"📖 <a href=\"https://te.legra.ph/How-to-Convert-a-Group-to-a-Supergroup-01-02\">How to convert a group</a>\n\n"+
				"<i>Join our support group if you need help.</i>",
			chatID,
		)

		_, _ = c.SendTextMessage(chatID, text, &td.SendTextMessageOpts{
			ReplyMarkup:           core.AddMeMarkup(c.Me.Usernames.EditableUsername),
			DisableWebPagePreview: true,
			ParseMode:             "HTML",
		})

		time.Sleep(1 * time.Second)
		_ = c.LeaveChat(chatID)
		return nil
	}

	if m.Content == nil {
		return nil
	}
	var message string
	switch m.Content.(type) {
	case *td.MessageVideoChatStarted:
		cache.ChatCache.ClearChat(chatID)
		message = "🎙 <b>Voice Chat Started</b>\n\n▸ Use <code>/play [song or URL]</code> to begin streaming."
	case *td.MessageVideoChatEnded:
		cache.ChatCache.ClearChat(chatID)
		message = "🔴 <b>Voice Chat Ended</b>\n\n<i>Queue cleared. See you next time!</i>"
	default:
		return nil
	}

	_, _ = c.SendTextMessage(chatID, message, &td.SendTextMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}
