/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package dl

import (
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

const (
	// downloadCacheTTL controls how long a resolved (platform, id) -> file
	// path mapping is trusted without re-downloading, enabling instant
	// replays/duplicate requests across chats.
	downloadCacheTTL = 6 * time.Hour
	// maxDownloadRetries is the number of attempts (including the first)
	// made for a single track before giving up.
	maxDownloadRetries = 3
	retryBaseDelay     = 1 * time.Second
	// maxParallelDownloads bounds how many downloads (across all chats) run
	// at once, so parallel/prefetch downloads can't overwhelm the host's
	// CPU, disk or network.
	maxParallelDownloads = 4
	// maxCachedDownloadPaths bounds the download-path cache by entry count
	// so long-running bots don't accumulate an unbounded map of stale
	// file paths/URLs in memory.
	maxCachedDownloadPaths = 2000
)

// downloadPathCache maps a stable "platform:id:video=bool" key to a verified
// local file path or playable URL. This is the "smart cache": it survives
// across chats/plays so the same track is never downloaded twice while the
// entry is still valid.
var downloadPathCache = cache.NewBoundedCache[string](downloadCacheTTL, maxCachedDownloadPaths)

// downloadSemaphore bounds the number of concurrent real downloads
// (parallel queue prefetching + simultaneous chats) to a sane limit.
var downloadSemaphore = make(chan struct{}, maxParallelDownloads)

// inFlightCall represents a download currently in progress for a given key.
type inFlightCall struct {
	done chan struct{}
	path string
	err  error
}

// inFlightGroup deduplicates concurrent downloads: if the same track is
// requested by multiple chats (or queue slots) at the same time, only one
// real download happens and every caller shares the result.
type inFlightGroup struct {
	mu    sync.Mutex
	calls map[string]*inFlightCall
}

var downloadGroup = &inFlightGroup{calls: make(map[string]*inFlightCall)}

// do runs fn for key, coalescing concurrent callers using the same key into
// a single execution.
func (g *inFlightGroup) do(key string, fn func() (string, error)) (string, error) {
	g.mu.Lock()
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		<-call.done
		return call.path, call.err
	}

	call := &inFlightCall{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	call.path, call.err = fn()
	close(call.done)

	g.mu.Lock()
	delete(g.calls, key)
	g.mu.Unlock()

	return call.path, call.err
}

// trackCacheKey builds a stable identity for a track, independent of which
// chat/user requested it, so duplicate requests share the same cached file
// and in-flight download.
func trackCacheKey(platform, id string, video bool) string {
	return fmt.Sprintf("%s:%s:video=%t", strings.ToLower(platform), strings.TrimSpace(id), video)
}

// resolveViaManager wraps a raw provider download (fn) with smart caching,
// duplicate/in-flight coalescing, retries and post-download validation.
// It's the single choke point every real download passes through, keeping
// the musicService interface itself free of this cross-cutting logic.
func resolveViaManager(platform, id string, video bool, fn func() (string, error)) (string, error) {
	key := trackCacheKey(platform, id, video)

	if cached, ok := downloadPathCache.Get(key); ok && isPathStillValid(cached) {
		return cached, nil
	}

	return downloadGroup.do(key, func() (string, error) {
		// Re-check the cache: another caller may have just finished
		// populating it while we were waiting to acquire the group lock.
		if cached, ok := downloadPathCache.Get(key); ok && isPathStillValid(cached) {
			return cached, nil
		}

		downloadSemaphore <- struct{}{}
		defer func() { <-downloadSemaphore }()

		path, err := downloadWithRetry(video, fn)
		if err != nil {
			return "", err
		}

		downloadPathCache.Set(key, path)
		return path, nil
	})
}

// isPathStillValid re-checks a cached result before trusting it: local
// files might have been cleaned up, but remote URLs are trusted for the
// life of the cache entry.
func isPathStillValid(path string) bool {
	if path == "" {
		return false
	}
	if isRemoteURL(path) {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// isRemoteURL reports whether path is a remote URL rather than a local
// filesystem path (many providers hand back a CDN URL for ntgcalls to
// stream directly, without an actual local download).
func isRemoteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// downloadWithRetry retries a download attempt with exponential backoff,
// validating the result each time and skipping retries for errors that are
// clearly permanent (bad input, unsupported platform, etc).
func downloadWithRetry(video bool, attemptFn func() (string, error)) (string, error) {
	var lastErr error
	delay := retryBaseDelay

	for attempt := 1; attempt <= maxDownloadRetries; attempt++ {
		path, err := attemptFn()
		if err == nil {
			if verr := validateMediaFile(path, video); verr != nil {
				slog.Warn("Downloaded media failed validation, retrying", "path", path, "attempt", attempt, "error", verr)
				if !isRemoteURL(path) {
					_ = os.Remove(path)
				}
				lastErr = verr
			} else {
				return path, nil
			}
		} else {
			lastErr = err
			if !isRetryableDownloadError(err) {
				return "", err
			}
			slog.Warn("Download attempt failed, retrying", "attempt", attempt, "maxRetries", maxDownloadRetries, "error", err)
		}

		if attempt < maxDownloadRetries {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", maxDownloadRetries, lastErr)
}

// isRetryableDownloadError decides whether a failure is transient (network
// blip, rate limit, temporary API hiccup) and thus worth retrying, versus a
// permanent failure (bad URL, unsupported platform) that will never
// succeed no matter how many times it's retried.
func isRetryableDownloadError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	nonRetryable := []string{
		"invalid cached url",
		"unsupported platform",
		"the provided url is invalid",
		"invalid url",
		"missing cdn url",
		"the query is empty",
		"no video results",
		"no track found",
		"invalid or unplayable link",
		"videoid is empty",
	}
	for _, s := range nonRetryable {
		if strings.Contains(msg, s) {
			return false
		}
	}
	return true
}

// validateMediaFile makes a best-effort check that a downloaded track is
// real, playable media before it's handed off to the voice chat engine.
// Local files are checked strictly (non-empty + ffprobe-decodable); remote
// URLs are only sanity-checked for well-formedness since a full network
// probe on every track would add avoidable latency.
func validateMediaFile(path string, video bool) error {
	if path == "" {
		return errors.New("empty file path")
	}

	if isRemoteURL(path) {
		if _, err := url.ParseRequestURI(path); err != nil {
			return fmt.Errorf("invalid remote media url: %w", err)
		}
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("downloaded file is empty")
	}

	return probeMediaStream(path, video)
}

// probeMediaStream runs a fast ffprobe check to confirm the expected
// stream type is present and decodable.
func probeMediaStream(path string, video bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamType, label := "a", "audio"
	if video {
		streamType, label = "v", "video"
	}

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", streamType+":0",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffprobe validation failed: %w", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("no %s stream found in downloaded file", label)
	}
	return nil
}

// PrefetchTracks kicks off background downloads for tracks that don't yet
// have a local file, bounded by the shared download semaphore, so playback
// can start instantly once a track becomes current. onDone (if non-nil) is
// invoked from a background goroutine with the result of each download.
func PrefetchTracks(bot *td.Client, tracks []*utils.CachedTrack, onDone func(track *utils.CachedTrack, path string, err error)) {
	for _, t := range tracks {
		if t == nil || t.FilePath != "" || t.Platform == utils.Telegram {
			continue
		}

		track := t
		go func() {
			path, err := DownloadCachedTrack(track, bot)
			if err != nil {
				slog.Warn("Prefetch download failed", "track", track.Name, "platform", track.Platform, "error", err)
			}
			if onDone != nil {
				onDone(track, path, err)
			}
		}()
	}
}
