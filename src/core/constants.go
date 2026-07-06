/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package core

// Callback prefix strings used with callbackquery.Prefix() for handler routing.
// Keep in sync with the CB* data constants below.
const (
	PrefixPlay     = "play_"
	PrefixVcPlay   = "vcplay_"
	PrefixSettings = "settings_"
	PrefixHelp     = "help_"
	PrefixQueue    = "queue_"
	PrefixPlayer   = "player_"
	PrefixRec      = "rec_"
)

// Callback data constants for all inline keyboard buttons.
// Use these instead of raw string literals so that renaming a callback
// data value is a single-site change and typos are caught at compile time.
const (
	// Playback-control callbacks (prefix "play_").
	CBPlaySkip      = "play_skip"
	CBPlayStop      = "play_stop"
	CBPlayPause     = "play_pause"
	CBPlayResume    = "play_resume"
	CBPlayMute      = "play_mute"
	CBPlayUnmute    = "play_unmute"
	CBPlayAddToList = "play_add_to_list"

	// Panel-control callbacks (prefix "vcplay_").
	CBVcPlayClose = "vcplay_close"

	// Settings callbacks (prefix "settings_").
	CBSettingsMain   = "settings_main"
	CBSettingsPlay   = "settings_play"
	CBSettingsDelete = "settings_delete"
	CBSettingsAdmin  = "settings_admin"
	CBSettingsLang   = "settings_lang"

	// Help-menu callbacks (prefix "help_").
	CBHelpAll      = "help_all"
	CBHelpBack     = "help_back"
	CBHelpUser     = "help_user"
	CBHelpAdmin    = "help_admin"
	CBHelpOwner    = "help_owner"
	CBHelpDevs     = "help_devs"
	CBHelpPlaylist = "help_playlist"

	// Now-Playing Player callbacks (prefix "player_").
	CBPlayerPauseResume = "player_pp"   // toggle pause / resume
	CBPlayerSkip        = "player_skip" // skip to next track
	CBPlayerPrev        = "player_prev" // previous track (stub)
	CBPlayerShuffle     = "player_shf"  // shuffle queue
	CBPlayerLoop        = "player_lp"   // cycle loop count
	CBPlayerQueue       = "player_que"  // open queue panel
	CBPlayerLyrics      = "player_lyr"  // lyrics (future)
	CBPlayerDownload    = "player_dl"   // download (future)
	CBPlayerVideo       = "player_vid"  // video toggle (future)
	CBPlayerFav         = "player_fav"  // add to favourites
	CBPlayerSettings    = "player_cfg"  // settings shortcut
	CBPlayerClose       = "player_cls"  // close player panel

	// Queue-management callbacks (prefix "queue_").
	// Navigation.
	CBQueuePage    = "queue_pg"  // data: queue_pg:N
	CBQueueStats   = "queue_sts" // show stats alert
	CBQueueRefresh = "queue_ref"
	CBQueueClose   = "queue_cls"
	CBQueueBack    = "queue_bk"

	// Sort / filter menus.
	CBQueueSortMenu   = "queue_srtm"
	CBQueueFilterMenu = "queue_fltm"
	CBQueueSort       = "queue_srt" // data: queue_srt:TYPE
	CBQueueFilter     = "queue_flt" // data: queue_flt:TYPE

	// Track-level actions.
	CBQueueDetail   = "queue_dtl" // data: queue_dtl:N (1-based page position)
	CBQueueRemove   = "queue_rm"  // data: queue_rm:N  (raw queue index, 1-based up-next)
	CBQueueMoveUp   = "queue_mu"  // data: queue_mu:N
	CBQueueMoveDown = "queue_md"  // data: queue_md:N
	CBQueueJump     = "queue_jmp" // data: queue_jmp:N
	CBQueueFav      = "queue_fav" // data: queue_fav:N
	CBQueueShare    = "queue_shr" // data: queue_shr:N

	// Admin bulk actions.
	CBQueueClear   = "queue_clr"
	CBQueueShuffle = "queue_shf"
	CBQueueReverse = "queue_rev"
	CBQueueLock    = "queue_lck"

	// Recommendation callbacks (prefix "rec_").
	CBRecPlay  = "rec_play"  // data: rec_play:token:idx
	CBRecClose = "rec_close" // data: rec_close
)
