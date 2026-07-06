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
	"strconv"
	"strings"
	"time"

	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"github.com/zefronxd/TGMUSIC/src/core/dl"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// recCandidates holds short-lived candidate lists keyed by a per-chat token
// so recommendation callback buttons can resolve the full track without
// needing to re-fetch metadata (and without bloating callback_data, which
// Telegram limits to 64 bytes).
var recCandidates = cache.NewCache[[]utils.MusicTrack](15 * time.Minute)

const maxRecommendations = 5

func init() {
	vc.RecommendHook = TriggerRecommendations
}

// recTrackKey builds a stable dedup/history key for a track.
func recTrackKey(platform, id string) string {
	return platform + ":" + id
}

// TriggerRecommendations is invoked (as a background goroutine) once the
// queue for a chat empties, right after the last track finished playing. It
// gathers "similar song" candidates from the most fitting metadata source
// for the last track's platform, filters out anything already suggested to
// this chat, and posts an inline "play similar" keyboard.
func TriggerRecommendations(bot *td.Client, chatID int64, lastTrack *utils.CachedTrack) {
	if lastTrack == nil {
		return
	}

	candidates, err := gatherRecommendations(lastTrack)
	if err != nil || len(candidates) == 0 {
		return
	}

	history := db.Instance.GetRecommendHistory(chatID)
	seen := make(map[string]bool, len(history))
	for _, k := range history {
		seen[k] = true
	}

	fresh := make([]utils.MusicTrack, 0, maxRecommendations)
	newKeys := make([]string, 0, maxRecommendations)
	for _, t := range candidates {
		key := recTrackKey(t.Platform, t.Id)
		if seen[key] {
			continue
		}
		seen[key] = true
		fresh = append(fresh, t)
		newKeys = append(newKeys, key)
		if len(fresh) >= maxRecommendations {
			break
		}
	}

	if len(fresh) == 0 {
		return
	}

	token := fmt.Sprintf("%d_%d", chatID, time.Now().Unix())
	recCandidates.Set(token, fresh)

	titles := make([]string, len(fresh))
	for i, t := range fresh {
		titles[i] = t.Title
	}

	var sb strings.Builder
	sb.WriteString("🎧 <b>Up Next: You Might Also Like</b>\n━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("Based on <i>%s</i>, here are some similar tracks:\n", htmlEscape(lastTrack.Name)))

	_, _ = bot.SendTextMessage(chatID, sb.String(), &td.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		ReplyMarkup:           core.RecommendationsKeyboard(token, titles),
	})

	_ = db.Instance.AddRecommendedTracks(chatID, newKeys...)
}

// gatherRecommendations resolves "similar" candidate tracks for the given
// seed track, using YouTube's native Mix/radio playlist for YouTube-sourced
// tracks, and Last.fm's track.getSimilar metadata (resolved into playable
// tracks via the normal search pipeline) for every other platform.
func gatherRecommendations(seed *utils.CachedTrack) ([]utils.MusicTrack, error) {
	if seed.Platform == utils.YouTube && seed.TrackID != "" {
		tracks, err := dl.RelatedYouTubeTracks(seed.TrackID, maxRecommendations*2)
		if err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}

	artist := seed.Channel
	title := seed.Name
	if artist == "" {
		// Fall back to splitting "Artist - Title" style names.
		if parts := strings.SplitN(title, "-", 2); len(parts) == 2 {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
		}
	}

	similar, err := dl.GetLastfmSimilarTracks(artist, title, maxRecommendations*2)
	if err != nil || len(similar) == 0 {
		return nil, err
	}

	resolved := make([]utils.MusicTrack, 0, len(similar))
	for _, s := range similar {
		query := fmt.Sprintf("%s %s", s.Artist, s.Name)
		wrapper := dl.NewDownloaderWrapper(query)
		result, err := wrapper.Search()
		if err != nil || len(result.Results) == 0 {
			continue
		}
		resolved = append(resolved, result.Results[0])
		if len(resolved) >= maxRecommendations*2 {
			break
		}
	}
	return resolved, nil
}

// recommendCallbackHandler handles taps on the "▶ <track>" / "Close" buttons
// posted by TriggerRecommendations.
func recommendCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if data == core.CBRecClose {
		_ = cb.Answer(c, 0, false, "", "")
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return td.EndGroups
	}

	rest := strings.TrimPrefix(data, core.CBRecPlay+":")
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		_ = cb.Answer(c, 0, true, "This suggestion has expired.", "")
		return td.EndGroups
	}
	token, idxStr := parts[0], parts[1]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		_ = cb.Answer(c, 0, true, "This suggestion has expired.", "")
		return td.EndGroups
	}

	tracks, ok := recCandidates.Get(token)
	if !ok || idx < 0 || idx >= len(tracks) {
		_ = cb.Answer(c, 0, true, "This suggestion has expired. Try /play instead.", "")
		return td.EndGroups
	}
	song := tracks[idx]

	chatID := cb.ChatId
	if _track := cache.ChatCache.GetTrackIfExists(chatID, song.Id); _track != nil {
		_ = cb.Answer(c, 0, true, "Already in the queue.", "")
		return td.EndGroups
	}

	_ = cb.Answer(c, 0, false, fmt.Sprintf("Adding %s…", song.Title), "")

	userName := "Recommendation"
	if u, err := c.GetUser(cb.SenderUserId); err == nil {
		userName = u.FirstName
	}

	saveCache := utils.CachedTrack{
		URL: song.Url, Name: song.Title, User: userName, UserID: cb.SenderUserId,
		Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration,
		Channel: song.Channel, Views: song.Views, Platform: song.Platform,
	}

	qLen := cache.ChatCache.AddSong(chatID, &saveCache)
	if qLen > 1 {
		dl.PrefetchTracks(c, []*utils.CachedTrack{&saveCache}, func(track *utils.CachedTrack, path string, err error) {
			if err == nil && path != "" {
				cache.ChatCache.SetTrackFilePath(chatID, track.TrackID, path)
			}
		})
		_, err := cb.EditMessageText(c, utils.QueueAddedText(&saveCache, qLen), &td.EditTextMessageOpts{
			ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return err
	}

	dlResult, err := dl.DownloadCachedTrack(&saveCache, c)
	if err != nil {
		cache.ChatCache.RemoveCurrentSong(chatID)
		_, err = cb.EditMessageText(c, fmt.Sprintf("❌ <b>Download Failed</b>\n\n<code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	saveCache.FilePath = dlResult

	if err := vc.Calls.PlayMedia(c, chatID, saveCache.FilePath, false, ""); err != nil {
		cache.ChatCache.RemoveCurrentSong(chatID)
		_, err = cb.EditMessageText(c, err.Error(), &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
		return err
	}

	_, err = cb.EditMessageText(c, utils.NowPlayingText(&saveCache), &td.EditTextMessageOpts{
		ParseMode: "HTML", ReplyMarkup: core.ControlButtons("play"), DisableWebPagePreview: true,
	})
	return err
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(s)
}
