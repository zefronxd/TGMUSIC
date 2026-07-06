/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import (
	"testing"

	"github.com/zefronxd/TGMUSIC/src/utils"
)

// makeTrack is a test helper that builds a minimal CachedTrack.
func makeTrack(name, platform string, isVideo bool, userID int64) *utils.CachedTrack {
	return &utils.CachedTrack{
		Name:     name,
		TrackID:  name, // unique enough for tests
		Platform: platform,
		IsVideo:  isVideo,
		UserID:   userID,
		User:     "tester",
		Duration: 180,
	}
}

// ─── ApplyFilter ─────────────────────────────────────────────────────────────

// TestApplyFilter_AllFilter verifies that FilterAll returns the original slice unchanged.
func TestApplyFilter_AllFilter(t *testing.T) {
	tracks := []*utils.CachedTrack{
		makeTrack("now", utils.YouTube, false, 1),
		makeTrack("yt1", utils.YouTube, false, 2),
		makeTrack("sp1", utils.Spotify, false, 3),
	}
	got := ApplyFilter(tracks, FilterAll)
	if len(got) != 3 {
		t.Fatalf("FilterAll: want 3 tracks, got %d", len(got))
	}
}

// TestApplyFilter_PreservesNowPlayingWhenItDoesNotMatch checks the critical invariant:
// index 0 (now-playing) is always kept even when it does NOT match the active filter.
func TestApplyFilter_PreservesNowPlayingWhenItDoesNotMatch(t *testing.T) {
	nowPlaying := makeTrack("now-spotify", utils.Spotify, false, 1) // will NOT match YouTube filter
	tracks := []*utils.CachedTrack{
		nowPlaying,
		makeTrack("yt1", utils.YouTube, false, 2),
		makeTrack("sp1", utils.Spotify, false, 3),
		makeTrack("yt2", utils.YouTube, false, 4),
	}

	got := ApplyFilter(tracks, FilterYouTube)

	// Invariant: index 0 must always be the original now-playing track.
	if got[0] != nowPlaying {
		t.Fatalf("now-playing track was replaced or removed by filter")
	}
	// Only YouTube up-next tracks should remain.
	if len(got) != 3 { // nowPlaying + yt1 + yt2
		t.Fatalf("want 3 tracks (nowPlaying+2 YT), got %d", len(got))
	}
	for i, tr := range got[1:] {
		if tr.Platform != utils.YouTube {
			t.Errorf("up-next[%d] platform %q does not match YouTube filter", i, tr.Platform)
		}
	}
}

// TestApplyFilter_PreservesNowPlayingWhenItMatches verifies behaviour when now-playing
// also satisfies the filter — it must still sit at index 0 and not appear twice.
func TestApplyFilter_PreservesNowPlayingWhenItMatches(t *testing.T) {
	nowPlaying := makeTrack("now-yt", utils.YouTube, false, 1)
	tracks := []*utils.CachedTrack{
		nowPlaying,
		makeTrack("yt1", utils.YouTube, false, 2),
		makeTrack("sp1", utils.Spotify, false, 3),
	}

	got := ApplyFilter(tracks, FilterYouTube)

	if got[0] != nowPlaying {
		t.Fatalf("now-playing track changed after filter")
	}
	// nowPlaying + yt1; sp1 filtered out.
	if len(got) != 2 {
		t.Fatalf("want 2 tracks, got %d", len(got))
	}
}

// TestApplyFilter_VideoFilter confirms that video/audio split works and does not
// affect the now-playing track regardless of its own media type.
func TestApplyFilter_VideoFilter(t *testing.T) {
	nowAudio := makeTrack("now-audio", utils.YouTube, false, 1)
	tracks := []*utils.CachedTrack{
		nowAudio,
		makeTrack("v1", utils.YouTube, true, 2),
		makeTrack("a1", utils.YouTube, false, 3),
		makeTrack("v2", utils.Spotify, true, 4),
	}

	got := ApplyFilter(tracks, FilterVideo)

	if got[0] != nowAudio {
		t.Fatal("now-playing removed by video filter")
	}
	// nowAudio + v1 + v2
	if len(got) != 3 {
		t.Fatalf("want 3 (now+2 videos), got %d", len(got))
	}
	for _, tr := range got[1:] {
		if !tr.IsVideo {
			t.Errorf("non-video track %q leaked through video filter", tr.Name)
		}
	}
}

