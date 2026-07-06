/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package queue

import (
        "fmt"
        "log/slog"
        "strconv"
        "strings"

        "github.com/zefronxd/TGMUSIC/config"
        "github.com/zefronxd/TGMUSIC/src/core"
        "github.com/zefronxd/TGMUSIC/src/core/cache"
        "github.com/zefronxd/TGMUSIC/src/core/db"
        "github.com/zefronxd/TGMUSIC/src/vc"

        td "github.com/AshokShau/gotdbot"
)

// ────────────────────────────────────────────────────────────────────────────
// Keyboard builders
// ────────────────────────────────────────────────────────────────────────────

func cbBtn(text, data string) td.InlineKeyboardButton {
        return td.InlineKeyboardButton{
                Text: text,
                Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(data)},
        }
}

// MainKeyboard returns the navigation + admin keyboard for the queue page view.
func MainKeyboard(page, totalPages int, locked bool, upNextCount int) *td.ReplyMarkupInlineKeyboard {
        rows := make([][]td.InlineKeyboardButton, 0, 5)

        // Row 1: pagination (only when there is more than one page).
        if totalPages > 1 {
                pageLabel := fmt.Sprintf("📄 %d/%d", page, totalPages)
                rows = append(rows, []td.InlineKeyboardButton{
                        cbBtn("⏮", core.CBQueuePage+":1"),
                        cbBtn("◀", fmt.Sprintf("%s:%d", core.CBQueuePage, max(1, page-1))),
                        cbBtn(pageLabel, core.CBQueueStats),
                        cbBtn("▶", fmt.Sprintf("%s:%d", core.CBQueuePage, min(totalPages, page+1))),
                        cbBtn("⏭", fmt.Sprintf("%s:%d", core.CBQueuePage, totalPages)),
                })
        }

        // Row 2: sort/filter/refresh.
        rows = append(rows, []td.InlineKeyboardButton{
                cbBtn("🔄 Refresh", core.CBQueueRefresh),
                cbBtn("🔀 Sort", core.CBQueueSortMenu),
                cbBtn("📂 Filter", core.CBQueueFilterMenu),
        })

        // Row 3: admin actions.
        lockLabel := "🔓 Unlock"
        if !locked {
                lockLabel = "🔒 Lock"
        }
        rows = append(rows, []td.InlineKeyboardButton{
                cbBtn("🗑 Clear", core.CBQueueClear),
                cbBtn("🔀 Shuffle", core.CBQueueShuffle),
                cbBtn("🔁 Reverse", core.CBQueueReverse),
                cbBtn(lockLabel, core.CBQueueLock),
        })

        // Row 4: clickable track-number buttons for detail view.
        if upNextCount > 0 {
                const btnsPerRow = 5
                numRows := (upNextCount + btnsPerRow - 1) / btnsPerRow
                for r := 0; r < numRows; r++ {
                        row := make([]td.InlineKeyboardButton, 0, btnsPerRow)
                        for b := 0; b < btnsPerRow; b++ {
                                pos := r*btnsPerRow + b + 1
                                if pos > upNextCount {
                                        break
                                }
                                row = append(row, cbBtn(fmt.Sprintf("%d", pos), fmt.Sprintf("%s:%d", core.CBQueueDetail, pos)))
                        }
                        rows = append(rows, row)
                }
        }

        // Last row: close button.
        rows = append(rows, []td.InlineKeyboardButton{
                cbBtn("❌ Close", core.CBQueueClose),
        })

        return &td.ReplyMarkupInlineKeyboard{Rows: rows}
}

// DetailKeyboard returns the keyboard for the single-track detail view.
func DetailKeyboard(idx, queueLen int) *td.ReplyMarkupInlineKeyboard {
        canUp := idx > 1
        canDown := idx < queueLen-1 // queueLen includes now-playing at index 0

        rows := make([][]td.InlineKeyboardButton, 0, 3)

        // Row 1: movement controls.
        moveRow := []td.InlineKeyboardButton{}
        if canUp {
                moveRow = append(moveRow, cbBtn("⬆ Move Up", fmt.Sprintf("%s:%d", core.CBQueueMoveUp, idx)))
        }
        if canDown {
                moveRow = append(moveRow, cbBtn("⬇ Move Down", fmt.Sprintf("%s:%d", core.CBQueueMoveDown, idx)))
        }
        moveRow = append(moveRow, cbBtn("↩ Jump Here", fmt.Sprintf("%s:%d", core.CBQueueJump, idx)))
        rows = append(rows, moveRow)

        // Row 2: user actions.
        rows = append(rows, []td.InlineKeyboardButton{
                cbBtn("🗑 Remove", fmt.Sprintf("%s:%d", core.CBQueueRemove, idx)),
                cbBtn("⭐ Favorite", fmt.Sprintf("%s:%d", core.CBQueueFav, idx)),
                cbBtn("🔗 Share", fmt.Sprintf("%s:%d", core.CBQueueShare, idx)),
        })

        // Row 3: back.
        rows = append(rows, []td.InlineKeyboardButton{
                cbBtn("◀ Back", core.CBQueueBack),
        })

        return &td.ReplyMarkupInlineKeyboard{Rows: rows}
}

