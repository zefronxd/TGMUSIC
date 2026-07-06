/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import (
	"sort"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/utils"
)

// Sort key constants.
const (
	SortDefault   = ""      // original insertion order
	SortAlpha     = "alpha" // alphabetical by name
	SortDuration  = "dur"   // ascending duration
	SortNewest    = "new"   // newest additions first (reverse insertion)
	SortOldest    = "old"   // oldest additions first (original insertion)
	SortRequester = "req"   // alphabetical by requester name
	SortSource    = "src"   // alphabetical by platform
)

// ApplySort returns the track list sorted by sortBy.
// The now-playing track (index 0) is always preserved at position 0.
// For SortDefault the original slice is returned without copying.
func ApplySort(tracks []*utils.CachedTrack, sortBy string) []*utils.CachedTrack {
	if len(tracks) <= 1 || sortBy == SortDefault {
		return tracks
	}

	// Copy the "up next" portion so we never mutate the live queue.
	tail := make([]*utils.CachedTrack, len(tracks)-1)
	copy(tail, tracks[1:])

	switch sortBy {
	case SortAlpha:
		sort.SliceStable(tail, func(i, j int) bool {
			return strings.ToLower(tail[i].Name) < strings.ToLower(tail[j].Name)
		})
	case SortDuration:
		sort.SliceStable(tail, func(i, j int) bool {
			return tail[i].Duration < tail[j].Duration
		})
	case SortNewest:
		// Reverse: newest item was appended last, so it sits at the end of the slice.
		for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
			tail[i], tail[j] = tail[j], tail[i]
		}
	case SortOldest:
		// Oldest-first is the original insertion order — nothing to do.
	case SortRequester:
		sort.SliceStable(tail, func(i, j int) bool {
			return strings.ToLower(tail[i].User) < strings.ToLower(tail[j].User)
		})
	case SortSource:
		sort.SliceStable(tail, func(i, j int) bool {
			return strings.ToLower(tail[i].Platform) < strings.ToLower(tail[j].Platform)
		})
	}

	result := make([]*utils.CachedTrack, 1, len(tracks))
	result[0] = tracks[0]
	return append(result, tail...)
}

// SortLabel returns a short, human-readable label for a sort key.
func SortLabel(sortBy string) string {
	switch sortBy {
	case SortAlpha:
		return "🔤 A–Z"
	case SortDuration:
		return "⏱ Duration"
	case SortNewest:
		return "🆕 Newest"
	case SortOldest:
		return "🕰 Oldest"
	case SortRequester:
		return "👤 Requester"
	case SortSource:
		return "🔗 Source"
	default:
		return "🔀 Default"
	}
}
