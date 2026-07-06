/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
        "github.com/zefronxd/TGMUSIC/config"
        "github.com/zefronxd/TGMUSIC/internal/queue"
        "github.com/zefronxd/TGMUSIC/src/core"
        "github.com/zefronxd/TGMUSIC/src/core/cache"
        "github.com/zefronxd/TGMUSIC/src/core/db"
        "github.com/zefronxd/TGMUSIC/src/core/dl"
        "github.com/zefronxd/TGMUSIC/src/core/status"
        "github.com/zefronxd/TGMUSIC/src/vc"
        "fmt"
        "html"
        "strings"

        "github.com/zefronxd/TGMUSIC/src/utils"

        td "github.com/AshokShau/gotdbot"
)

// playHandler handles the /play command.
func playHandler(c *td.Client, m *td.Message) error {
        if !playMode(c, m) {
                return td.EndGroups
        }
        return handlePlay(c, m, false)
}

// vPlayHandler handles the /vplay command.
func vPlayHandler(c *td.Client, m *td.Message) error {
        if !playMode(c, m) {
                return td.EndGroups
        }

        if !config.EnableVideoPlayback {
                _, _ = m.ReplyText(c,
                        "🎥 <b>Video Playback Disabled</b>\n"+
                                "━━━━━━━━━━━━━━━━━━━━━━\n\n"+
                                "Video streaming is currently turned off to ensure a smooth audio experience for everyone.\n\n"+
                                "<i>Use /play to stream audio instead.</i>",
                        &td.SendTextMessageOpts{ParseMode: "HTML"},
                )
                return td.EndGroups
        }
        return handlePlay(c, m, true)
}

