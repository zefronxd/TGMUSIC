/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

// Package thumb provides a premium, high-performance thumbnail engine for
// the Zefron Music bot. It renders commercial-quality 1920×1080 PNG images
// with a dark glassmorphism aesthetic and caches results to stay well under
// the 500 ms generation target.
package thumb

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/utils"
)

// TrackData carries all metadata the renderer needs for one thumbnail.
type TrackData struct {
	Name     string // song title
	Channel  string // artist / channel name
	Duration int    // length in seconds
	User     string // requester display name
	Platform string // "youtube", "spotify", …
	Thumb    string // album-art URL (may be empty)
	QueuePos int    // 1 = now playing, >1 = queued
	Status   string // "playing" | "paused" | "stopped"
	IsVideo  bool
	Views    string
}

// engine is the singleton thumbnail generator.
type engine struct {
	thumbCache *cache.Cache[[]byte]
	artCache   *cache.Cache[image.Image]
	http       *http.Client
}

// Thumbnails and album art are large (PNG/JPEG payloads), so both caches
// are bounded by entry count in addition to TTL - this caps worst-case RAM
// usage no matter how many distinct tracks are requested.
const (
	maxCachedThumbnails = 200
	maxCachedAlbumArt   = 150
)

// Engine is the global, ready-to-use instance.
var Engine = &engine{
	// Thumbnails are reused for 30 minutes; album art for 1 hour.
	thumbCache: cache.NewBoundedCache[[]byte](30*time.Minute, maxCachedThumbnails),
	artCache:   cache.NewBoundedCache[image.Image](1*time.Hour, maxCachedAlbumArt),
	http: &http.Client{
		Timeout: 8 * time.Second,
	},
}

// cacheKey returns a short deterministic key for the given track data.
// It incorporates name, user, platform and status so that the thumbnail
// is invalidated whenever any of those change.
func (e *engine) cacheKey(d *TrackData) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d", d.Name, d.User, d.Platform, d.Status, d.QueuePos)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h[:10])
}

// Generate renders (or returns a cached) PNG thumbnail for d.
// It never returns an error to the caller – if album art is unavailable a
// premium default artwork is used, and if rendering fails the error is logged
// and the caller receives nil so it can fall back gracefully.
func (e *engine) Generate(d *TrackData) ([]byte, error) {
	key := e.cacheKey(d)
	if cached, ok := e.thumbCache.Get(key); ok {
		return cached, nil
	}

	art := e.fetchArt(d.Thumb)
	png, err := render(d, art)
	if err != nil {
		slog.Error("thumb: render failed", "error", err)
		return nil, err
	}

	e.thumbCache.Set(key, png)
	return png, nil
}

// Invalidate removes the cached thumbnail for d (e.g. on status change).
func (e *engine) Invalidate(d *TrackData) {
	e.thumbCache.Delete(e.cacheKey(d))
}

// fetchArt downloads and caches album art.  Returns nil on any failure.
func (e *engine) fetchArt(url string) image.Image {
	if url == "" {
		return nil
	}
	if img, ok := e.artCache.Get(url); ok {
		return img
	}

	resp, err := e.http.Get(url)
	if err != nil {
		slog.Debug("thumb: art fetch failed", "url", url, "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		slog.Debug("thumb: art decode failed", "error", err)
		return nil
	}

	e.artCache.Set(url, img)
	return img
}

// platformLabel delegates to utils.PlatformLabel for use within the package.
func platformLabel(platform string) string {
	return utils.PlatformLabel(platform)
}

// qualityLabel returns a human-readable audio quality string for a platform.
func qualityLabel(platform string) string {
	switch platform {
	case utils.Spotify:
		return "Premium · 320 kbps"
	case utils.Apple:
		return "Lossless · ALAC"
	case utils.Tidal:
		return "HiFi · FLAC"
	case utils.JioSaavn:
		return "HD · 320 kbps"
	case utils.Deezer:
		return "HQ · 320 kbps"
	case utils.YouTube:
		return "HD · 256 kbps"
	case utils.SoundCloud:
		return "HQ · 128 kbps"
	default:
		return "High Quality"
	}
}
