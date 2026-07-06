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
	"runtime"
	"time"

	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/db"

	td "github.com/AshokShau/gotdbot"
)

// pingHandler handles the /ping command.
func pingHandler(c *td.Client, m *td.Message) error {

	start := time.Now()

	msg, err := m.ReplyText(c, "Pinging… please wait…", nil)
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := getFormattedDuration(time.Since(startTime))

	response := fmt.Sprintf(
		"📊 <b>System Status</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"▸ <b>Latency</b>    <code>%d ms</code>\n"+
			"▸ <b>Uptime</b>     <code>%s</code>\n"+
			"▸ <b>Goroutines</b> <code>%d</code>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━",
		latency, uptime, runtime.NumGoroutine(),
	)

	_, err = msg.EditText(c, response, &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

// startHandler handles the /start command.
func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId

	if m.IsPrivate() {
		go func(chatID int64) {
			_ = db.Instance.AddUser(chatID)
		}(chatID)

		response := fmt.Sprintf(
			"👋 <b>Hello, %s!</b>\n\n"+
				"I'm <b>%s</b> — your premium music companion.\n\n"+
				"🎵 Stream music in Telegram voice chats\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"<b>Supported Platforms</b>\n"+
				"▸ YouTube  ·  Spotify  ·  Apple Music\n"+
				"▸ SoundCloud  ·  Deezer  ·  JioSaavn\n"+
				"▸ Twitch  ·  Kick  ·  MX Player  ·  Tidal\n\n"+
				"<i>Tap the Help button to explore all commands.</i>",
			firstName(c, m),
			c.Me.FirstName,
		)

		_, err := m.ReplyPhoto(c, td.InputFileRemote{Id: config.StartImg}, &td.SendPhotoOpts{
			ParseMode:   "HTML",
			Caption:     response,
			ReplyMarkup: core.AddMeMarkup(c.Me.Usernames.EditableUsername),
		})

		return err
	}

	go func(chatID int64) {
		_ = db.Instance.AddChat(chatID)
	}(chatID)

	uptime := getFormattedDuration(time.Since(startTime))
	response := fmt.Sprintf(
		"🎵 <b>%s</b> is online\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"▸ <b>Status</b>  <code>Ready</code>\n"+
			"▸ <b>Uptime</b>  <code>%s</code>\n\n"+
			"<i>Use /play to start streaming music in this group.</i>",
		c.Me.FirstName,
		uptime,
	)

	_, err := m.ReplyText(c, response, &td.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		ReplyMarkup:           core.SupportBtn(),
	})

	return err
}
