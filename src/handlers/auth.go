/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func authListHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if m.IsPrivate() {
		return nil
	}

	chatID := m.ChatId

	authUser := db.Instance.GetAuthUsers(chatID)
	if authUser == nil || len(authUser) == 0 {
		_, _ = m.ReplyText(c, "🔐 <b>Access Control</b>\n\nNo authorized users found in this chat.", replyOpts)
		return nil
	}

	text := "🔐 <b>Authorized Users</b>\n" +
		"━━━━━━━━━━━━━━━━━━━━━━\n\n"
	for _, uid := range authUser {
		text += fmt.Sprintf("▸ <a href=\"tg://user?id=%d\">%d</a>\n", uid, uid)
	}

	_, _ = m.ReplyText(c, text, replyOpts)
	return td.EndGroups
}

func addAuthHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if m.IsPrivate() {
		return td.EndGroups
	}

	chatID := m.ChatId

	UserStatus, err := cache.GetUserAdmin(c, chatID, m.SenderID(), false)
	if err != nil {
		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "❌ <b>Error</b>\n\nUnable to verify administrator status.", replyOpts)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
	default:
		_, _ = m.ReplyText(c, "🔒 <b>Permission Denied</b>\n\nYou must be an administrator to use this command.", replyOpts)
		return td.EndGroups
	}

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Error</b>\n\n%s", err.Error()), replyOpts)
		return nil
	}

	if db.Instance.IsAuthUser(chatID, userID) {
		_, _ = m.ReplyText(c, fmt.Sprintf("ℹ️ <b>Already Authorized</b>\n\nUser <code>%d</code> is already in the authorized list.", userID), replyOpts)
		return nil
	}

	if err = db.Instance.AddAuthUser(chatID, userID); err != nil {
		c.Logger.Error("Failed to add authorized user", "error", err)
		_, _ = m.ReplyText(c, "❌ <b>Failed</b>\n\nCould not authorize this user.", replyOpts)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("✅ <b>User Authorized</b>\n\n▸ ID: <code>%d</code>", userID), replyOpts)
	return err
}

func removeAuthHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}

	if m.IsPrivate() {
		return td.EndGroups
	}

	chatID := m.ChatId

	UserStatus, err := cache.GetUserAdmin(c, chatID, m.SenderID(), false)
	if err != nil {
		c.Logger.Warn("GetUserAdmin error", "error", err)
		_, _ = m.ReplyText(c, "❌ <b>Error</b>\n\nUnable to verify administrator status.", replyOpts)
		return td.EndGroups
	}

	switch UserStatus.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
	default:
		_, _ = m.ReplyText(c, "🔒 <b>Permission Denied</b>\n\nYou must be an administrator to use this command.", replyOpts)
		return td.EndGroups
	}

	userID, err := getTargetUserID(c, m)
	if err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Error</b>\n\n%s", err.Error()), replyOpts)
		return nil
	}

	if !db.Instance.IsAuthUser(chatID, userID) {
		_, _ = m.ReplyText(c, fmt.Sprintf("ℹ️ <b>Not Authorized</b>\n\nUser <code>%d</code> is not in the authorized list.", userID), replyOpts)
		return nil
	}

	if err := db.Instance.RemoveAuthUser(chatID, userID); err != nil {
		c.Logger.Error("Failed to remove authorized user", "error", err)
		_, _ = m.ReplyText(c, "❌ <b>Failed</b>\n\nCould not remove this user.", replyOpts)
		return nil
	}

	_, err = m.ReplyText(c, fmt.Sprintf("🗑 <b>Authorization Removed</b>\n\n▸ ID: <code>%d</code>", userID), replyOpts)
	return err
}
