/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package playerui

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/zefronxd/TGMUSIC/src/core/cache"
	td "github.com/AshokShau/gotdbot"
)

// PlayedTimeFn is injected from the package that owns vc.Calls (handlers or vc),
// allowing the progress loop to query live playback time without importing the
// CGO-dependent vc package.  Set this before starting any progress loop.
type PlayedTimeFn func(chatID int64) (int, error)

const (
	// progressInterval is how often the progress loop wakes to consider an edit.
	progressInterval = 5 * time.Second

	// minEditGap is the minimum wall-clock gap between two consecutive Telegram
	// edits for the same message, honouring Telegram's flood-wait thresholds.
	minEditGap = 4 * time.Second
)

// StartProgressLoop launches (or restarts) the background goroutine that
// polls playback time and edits the player panel message every few seconds.
//
// fn must be non-nil and must not import the vc package directly —
// pass a closure such as:
//
//	func(id int64) (int, error) { t, err := vc.Calls.PlayedTime(id); return int(t), err }
func (m *PlayerManager) StartProgressLoop(c *td.Client, chatID int64, fn PlayedTimeFn) {
	s := m.getOrCreate(chatID)

	s.mu.Lock()
	// Terminate any existing loop goroutine before starting a new one.
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
		default:
			close(s.stopCh)
		}
	}
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	s.mu.Unlock()

	go m.progressLoop(c, chatID, fn, stopCh)
}

// StopProgressLoop terminates the background progress goroutine for chatID.
// The player panel message is left intact; call this on pause or when the
// panel should freeze (the full Clear method removes all state).
func (m *PlayerManager) StopProgressLoop(chatID int64) {
	s := m.get(chatID)
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
		s.stopCh = nil
	}
	s.mu.Unlock()
}

// progressLoop is the background worker that edits the player panel message.
// It exits when stopCh is closed, when the chat is no longer active, or when
// the player message disappears from the Manager.
func (m *PlayerManager) progressLoop(c *td.Client, chatID int64, fn PlayedTimeFn, stopCh <-chan struct{}) {
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()

	var lastEditAt time.Time

	slog.Debug("player: progress loop started", "chat_id", chatID)

	for {
		select {
		case <-stopCh:
			slog.Debug("player: progress loop stopped", "chat_id", chatID)
			return
		case <-ticker.C:
		}

		// Respect flood-wait by enforcing a minimum inter-edit gap.
		if time.Since(lastEditAt) < minEditGap {
			continue
		}

		s := m.get(chatID)
		if s == nil {
			slog.Debug("player: session gone, stopping loop", "chat_id", chatID)
			return
		}

		s.mu.Lock()
		msg := s.msg
		track := s.track
		isPaused := s.isPaused
		s.mu.Unlock()

		if msg == nil || track == nil {
			continue
		}

		// Do not tick progress while paused — time is frozen.
		if isPaused {
			continue
		}

		// Exit cleanly once the chat queue empties (stream ended).
		if !cache.ChatCache.IsActive(chatID) {
			slog.Debug("player: chat no longer active, exiting loop", "chat_id", chatID)
			return
		}

		// Obtain the precise elapsed time from the vc layer.
		elapsedSecs := 0
		if fn != nil {
			if t, err := fn(chatID); err == nil {
				elapsedSecs = t
			}
		}

		text := RenderPlayer(track, chatID, elapsedSecs)
		hash := textHash(text)

		s.mu.Lock()
		lastHash := s.lastHash
		s.mu.Unlock()

		// Skip if nothing changed.
		if hash == lastHash {
			continue
		}

		_, err := msg.EditText(c, text, &td.EditTextMessageOpts{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           PlayerKeyboard(false),
		})
		if err != nil {
			// Transient errors (no-diff, brief flood) are normal — log and continue.
			slog.Debug("player: progress edit failed", "chat_id", chatID, "error", err)
			continue
		}

		s.mu.Lock()
		s.lastHash = hash
		s.mu.Unlock()

		lastEditAt = time.Now()
		slog.Debug("player: progress updated", "chat_id", chatID, "elapsed_secs", elapsedSecs)
	}
}

// textHash returns a short deterministic fingerprint of text used for change
// detection.  It is NOT a security primitive — collision probability is fine
// for the deduplication use-case here.
func textHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h[:8])
}
