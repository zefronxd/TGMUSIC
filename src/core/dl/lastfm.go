/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package dl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/zefronxd/TGMUSIC/config"
)

// SimilarTrack is a lightweight (artist, title) pair returned by an external
// "similar tracks" metadata provider (currently Last.fm).
type SimilarTrack struct {
	Name   string
	Artist string
}

type lastfmSimilarResponse struct {
	SimilarTracks struct {
		Track []struct {
			Name   string `json:"name"`
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"track"`
	} `json:"similartracks"`
}

// GetLastfmSimilarTracks queries Last.fm's track.getSimilar endpoint for
// tracks similar to the given artist/title pair. It requires
// config.LastfmApiKey to be configured (LASTFM_API_KEY env var); otherwise it
// returns an error so callers can fall back to another recommendation source.
func GetLastfmSimilarTracks(artist, title string, limit int) ([]SimilarTrack, error) {
	if config.LastfmApiKey == "" {
		return nil, errors.New("LASTFM_API_KEY is not configured")
	}
	if artist == "" || title == "" {
		return nil, errors.New("artist and title are required")
	}
	if limit <= 0 {
		limit = 5
	}

	q := url.Values{
		"method":      {"track.getsimilar"},
		"artist":      {artist},
		"track":       {title},
		"api_key":     {config.LastfmApiKey},
		"format":      {"json"},
		"limit":       {fmt.Sprintf("%d", limit)},
		"autocorrect": {"1"},
	}
	fullURL := "https://ws.audioscrobbler.com/2.0/?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("last.fm request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm returned status %s", resp.Status)
	}

	var parsed lastfmSimilarResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode last.fm response: %w", err)
	}

	out := make([]SimilarTrack, 0, len(parsed.SimilarTracks.Track))
	for _, t := range parsed.SimilarTracks.Track {
		if t.Name == "" || t.Artist.Name == "" {
			continue
		}
		out = append(out, SimilarTrack{Name: t.Name, Artist: t.Artist.Name})
	}
	return out, nil
}
