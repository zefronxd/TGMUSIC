/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/src/utils"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	data := cb.DataString()
	if strings.HasPrefix(data, core.PrefixSettings) {
		return nil
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	if !cache.ChatCache.IsActive(chatID) {
		const msg = "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat."
		_ = cb.Answer(c, 0, false, "No active playback.", "")
		_, _ = cb.EditMessageText(c, msg, &td.EditTextMessageOpts{
			ReplyMarkup:           core.ControlButtons(""),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		const msg = "⚠️ <b>No Active Stream</b>\n\nThere is no active playback in the voice chat."
		_ = cb.Answer(c, 0, false, "No active playback.", "")
		_, _ = cb.EditMessageText(c, msg, &td.EditTextMessageOpts{
			ReplyMarkup:           core.ControlButtons(""),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
		return nil
	}

	// buildStatus prepends a status header to the shared track card.
	buildStatus := func(emoji, status string) string {
		name := html.EscapeString(currentTrack.Name)
		user := html.EscapeString(currentTrack.User)
		title := name
		if currentTrack.URL != "" {
			title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(currentTrack.URL), name)
		}
		return fmt.Sprintf(
			"%s <b>%s</b>\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"🎶 <b>%s</b>\n"+
				"⏱ <code>%s</code>\n"+
				"👤 <i>%s</i>",
			emoji, status,
			title,
			utils.SecToMin(currentTrack.Duration),
			user,
		)
	}

	actor := html.EscapeString(user.FirstName)

	switch {
	case strings.Contains(data, core.CBPlaySkip):
		if err := vc.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to skip.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Skip Failed</b>\n\nUnable to skip the current track.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true,
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "⏭ Track skipped.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, core.CBPlayStop):
		if err := vc.Calls.Stop(chatID, false); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to stop.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Stop Failed</b>\n\nUnable to stop playback.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true,
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "⏹ Stream ended.", "")
		msg := fmt.Sprintf("⏹ <b>Stream Ended</b>\n━━━━━━━━━━━━━━━━━━━━━━\n\n👤 <i>Stopped by %s</i>", actor)
		_, err := cb.EditMessageText(c, msg, &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return err

	case strings.Contains(data, core.CBPlayPause):
		if _, err = vc.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to pause.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Pause Failed</b>\n\nUnable to pause playback.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true,
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "⏸ Paused.", "")
		text := buildStatus("⏸", "Paused") + fmt.Sprintf("\n\n👤 <i>Paused by %s</i>", actor)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return nil

	case strings.Contains(data, core.CBPlayResume):
		if _, err := vc.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to resume.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Resume Failed</b>\n\nUnable to resume playback.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons("pause"), ParseMode: "HTML", DisableWebPagePreview: true,
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "▶️ Resumed.", "")
		text := buildStatus("🎵", "Now Playing") + fmt.Sprintf("\n\n👤 <i>Resumed by %s</i>", actor)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return nil

	case strings.Contains(data, core.CBPlayMute):
		if _, err := vc.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to mute.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Mute Failed</b>\n\nUnable to mute playback.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true,
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "🔇 Muted.", "")
		text := buildStatus("🔇", "Muted") + fmt.Sprintf("\n\n👤 <i>Muted by %s</i>", actor)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons("mute"), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return nil

	case strings.Contains(data, core.CBPlayUnmute):
		if _, err := vc.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to unmute.", "")
			_, _ = cb.EditMessageText(c, "❌ <b>Unmute Failed</b>\n\nUnable to unmute playback.", &td.EditTextMessageOpts{
				ReplyMarkup: core.ControlButtons("unmute"), ParseMode: "HTML",
			})
			return nil
		}
		_ = cb.Answer(c, 0, false, "🔊 Unmuted.", "")
		text := buildStatus("🎵", "Now Playing") + fmt.Sprintf("\n\n👤 <i>Unmuted by %s</i>", actor)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons("unmute"), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return nil

	case strings.Contains(data, core.CBPlayAddToList):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to fetch playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 0, false, "Unable to create playlist.", "")
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}
		if err = db.Instance.AddSongToPlaylist(playlistID, song); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to add track to playlist.", "")
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Playlist not found.", "")
			return nil
		}
		_ = cb.Answer(c, 0, false, fmt.Sprintf("Added \"%s\" to \"%s\".", song.Name, playlist.Name), "")
		return nil
	}

	// Fallback: re-render the now-playing card.
	text := buildStatus("🎵", "Now Playing")
	_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
		ReplyMarkup: core.ControlButtons("resume"), ParseMode: "HTML", DisableWebPagePreview: true,
	})
	return nil
}

func vcPlayHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if strings.Contains(data, core.CBVcPlayClose) {
		_ = cb.Answer(c, 0, false, "Closing panel.", "")
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	slog.Info("Received vcplay callback", "data", data)
	return nil
}
