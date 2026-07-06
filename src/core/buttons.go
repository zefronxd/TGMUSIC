/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package core

import (
	"fmt"
	"github.com/zefronxd/TGMUSIC/config"
	"github.com/zefronxd/TGMUSIC/src/utils"

	"github.com/AshokShau/gotdbot"
)

// cb creates a callback-data inline button.
func cb(text, data string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
	}
}

// url creates a URL inline button.
func url(text, link string) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
	}
}

// Pre-built buttons that never change.
var (
	CloseBtn    = cb("Close", CBVcPlayClose)
	HomeBtn     = cb("Home", CBHelpBack)
	HelpBtn     = cb("Help", CBHelpAll)
	UserBtn     = cb("Users", CBHelpUser)
	AdminBtn    = cb("Admins", CBHelpAdmin)
	OwnerBtn    = cb("Owner", CBHelpOwner)
	DevsBtn     = cb("Devs", CBHelpDevs)
	PlaylistBtn = cb("Playlist", CBHelpPlaylist)

	SourceCodeBtn = url("Source Code", "https://github.com/zefronxd/TGMUSIC")
)

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string, cmdDelete bool, language string) *gotdbot.ReplyMarkupInlineKeyboard {
	playText := "Everyone"
	if playMode == utils.Admins {
		playText = "Admins"
	}

	deleteText := "False"
	if cmdDelete {
		deleteText = "True"
	}

	adminText := "Everyone"
	if adminMode == utils.Admins {
		adminText = "Admins"
	}

	langText := "English"
	if language != "en" && language != "" {
		langText = language
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{cb("Play Mode ➜", CBSettingsMain), cb(playText, CBSettingsPlay)},
			{cb("Command Delete ➜", CBSettingsMain), cb(deleteText, CBSettingsDelete)},
			{cb("Admin Mode ➜", CBSettingsMain), cb(adminText, CBSettingsAdmin)},
			{cb("Language ➜", CBSettingsMain), cb(langText, CBSettingsLang)},
			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{UserBtn, AdminBtn, OwnerBtn},
			{PlaylistBtn, DevsBtn, CloseBtn},
			{HomeBtn},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{HelpBtn, HomeBtn},
			{CloseBtn, SourceCodeBtn},
		},
	}
}

// ControlButtons returns the inline keyboard for a given playback state.
// Recognised modes: "play", "pause", "resume", "mute", "unmute".
// Any other value returns a minimal keyboard with only the Close button.
func ControlButtons(mode string) *gotdbot.ReplyMarkupInlineKeyboard {
	skipBtn := cb("‣‣I", CBPlaySkip)
	stopBtn := cb("▢", CBPlayStop)
	pauseBtn := cb("II", CBPlayPause)
	resumeBtn := cb("▷", CBPlayResume)
	muteBtn := cb("🔇", CBPlayMute)
	unmuteBtn := cb("🔊", CBPlayUnmute)
	addToPlaylistBtn := cb("➕", CBPlayAddToList)

	switch mode {
	case "play":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{addToPlaylistBtn, CloseBtn},
			},
		}
	case "pause":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, resumeBtn},
				{CloseBtn},
			},
		}
	case "resume":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, pauseBtn},
				{CloseBtn},
			},
		}
	case "mute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, unmuteBtn},
				{CloseBtn},
			},
		}
	case "unmute":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{skipBtn, stopBtn, muteBtn},
				{CloseBtn},
			},
		}
	default:
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{CloseBtn},
			},
		}
	}
}

// RecommendationsKeyboard builds an inline keyboard listing up to five
// recommended tracks. Each button's callback data encodes the short-lived
// token identifying the candidate list plus the track's index within it, so
// the callback handler can resolve the full track from a temporary cache.
func RecommendationsKeyboard(token string, titles []string) *gotdbot.ReplyMarkupInlineKeyboard {
	rows := make([][]gotdbot.InlineKeyboardButton, 0, len(titles)+1)
	for i, title := range titles {
		label := fmt.Sprintf("▶ %s", title)
		if len(label) > 40 {
			label = label[:37] + "..."
		}
		rows = append(rows, []gotdbot.InlineKeyboardButton{
			cb(label, fmt.Sprintf("%s:%s:%d", CBRecPlay, token, i)),
		})
	}
	rows = append(rows, []gotdbot.InlineKeyboardButton{cb("Close", CBRecClose)})
	return &gotdbot.ReplyMarkupInlineKeyboard{Rows: rows}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	addMeBtn := url("Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ", fmt.Sprintf("https://t.me/%s?startgroup=true", username))
	channelBtn := url("Updates", config.SupportChannel)
	groupBtn := url("Group", config.SupportGroup)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{HelpBtn},
			{channelBtn, groupBtn},
			{SourceCodeBtn},
		},
	}
}
