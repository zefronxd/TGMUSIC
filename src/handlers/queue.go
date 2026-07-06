/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/internal/queue"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// queueHandler sends the interactive queue management UI for the current chat.
func queueHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, _ = m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return nil
	}

	rawQueue := cache.ChatCache.GetQueue(chatID)
	if len(rawQueue) == 0 {
		_, _ = m.ReplyText(c, "📋 <b>Queue Empty</b>\n\nThere are no tracks in the queue.\n\n<i>Use /play to add a track.</i>", replyOpts)
		return nil
	}

	chat, err := c.GetChat(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "Error fetching chat information.", nil)
		return nil
	}

	// Invalidate stale cache so the first render is always fresh.
	queue.Manager.Invalidate(chatID)
	queue.Manager.SetPage(chatID, 1)

	playedSecs := 0
	if t, err := vc.Calls.PlayedTime(chatID); err == nil {
		playedSecs = int(t)
	}

	sortBy := queue.Manager.GetSort(chatID)
	filterBy := queue.Manager.GetFilter(chatID)
	locked := queue.Manager.IsLocked(chatID)

	view := queue.ApplySort(rawQueue, sortBy)
	view = queue.ApplyFilter(view, filterBy)

	page := 1
	totalPages := queue.PageCount(len(view))
	upNext := queue.PageSlice(view, page)

	text := queue.RenderPage(chat.Title, view, page, playedSecs, sortBy, filterBy, locked)
	queue.Manager.SetCachedText(chatID, page, text)

	kb := queue.MainKeyboard(page, totalPages, locked, len(upNext))
	_, err = m.ReplyText(c, text, &td.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		ReplyMarkup:           kb,
	})
	return err
}
