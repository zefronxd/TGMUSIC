/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package status

import "time"

// Config holds tuneable parameters for the status engine.
// All fields have safe production defaults in DefaultConfig.
type Config struct {
	// AnimationEnabled controls whether dot-cycling edits are sent.
	// Set to false to send a static loading message only.
	AnimationEnabled bool

	// AnimationInterval is the delay between animation frames.
	// Keep ≥ 3 s to stay comfortably below Telegram's per-message edit rate.
	AnimationInterval time.Duration

	// MaxEdits is the maximum number of animation edits before the goroutine
	// stops cycling on its own (safety valve against indefinite animation).
	MaxEdits int

	// MaxLifetime is the absolute upper bound for a status goroutine.
	// The goroutine exits after this duration regardless of operation state.
	MaxLifetime time.Duration

	// AutoCleanup, when true, deletes the status message automatically
	// after it has been resolved (Done/Error).
	AutoCleanup bool

	// CleanupDelay is how long to wait before deleting the message when
	// AutoCleanup is enabled.
	CleanupDelay time.Duration
}

// DefaultConfig is the recommended production configuration.
var DefaultConfig = Config{
	AnimationEnabled:  true,
	AnimationInterval: 3 * time.Second,
	MaxEdits:          20,
	MaxLifetime:       5 * time.Minute,
	AutoCleanup:       false,
	CleanupDelay:      8 * time.Second,
}
