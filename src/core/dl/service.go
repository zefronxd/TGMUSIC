/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package dl

import (
	"github.com/zefronxd/TGMUSIC/config"
	"github.com/zefronxd/TGMUSIC/src/core/cache"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"time"
)

const (
	// searchCacheTTL/trackInfoCacheTTL keep repeat searches and metadata
	// lookups for the same query fast without re-hitting external
	// providers, while staying short enough that results don't go stale.
	searchCacheTTL    = 5 * time.Minute
	trackInfoCacheTTL = 15 * time.Minute

	maxCachedSearches   = 500
	maxCachedTrackInfos = 1000
)

// searchResultCache and trackInfoCache are keyed by the raw query string
// (URL, search term or ID) that produced them. They're shared across every
// DownloaderWrapper instance so identical requests from different chats
// reuse the same result instead of hitting YouTube/Spotify/the API gateway
// again.
var (
	searchResultCache = cache.NewBoundedCache[utils.PlatformTracks](searchCacheTTL, maxCachedSearches)
	trackInfoCache    = cache.NewBoundedCache[utils.TrackInfo](trackInfoCacheTTL, maxCachedTrackInfos)
)

// musicService defines a standard interface for interacting with various music services.
// This allows for a unified approach to handling different platforms like YouTube, Spotify, etc.
type musicService interface {
	// isValid determines if the service can handle the given query.
	isValid() bool
	// getInfo retrieves metadata for a track or playlist.
	getInfo() (utils.PlatformTracks, error)
	// search queries the service for a track.
	search() (utils.PlatformTracks, error)
	// getTrack fetches detailed information for a single track.
	getTrack() (utils.TrackInfo, error)
	// downloadTrack handles the download of a track.
	downloadTrack(trackInfo utils.TrackInfo, video bool) (string, error)
}

// DownloaderWrapper provides a unified interface for music service interactions.
type DownloaderWrapper struct {
	service musicService
	query   string
}

// NewDownloaderWrapper selects the appropriate musicService based on the query format or configuration defaults.
func NewDownloaderWrapper(query string) *DownloaderWrapper {
	yt := newYouTubeData(query)
	api := newApiData(query)
	direct := newDirectLink(query)

	var chosen musicService
	if yt.isValid() {
		chosen = yt
	} else if api.isValid() {
		chosen = api
	} else if direct.isValid() {
		chosen = direct
	} else {
		switch config.DefaultService {
		case "spotify":
			chosen = api
		default:
			chosen = yt
		}
	}

	return &DownloaderWrapper{
		service: chosen,
		query:   query,
	}
}

// IsValid checks if the underlying service can handle the query.
func (d *DownloaderWrapper) IsValid() bool {
	return d.service != nil && d.service.isValid()
}

// GetInfo retrieves metadata by delegating the call to the wrapped service.
func (d *DownloaderWrapper) GetInfo() (utils.PlatformTracks, error) {
	return d.service.getInfo()
}

// Search performs a search by delegating the call to the wrapped service,
// transparently caching results per query so repeated searches (common for
// popular tracks, or multiple chats searching the same thing) skip the
// external API entirely for a few minutes.
func (d *DownloaderWrapper) Search() (utils.PlatformTracks, error) {
	key := "search:" + d.query
	if cached, ok := searchResultCache.Get(key); ok {
		return cached, nil
	}

	result, err := d.service.search()
	if err != nil {
		return result, err
	}

	searchResultCache.Set(key, result)
	return result, nil
}

// GetTrack retrieves detailed track information by delegating the call to
// the wrapped service, transparently caching the metadata per query.
func (d *DownloaderWrapper) GetTrack() (utils.TrackInfo, error) {
	key := "track:" + d.query
	if cached, ok := trackInfoCache.Get(key); ok {
		return cached, nil
	}

	info, err := d.service.getTrack()
	if err != nil {
		return info, err
	}

	trackInfoCache.Set(key, info)
	return info, nil
}

// DownloadTrack downloads a track by delegating the call to the wrapped service.
// It returns the file path of the downloaded track or an error if the download fails.
func (d *DownloaderWrapper) DownloadTrack(info utils.TrackInfo, video bool) (string, error) {
	return d.service.downloadTrack(info, video)
}
