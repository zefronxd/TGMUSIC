/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package status

// Type identifies a status kind. Adding a new status requires only adding
// a new constant here and its definition in the definitions map — no other
// file needs to change.
type Type int

const (
	TypeSearching           Type = iota // 🔍 Searching Music
	TypeDownloadingAudio                // 📥 Downloading Audio
	TypeDownloadingVideo                // 📥 Downloading Video
	TypeFetchingMetadata                // 📡 Fetching Metadata
	TypeGeneratingThumbnail             // 🖼 Generating Thumbnail
	TypeJoiningVoiceChat                // 🎙 Joining Voice Chat
	TypeConnecting                      // 🔗 Connecting
	TypeBuffering                       // ⚡ Buffering
	TypeStartingPlayback                // ▶ Starting Playback
	TypeResumingPlayback                // ▶ Resuming Playback
	TypeSkippingTrack                   // ⏭ Skipping Track
	TypeLeavingVoiceChat                // 🔇 Leaving Voice Chat
	TypeClearingQueue                   // 🗑 Clearing Queue
	TypeImportingPlaylist               // 🗂 Importing Playlist
	TypeExportingPlaylist               // 📤 Exporting Playlist
	TypeUpdatingQueue                   // 🔄 Updating Queue
	TypeCachingAudio                    // 💾 Caching Audio
	TypeOptimizingStream                // ⚡ Optimizing Stream
	TypeLoadingLyrics                   // 📜 Loading Lyrics
	TypeSearchingPlaylist               // 🔍 Searching Playlist
	TypeRestoringSession                // 🔧 Restoring Session
	TypeReloadingCache                  // 🔄 Reloading Cache
)

// statusDef holds the visual definition of one status type.
type statusDef struct {
	Emoji string
	Label string
	Hint  string
}

// definitions maps every Type to its display properties.
// To add a new status: add a constant above and an entry here.
var definitions = map[Type]statusDef{
	TypeSearching:           {"🔍", "Searching Music", "Please wait\u2026"},
	TypeDownloadingAudio:    {"📥", "Downloading Audio", "Please wait\u2026"},
	TypeDownloadingVideo:    {"📥", "Downloading Video", "Please wait\u2026"},
	TypeFetchingMetadata:    {"📡", "Fetching Metadata", "Please wait\u2026"},
	TypeGeneratingThumbnail: {"🖼", "Generating Thumbnail", "Please wait\u2026"},
	TypeJoiningVoiceChat:    {"🎙", "Joining Voice Chat", "Please wait\u2026"},
	TypeConnecting:          {"🔗", "Connecting", "Please wait\u2026"},
	TypeBuffering:           {"⚡", "Buffering", "Please wait\u2026"},
	TypeStartingPlayback:    {"▶", "Starting Playback", "Please wait\u2026"},
	TypeResumingPlayback:    {"▶", "Resuming Playback", "Please wait\u2026"},
	TypeSkippingTrack:       {"⏭", "Skipping Track", "Please wait\u2026"},
	TypeLeavingVoiceChat:    {"🔇", "Leaving Voice Chat", "Please wait\u2026"},
	TypeClearingQueue:       {"🗑", "Clearing Queue", "Please wait\u2026"},
	TypeImportingPlaylist:   {"🗂", "Importing Playlist", "Please wait\u2026"},
	TypeExportingPlaylist:   {"📤", "Exporting Playlist", "Please wait\u2026"},
	TypeUpdatingQueue:       {"🔄", "Updating Queue", "Please wait\u2026"},
	TypeCachingAudio:        {"💾", "Caching Audio", "Please wait\u2026"},
	TypeOptimizingStream:    {"⚡", "Optimizing Stream", "Please wait\u2026"},
	TypeLoadingLyrics:       {"📜", "Loading Lyrics", "Please wait\u2026"},
	TypeSearchingPlaylist:   {"🔍", "Searching Playlist", "Please wait\u2026"},
	TypeRestoringSession:    {"🔧", "Restoring Session", "Please wait\u2026"},
	TypeReloadingCache:      {"🔄", "Reloading Cache", "Please wait\u2026"},
}