// SortKeyboard returns the sort-selection keyboard.
func SortKeyboard(current string) *td.ReplyMarkupInlineKeyboard {
        check := func(k string) string {
                if k == current {
                        return "✓ "
                }
                return ""
        }
        return &td.ReplyMarkupInlineKeyboard{
                Rows: [][]td.InlineKeyboardButton{
                        {
                                cbBtn(check(SortDefault)+"Default", fmt.Sprintf("%s:%s", core.CBQueueSort, SortDefault)),
                                cbBtn(check(SortAlpha)+"A–Z", fmt.Sprintf("%s:%s", core.CBQueueSort, SortAlpha)),
                                cbBtn(check(SortDuration)+"Duration", fmt.Sprintf("%s:%s", core.CBQueueSort, SortDuration)),
                        },
                        {
                                cbBtn(check(SortNewest)+"Newest", fmt.Sprintf("%s:%s", core.CBQueueSort, SortNewest)),
                                cbBtn(check(SortOldest)+"Oldest", fmt.Sprintf("%s:%s", core.CBQueueSort, SortOldest)),
                                cbBtn(check(SortRequester)+"Requester", fmt.Sprintf("%s:%s", core.CBQueueSort, SortRequester)),
                                cbBtn(check(SortSource)+"Source", fmt.Sprintf("%s:%s", core.CBQueueSort, SortSource)),
                        },
                        {cbBtn("◀ Back", core.CBQueueBack)},
                },
        }
}

// FilterKeyboard returns the filter-selection keyboard.
func FilterKeyboard(current string) *td.ReplyMarkupInlineKeyboard {
        check := func(k string) string {
                if k == current {
                        return "✓ "
                }
                return ""
        }
        return &td.ReplyMarkupInlineKeyboard{
                Rows: [][]td.InlineKeyboardButton{
                        {
                                cbBtn(check(FilterAll)+"All", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterAll)),
                                cbBtn(check(FilterYouTube)+"YouTube", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterYouTube)),
                                cbBtn(check(FilterSpotify)+"Spotify", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterSpotify)),
                        },
                        {
                                cbBtn(check(FilterTelegram)+"Telegram", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterTelegram)),
                                cbBtn(check(FilterVideo)+"Video", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterVideo)),
                                cbBtn(check(FilterAudio)+"Audio", fmt.Sprintf("%s:%s", core.CBQueueFilter, FilterAudio)),
                        },
                        {cbBtn("◀ Back", core.CBQueueBack)},
                },
        }
}

// ────────────────────────────────────────────────────────────────────────────
// Admin helpers
// ────────────────────────────────────────────────────────────────────────────

// isQueueAdmin reports whether userID may perform admin-level queue actions in
// chatID. Checks: bot owner, db-registered admin, auth-user, Telegram admin.
func isQueueAdmin(c *td.Client, chatID, userID int64) bool {
        if userID == config.OwnerId {
                return true
        }
        if db.Instance.IsAdmin(chatID, userID) || db.Instance.IsAuthUser(chatID, userID) {
                return true
        }
        member, err := cache.GetUserAdmin(c, chatID, userID, false)
        if err != nil {
                return false
        }
        switch member.Status.(type) {
        case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator:
                return true
        }
        return false
}

// denyAdmin answers the callback with a "permission denied" alert.
func denyAdmin(c *td.Client, cb *td.UpdateNewCallbackQuery) {
        _ = cb.Answer(c, 0, true, "🚫 Admin action — only admins may do this.", "")
}

// ────────────────────────────────────────────────────────────────────────────
// View helpers
// ────────────────────────────────────────────────────────────────────────────

