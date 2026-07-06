/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import "github.com/zefronxd/TGMUSIC/src/utils"

// PageCount returns the total number of pages for the given queue.
// The now-playing track (index 0) is always on every page; only the
// "up next" portion is paginated at PageSize tracks per page.
func PageCount(queueLen int) int {
	if queueLen <= 0 {
		return 0
	}
	queued := queueLen - 1 // exclude now-playing
	if queued <= 0 {
		return 1
	}
	return (queued + PageSize - 1) / PageSize
}

// ClampPage returns page clamped to [1, PageCount(queueLen)].
// Returns 1 when queueLen is 0.
func ClampPage(page, queueLen int) int {
	total := PageCount(queueLen)
	if total <= 0 {
		return 1
	}
	if page < 1 {
		return 1
	}
	if page > total {
		return total
	}
	return page
}

// PageSlice returns the "up next" sub-slice for the given 1-based page.
// The returned slice is a window into the sorted/filtered view — it does NOT
// include the now-playing track (tracks[0]).
func PageSlice(tracks []*utils.CachedTrack, page int) []*utils.CachedTrack {
	if len(tracks) <= 1 {
		return nil
	}
	queue := tracks[1:] // up-next portion
	start := (page - 1) * PageSize
	if start >= len(queue) {
		return nil
	}
	end := start + PageSize
	if end > len(queue) {
		end = len(queue)
	}
	return queue[start:end]
}

// QueueOffset returns the 1-based index of the first "up next" entry on page p.
func QueueOffset(page int) int {
	return (page-1)*PageSize + 1
}
