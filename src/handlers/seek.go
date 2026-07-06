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
	"strconv"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/vc"

	td "github.com/AshokShau/gotdbot"
)

// seekHandler handles the /seek command.
func seekHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}
	chatID := m.ChatId

	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return err
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.ReplyText(c, "⚠️ <b>No Active Stream</b>\n\nThe bot is not streaming in the voice chat.", replyOpts)
		return err
	}

	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "⏩ <b>Seek</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"<b>Usage:</b> <code>/seek [seconds]</code>\n"+
			"<b>Example:</b> <code>/seek 30</code>\n\n"+
			"<i>Minimum seek: 10 seconds</i>", replyOpts)
		return nil
	}

	seekTime, err := strconv.Atoi(args)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ <b>Invalid Value</b>\n\nPlease provide a valid number of seconds.", replyOpts)
		return nil
	}

	if seekTime < 10 {
		_, _ = m.ReplyText(c, "❌ <b>Too Short</b>\n\nMinimum seek time is <code>10</code> seconds.", replyOpts)
		return nil
	}

	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "❌ <b>Error</b>\n\nFailed to fetch the current stream position.", replyOpts)
		return nil
	}

	toSeek := int(currDur) + seekTime
	if toSeek >= playingSong.Duration {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Seek Limit Exceeded</b>\n\nMaximum position for this track is <code>%s</code>.", utils.SecToMin(playingSong.Duration)), replyOpts)
		return nil
	}

	if err = vc.Calls.SeekStream(
		c,
		chatID,
		playingSong.FilePath,
		toSeek,
		playingSong.Duration,
		playingSong.IsVideo,
	); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("❌ <b>Seek Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return nil
	}

	_, _ = m.ReplyText(c, fmt.Sprintf("⏩ <b>Seeked</b>  +<code>%s</code>  →  <code>%s</code>\n\n👤 <i>%s</i>", utils.SecToMin(seekTime), utils.SecToMin(toSeek), firstName(c, m)), replyOpts)
	return nil
}
