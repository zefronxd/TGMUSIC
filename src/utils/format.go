/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package utils

import (
	"fmt"
	"html"
	"strings"
)

// PlatformLabel returns a human-readable display name for a platform constant.
func PlatformLabel(platform string) string {
	switch platform {
	case YouTube:
		return "YouTube"
	case Spotify:
		return "Spotify"
	case JioSaavn:
		return "JioSaavn"
	case Apple:
		return "Apple Music"
	case SoundCloud:
		return "SoundCloud"
	case Telegram:
		return "Telegram"
	case Deezer:
		return "Deezer"
	case Gaana:
		return "Gaana"
	case Tidal:
		return "Tidal"
	case MXPlayer:
		return "MX Player"
	case Twitch:
		return "Twitch"
	case TwitchClip:
		return "Twitch Clip"
	case Kick:
		return "Kick"
	case KickClip:
		return "Kick Clip"
	case DirectLink:
		return "Direct Link"
	default:
		if platform == "" {
			return "Unknown"
		}
		// Fallback: replace underscores and capitalise each word.
		words := strings.Fields(strings.ReplaceAll(platform, "_", " "))
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		return strings.Join(words, " ")
	}
}

// PlatformEmoji returns a small emoji that represents the given platform.
func PlatformEmoji(platform string) string {
	switch platform {
	case YouTube:
		return "▶"
	case Spotify:
		return "🟢"
	case JioSaavn:
		return "🎵"
	case Apple:
		return "🍎"
	case SoundCloud:
		return "🔶"
	case Telegram:
		return "📎"
	case Deezer:
		return "🎧"
	case Tidal:
		return "🌊"
	case Twitch, TwitchClip:
		return "🟣"
	case Kick, KickClip:
		return "🟩"
	default:
		return "🎵"
	}
}

// NowPlayingText builds the standard "Now Playing" HTML message for a track.
// All track fields are HTML-escaped internally; callers must not pre-escape them.
func NowPlayingText(track *CachedTrack) string {
	name := html.EscapeString(track.Name)
	user := html.EscapeString(track.User)

	title := name
	if track.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(track.URL), name)
	}

	var sb strings.Builder
	sb.WriteString("🎵 <b>Now Playing</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprintf(&sb, "🎶 <b>%s</b>", title)

	if track.Channel != "" {
		fmt.Fprintf(&sb, "\n🎙 <i>%s</i>", html.EscapeString(track.Channel))
	}

	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "⏱ <code>%s</code>", SecToMin(track.Duration))

	if track.Views != "" && track.Views != "0" {
		fmt.Fprintf(&sb, "  ·  👁 %s", html.EscapeString(track.Views))
	}

	if track.Platform != "" && track.Platform != Telegram {
		fmt.Fprintf(&sb, "\n%s <b>%s</b>", PlatformEmoji(track.Platform), PlatformLabel(track.Platform))
	}

	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "👤 <i>%s</i>", user)
	return sb.String()
}

// QueueAddedText builds the "Added to Queue" HTML message shown when a track is
// enqueued behind an already-playing track.
// All track fields are HTML-escaped internally; callers must not pre-escape them.
func QueueAddedText(track *CachedTrack, position int) string {
	name := html.EscapeString(track.Name)
	user := html.EscapeString(track.User)

	title := name
	if track.URL != "" {
		title = fmt.Sprintf("<a href='%s'>%s</a>", html.EscapeString(track.URL), name)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "✅ <b>Added to Queue</b>  ·  <code>#%d</code>\n", position)
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Fprintf(&sb, "🎶 <b>%s</b>\n\n", title)
	fmt.Fprintf(&sb, "⏱ <code>%s</code>", SecToMin(track.Duration))

	if track.Platform != "" && track.Platform != Telegram {
		fmt.Fprintf(&sb, "  ·  %s %s", PlatformEmoji(track.Platform), PlatformLabel(track.Platform))
	}

	fmt.Fprintf(&sb, "\n👤 <i>%s</i>", user)
	return sb.String()
}