// editOpts returns standard HTML edit options with no web-page preview.
var editOpts = &td.EditTextMessageOpts{
        ParseMode:             "HTML",
        DisableWebPagePreview: true,
}

// renderAndEdit renders the current queue page for chatID and edits the
// callback message in-place. It uses the Manager's page cache to avoid
// rebuilding identical pages.
func renderAndEdit(c *td.Client, cb *td.UpdateNewCallbackQuery, chatID int64, chatTitle string) error {
        rawQueue := cache.ChatCache.GetQueue(chatID)

        // Detect queue changes and invalidate the text cache automatically.
        hash := queueHash(rawQueue)
        Manager.checkHash(chatID, hash)

        page := ClampPage(Manager.GetPage(chatID), len(rawQueue))
        Manager.SetPage(chatID, page)

        sortBy := Manager.GetSort(chatID)
        filterBy := Manager.GetFilter(chatID)
        locked := Manager.IsLocked(chatID)

        // Apply sort and filter on a copy — never mutate the live queue.
        view := ApplySort(rawQueue, sortBy)
        view = ApplyFilter(view, filterBy)

        totalPages := PageCount(len(view))
        page = ClampPage(page, len(view))

        // Try the text cache first.
        text := Manager.GetCachedText(chatID, page)
        if text == "" {
                playedSecs := 0
                if t, err := vc.Calls.PlayedTime(chatID); err == nil {
                        playedSecs = int(t)
                }
                text = RenderPage(chatTitle, view, page, playedSecs, sortBy, filterBy, locked)
                Manager.SetCachedText(chatID, page, text)
        }

        upNext := PageSlice(view, page)
        kb := MainKeyboard(page, totalPages, locked, len(upNext))

        opts := &td.EditTextMessageOpts{
                ReplyMarkup:           kb,
                ParseMode:             "HTML",
                DisableWebPagePreview: true,
        }
        _, err := cb.EditMessageText(c, text, opts)
        return err
}

// ────────────────────────────────────────────────────────────────────────────
// Main callback dispatcher
// ────────────────────────────────────────────────────────────────────────────

