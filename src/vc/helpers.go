/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package vc

import (
	"context"
	"fmt"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/dl"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"os/exec"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
	"github.com/amarnathcjd/gogram/telegram"
)

// downloadAndPrepareSong downloads a track if it has no local file path yet.
// Returns an error (and edits the status reply) when the download fails.
func (c *TelegramCalls) downloadAndPrepareSong(bot *td.Client, song *utils.CachedTrack, reply *td.Message) error {
	if song.FilePath != "" {
		return nil
	}

	dlPath, err := dl.DownloadCachedTrack(song, bot)
	song.FilePath = dlPath
	if err != nil || song.FilePath == "" {
		_, _ = reply.EditText(bot, "⚠️ Download failed. Skipping track…", nil)
		return err
	}
	return nil
}

// PlayNext plays the next song in the queue, honouring the loop counter.
// When the queue is empty it stops the call and notifies the chat.
func (c *TelegramCalls) PlayNext(bot *td.Client, chatID int64) error {
	loop := cache.ChatCache.GetLoopCount(chatID)
	if loop > 0 {
		cache.ChatCache.SetLoopCount(chatID, loop-1)
		if current := cache.ChatCache.GetPlayingTrack(chatID); current != nil {
			return c.playSong(bot, chatID, current)
		}
	}

	if next := cache.ChatCache.GetUpcomingTrack(chatID); next != nil {
		cache.ChatCache.RemoveCurrentSong(chatID)
		return c.playSong(bot, chatID, next)
	}

	cache.ChatCache.RemoveCurrentSong(chatID)
	return c.handleNoSong(bot, chatID)
}

// RecommendHook, when set (by handlers.LoadModules via its init()), is
// invoked once a chat's queue empties so "similar songs" suggestions can be
// posted. Wired this way to avoid an import cycle: handlers already imports
// vc, so vc cannot import handlers back directly.
var RecommendHook func(bot *td.Client, chatID int64, lastTrack *utils.CachedTrack)

// handleNoSong stops playback and notifies the chat that the queue is empty.
func (c *TelegramCalls) handleNoSong(bot *td.Client, chatID int64) error {
	lastTrack := cache.ChatCache.GetPlayingTrack(chatID)
	_ = c.Stop(chatID, false)
	_, _ = bot.SendTextMessage(chatID, "🎵 Queue finished. Add more songs with /play.", nil)

	if RecommendHook != nil && lastTrack != nil {
		go RecommendHook(bot, chatID, lastTrack)
	}
	return nil
}

// handleFlood pauses execution for short flood-wait errors (<= 5 s).
// Returns false when the wait is too long or not a flood error.
func handleFlood(err error) bool {
	wait := telegram.GetFloodWait(err)
	if wait <= 0 {
		return false
	}
	if wait > 5 {
		logger.Warn("Flood wait too long, skipping sleep", "seconds", wait)
		return false
	}
	logger.Warn("Flood wait, sleeping", "seconds", wait)
	time.Sleep(time.Duration(wait+1) * time.Second)
	return true
}

// getVideoDimensions uses ffprobe to return the width and height of a video file.
func getVideoDimensions(filePath string) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		filePath,
	)

	out, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to get video dimensions", "file", filePath, "error", err)
		return 0, 0
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		logger.Warn("Unexpected ffprobe dimension output", "file", filePath, "output", string(out))
		return 0, 0
	}

	width, _ := strconv.Atoi(parts[0])
	height, _ := strconv.Atoi(parts[1])
	return width, height
}

// UpdateMembership updates the cached membership status for a user in a chat.
func (c *TelegramCalls) UpdateMembership(chatId, userId int64, status td.ChatMemberStatus) {
	if c.statusCache == nil {
		return
	}
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)
	c.statusCache.Set(cacheKey, status)
	logger.Info("Membership cache updated", "chat_id", chatId, "user_id", userId)
}

// UpdateInviteLink stores or removes the cached invite link for a chat.
func (c *TelegramCalls) UpdateInviteLink(chatId int64, link string) {
	cacheKey := strconv.FormatInt(chatId, 10)
	if link == "" {
		c.inviteCache.Delete(cacheKey)
		return
	}
	c.inviteCache.Set(cacheKey, link)
}
