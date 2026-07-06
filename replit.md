# TgMusicBot / Zefron Music

A production Telegram music-streaming bot written in Go.  
Streams audio (and video) into Telegram voice chats from YouTube, Spotify, Apple Music, SoundCloud, Deezer, JioSaavn, Tidal, Twitch, Kick, and more.

## Stack

- **Language:** Go 1.25 (module `github.com/zefronxd/TGMUSIC`)
- **Telegram client:** `github.com/AshokShau/gotdbot` (TDLib wrapper)
- **Voice chat:** `ntgcalls` (CGo bindings, requires `libtdjson.so.1.8.65`)
- **Database:** MongoDB (`go.mongodb.org/mongo-driver/v2`)
- **Image generation:** `github.com/fogleman/gg` + `golang.org/x/image`

## Project structure

```
config/          environment variable loading
main.go          entry point
setup_ntgcalls.go  CGo / ntgcalls bootstrap
src/
  core/
    buttons.go        inline keyboard builders
    constants.go      callback-data constants
    cache/            generic TTL cache + music queue
    db/               MongoDB helpers
    dl/               downloaders (yt-dlp, Spotify, API)
    thumb/            *** premium thumbnail engine ***
  handlers/     command & callback handlers
  utils/        shared models, formatting, durations
  vc/           voice chat management (ntgcalls)
  init.go       module bootstrapper
```

## Premium Thumbnail Engine (`src/core/thumb/`)

Generates **1920 × 1080 PNG** thumbnails with a premium dark/glassmorphism aesthetic:

| File | Responsibility |
|------|---------------|
| `engine.go` | `Engine` singleton, `Generate()`, dual TTL cache (thumbnails 30 min, album art 1 hr) |
| `renderer.go` | Drawing pipeline: background gradient → ambient glows → album art → glass panel → track info → spectrum bars → vignette |
| `fonts.go` | Embedded Go fonts (goregular / gobold / goitalic) via `golang.org/x/image/font/opentype` |

**Integration:** `src/handlers/thumb_sender.go` fires `sendNowPlayingThumb` as a goroutine from `handleSingleTrack` after the text control panel is already delivered to the user, so thumbnail generation never blocks playback.

**Cache keys** incorporate: song name · requester · platform · playback status · queue position.

## Running locally

The bot requires `ntgcalls` native libraries and `libtdjson.so.1.8.65` — not available in Replit's default environment. It is designed to be deployed to a Linux server or container.

### Required environment variables

| Variable | Description |
|----------|-------------|
| `API_ID` | Telegram API ID |
| `API_HASH` | Telegram API hash |
| `TOKEN` | Bot token |
| `MONGO_URI` | MongoDB connection string |
| `OWNER_ID` | Owner Telegram user ID |
| `LOGGER_ID` | Log group/channel ID |
| `STRING1`–`STRING10` | Pyrogram / Gogram session strings |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `DEFAULT_SERVICE` | `youtube` | Default search platform (`youtube` \| `spotify`) |
| `API_URL` | `https://api.onegrab.fun` | External media API |
| `API_KEY` | — | API key for the above |
| `DL_BOT_TOKEN` | — | Separate bot token for downloads |
| `SONG_DURATION_LIMIT` | `3600` | Max song length in seconds |
| `DOWNLOADS_DIR` | `database` | Local download directory |
| `COOKIES_URL` | — | Comma-separated cookie file URLs for yt-dlp |
| `SUPPORT_GROUP` | — | Support group link |
| `SUPPORT_CHANNEL` | — | Announcements channel link |
| `START_IMG` | Pinterest URL | Photo shown on /start in private chats |
| `ENABLE_VPLAY` | `true` | Enable /vplay video streaming |
| `PORT` | `6060` | pprof debug server port |

## User preferences

- Preserve the existing Go module path `github.com/zefronxd/TGMUSIC`
- Keep all existing commands, handlers, and callback data constants
- Maintain backward compatibility when extending modules
- Use the existing `cache.Cache[T]` generic for any new caching needs
- New themes for the thumbnail engine: add a new file in `src/core/thumb/` — no other files need to change
