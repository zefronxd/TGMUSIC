/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"github.com/zefronxd/TGMUSIC/src/core/db"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var (
	broadcastCancelFlag atomic.Bool
	broadcastInProgress atomic.Bool
)

func getFloodWait(err error) int {
	if err == nil {
		return 0
	}

	type retryError interface {
		GetRetryAfter() int
	}

	if re, ok := err.(retryError); ok {
		return re.GetRetryAfter()
	}

	if tdErr, ok := err.(*td.Error); ok {
		return tdErr.GetRetryAfter()
	}

	if tdErr, ok := err.(td.Error); ok {
		return tdErr.GetRetryAfter()
	}

	return 0
}

func cancelBroadcastHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if !broadcastInProgress.Load() {
		_, _ = m.ReplyText(c, "ℹ️ <b>No Broadcast Active</b>\n\nThere is no broadcast in progress.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	broadcastCancelFlag.Store(true)
	_, _ = m.ReplyText(c, "🛑 <b>Broadcast Cancelled</b>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	return td.EndGroups
}

func broadcastHandler(c *td.Client, m *td.Message) error {
	if !isDev(c, m) {
		return td.EndGroups
	}

	if broadcastInProgress.Load() {
		_, _ = m.ReplyText(c, "⚠️ <b>Broadcast Running</b>\n\nA broadcast is already in progress. Use <code>/cancelbc</code> to stop it.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	reply, err := m.GetRepliedMessage(c)
	if err != nil {
		usage := "📢 <b>Broadcast</b>\n" +
			"━━━━━━━━━━━━━━━━━━━━━━\n\n" +
			"<b>Usage:</b> Reply to a message, then:\n\n" +
			"▸ <code>/broadcast</code> — groups + users\n" +
			"▸ <code>/broadcast -chat</code> — groups only\n" +
			"▸ <code>/broadcast -user</code> — users only\n" +
			"▸ <code>/broadcast -copy</code> — send as copy\n\n" +
			"<i>Reply to the message you want to broadcast.</i>"

		_, _ = m.ReplyText(c, usage, &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	args := strings.Fields(Args(m))

	copyMode := false
	mode := "both" // default

	for _, a := range args {
		switch a {
		case "-copy":
			copyMode = true
		case "-chat":
			mode = "chat"
		case "-user":
			mode = "user"
		case "-both":
			mode = "both"
		}
	}

	chats, _ := db.Instance.GetAllChats()
	users, _ := db.Instance.GetAllUsers()

	groupsMap := make(map[int64]bool)
	for _, id := range chats {
		groupsMap[id] = true
	}

	var targets []int64

	switch mode {
	case "chat":
		targets = append(targets, chats...)
	case "user":
		targets = append(targets, users...)
	case "both":
		targets = append(targets, chats...)
		targets = append(targets, users...)
	}

	if len(targets) == 0 {
		_, _ = m.ReplyText(c, "⚠️ <b>No Targets</b>\n\nNo chats or users found to broadcast to.", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return td.EndGroups
	}

	broadcastCancelFlag.Store(false)
	broadcastInProgress.Store(true)

	sentMsg, _ := m.ReplyText(c, fmt.Sprintf("📢 <b>Broadcast Started</b>\n\n▸ Targets: <code>%d</code>", len(targets)), &td.SendTextMessageOpts{ParseMode: "HTML"})

	go func() {
		defer broadcastInProgress.Store(false)

		var failedBuilder strings.Builder
		count, ucount := 0, 0

		for _, chatID := range targets {
			if broadcastCancelFlag.Load() {
				_, _ = sentMsg.EditText(
					c,
					fmt.Sprintf("🛑 <b>Broadcast Cancelled</b>\n\n▸ Groups: <code>%d</code>\n▸ Users: <code>%d</code>", count, ucount),
					&td.EditTextMessageOpts{ParseMode: "HTML"},
				)
				return
			}

			var errSend error
			if copyMode {
				_, errSend = reply.Copy(c, chatID, &td.SendCopyOpts{
					ReplyMarkup: reply.ReplyMarkup,
				})
			} else {
				_, errSend = reply.Forward(c, chatID, &td.ForwardMessageOpts{})
			}

			if errSend == nil {
				if groupsMap[chatID] {
					count++
				} else {
					ucount++
				}
				time.Sleep(200 * time.Millisecond)
			} else {
				wait := getFloodWait(errSend)
				if wait > 0 {
					time.Sleep(time.Duration(wait+30) * time.Second)
					continue
				}
				failedBuilder.WriteString(fmt.Sprintf("%d - %v\n", chatID, errSend))
			}
		}

		text := fmt.Sprintf("✅ <b>Broadcast Complete</b>\n\n▸ Groups: <code>%d</code>\n▸ Users: <code>%d</code>", count, ucount)
		failedStr := failedBuilder.String()

		if failedStr != "" {
			errFile := filepath.Join(
				os.TempDir(),
				fmt.Sprintf("errors_%d.txt", time.Now().UnixNano()),
			)

			if err := os.WriteFile(errFile, []byte(failedStr), 0644); err == nil {
				defer os.Remove(errFile)

				_, errSendDoc := m.ReplyDocument(
					c,
					td.InputFileLocal{Path: errFile},
					&td.SendDocumentOpts{Caption: text},
				)

				if errSendDoc != nil {
					_, _ = sentMsg.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML"})
				}
			} else {
				_, _ = sentMsg.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML"})
			}
		} else {
			_, _ = sentMsg.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML"})
		}
	}()

	return td.EndGroups
}
