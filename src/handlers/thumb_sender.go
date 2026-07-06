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
	"github.com/zefronxd/TGMUSIC/src/core"
	"github.com/zefronxd/TGMUSIC/src/core/thumb"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"os"

	td "github.com/AshokShau/gotdbot"
)

// sendNowPlayingThumb generates a premium thumbnail for track and delivers it
// as a photo reply to m.  It must be called as a goroutine so it never blocks
// the caller.  All failures are silently swallowed — the text control panel
// that was already delivered remains fully functional regardless.
func sendNowPlayingThumb(c *td.Client, m *td.Message, track *utils.CachedTrack, queuePos int) {
	data := &thumb.TrackData{
		Name:     track.Name,
		Channel:  track.Channel,
		Duration: track.Duration,
		User:     track.User,
		Platform: track.Platform,
		Thumb:    track.Thumbnail,
		QueuePos: queuePos,
		Status:   "playing",
		IsVideo:  track.IsVideo,
		Views:    track.Views,
	}

	png, err := thumb.Engine.Generate(data)
	if err != nil || len(png) == 0 {
		return
	}

	// Use os.CreateTemp for a guaranteed collision-free, path-traversal-safe
	// temporary file.  TrackID is intentionally not used in the path.
	tmp, err := os.CreateTemp(config.DownloadsDir, "thumb_*.png")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(png); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}

	_, _ = m.ReplyPhoto(c, td.InputFileLocal{Path: tmpPath}, &td.SendPhotoOpts{
		Caption:     utils.NowPlayingText(track),
		ParseMode:   "HTML",
		ReplyMarkup: core.ControlButtons("play"),
	})
}