// TestApplyFilter_EmptyQueue returns an empty slice without panicking.
func TestApplyFilter_EmptyQueue(t *testing.T) {
	got := ApplyFilter([]*utils.CachedTrack{}, FilterYouTube)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d", len(got))
	}
}

// TestApplyFilter_OnlyNowPlaying returns a one-element slice with just now-playing
// when no up-next tracks match.
func TestApplyFilter_OnlyNowPlaying(t *testing.T) {
	nowPlaying := makeTrack("now-sp", utils.Spotify, false, 1)
	tracks := []*utils.CachedTrack{
		nowPlaying,
		makeTrack("yt1", utils.YouTube, false, 2),
		makeTrack("yt2", utils.YouTube, false, 3),
	}

	got := ApplyFilter(tracks, FilterSpotify) // only Spotify up-next — none exist

	if len(got) != 1 {
		t.Fatalf("want 1 (now-playing only), got %d", len(got))
	}
	if got[0] != nowPlaying {
		t.Fatal("now-playing replaced")
	}
}

// ─── PageSlice after filter ───────────────────────────────────────────────────

// TestPageSlice_AfterFilter verifies that PageSlice correctly skips index 0 (now-playing)
// in a filtered view where now-playing does not match the filter.
func TestPageSlice_AfterFilter(t *testing.T) {
	nowPlaying := makeTrack("now-sp", utils.Spotify, false, 1)
	tracks := []*utils.CachedTrack{
		nowPlaying,
		makeTrack("yt1", utils.YouTube, false, 2),
		makeTrack("yt2", utils.YouTube, false, 3),
		makeTrack("sp1", utils.Spotify, false, 4),
	}

	filtered := ApplyFilter(tracks, FilterYouTube) // [nowSp, yt1, yt2]
	page := PageSlice(filtered, 1)

	// Up-next portion should be yt1, yt2 — not now-playing.
	if len(page) != 2 {
		t.Fatalf("want 2 up-next tracks after filter, got %d", len(page))
	}
	for _, tr := range page {
		if tr == nowPlaying {
			t.Error("now-playing track appeared in up-next page slice")
		}
	}
}

// ─── PageCount after filter ───────────────────────────────────────────────────

func TestPageCount_AfterFilter(t *testing.T) {
	// 1 now-playing + 10 YouTube + 5 Spotify = 16 total
	tracks := make([]*utils.CachedTrack, 0, 16)
	tracks = append(tracks, makeTrack("now", utils.Spotify, false, 1))
	for i := 0; i < 10; i++ {
		tracks = append(tracks, makeTrack("yt", utils.YouTube, false, int64(i+2)))
	}
	for i := 0; i < 5; i++ {
		tracks = append(tracks, makeTrack("sp", utils.Spotify, false, int64(i+12)))
	}

	filtered := ApplyFilter(tracks, FilterYouTube) // [now(sp)] + 10 YT = 11 items
	// queued = 10 → 1 page of PageSize=10
	if pc := PageCount(len(filtered)); pc != 1 {
		t.Errorf("want 1 page, got %d", pc)
	}

	// 21 YouTube up-next → 3 pages
	bigTracks := make([]*utils.CachedTrack, 0, 22)
	bigTracks = append(bigTracks, makeTrack("now", utils.Spotify, false, 1))
	for i := 0; i < 21; i++ {
		bigTracks = append(bigTracks, makeTrack("yt", utils.YouTube, false, int64(i+2)))
	}
	filtered2 := ApplyFilter(bigTracks, FilterYouTube) // [now(sp)] + 21 YT = 22 items, queued=21
	if pc := PageCount(len(filtered2)); pc != 3 {
		t.Errorf("want 3 pages for 21 queued, got %d", pc)
	}
}
