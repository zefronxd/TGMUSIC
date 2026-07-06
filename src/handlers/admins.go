/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/src/core/status"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"github.com/zefronxd/TGMUSIC/src/vc"
	"fmt"
	"time"

	"github.com/zefronxd/TGMUSIC/src/core/cache"

	"github.com/AshokShau/gotdbot"
)

const reloadCooldown = 3 * time.Minute

var reloadRateLimit = cache.NewCache[time.Time](reloadCooldown)

func reloadAdminCacheHandler(c *gotdbot.Client, m *gotdbot.Message) error {
	if m.IsPrivate() {
		return gotdbot.EndGroups
	}

	reloadKey := fmt.Sprintf("reload:%d", m.ChatId)
	if lastUsed, ok := reloadRateLimit.Get(reloadKey); ok {
		timePassed := time.Since(lastUsed)
		if timePassed < reloadCooldown {
			remaining := int((reloadCooldown - timePassed).Seconds())
			_, _ = m.ReplyText(c, fmt.Sprintf("⏳ <b>Rate Limited</b>\n\nPlease wait <code>%s</code> before using this command again.", utils.SecToMin(remaining)), &gotdbot.SendTextMessageOpts{ParseMode: "HTML"})
			return nil
		}
	}

	reloadRateLimit.Set(reloadKey, time.Now())

	reply, err := status.New(c, m, status.TypeReloadingCache)
	if err != nil {
		c.Logger.Warn("Failed to send reloading message for chat", "chat_id", m.ChatId, "error", err)
		return gotdbot.EndGroups
	}

	cache.ClearAdminCache(m.ChatId)
	vc.Calls.UpdateInviteLink(m.ChatId, "")

	admins, err := cache.GetAdmins(c, m.ChatId, true)
	if err != nil {
		c.Logger.Warn("Failed to reload the admin cache for chat", "chat_id", m.ChatId, "error", err)
		_, _ = reply.EditText(c, "❌ <b>Reload Failed</b>\n\nCould not refresh the administrator cache.", &gotdbot.EditTextMessageOpts{ParseMode: "HTML"})
		return gotdbot.EndGroups
	}

	c.Logger.Info("Reloaded admins for chat", "count", len(admins), "chat_id", m.ChatId)
	_, _ = reply.EditText(c, fmt.Sprintf("✅ <b>Admin Cache Refreshed</b>\n\n▸ <code>%d</code> admins loaded.", len(admins)), &gotdbot.EditTextMessageOpts{ParseMode: "HTML"})
	return gotdbot.EndGroups
}

// privacyHandler handles the /privacy command.
func privacyHandler(c *gotdbot.Client, m *gotdbot.Message) error {

	botName := c.Me.FirstName

	text := fmt.Sprintf(
		"🔒 <b>Privacy Policy</b>  ·  %s\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"▸ <b>Data Storage</b>\n"+
			"  We do not store personal data or track browsing activity.\n\n"+
			"▸ <b>Collection</b>\n"+
			"  Only your Telegram <b>User ID</b> and <b>Chat ID</b> are collected to provide music services. No names, phone numbers, or locations are stored.\n\n"+
			"▸ <b>Usage</b>\n"+
			"  Data is used strictly for bot functionality — no marketing or commercial use.\n\n"+
			"▸ <b>Sharing</b>\n"+
			"  We do not share, sell, or trade data with third parties.\n\n"+
			"▸ <b>Security</b>\n"+
			"  Standard encryption is used to protect data. No online service is 100%% secure.\n\n"+
			"▸ <b>Cookies</b>\n"+
			"  %s does not use cookies or tracking technologies.\n\n"+
			"▸ <b>Your Rights</b>\n"+
			"  You may request data deletion or block the bot to revoke access.\n\n"+
			"▸ <b>Updates</b>\n"+
			"  Policy changes will be announced in the bot.\n\n"+
			"▸ <b>Contact</b>\n"+
			"  Questions? <a href=\"https://t.me/GuardxSupport\">Join our Support Group</a>.\n\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n"+
			"<i>This policy ensures a safe experience with %s.</i>",
		botName, botName, botName)

	_, err := m.ReplyText(c, text, &gotdbot.SendTextMessageOpts{ParseMode: "html", DisableWebPagePreview: true})
	return err
}
