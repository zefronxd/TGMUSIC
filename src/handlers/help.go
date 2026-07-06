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
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core"

	td "github.com/AshokShau/gotdbot"
)

type helpCategory struct {
	Title   string
	Content string
	Markup  *td.ReplyMarkupInlineKeyboard
}

func getHelpCategories() map[string]helpCategory {
	back := core.BackHelpMenuKeyboard()
	return map[string]helpCategory{
		core.CBHelpUser: {
			Title: "User Commands",
			Content: "🎵 <b>Playback</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/play [song or URL]</code> — Play audio\n" +
				"▸ <code>/vplay [song or URL]</code> — Play video\n\n" +
				"📋 <b>Queue</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/queue</code> — View current queue\n\n" +
				"🔧 <b>Utilities</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/start</code> — Start the bot\n" +
				"▸ <code>/ping</code> — Check bot latency\n" +
				"▸ <code>/privacy</code> — Privacy policy",
			Markup: back,
		},
		core.CBHelpAdmin: {
			Title: "Admin Commands",
			Content: "🎛 <b>Playback Controls</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/skip</code> — Skip current track\n" +
				"▸ <code>/pause</code> — Pause playback\n" +
				"▸ <code>/resume</code> — Resume playback\n" +
				"▸ <code>/stop</code> · <code>/end</code> — Stop & clear queue\n" +
				"▸ <code>/mute</code> · <code>/unmute</code> — Toggle mute\n" +
				"▸ <code>/seek [sec]</code> — Seek forward by seconds\n" +
				"▸ <code>/speed [0.5–4.0]</code> — Set playback speed\n\n" +
				"📋 <b>Queue Management</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/remove [n]</code> — Remove track #n\n" +
				"▸ <code>/loop [0–10]</code> — Set loop count\n\n" +
				"🔐 <b>Access Control</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/auth</code> — Authorize a user\n" +
				"▸ <code>/unauth</code> — Remove authorization\n" +
				"▸ <code>/authlist</code> — List authorized users\n" +
				"▸ <code>/reload</code> — Refresh admin cache",
			Markup: back,
		},
		core.CBHelpDevs: {
			Title: "Developer Commands",
			Content: "🖥 <b>System</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/stats</code> — Runtime statistics\n" +
				"▸ <code>/av</code> — Active voice chats\n\n" +
				"🔧 <b>Maintenance</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/logger [on|off]</code> — Toggle play logger\n" +
				"▸ <code>/leaveall</code> — Leave all voice chats\n" +
				"▸ <code>/clearass</code> — Clear assistant assignments\n" +
				"▸ <code>/broadcast [msg]</code> — Broadcast to all chats",
			Markup: back,
		},
		core.CBHelpOwner: {
			Title: "Owner Commands",
			Content: "⚙️ <b>Settings</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/settings</code> — Chat settings panel",
			Markup: back,
		},
		core.CBHelpPlaylist: {
			Title: "Playlist Commands",
			Content: "🗂 <b>Playlist Management</b>\n" +
				"━━━━━━━━━━━━━━━━━━━━━━\n" +
				"▸ <code>/createplaylist [name]</code> — Create a playlist\n" +
				"▸ <code>/deleteplaylist [id]</code> — Delete a playlist\n" +
				"▸ <code>/addtoplaylist [id] [url]</code> — Add a track\n" +
				"▸ <code>/removefromplaylist [id] [n|url]</code> — Remove a track\n" +
				"▸ <code>/playlistinfo [id]</code> — Show playlist details\n" +
				"▸ <code>/myplaylists</code> — List your playlists",
			Markup: back,
		},
	}
}

func helpCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	botIntro := func() string {
		return fmt.Sprintf(
			"👋 <b>Hello, %s!</b>\n\n"+
				"I'm <b>%s</b> — your premium music companion for Telegram.\n\n"+
				"🎵 Stream audio & video · Multi-source · High performance\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"<b>Supported Platforms</b>\n"+
				"▸ YouTube  ·  Spotify  ·  Apple Music\n"+
				"▸ SoundCloud  ·  Deezer  ·  JioSaavn  ·  Tidal\n\n"+
				"<i>Choose a category below to explore all commands.</i>",
			user.FirstName, c.Me.FirstName,
		)
	}

	switch {
	case strings.Contains(data, core.CBHelpAll):
		_ = cb.Answer(c, 0, false, "Opening help menu…", "")
		_, _ = cb.EditMessageCaption(c, botIntro(), &td.EditCaptionOpts{
			ReplyMarkup: core.HelpMenuKeyboard(),
			ParseMode:   "HTML",
		})
		return nil

	case strings.Contains(data, core.CBHelpBack):
		_ = cb.Answer(c, 0, false, "Returning to main menu…", "")
		_, _ = cb.EditMessageCaption(c, botIntro(), &td.EditCaptionOpts{
			ReplyMarkup: core.AddMeMarkup(c.Me.Usernames.EditableUsername),
			ParseMode:   "HTML",
		})
		return nil
	}

	categories := getHelpCategories()
	if cat, ok := categories[data]; ok {
		_ = cb.Answer(c, 0, false, cat.Title, "")
		body := fmt.Sprintf("<b>%s</b>\n\n%s\n\n<i>Use the buttons below to go back.</i>", cat.Title, cat.Content)
		_, _ = cb.EditMessageCaption(c, body, &td.EditCaptionOpts{
			ReplyMarkup: cat.Markup,
			ParseMode:   "HTML",
		})
		return nil
	}

	_ = cb.Answer(c, 0, true, "Unknown help category.", "")
	return nil
}
