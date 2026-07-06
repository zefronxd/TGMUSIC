/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package vc

/*
#cgo linux LDFLAGS: -L . -lntgcalls -lm -lz
#cgo darwin LDFLAGS: -L . -lntgcalls -lc++ -lz -lbz2 -liconv -framework AVFoundation -framework AudioToolbox -framework CoreAudio -framework QuartzCore -framework CoreMedia -framework VideoToolbox -framework AppKit -framework Metal -framework MetalKit -framework OpenGL -framework IOSurface -framework ScreenCaptureKit

// Currently is supported only dynamically linked library on Windows due to
// https://github.com/golang/go/issues/63903
#cgo windows LDFLAGS: -L. -lntgcalls
#include "ntgcalls/ntgcalls.h"
#include "glibc_compatibility.h"
*/
import "C"

import (
	"github.com/zefronxd/TGMUSIC/config"
	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"github.com/zefronxd/TGMUSIC/src/core/dl"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"github.com/zefronxd/TGMUSIC/src/vc/ntgcalls"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

// getClientIndex selects an assistant client index (0-based) for a given chat.
func (c *TelegramCalls) getClientIndex(chatID int64) (int, error) {
	c.mu.RLock()
	totalClients := len(c.assistants)
	c.mu.RUnlock()

	if totalClients == 0 {
		return -1, errors.New("no assistant clients are available")
	}

	assignedIndex, err := db.Instance.GetAssistant(chatID)
	if err != nil {
		logger.Warn("Failed to get assigned assistant from DB", "chat_id", chatID, "error", err)
		assignedIndex = -1
	}

	if assignedIndex >= 0 && assignedIndex < totalClients {
		return assignedIndex, nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalClients)))
	if err != nil {
		logger.Warn("Failed to generate random assistant index, using 0", "error", err)
		newClientIndex := 0
		if assignedIndex == -1 && chatID != 0 {
			if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
				logger.Warn("Failed to persist assistant assignment", "chat_id", chatID, "error", err)
			}
		}
		return newClientIndex, nil
	}

	newClientIndex := int(n.Int64())
	if chatID != 0 {
		if _, err := db.Instance.AssignAssistant(chatID, newClientIndex); err != nil {
			logger.Warn("Failed to persist assistant assignment", "chat_id", chatID, "error", err)
		}
	}
	return newClientIndex, nil
}

// GetGroupAssistant retrieves the assistant and its index for a given chat.
func (c *TelegramCalls) GetGroupAssistant(chatID int64) (*Assistant, int, error) {
	clientIndex, err := c.getClientIndex(chatID)
	if err != nil {
		return nil, -1, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	call, ok := c.assistants[clientIndex]
	if !ok {
		return nil, -1, fmt.Errorf("no ntgcalls instance for assistant index %d", clientIndex)
	}
	return call, clientIndex, nil
}

// playSong downloads and plays a single song from the queue.
// It posts a status message to the chat, then updates it once playback begins.
func (c *TelegramCalls) playSong(bot *td.Client, chatID int64, song *utils.CachedTrack) error {
	reply, err := bot.SendTextMessage(chatID, fmt.Sprintf("⏳ Downloading · %s", song.Name), nil)
	if err != nil {
		logger.Warn("Failed to send download status message", "chat_id", chatID, "error", err)
		return err
	}

	if err = c.downloadAndPrepareSong(bot, song, reply); err != nil {
		return c.PlayNext(bot, chatID)
	}

	if err = c.PlayMedia(bot, chatID, song.FilePath, song.IsVideo, ""); err != nil {
		_, _ = reply.EditText(bot, err.Error(), &td.EditTextMessageOpts{
			ParseMode: "HTML", DisableWebPagePreview: true,
		})
		return nil
	}

	if song.Duration == 0 {
		song.Duration = utils.GetMediaDuration(song.FilePath)
	}

	_, err = reply.EditText(bot, utils.NowPlayingText(song), &td.EditTextMessageOpts{
		ReplyMarkup:           core.ControlButtons("play"),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	if err != nil {
		logger.Warn("Failed to update now-playing message", "chat_id", chatID, "error", err)
	}

	c.prefetchUpcoming(bot, chatID)
	return nil
}

// prefetchUpcoming downloads the next couple of queued tracks in the
// background while the current one plays, so playback can advance
// instantly once a track becomes current instead of waiting on a fresh
// download.
func (c *TelegramCalls) prefetchUpcoming(bot *td.Client, chatID int64) {
	queue := cache.ChatCache.GetQueue(chatID)
	if len(queue) < 2 {
		return
	}

	upcoming := queue[1:]
	if len(upcoming) > 2 {
		upcoming = upcoming[:2]
	}

	dl.PrefetchTracks(bot, upcoming, func(track *utils.CachedTrack, path string, err error) {
		if err == nil && path != "" {
			cache.ChatCache.SetTrackFilePath(chatID, track.TrackID, path)
		}
	})
}

// Stop halts media playback in a voice chat and clears the chat's queue.
func (c *TelegramCalls) Stop(chatId int64, banned bool) error {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return err
	}

	cache.ChatCache.ClearChat(chatId)
	if err = call.stopCall(chatId, banned); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		logger.Warn("Failed to stop call", "chat_id", chatId, "assistant_index", index, "error", err)
		return fmt.Errorf("stop call: %w", err)
	}
	return nil
}

// Pause temporarily stops media playback in a voice chat.
func (c *TelegramCalls) Pause(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Pause(chatId)
	if err != nil {
		logger.Warn("Failed to pause call", "chat_id", chatId, "assistant_index", index, "error", err)
		return res, fmt.Errorf("pause: %w", err)
	}
	return res, nil
}

// Resume continues a paused media playback in a voice chat.
func (c *TelegramCalls) Resume(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Resume(chatId)
	if err != nil {
		logger.Warn("Failed to resume call", "chat_id", chatId, "assistant_index", index, "error", err)
		return res, fmt.Errorf("resume: %w", err)
	}
	return res, nil
}

// Mute silences the media playback in a voice chat.
func (c *TelegramCalls) Mute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.Mute(chatId)
	if err != nil {
		logger.Warn("Failed to mute call", "chat_id", chatId, "assistant_index", index, "error", err)
		return res, fmt.Errorf("mute: %w", err)
	}
	return res, nil
}

