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
	"time"
)

const (
	lyricsCacheTTL  = 24 * time.Hour
	maxCachedLyrics = 500
)

// lyricsCache stores fetched lyrics keyed by a track identity (e.g.
// "platform:id"). Lyrics rarely change once published, so entries are kept
// much longer than search/metadata results, but are still bounded and will
// expire automatically. No lyrics provider is wired up yet (the /lyrics UI
// is a stub), but this cache is ready to be used the moment one is added -
// callers should use GetCachedLyrics/SetCachedLyrics rather than reaching
// into a new cache instance.
var lyricsCache = cache.NewBoundedCache[string](lyricsCacheTTL, maxCachedLyrics)

// GetCachedLyrics returns previously cached lyrics for key, if present and
// not expired.
func GetCachedLyrics(key string) (string, bool) {
	return lyricsCache.Get(key)
}

// SetCachedLyrics stores lyrics for key using the default lyrics TTL.
func SetCachedLyrics(key, lyrics string) {
	lyricsCache.Set(key, lyrics)
}
