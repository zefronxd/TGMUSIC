/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

// Package status provides a premium animated loading and status message system
// for Zefron Music.  It wraps a Telegram message with a dot-cycling animation
// goroutine and exposes an EditText method that is drop-in compatible with the
// (*td.Message).EditText signature used throughout the handlers package.
//
// Adding a new status type requires only two changes: a constant in types.go
// and an entry in the definitions map.  No other files need to be touched.
package status

import (
	"log/slog"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// Updater is satisfied by both *td.Message and *Status.  Handler functions
// accept this interface so they work unchanged whether they receive a plain
// Telegram message or an animated status wrapper.
type Updater interface {
	EditText(c *td.Client, text string, opts *td.EditTextMessageOpts) (*td.Message, error)
}

// Status wraps a Telegram message with an animated loading state.
// Its EditText method matches the signature of (*td.Message).EditText exactly,
// so callers need no special handling beyond the initial New() call.
//
// All edits to the underlying Telegram message (animation frames and the final
// resolution) are serialised behind mu + resolved, preventing a late animation
// tick from overwriting the resolved success/error text.
type Status struct {
	msg *td.Message
	c   *td.Client
	d   statusDef
	cfg Config

	// mu serialises every call to s.msg.EditText so that an in-flight
	// animation tick cannot land after the caller's resolution edit.
	mu sync.Mutex
	// resolved is set to true inside mu before the resolution edit; animate()
	// checks it (also inside mu) and exits immediately if it is true.
	resolved bool

	stopOnce sync.Once
	done     chan struct{}
}

// New sends the initial premium loading message to the chat and launches the
// animation goroutine.  The caller replaces its m.ReplyText(…) call with this
// and uses the returned *Status wherever an Updater (or *td.Message) is expected.
func New(c *td.Client, m *td.Message, t Type, cfg ...Config) (*Status, error) {
	d, ok := definitions[t]
	if !ok {
		d = statusDef{"⏳", "Loading", "Please wait\u2026"}
	}

	active := DefaultConfig
	if len(cfg) > 0 {
		active = cfg[0]
		// Normalise to prevent zero/negative durations causing immediate exit
		// or a panic in time.NewTicker / time.NewTimer.
		if active.AnimationInterval <= 0 {
			active.AnimationInterval = DefaultConfig.AnimationInterval
		}
		if active.MaxEdits <= 0 {
			active.MaxEdits = DefaultConfig.MaxEdits
		}
		if active.MaxLifetime <= 0 {
			active.MaxLifetime = DefaultConfig.MaxLifetime
		}
		if active.CleanupDelay <= 0 {
			active.CleanupDelay = DefaultConfig.CleanupDelay
		}
	}

	msg, err := m.ReplyText(c, renderLoading(d, 0), &td.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	if err != nil {
		return nil, err
	}

	slog.Debug("status: started", "type", t, "chat_id", m.ChatId, "msg_id", msg.Id)

	s := &Status{
		msg:  msg,
		c:    c,
		d:    d,
		cfg:  active,
		done: make(chan struct{}),
	}

	if active.AnimationEnabled {
		go s.animate()
	}

	return s, nil
}

// EditText stops the animation and replaces the loading message with text.
// The signature matches (*td.Message).EditText so Status satisfies the Updater
// interface and can be passed to any handler function accepting Updater.
//
// Resolution edit and animation ticks are serialised via mu+resolved, so the
// caller's text is guaranteed to be the last edit on the message.
func (s *Status) EditText(c *td.Client, text string, opts *td.EditTextMessageOpts) (*td.Message, error) {
	// Signal the goroutine to stop looping (fast path; it may already be past
	// the select when we acquire mu below, which is why resolved is also checked).
	s.stop()

	if opts == nil {
		opts = &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true}
	}

	s.mu.Lock()
	s.resolved = true
	edited, err := s.msg.EditText(c, text, opts)
	s.mu.Unlock()

	if err != nil {
		slog.Warn("status: EditText failed", "chat_id", s.msg.ChatId, "error", err)
		return nil, err
	}

	slog.Debug("status: resolved", "chat_id", s.msg.ChatId, "msg_id", s.msg.Id)

	if s.cfg.AutoCleanup {
		go s.scheduleDelete(c, s.cfg.CleanupDelay)
	}

	return edited, nil
}

// Msg returns the underlying Telegram message.  Use only when the raw message
// is required (e.g. reading ChatId / Id); prefer EditText for updates.
func (s *Status) Msg() *td.Message {
	return s.msg
}

// Stop cancels the animation goroutine without editing the message.
// Safe to call multiple times.
func (s *Status) Stop() {
	s.stop()
}

// stop closes the done channel exactly once via sync.Once.
func (s *Status) stop() {
	s.stopOnce.Do(func() {
		close(s.done)
		slog.Debug("status: animation stopped", "chat_id", s.msg.ChatId)
	})
}

// animate runs the dot-cycling loop.  It exits when stop() is called,
// MaxLifetime elapses, or MaxEdits is exhausted.
//
// Every actual Telegram edit is performed while holding mu.  If resolved is
// already true when the lock is acquired the goroutine exits immediately,
// preventing a late tick from overwriting the caller's resolution text.
func (s *Status) animate() {
	ticker := time.NewTicker(s.cfg.AnimationInterval)
	defer ticker.Stop()

	deadline := time.NewTimer(s.cfg.MaxLifetime)
	defer deadline.Stop()

	frame := 1
	edits := 0

	for {
		select {
		case <-s.done:
			return

		case <-deadline.C:
			slog.Info("status: max lifetime reached, stopping animation",
				"chat_id", s.msg.ChatId)
			s.stop()
			return

		case <-ticker.C:
			if edits >= s.cfg.MaxEdits {
				slog.Debug("status: max edits reached, stopping animation",
					"chat_id", s.msg.ChatId)
				s.stop()
				return
			}

			// Serialise with EditText: if already resolved, do not overwrite.
			s.mu.Lock()
			if s.resolved {
				s.mu.Unlock()
				return
			}
			_, err := s.msg.EditText(s.c, renderLoading(s.d, frame), &td.EditTextMessageOpts{
				ParseMode:             "HTML",
				DisableWebPagePreview: true,
			})
			s.mu.Unlock()

			if err != nil {
				// Transient edit errors (no-diff, brief flood) are normal.
				slog.Debug("status: animation edit skipped", "error", err,
					"chat_id", s.msg.ChatId)
			}

			frame++
			edits++
		}
	}
}

// scheduleDelete deletes the status message after delay.
func (s *Status) scheduleDelete(c *td.Client, delay time.Duration) {
	time.Sleep(delay)
	if err := c.DeleteMessages(s.msg.ChatId, []int64{s.msg.Id}, nil); err != nil {
		slog.Debug("status: auto-delete failed (may already be gone)", "error", err)
	}
}
