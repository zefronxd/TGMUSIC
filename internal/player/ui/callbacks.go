/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import (
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	td "github.com/AshokShau/gotdbot"
)

// ─── Injected vc functions ───────────────────────────────────────────────────
//
// These are set from the handlers package (load.go) at startup, allowing the
// callback handler to control vc playback without importing the CGO-dependent
// vc package.  All three must be non-nil before any player_ callback fires.

// FnPause pauses the voice chat for chatID.
var FnPause func(chatID int64) (bool, error)

// FnResume resumes the voice chat for chatID.
var FnResume func(chatID int64) (bool, error)

// FnPlayNext skips to the next track in the queue for chatID.
var FnPlayNext func(c *td.Client, chatID int64) error

// ─── Handler ─────────────────────────────────────────────────────────────────

// HandlePlayerCallback is the entry point for all player_ callback queries.
// Register it with callbackquery.Prefix(core.PrefixPlayer) in load.go.
func HandlePlayerCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if FnPause == nil || FnResume == nil || FnPlayNext == nil {
		_ = cb.Answer(c, 0, true, "⚠️ Player not initialised.", "")
		return nil
	}

	data := cb.DataString()
	chatID := cb.ChatId

	// Strip "player_" prefix to get the raw command.
	cmd := strings.TrimPrefix(data, core.PrefixPlayer)

	if !cache.ChatCache.IsActive(chatID) {
		_ = cb.Answer(c, 0, true, "⚠️ No active stream.", "")
		_, _ = cb.EditMessageText(c, "⚠️ <b>No Active Stream</b>\n\nThere is no active playback.", &td.EditTextMessageOpts{
			ParseMode: "HTML",
		})
		return nil
	}

	track := cache.ChatCache.GetPlayingTrack(chatID)
	if track == nil {
		_ = cb.Answer(c, 0, true, "⚠️ No active stream.", "")
		return nil
	}

	switch cmd {

	// ── Transport ────────────────────────────────────────────────────────────

	case "pp": // pause / resume toggle
		isPaused := Manager.IsPaused(chatID)

		if isPaused {
			if _, err := FnResume(chatID); err != nil {
				_ = cb.Answer(c, 0, true, "❌ Resume failed.", "")
				return nil
			}
			Manager.OnResumed(chatID)
			_ = cb.Answer(c, 0, false, "▶ Resumed.", "")
			slog.Info("player: resumed", "chat_id", chatID, "user_id", cb.SenderUserId)
		} else {
			if _, err := FnPause(chatID); err != nil {
				_ = cb.Answer(c, 0, true, "❌ Pause failed.", "")
				return nil
			}
			Manager.OnPaused(chatID)
			_ = cb.Answer(c, 0, false, "⏸ Paused.", "")
			slog.Info("player: paused", "chat_id", chatID, "user_id", cb.SenderUserId)
		}

		// Re-render immediately after state change.
		nowPaused := Manager.IsPaused(chatID)
		elapsed := Manager.GetElapsed(chatID)
		var text string
		if nowPaused {
			text = RenderPaused(track, chatID, elapsed, "")
		} else {
			text = RenderPlayer(track, chatID, elapsed)
		}
		Manager.Invalidate(chatID)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           PlayerKeyboard(nowPaused),
		})

	case "skip":
		if err := FnPlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, true, "❌ Skip failed.", "")
			return nil
		}
		_ = cb.Answer(c, 0, false, "⏭ Skipped.", "")
		slog.Info("player: track skipped", "chat_id", chatID, "user_id", cb.SenderUserId)
		// playSong will edit this message when the next track starts.

	case "prev":
		// Previous-track is not supported by ntgcalls; restart current instead.
		_ = cb.Answer(c, 0, true, "⏮ Previous track not available.", "")

	// ── Mode controls ────────────────────────────────────────────────────────

	case "shf":
		cache.ChatCache.ShuffleQueue(chatID)
		Manager.Invalidate(chatID)
		_ = cb.Answer(c, 0, false, "🔀 Queue shuffled.", "")
		slog.Info("player: queue shuffled", "chat_id", chatID)

	case "lp":
		current := cache.ChatCache.GetLoopCount(chatID)
		next := nextLoopCount(current)
		cache.ChatCache.SetLoopCount(chatID, next)
		Manager.Invalidate(chatID)
		_ = cb.Answer(c, 0, false, fmt.Sprintf("🔁 Loop: %s", loopLabel(next)), "")
		slog.Info("player: loop changed", "chat_id", chatID, "loop", next)
		// Immediately re-render so the user sees the new loop label.
		elapsed := Manager.GetElapsed(chatID)
		text := RenderPlayer(track, chatID, elapsed)
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           PlayerKeyboard(Manager.IsPaused(chatID)),
		})

	case "que":
		// The queue panel is managed by the queue_ callback system.
		_ = cb.Answer(c, 0, true, "📜 Use /queue to view the full queue.", "")

	// ── Content features (stubs — ready for one-file extension) ─────────────

	case "lyr":
		_ = cb.Answer(c, 0, true, "🎧 Lyrics — coming soon.", "")
		slog.Debug("player: lyrics stub", "chat_id", chatID)

	case "dl":
		_ = cb.Answer(c, 0, true, "📥 Download — coming soon.", "")
		slog.Debug("player: download stub", "chat_id", chatID)

	case "vid":
		_ = cb.Answer(c, 0, true, "📺 Video toggle — coming soon.", "")
		slog.Debug("player: video stub", "chat_id", chatID)

	// ── Extras ───────────────────────────────────────────────────────────────

	case "fav":
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 0, true, "❌ Could not fetch playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 0, true, "❌ Could not create playlist.", "")
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      track.URL,
			Name:     track.Name,
			TrackID:  track.TrackID,
			Duration: track.Duration,
			Platform: track.Platform,
		}
		if err = db.Instance.AddSongToPlaylist(playlistID, song); err != nil {
			_ = cb.Answer(c, 0, true, "❌ Could not save to playlist.", "")
			return nil
		}

		playlistName := "My Playlist"
		if pl, plErr := db.Instance.GetPlaylist(playlistID); plErr == nil && pl != nil {
			playlistName = pl.Name
		}

		_ = cb.Answer(c, 0, false,
			fmt.Sprintf("❤️ Added \"%s\" to \"%s\".",
				html.EscapeString(track.Name), html.EscapeString(playlistName)), "")
		slog.Info("player: track favorited", "chat_id", chatID, "track", track.Name)

	case "cfg":
		_ = cb.Answer(c, 0, true, "⚙ Use /settings to open the settings panel.", "")

	case "cls":
		_ = cb.Answer(c, 0, false, "Closing player.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		Manager.StopProgressLoop(chatID)
		slog.Info("player: panel closed", "chat_id", chatID, "user_id", cb.SenderUserId)

	default:
		slog.Debug("player: unknown callback", "data", data)
	}

	return nil
}