// HandleQueueCallback is the single entry point for all "queue_" callbacks.
// Register it in load.go with:
//
//      c.OnUpdateNewCallbackQuery(queue.HandleQueueCallback, callbackquery.Prefix(core.PrefixQueue))
func HandleQueueCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
        if cb.IsPrivate() {
                _ = cb.Answer(c, 0, true, "Queue controls only work in groups.", "")
                return td.EndGroups
        }

        chatID := cb.ChatId
        userID := cb.SenderUserId

        // Resolve requester's display name for action feedback.
        userName := "User"
        if u, err := c.GetUser(userID); err == nil {
                userName = u.FirstName
        }

        // Resolve chat title for the header.
        chatTitle := "Queue"
        if ch, err := c.GetChat(chatID); err == nil {
                chatTitle = ch.Title
        }

        // Parse "queue_CMD" or "queue_CMD:ARG"
        data := cb.DataString()
        rest := strings.TrimPrefix(data, core.PrefixQueue)
        parts := strings.SplitN(rest, ":", 2)
        cmd := parts[0]
        arg := ""
        if len(parts) > 1 {
                arg = parts[1]
        }

        // parseInt parses arg as int; returns (value, ok).
        parseInt := func() (int, bool) {
                n, err := strconv.Atoi(arg)
                return n, err == nil
        }

        switch cmd {

        // ── Navigation ──────────────────────────────────────────────────────────

        case "pg": // queue_pg:N — navigate to page N
                n, ok := parseInt()
                if !ok {
                        _ = cb.Answer(c, 0, true, "Invalid page.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                view := ApplySort(rawQueue, Manager.GetSort(chatID))
                view = ApplyFilter(view, Manager.GetFilter(chatID))
                clamped := ClampPage(n, len(view))
                Manager.SetPage(chatID, clamped)
                _ = cb.Answer(c, 0, false, "", "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "ref": // queue_ref — refresh
                Manager.Invalidate(chatID)
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if len(rawQueue) == 0 {
                        _ = cb.Answer(c, 0, false, "Queue is empty.", "")
                        _, _ = cb.EditMessageText(c, "📋 <b>Queue Empty</b>\n\nThere are no tracks in the queue.\n\n<i>Use /play to add a track.</i>", editOpts)
                        return nil
                }
                _ = cb.Answer(c, 0, false, "🔄 Refreshed.", "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "sts": // queue_sts — show stats alert
                rawQueue := cache.ChatCache.GetQueue(chatID)
                view := ApplySort(rawQueue, Manager.GetSort(chatID))
                view = ApplyFilter(view, Manager.GetFilter(chatID))
                page := ClampPage(Manager.GetPage(chatID), len(view))
                totalPages := PageCount(len(view))

                statsText := Manager.GetCachedStats(chatID)
                if statsText == "" {
                        statsText = RenderStatsAlert(view, page, totalPages)
                        Manager.SetCachedStats(chatID, statsText)
                }
                _ = cb.Answer(c, 0, true, statsText, "")
                return nil

        case "cls": // queue_cls — close (delete the message)
                _ = cb.Answer(c, 0, false, "Closed.", "")
                _ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
                return nil

        // ── Sort / Filter ────────────────────────────────────────────────────────

        case "srtm": // queue_srtm — open sort menu
                _ = cb.Answer(c, 0, false, "", "")
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if len(rawQueue) == 0 {
                        _ = cb.Answer(c, 0, true, "Queue is empty.", "")
                        return nil
                }
                _, err := cb.EditMessageText(c, "🔀 <b>Sort Queue</b>\n\nChoose a sort order for the display view.\n<i>The playback order is not affected by sorting.</i>", &td.EditTextMessageOpts{
                        ReplyMarkup:           SortKeyboard(Manager.GetSort(chatID)),
                        ParseMode:             "HTML",
                        DisableWebPagePreview: true,
                })
                return err

        case "srt": // queue_srt:TYPE — apply sort
                sortBy := arg
                switch sortBy {
                case SortDefault, SortAlpha, SortDuration, SortNewest, SortOldest, SortRequester, SortSource:
                default:
                        _ = cb.Answer(c, 0, true, "Unknown sort type.", "")
                        return nil
                }
                Manager.SetSort(chatID, sortBy)
                Manager.SetPage(chatID, 1)
                _ = cb.Answer(c, 0, false, fmt.Sprintf("Sort: %s", SortLabel(sortBy)), "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "fltm": // queue_fltm — open filter menu
                _ = cb.Answer(c, 0, false, "", "")
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if len(rawQueue) == 0 {
                        _ = cb.Answer(c, 0, true, "Queue is empty.", "")
                        return nil
                }
                _, err := cb.EditMessageText(c, "📂 <b>Filter Queue</b>\n\nShow only tracks matching the selected type.", &td.EditTextMessageOpts{
                        ReplyMarkup:           FilterKeyboard(Manager.GetFilter(chatID)),
                        ParseMode:             "HTML",
                        DisableWebPagePreview: true,
                })
                return err

        case "flt": // queue_flt:TYPE — apply filter
                filterBy := arg
                switch filterBy {
                case FilterAll, FilterYouTube, FilterSpotify, FilterTelegram, FilterVideo, FilterAudio:
                default:
                        _ = cb.Answer(c, 0, true, "Unknown filter type.", "")
                        return nil
                }
                Manager.SetFilter(chatID, filterBy)
                Manager.SetPage(chatID, 1)
                _ = cb.Answer(c, 0, false, fmt.Sprintf("Filter: %s", FilterLabel(filterBy)), "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "bk": // queue_bk — back to main page view
                _ = cb.Answer(c, 0, false, "", "")
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if len(rawQueue) == 0 {
                        _, _ = cb.EditMessageText(c, "📋 <b>Queue Empty</b>\n\nThere are no tracks in the queue.\n\n<i>Use /play to add a track.</i>", editOpts)
                        return nil
                }
                return renderAndEdit(c, cb, chatID, chatTitle)

        // ── Track Detail View ────────────────────────────────────────────────────

        case "dtl": // queue_dtl:N — show detail for track at 1-based up-next position N on current page
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid track number.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                // Resolve sorted/filtered view and locate the track on the current page.
                page := Manager.GetPage(chatID)
                view := ApplySort(rawQueue, Manager.GetSort(chatID))
                view = ApplyFilter(view, Manager.GetFilter(chatID))
                upNext := PageSlice(view, page)
                if n < 1 || n > len(upNext) {
                        _ = cb.Answer(c, 0, true, "Track not found on this page.", "")
                        return nil
                }
                selectedTrack := upNext[n-1]

                // Map the view pointer back to its raw queue index so that action
                // buttons always target the correct slot regardless of active sort/filter.
                rawIdx := -1
                for i, t := range rawQueue {
                        if t == selectedTrack {
                                rawIdx = i
                                break
                        }
                }
                if rawIdx < 1 {
                        _ = cb.Answer(c, 0, true, "Track not found in queue.", "")
                        return nil
                }

                _ = cb.Answer(c, 0, false, "", "")
                // Render detail from the raw queue so indices shown are raw indices.
                text := RenderDetail(rawQueue, rawIdx)
                kb := DetailKeyboard(rawIdx, len(rawQueue))
                _, err := cb.EditMessageText(c, text, &td.EditTextMessageOpts{
                        ReplyMarkup:           kb,
                        ParseMode:             "HTML",
                        DisableWebPagePreview: true,
                })
                return err

        // ── Per-track actions ────────────────────────────────────────────────────

        case "rm": // queue_rm:N — remove track at raw-queue index N (1-based up-next)
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid track index.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if n >= len(rawQueue) {
                        _ = cb.Answer(c, 0, true, "Track not found.", "")
                        return nil
                }
                target := rawQueue[n]
                // Non-admins may only remove their own tracks.
                // Prefer the stable UserID field; fall back to display name only when
                // UserID was not captured (legacy tracks added before this version).
                isOwner := (target.UserID != 0 && target.UserID == userID) ||
                        (target.UserID == 0 && target.User == userName)
                if !isOwner && !isQueueAdmin(c, chatID, userID) {
                        _ = cb.Answer(c, 0, true, "🚫 You can only remove your own tracks.", "")
                        return nil
                }
                if !cache.ChatCache.RemoveTrack(chatID, n) {
                        _ = cb.Answer(c, 0, true, "Failed to remove track.", "")
                        return nil
                }
                Manager.Invalidate(chatID)
                _ = cb.Answer(c, 0, false, fmt.Sprintf("🗑 Removed: %s", trunc(target.Name, 30)), "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "mu": // queue_mu:N — move track at index N one position up (toward head)
                n, ok := parseInt()
                if !ok || n < 2 {
                        _ = cb.Answer(c, 0, true, "Cannot move further up.", "")
                        return nil
                }
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                if !cache.ChatCache.MoveTrack(chatID, n, n-1) {
                        _ = cb.Answer(c, 0, true, "Move failed.", "")
                        return nil
                }
                Manager.Invalidate(chatID)
                _ = cb.Answer(c, 0, false, "⬆ Moved up.", "")
                // Reopen detail view for the new index.
                rawQueue := cache.ChatCache.GetQueue(chatID)
                newIdx := n - 1
                if newIdx < 1 || newIdx >= len(rawQueue) {
                        return renderAndEdit(c, cb, chatID, chatTitle)
                }
                text := RenderDetail(rawQueue, newIdx)
                kb := DetailKeyboard(newIdx, len(rawQueue))
                _, err := cb.EditMessageText(c, text, &td.EditTextMessageOpts{
                        ReplyMarkup: kb, ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err

        case "md": // queue_md:N — move track at index N one position down (toward tail)
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid index.", "")
                        return nil
                }
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if n >= len(rawQueue)-1 {
                        _ = cb.Answer(c, 0, true, "Cannot move further down.", "")
                        return nil
                }
                if !cache.ChatCache.MoveTrack(chatID, n, n+1) {
                        _ = cb.Answer(c, 0, true, "Move failed.", "")
                        return nil
                }
                Manager.Invalidate(chatID)
                _ = cb.Answer(c, 0, false, "⬇ Moved down.", "")
                rawQueue = cache.ChatCache.GetQueue(chatID)
                newIdx := n + 1
                if newIdx >= len(rawQueue) {
                        return renderAndEdit(c, cb, chatID, chatTitle)
                }
                text := RenderDetail(rawQueue, newIdx)
                kb := DetailKeyboard(newIdx, len(rawQueue))
                _, err := cb.EditMessageText(c, text, &td.EditTextMessageOpts{
                        ReplyMarkup: kb, ParseMode: "HTML", DisableWebPagePreview: true,
                })
                return err

        case "jmp": // queue_jmp:N — jump to track at raw-queue index N
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid track index.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if n >= len(rawQueue) {
                        _ = cb.Answer(c, 0, true, "Track not found.", "")
                        return nil
                }
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                // Remove all "up next" tracks before the target.
                cache.ChatCache.TrimQueueBefore(chatID, n)
                Manager.Invalidate(chatID)
                // Skip the current track so the target starts playing.
                if err := vc.Calls.PlayNext(c, chatID); err != nil {
                        slog.Warn("queue jump: PlayNext failed", "chat_id", chatID, "error", err)
                }
                _ = cb.Answer(c, 0, false, "↩ Jumped to track.", "")
                _ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
                return nil

        case "fav": // queue_fav:N — add track N to the user's first playlist
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid track index.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if n >= len(rawQueue) {
                        _ = cb.Answer(c, 0, true, "Track not found.", "")
                        return nil
                }
                t := rawQueue[n]
                playlists, err := db.Instance.GetUserPlaylists(userID)
                if err != nil {
                        _ = cb.Answer(c, 0, true, "Could not fetch playlists.", "")
                        return nil
                }
                var playlistID string
                if len(playlists) == 0 {
                        playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", userID)
                        if err != nil {
                                _ = cb.Answer(c, 0, true, "Could not create playlist.", "")
                                return nil
                        }
                } else {
                        playlistID = playlists[0].ID
                }
                song := db.Song{
                        URL: t.URL, Name: t.Name, TrackID: t.TrackID,
                        Duration: t.Duration, Platform: t.Platform,
                }
                if err := db.Instance.AddSongToPlaylist(playlistID, song); err != nil {
                        _ = cb.Answer(c, 0, true, "Could not add to playlist.", "")
                        return nil
                }
                _ = cb.Answer(c, 0, false, fmt.Sprintf("⭐ Added \"%s\" to your playlist.", trunc(t.Name, 30)), "")
                return nil

        case "shr": // queue_shr:N — share track URL as an alert
                n, ok := parseInt()
                if !ok || n < 1 {
                        _ = cb.Answer(c, 0, true, "Invalid track index.", "")
                        return nil
                }
                rawQueue := cache.ChatCache.GetQueue(chatID)
                if n >= len(rawQueue) {
                        _ = cb.Answer(c, 0, true, "Track not found.", "")
                        return nil
                }
                t := rawQueue[n]
                shareText := t.URL
                if shareText == "" {
                        shareText = "No shareable URL available for this track."
                } else {
                        shareText = fmt.Sprintf("%s\n%s", trunc(t.Name, 40), shareText)
                }
                _ = cb.Answer(c, 0, true, shareText, "")
                return nil

        // ── Admin bulk actions ───────────────────────────────────────────────────

        case "clr": // queue_clr — clear the entire queue and stop playback
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                if err := vc.Calls.Stop(chatID, false); err != nil {
                        _ = cb.Answer(c, 0, true, "Failed to stop playback.", "")
                        return nil
                }
                Manager.Invalidate(chatID)
                Manager.Remove(chatID)
                _ = cb.Answer(c, 0, false, "🗑 Queue cleared.", "")
                _, err := cb.EditMessageText(c, fmt.Sprintf(
                        "🗑 <b>Queue Cleared</b>\n\n👤 <i>Cleared by %s</i>",
                        userName,
                ), editOpts)
                return err

        case "shf": // queue_shf — shuffle up-next tracks
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                cache.ChatCache.ShuffleQueue(chatID)
                Manager.Invalidate(chatID)
                _ = cb.Answer(c, 0, false, "🔀 Queue shuffled.", "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "rev": // queue_rev — reverse up-next tracks
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                cache.ChatCache.ReverseQueue(chatID)
                Manager.Invalidate(chatID)
                _ = cb.Answer(c, 0, false, "🔁 Queue reversed.", "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        case "lck": // queue_lck — toggle queue lock
                if !isQueueAdmin(c, chatID, userID) {
                        denyAdmin(c, cb)
                        return nil
                }
                locked := Manager.ToggleLock(chatID)
                Manager.Invalidate(chatID)
                status := "🔒 Queue locked."
                if !locked {
                        status = "🔓 Queue unlocked."
                }
                _ = cb.Answer(c, 0, false, status, "")
                return renderAndEdit(c, cb, chatID, chatTitle)

        default:
                slog.Warn("queue: unhandled callback command", "cmd", cmd, "data", data)
                _ = cb.Answer(c, 0, true, "Unknown queue action.", "")
        }

        return nil
}

// min/max are provided as local helpers for Go versions below 1.21.
func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}

func max(a, b int) int {
        if a > b {
                return a
        }
        return b
}
