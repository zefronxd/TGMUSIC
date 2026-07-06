/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import "sync"

// chatState holds the per-chat display state for the interactive queue viewer.
// It is kept entirely in-memory; all state is transient and resets on restart.
type chatState struct {
	mu         sync.RWMutex
	page       int            // current page, 1-based
	sortBy     string         // active sort key (SortDefault … SortSource)
	filterBy   string         // active filter key (FilterAll … FilterAudio)
	locked     bool           // true = queue is locked; only admins may add tracks
	queueHash  string         // last observed queue fingerprint for cache invalidation
	textCache  map[int]string // page number → rendered page text
	statsAlert string         // cached stats alert text
}

// QueueManager manages per-chat queue display state across any number of
// simultaneous groups. All exported methods are safe for concurrent use.
type QueueManager struct {
	mu     sync.RWMutex
	states map[int64]*chatState
}

// Manager is the package-level singleton used by the callback handler and
// the command handler.
var Manager = &QueueManager{
	states: make(map[int64]*chatState),
}

// getOrCreate returns the chatState for chatID, creating it if absent.
// Caller must NOT hold qm.mu when calling.
func (qm *QueueManager) getOrCreate(chatID int64) *chatState {
	qm.mu.RLock()
	st, ok := qm.states[chatID]
	qm.mu.RUnlock()
	if ok {
		return st
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()
	// Double-checked locking.
	if st, ok = qm.states[chatID]; ok {
		return st
	}
	st = &chatState{
		page:      1,
		textCache: make(map[int]string),
	}
	qm.states[chatID] = st
	return st
}

// invalidateLocked clears the page text cache and stats alert.
// Caller must hold st.mu (write).
func invalidateLocked(st *chatState) {
	st.textCache = make(map[int]string)
	st.statsAlert = ""
}

// Invalidate drops all cached rendered pages for chatID.
// Call this whenever the underlying queue changes.
func (qm *QueueManager) Invalidate(chatID int64) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	invalidateLocked(st)
	st.mu.Unlock()
}

// checkHash compares the current queue fingerprint against the cached one.
// If they differ the text cache is cleared and the stored hash is updated.
// Caller must NOT hold st.mu.
func (qm *QueueManager) checkHash(chatID int64, hash string) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.queueHash != hash {
		invalidateLocked(st)
		st.queueHash = hash
	}
}

// GetPage returns the current page for chatID, defaulting to 1.
func (qm *QueueManager) GetPage(chatID int64) int {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.page < 1 {
		return 1
	}
	return st.page
}

// SetPage stores the current page for chatID.
func (qm *QueueManager) SetPage(chatID int64, page int) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	st.page = page
	st.mu.Unlock()
}

// GetSort returns the active sort key for chatID.
func (qm *QueueManager) GetSort(chatID int64) string {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.sortBy
}

// SetSort stores the sort key and invalidates the page cache.
func (qm *QueueManager) SetSort(chatID int64, sortBy string) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	st.sortBy = sortBy
	invalidateLocked(st)
	st.mu.Unlock()
}

// GetFilter returns the active filter key for chatID.
func (qm *QueueManager) GetFilter(chatID int64) string {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.filterBy
}

// SetFilter stores the filter key and invalidates the page cache.
func (qm *QueueManager) SetFilter(chatID int64, filterBy string) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	st.filterBy = filterBy
	invalidateLocked(st)
	st.mu.Unlock()
}

// IsLocked reports whether the queue for chatID is locked.
func (qm *QueueManager) IsLocked(chatID int64) bool {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.locked
}

// ToggleLock flips the lock state for chatID and returns the new state.
func (qm *QueueManager) ToggleLock(chatID int64) bool {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.locked = !st.locked
	return st.locked
}

// GetCachedText returns a previously rendered page text, or "" on a miss.
func (qm *QueueManager) GetCachedText(chatID int64, page int) string {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.textCache[page]
}

// SetCachedText stores rendered page text.
func (qm *QueueManager) SetCachedText(chatID int64, page int, text string) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	st.textCache[page] = text
	st.mu.Unlock()
}

// GetCachedStats returns the cached stats alert string, or "".
func (qm *QueueManager) GetCachedStats(chatID int64) string {
	st := qm.getOrCreate(chatID)
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.statsAlert
}

// SetCachedStats stores the stats alert string.
func (qm *QueueManager) SetCachedStats(chatID int64, text string) {
	st := qm.getOrCreate(chatID)
	st.mu.Lock()
	st.statsAlert = text
	st.mu.Unlock()
}

// Remove drops the state entry for chatID, reclaiming memory after the
// queue is cleared or the stream ends.
func (qm *QueueManager) Remove(chatID int64) {
	qm.mu.Lock()
	delete(qm.states, chatID)
	qm.mu.Unlock()
}
