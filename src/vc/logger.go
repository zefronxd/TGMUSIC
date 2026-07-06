/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package vc

import (
	"github.com/zefronxd/TGMUSIC/config"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

// sendLogger sends a structured play-event log to the designated logger chat.
func sendLogger(client *td.Client, chatID int64, song *utils.CachedTrack) {
	if chatID == 0 || song == nil || chatID == config.LoggerId {
		return
	}

	emoji := utils.PlatformEmoji(song.Platform)
	platform := utils.PlatformLabel(song.Platform)

	text := fmt.Sprintf(
		"%s <b>Now streaming</b> in <code>%d</code>\n\n"+
			"🎶 <a href='%s'>%s</a>\n"+
			"⏱ %s  ·  👤 %s\n"+
			"%s %s  ·  📹 %v",
		emoji, chatID,
		song.URL, song.Name,
		utils.SecToMin(song.Duration), song.User,
		emoji, platform, song.IsVideo,
	)

	_, err := client.SendTextMessage(config.LoggerId, text, &td.SendTextMessageOpts{
		DisableWebPagePreview: true,
		ParseMode:             "HTML",
	})
	if err != nil {
		logger.Warn("Failed to send play log to logger chat", "error", err)
	}
}
