/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import (
	"sync"
	"time"

	"github.com/zefronxd/TGMUSIC/src/utils"
	td "github.com/AshokShau/gotdbot"
)

// playerSession tracks the live state of the now-playing panel for one chat.
// All fields after creation are accessed under mu.
type playerSession struct {
	mu sync.Mutex

	// msg is the Telegram message that IS the player panel.
	// Stored so the progress loop can edit it without needing a callback ctx.
	msg *td.Message

	// track is the currently playing track, mirroring the queue's track[0].
	track *utils.CachedTrack

	// startedAt is the wall-clock moment when the current track began.
	startedAt time.Time

	// pausedAt is the moment pause was requested (zero when not paused).
	pausedAt time.Time

	// isPaused indicates whether playback is currently paused.
	isPaused bool

	// lastHash is the SHA-256 prefix of the last rendered text,
	// used to skip no-op Telegram API edits.
	lastHash string

	// stopCh is closed to terminate the background progress goroutine.
	stopCh chan struct{}
}

// elapsed returns the elapsed playback seconds using the stored wall-clock
// times.  This is a low-fidelity fallback; the progress loop overwrites it
// with the precise value from vc.Calls.PlayedTime.
func (s *playerSession) elapsed() int {
	if s.startedAt.IsZero() {
		return 0
	}
	if s.isPaused && !s.pausedAt.IsZero() {
		return int(s.pausedAt.Sub(s.startedAt).Seconds())
	}
	return int(time.Since(s.startedAt).Seconds())
}

// PlayerManager holds per-chat player sessions and coordinates message
// reuse, progress ticking, and state changes across concurrent goroutines.
//
// Use the package-level Manager singleton — do not create additional instances.
type PlayerManager struct {
	mu       sync.RWMutex
	sessions map[int64]*playerSession
}

// Manager is the global singleton.  It is safe for concurrent use from any
// number of goroutines and supports thousands of simultaneous chat sessions.
var Manager = &PlayerManager{
	sessions: make(map[int64]*playerSession),
}

// get returns the session for chatID, or nil if none exists.
func (m *PlayerManager) get(chatID int64) *playerSession {
	m.mu.RLock()
	s := m.sessions[chatID]
	m.mu.RUnlock()
	return s
}

// getOrCreate returns the session for chatID, creating it if absent.
func (m *PlayerManager) getOrCreate(chatID int64) *playerSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[chatID]; ok {
		return s
	}
	s := &playerSession{}
	m.sessions[chatID] = s
	return s
}

// SetMessage records the Telegram message that IS the player panel for a
// chat.  The progress loop uses this message to edit progress updates in-place.
func (m *PlayerManager) SetMessage(chatID int64, msg *td.Message) {
	s := m.getOrCreate(chatID)
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

// GetMessage returns the player panel message for chatID, or nil if none has
// been registered yet.
func (m *PlayerManager) GetMessage(chatID int64) *td.Message {
	s := m.get(chatID)
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.msg
}

// OnTrackStarted resets the elapsed timer and stores the track metadata when
// a new track begins.  Call this immediately before sending / editing the
// player panel message for the new track.
func (m *PlayerManager) OnTrackStarted(chatID int64, track *utils.CachedTrack) {
	s := m.getOrCreate(chatID)
	s.mu.Lock()
	s.track = track
	s.startedAt = time.Now()
	s.pausedAt = time.Time{}
	s.isPaused = false
	s.lastHash = "" // force re-render on next tick
	s.mu.Unlock()
}

// OnPaused records the pause instant so that elapsed time is frozen correctly.
// Safe to call multiple times (idempotent).
func (m *PlayerManager) OnPaused(chatID int64) {
	s := m.get(chatID)
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.isPaused {
		s.isPaused = true
		s.pausedAt = time.Now()
		s.lastHash = ""
	}
	s.mu.Unlock()
}

// OnResumed advances startedAt by the paused duration so elapsed stays
// continuous.  Safe to call multiple times (idempotent).
func (m *PlayerManager) OnResumed(chatID int64) {
	s := m.get(chatID)
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.isPaused && !s.pausedAt.IsZero() && !s.startedAt.IsZero() {
		pausedFor := time.Since(s.pausedAt)
		s.startedAt = s.startedAt.Add(pausedFor)
		s.isPaused = false
		s.pausedAt = time.Time{}
		s.lastHash = ""
	}
	s.mu.Unlock()
}

// Invalidate clears the dedup hash so the next progress tick forces an edit
// even if the rendered text appears unchanged (e.g. after a loop toggle).
func (m *PlayerManager) Invalidate(chatID int64) {
	s := m.get(chatID)
	if s == nil {
		return
	}
	s.mu.Lock()
	s.lastHash = ""
	s.mu.Unlock()
}

// IsPaused reports whether playback is currently paused for chatID.
func (m *PlayerManager) IsPaused(chatID int64) bool {
	s := m.get(chatID)
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isPaused
}

// GetElapsed returns the wall-clock elapsed seconds for chatID.
// Callers that have access to vc.Calls.PlayedTime should prefer that for
// accuracy; this is the fallback used by the renderer.
func (m *PlayerManager) GetElapsed(chatID int64) int {
	s := m.get(chatID)
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.elapsed()
}

// GetTrack returns the currently registered track for a chat, or nil.
func (m *PlayerManager) GetTrack(chatID int64) *utils.CachedTrack {
	s := m.get(chatID)
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.track
}

// Clear removes all state for a chat and terminates the progress goroutine.
// Call on stop, end, or any event that permanently ends the session for a chat.
func (m *PlayerManager) Clear(chatID int64) {
	m.mu.Lock()
	s := m.sessions[chatID]
	delete(m.sessions, chatID)
	m.mu.Unlock()

	if s == nil {
		return
	}
	s.mu.Lock()
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	}
	s.mu.Unlock()
}