// Unmute restores the audio of a muted media playback in a voice chat.
func (c *TelegramCalls) Unmute(chatId int64) (bool, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return false, err
	}

	res, err := call.binding.UnMute(chatId)
	if err != nil {
		logger.Warn("Failed to unmute call", "chat_id", chatId, "assistant_index", index, "error", err)
		return res, fmt.Errorf("unmute: %w", err)
	}
	return res, nil
}

// PlayedTime retrieves the elapsed time (seconds) of the current playback.
func (c *TelegramCalls) PlayedTime(chatId int64) (uint64, error) {
	call, index, err := c.GetGroupAssistant(chatId)
	if err != nil {
		return 0, err
	}

	t, err := call.binding.Time(chatId, 0)
	if err != nil {
		logger.Warn("Failed to get played time", "chat_id", chatId, "assistant_index", index, "error", err)
		return 0, fmt.Errorf("get played time: %w", err)
	}
	return t, nil
}

// SeekStream jumps to a specific time offset in the current media stream.
func (c *TelegramCalls) SeekStream(bot *td.Client, chatID int64, filePath string, toSeek, duration int, isVideo bool) error {
	if toSeek < 0 || duration <= 0 {
		return errors.New("invalid seek position or duration")
	}

	// Input seeking (-ss before -i) is always used: it's fast (keyframe
	// based) for both local files and URLs, and avoids opening the same
	// source twice with a duplicate -i flag.
	ffmpegParams := fmt.Sprintf("-ss %d -to %d", toSeek, duration)

	return c.PlayMedia(bot, chatID, filePath, isVideo, ffmpegParams)
}

// ChangeSpeed modifies the playback speed of the current stream.
func (c *TelegramCalls) ChangeSpeed(bot *td.Client, chatID int64, speed float64) error {
	if speed < 0.5 || speed > 4.0 {
		return errors.New("speed must be between 0.5 and 4.0")
	}

	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		return errors.New("the bot isn't streaming in the voice chat")
	}

	videoPTS := 1 / speed

	var audioFilterBuilder strings.Builder
	remaining := speed
	for remaining > 2.0 {
		audioFilterBuilder.WriteString("atempo=2.0,")
		remaining /= 2.0
	}
	for remaining < 0.5 {
		audioFilterBuilder.WriteString("atempo=0.5,")
		remaining /= 0.5
	}
	audioFilterBuilder.WriteString(fmt.Sprintf("atempo=%f,", remaining))
	// Keep loudness/clipping safety even when the tempo is changed.
	audioFilterBuilder.WriteString(defaultAudioFilters)

	ffmpegFilters := fmt.Sprintf("-filter:v setpts=%f*PTS -filter:a %s", videoPTS, audioFilterBuilder.String())
	return c.PlayMedia(bot, chatID, playingSong.FilePath, playingSong.IsVideo, ffmpegFilters)
}

// RegisterHandlers sets up the event handlers for the voice call client.
func (c *TelegramCalls) RegisterHandlers(client *td.Client) {
	c.startAutoLeave(context.Background(), client)

	for _, call := range c.assistants {
		call.OnStreamEnd(func(chatID int64, streamType ntgcalls.StreamType, device ntgcalls.StreamDevice) {
			if streamType == ntgcalls.VideoStream {
				return
			}
			if err := c.PlayNext(client, chatID); err != nil {
				call.App.Logger.Warnf("[OnStreamEnd] Failed to play next song: %v", err)
			}
		})

		if _, err := call.App.SendMessage(client.Me.Usernames.EditableUsername, "/start"); err != nil {
			call.App.Logger.Warnf("Failed to ping bot on startup: %v", err)
		}

		if _, err := call.App.SendMessage(config.LoggerId, "Userbot started."); err != nil {
			call.App.Logger.Warnf("Failed to send startup log: %v", err)
		}
	}
}