func handlePlay(c *td.Client, m *td.Message, isVideo bool) error {
        chatID := m.ChatId

        if queueLen := cache.ChatCache.GetQueueLength(chatID); queueLen > 10 {
                _, _ = m.ReplyText(c, "📋 <b>Queue Full</b>\n\nMax 10 tracks in queue. Use <code>/end</code> to clear.", &td.SendTextMessageOpts{ParseMode: "HTML"})
                return td.EndGroups
        }

        // Enforce queue lock: only admins/auth-users/Telegram admins may add tracks.
        if queue.Manager.IsLocked(chatID) {
                senderID := m.SenderID()
                allowed := senderID == config.OwnerId ||
                        db.Instance.IsAdmin(chatID, senderID) ||
                        db.Instance.IsAuthUser(chatID, senderID)
                if !allowed {
                        if _, err := cache.GetUserAdmin(c, chatID, senderID, false); err != nil {
                                _, _ = m.ReplyText(c,
                                        "🔒 <b>Queue Locked</b>\n\nThe queue is currently locked. Only admins can add tracks.",
                                        replyOpts,
                                )
                                return td.EndGroups
                        }
                }
        }

        isReply := m.ReplyToMessageID() != 0
        args := Args(m)
        url := getUrl(c, m, isReply)

        rMsg := m
        var err error
        if isReply && args == "" && url == "" {
                r, err := m.GetRepliedMessage(c)
                if err == nil && r != nil {
                        args = r.Text()
                }
        }

        input := coalesce(url, args)

        if strings.HasPrefix(input, "tgpl_") {
                playlist, err := db.Instance.GetPlaylist(input)
                if err != nil {
                        _, err = m.ReplyText(c, "❌ <b>Playlist Not Found</b>\n\nCould not find a playlist with that ID.", &td.SendTextMessageOpts{ParseMode: "HTML"})
                        return err
                }

                tracks := db.ConvertSongsToTracks(playlist.Songs)
                if len(tracks) == 0 {
                        _, err = m.ReplyText(c, "📋 <b>Playlist Empty</b>\n\nThis playlist has no tracks.", &td.SendTextMessageOpts{ParseMode: "HTML"})
                        return err
                }

                updater, err := status.New(c, m, status.TypeImportingPlaylist)
                if err != nil {
                        c.Logger.Warn("Failed to send loading message", "error", err)
                        return td.EndGroups
                }
                return handleMultipleTracks(c, m, updater, tracks, chatID, isVideo)
        }

        if match := utils.TelegramMessageRegex.FindStringSubmatch(input); match != nil {
                rMsg, err = utils.GetMessage(c, input)
                if err != nil {
                        c.Logger.Warn("Failed to parse Telegram message link", "error", err)
                        _, err = m.ReplyText(c, "❌ <b>Invalid Link</b>\n\nCould not resolve that Telegram message link.", &td.SendTextMessageOpts{ParseMode: "HTML"})
                        return err
                }
        } else if isReply {
                rMsg, err = m.GetRepliedMessage(c)
                if err != nil {
                        _, err = m.ReplyText(c, "❌ <b>Invalid Reply</b>\n\nCould not fetch the replied message.", &td.SendTextMessageOpts{ParseMode: "HTML"})
                        return err
                }
        }

        if isValidMedia(rMsg) {
                isReply = true
        }

        if url == "" && args == "" && (!isReply || !isValidMedia(rMsg)) {
                _, _ = m.ReplyText(c,
                        "🎵 <b>Play</b>\n"+
                                "━━━━━━━━━━━━━━━━━━━━━━\n\n"+
                                "<b>Usage:</b> <code>/play [song or URL]</code>\n\n"+
                                "<b>Supported Platforms</b>\n"+
                                "▸ YouTube  ·  Spotify  ·  Apple Music\n"+
                                "▸ SoundCloud  ·  Deezer  ·  JioSaavn\n"+
                                "▸ Twitch  ·  Kick  ·  Tidal",
                        &td.SendTextMessageOpts{ReplyMarkup: core.SupportKeyboard(), ParseMode: "HTML"},
                )
                return td.EndGroups
        }

        updater, err := status.New(c, m, status.TypeSearching)
        if err != nil {
                c.Logger.Warn("Failed to send searching message", "error", err)
                return td.EndGroups
        }

        if isReply && isValidMedia(rMsg) {
                return handleMedia(c, m, updater, rMsg, chatID, isVideo)
        }

        wrapper := dl.NewDownloaderWrapper(input)
        if url != "" {
                if !wrapper.IsValid() {
                        _, _ = updater.EditText(c,
                                "❌ <b>Unsupported Platform</b>\n"+
                                        "━━━━━━━━━━━━━━━━━━━━━━\n\n"+
                                        "<b>Supported Platforms</b>\n"+
                                        "▸ YouTube  ·  Spotify  ·  Apple Music\n"+
                                        "▸ SoundCloud  ·  Deezer  ·  JioSaavn",
                                &td.EditTextMessageOpts{ReplyMarkup: core.SupportKeyboard(), ParseMode: "HTML"},
                        )
                        return td.EndGroups
                }

                trackInfo, err := wrapper.GetInfo()
                if err != nil {
                        _, _ = updater.EditText(c, fmt.Sprintf("❌ <b>Fetch Failed</b>\n\n<code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
                        return td.EndGroups
                }

                if len(trackInfo.Results) == 0 {
                        _, _ = updater.EditText(c, "😕 <b>No Results</b>\n\nNo tracks found for that link.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                        return td.EndGroups
                }

                return handleUrl(c, m, updater, trackInfo, chatID, isVideo)
        }

        return handleTextSearch(c, m, updater, wrapper, chatID, isVideo)
}

// handleMedia handles playing media from a replied-to Telegram message.
func handleMedia(c *td.Client, m *td.Message, updater status.Updater, dlMsg *td.Message, chatId int64, isVideo bool) error {
        file, fileName := getFile(dlMsg)
        if file == nil {
                _, err := updater.EditText(c, "❌ <b>No Media Found</b>\n\nNo valid media file was found in that message.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        if file.Size > config.MaxFileSize {
                _, err := updater.EditText(c, fmt.Sprintf("❌ <b>File Too Large</b>\n\nMax allowed size is <code>%d MB</code>.", config.MaxFileSize/(1024*1024)), &td.EditTextMessageOpts{ParseMode: "HTML"})
                if err != nil {
                        c.Logger.Warn("Failed to edit message", "error", err)
                }
                return nil
        }

        fileId := dlMsg.RemoteFileID()
        if _track := cache.ChatCache.GetTrackIfExists(chatId, fileId); _track != nil {
                _, err := updater.EditText(c, "ℹ️ <b>Already in Queue</b>\n\nThis track is already playing or in the queue.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        dur := utils.GetFileDur(dlMsg)
        link, err := dlMsg.GetLink(c)
        if err != nil {
                c.Logger.Warn("Failed to get file link", "error", err)
                link.Link = ""
        }

        saveCache := utils.CachedTrack{
                URL: link.Link, Name: fileName, User: firstName(c, m), UserID: m.SenderID(),
                TrackID: fileId, Duration: dur, IsVideo: isVideo, Platform: utils.Telegram,
        }

        qLen := cache.ChatCache.AddSong(chatId, &saveCache)
        if qLen > 1 {
                _, err := updater.EditText(c, utils.QueueAddedText(&saveCache, qLen), &td.EditTextMessageOpts{
                        ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err
        }

        file, err = dlMsg.Download(c, 1, 0, 0, true)
        if err != nil {
                cache.ChatCache.RemoveCurrentSong(chatId)
                _, err = updater.EditText(c, fmt.Sprintf("❌ <b>Download Failed</b>\n\n<code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        filePath := file.Local.Path
        if dur == 0 {
                dur = utils.GetMediaDuration(filePath)
                saveCache.Duration = dur
        }
        saveCache.FilePath = filePath

        if err = vc.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
                cache.ChatCache.RemoveCurrentSong(chatId)
                _, err = updater.EditText(c, err.Error(), &td.EditTextMessageOpts{
                        ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err
        }

        _, err = updater.EditText(c, utils.NowPlayingText(&saveCache), &td.EditTextMessageOpts{
                ParseMode: "HTML", ReplyMarkup: core.ControlButtons("play"), DisableWebPagePreview: true,
        })
        return err
}

// handleTextSearch handles a plain-text song search.
func handleTextSearch(c *td.Client, m *td.Message, updater status.Updater, wrapper *dl.DownloaderWrapper, chatId int64, isVideo bool) error {
        searchResult, err := wrapper.Search()
        if err != nil {
                _, err = updater.EditText(c, fmt.Sprintf("❌ <b>Search Failed</b>\n\n<code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        if len(searchResult.Results) == 0 {
                _, err = updater.EditText(c, "😕 <b>No Results</b>\n\nNo tracks found. Try a different search term.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        song := searchResult.Results[0]
        if _track := cache.ChatCache.GetTrackIfExists(chatId, song.Id); _track != nil {
                _, err := updater.EditText(c, "ℹ️ <b>Already in Queue</b>\n\nThis track is already playing or in the queue.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        return handleSingleTrack(c, m, updater, song, "", chatId, isVideo)
}

// handleUrl handles playback initiated from a platform URL.
func handleUrl(c *td.Client, m *td.Message, updater status.Updater, trackInfo utils.PlatformTracks, chatId int64, isVideo bool) error {
        if len(trackInfo.Results) == 1 {
                track := trackInfo.Results[0]
                if _track := cache.ChatCache.GetTrackIfExists(chatId, track.Id); _track != nil {
                        _, err := updater.EditText(c, "ℹ️ <b>Already in Queue</b>\n\nThis track is already playing or in the queue.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                        return err
                }
                return handleSingleTrack(c, m, updater, track, "", chatId, isVideo)
        }
        return handleMultipleTracks(c, m, updater, trackInfo.Results, chatId, isVideo)
}

// handleSingleTrack enqueues and starts (or queues) a single track.
func handleSingleTrack(c *td.Client, m *td.Message, updater status.Updater, song utils.MusicTrack, filePath string, chatId int64, isVideo bool) error {
        if song.Duration > int(config.SongDurationLimit) {
                _, err := updater.EditText(c, fmt.Sprintf(
                        "⏱ <b>Track Too Long</b>\n\nMax allowed duration is <code>%d min</code>.", config.SongDurationLimit/60,
                ), &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        saveCache := utils.CachedTrack{
                URL: song.Url, Name: song.Title, User: firstName(c, m), UserID: m.SenderID(),
                FilePath: filePath, Thumbnail: song.Thumbnail, TrackID: song.Id, Duration: song.Duration,
                Channel: song.Channel, Views: song.Views, IsVideo: isVideo, Platform: song.Platform,
        }

        qLen := cache.ChatCache.AddSong(chatId, &saveCache)
        if qLen > 1 {
                // Not up next yet - start downloading it in the background now so
                // it's ready the instant it becomes the current track.
                dl.PrefetchTracks(c, []*utils.CachedTrack{&saveCache}, func(track *utils.CachedTrack, path string, err error) {
                        if err == nil && path != "" {
                                cache.ChatCache.SetTrackFilePath(chatId, track.TrackID, path)
                        }
                })

                _, err := updater.EditText(c, utils.QueueAddedText(&saveCache, qLen), &td.EditTextMessageOpts{
                        ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err
        }

        if saveCache.FilePath == "" {
                dlResult, err := dl.DownloadCachedTrack(&saveCache, c)
                if err != nil {
                        cache.ChatCache.RemoveCurrentSong(chatId)
                        _, err = updater.EditText(c, fmt.Sprintf("❌ <b>Download Failed</b>\n\n<code>%s</code>", err.Error()), &td.EditTextMessageOpts{ParseMode: "HTML"})
                        return err
                }
                saveCache.FilePath = dlResult
        }

        if err := vc.Calls.PlayMedia(c, chatId, saveCache.FilePath, saveCache.IsVideo, ""); err != nil {
                cache.ChatCache.RemoveCurrentSong(chatId)
                _, err = updater.EditText(c, err.Error(), &td.EditTextMessageOpts{
                        ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err
        }

        _, err := updater.EditText(c, utils.NowPlayingText(&saveCache), &td.EditTextMessageOpts{
                ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true,
        })
        if err != nil {
                c.Logger.Warn("Failed to edit now-playing message", "error", err)
        }

        // Fire-and-forget: generate and send the premium thumbnail without blocking.
        go sendNowPlayingThumb(c, m, &saveCache, 1)

        return err
}

// handleMultipleTracks enqueues a batch of tracks (playlist / album).
func handleMultipleTracks(c *td.Client, m *td.Message, updater status.Updater, tracks []utils.MusicTrack, chatId int64, isVideo bool) error {
        if len(tracks) == 0 {
                _, err := updater.EditText(c, "😕 <b>No Tracks Found</b>\n\nThis playlist or album appears to be empty.", &td.EditTextMessageOpts{ParseMode: "HTML"})
                return err
        }

        var tracksToAdd []*utils.CachedTrack
        var skippedCount int

        shouldPlayFirst := false
        var firstTrack *utils.CachedTrack

        for _, track := range tracks {
                if track.Duration > int(config.SongDurationLimit) {
                        skippedCount++
                        continue
                }
                saveCache := &utils.CachedTrack{
                        Name: track.Title, TrackID: track.Id, Duration: track.Duration,
                        Thumbnail: track.Thumbnail, User: firstName(c, m), UserID: m.SenderID(),
                        Platform: track.Platform, IsVideo: isVideo, URL: track.Url,
                        Channel: track.Channel, Views: track.Views,
                }
                tracksToAdd = append(tracksToAdd, saveCache)
        }

        if len(tracksToAdd) == 0 {
                msg := "No valid tracks found."
                if skippedCount > 0 {
                        msg = fmt.Sprintf("All %d tracks were skipped (exceeded duration limit of %d min).", skippedCount, config.SongDurationLimit/60)
                }
                _, err := updater.EditText(c, msg, nil)
                return err
        }

        qLenAfter := cache.ChatCache.AddSongs(chatId, tracksToAdd)
        startLen := qLenAfter - len(tracksToAdd)

        if startLen == 0 {
                shouldPlayFirst = true
                firstTrack = tracksToAdd[0]
                firstTrack.Loop = 1
        }

        // Prefetch the tracks queued right behind the current (or soon-to-be
        // current) one in parallel, in the background, bounded by the shared
        // download semaphore.
        prefetchCandidates := tracksToAdd
        if shouldPlayFirst && len(prefetchCandidates) > 0 {
                prefetchCandidates = prefetchCandidates[1:]
        }
        if len(prefetchCandidates) > 3 {
                prefetchCandidates = prefetchCandidates[:3]
        }
        dl.PrefetchTracks(c, prefetchCandidates, func(track *utils.CachedTrack, path string, err error) {
                if err == nil && path != "" {
                        cache.ChatCache.SetTrackFilePath(chatId, track.TrackID, path)
                }
        })

        var sb strings.Builder
        sb.WriteString("<u><b>Added to Queue:</b></u>\n<blockquote expandable>\n")

        totalDuration := 0
        for i, track := range tracksToAdd {
                currentQLen := startLen + i + 1
                fmt.Fprintf(&sb, "<b>%d.</b> %s\n└ %s\n",
                        currentQLen,
                        html.EscapeString(track.Name),
                        utils.SecToMin(track.Duration),
                )
                totalDuration += track.Duration
        }
        sb.WriteString("</blockquote>")

        queueSummary := fmt.Sprintf(
                "\n<b>Queue Total:</b> %d\n<b>Duration:</b> %s\n<b>Requested by:</b> %s",
                qLenAfter, utils.SecToMin(totalDuration), html.EscapeString(firstName(c, m)),
        )
        sb.WriteString(queueSummary)

        if skippedCount > 0 {
                fmt.Fprintf(&sb, "\n\n<b>Skipped %d tracks</b> (exceeded duration limit).", skippedCount)
        }

        fullMessage := sb.String()
        if len(fullMessage) > 4096 {
                fullMessage = queueSummary
        }

        if shouldPlayFirst && firstTrack != nil {
                _ = vc.Calls.PlayNext(c, chatId)
        }

        _, err := updater.EditText(c, fullMessage, &td.EditTextMessageOpts{
                ParseMode: "HTML", ReplyMarkup: core.ControlButtons("play"), DisableWebPagePreview: true,
        })
        return err
}
