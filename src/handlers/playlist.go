/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zefronxd/TGMUSIC/src/core/db"
	"github.com/zefronxd/TGMUSIC/src/core/dl"

	td "github.com/AshokShau/gotdbot"
)

func createPlaylistHandler(c *td.Client, m *td.Message) error {

	userID := m.SenderID()

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(c, "<b>Usage:</b> /createplaylist [playlist name]", replyOpts)
		return err
	}

	userPlaylists, err := db.Instance.GetUserPlaylists(userID)
	if err != nil {
		_, err = m.ReplyText(c, "❌ <b>Fetch Failed</b>\n\nUnable to retrieve your playlists. Please try again later.", replyOpts)
		return err
	}

	if len(userPlaylists) >= 10 {
		_, _ = m.ReplyText(c, "⚠️ <b>Playlist Limit Reached</b>\n\nYou can have a maximum of <code>10</code> playlists.", replyOpts)
		return td.EndGroups
	}

	if len([]rune(args)) > 40 {
		args = string([]rune(args)[:40])
	}

	playlistID, err := db.Instance.CreatePlaylist(args, userID)
	if err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("❌ <b>Creation Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"✅ <b>Playlist Created</b>\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"▸ <b>Name</b>  %s\n"+
				"▸ <b>ID</b>    <code>%s</code>",
			args,
			playlistID,
		),
		replyOpts,
	)

	return td.EndGroups
}

func deletePlaylistHandler(c *td.Client, m *td.Message) error {

	userID := m.SenderID()

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /deleteplaylist [playlist id]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlist, err := db.Instance.GetPlaylist(args)
	if err != nil {
		_, err := m.ReplyText(c, "❌ <b>Not Found</b>\n\nNo playlist found with that ID.", replyOpts)
		return err
	}

	if playlist.UserID != userID {
		_, err := m.ReplyText(c, "🔒 <b>Permission Denied</b>\n\nYou can only delete playlists you created.", replyOpts)
		return err
	}

	err = db.Instance.DeletePlaylist(args, userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("❌ <b>Delete Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf("🗑 <b>Playlist Deleted</b>\n\n▸ <b>%s</b>", playlist.Name),
		replyOpts,
	)

	return err
}
func addToPlaylistHandler(c *td.Client, m *td.Message) error {

	userID := m.SenderID()

	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /addtoplaylist [playlist id] [song url]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlistID := args[0]
	songURL := args[1]

	playlist, err := db.Instance.GetPlaylist(playlistID)
	if err != nil {
		_, err := m.ReplyText(c, "❌ <b>Not Found</b>\n\nNo playlist found with that ID.", replyOpts)
		return err
	}

	if playlist.UserID != userID {
		_, err := m.ReplyText(c, "🔒 <b>Permission Denied</b>\n\nYou can only modify playlists you created.", replyOpts)
		return err
	}

	wrapper := dl.NewDownloaderWrapper(songURL)
	if !wrapper.IsValid() {
		_, err := m.ReplyText(c, "❌ <b>Unsupported Platform</b>\n\nThe provided URL is invalid or the platform is not supported.", replyOpts)
		return err
	}

	trackInfo, err := wrapper.GetInfo()
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("❌ <b>Fetch Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return err
	}

	if trackInfo.Results == nil || len(trackInfo.Results) == 0 {
		_, err := m.ReplyText(c, "😕 <b>No Tracks Found</b>\n\nNo playable tracks found for that link.", replyOpts)
		return err
	}

	song := db.Song{
		URL:      trackInfo.Results[0].Url,
		Name:     trackInfo.Results[0].Title,
		TrackID:  trackInfo.Results[0].Id,
		Duration: trackInfo.Results[0].Duration,
		Platform: trackInfo.Results[0].Platform,
	}

	err = db.Instance.AddSongToPlaylist(playlistID, song)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("❌ <b>Add Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return err
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"✅ <b>Track Added</b>\n\n"+
				"▸ <b>%s</b>\n"+
				"  <i>→ %s</i>",
			song.Name,
			playlist.Name,
		),
		replyOpts,
	)

	return err
}

func removeFromPlaylistHandler(c *td.Client, m *td.Message) error {

	userID := m.SenderID()

	args := strings.SplitN(Args(m), " ", 2)
	if len(args) != 2 {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /removefromplaylist [playlist id] [song number or url]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlistID := args[0]
	songIdentifier := args[1]

	playlist, err := db.Instance.GetPlaylist(playlistID)
	if err != nil {
		_, err = m.ReplyText(c, "❌ <b>Not Found</b>\n\nNo playlist found with that ID.", replyOpts)
		return err
	}

	if playlist.UserID != userID {
		_, err = m.ReplyText(c, "🔒 <b>Permission Denied</b>\n\nYou do not own this playlist.", replyOpts)
		return err
	}

	songIndex, err := strconv.Atoi(songIdentifier)
	var trackID string

	if err == nil {
		if songIndex < 1 || songIndex > len(playlist.Songs) {
			_, err := m.ReplyText(c, "❌ <b>Invalid Number</b>\n\nPlease provide a valid song number.", replyOpts)
			return err
		}
		trackID = playlist.Songs[songIndex-1].TrackID
	} else {
		for _, song := range playlist.Songs {
			if song.URL == songIdentifier || song.TrackID == songIdentifier {
				trackID = song.TrackID
				break
			}
		}
	}

	if trackID == "" {
		_, err = m.ReplyText(c, "😕 <b>Not Found</b>\n\nThat song was not found in the playlist.", replyOpts)
		return err
	}

	err = db.Instance.RemoveSongFromPlaylist(playlistID, trackID)
	if err != nil {
		_, err = m.ReplyText(c, fmt.Sprintf("❌ <b>Remove Failed</b>\n\n<code>%s</code>", err.Error()), replyOpts)
		return err
	}

	_, err = m.ReplyText(c, fmt.Sprintf("🗑 <b>Track Removed</b>\n\n<i>From playlist: %s</i>", playlist.Name), replyOpts)
	return err
}

func playlistInfoHandler(c *td.Client, m *td.Message) error {

	args := Args(m)
	if args == "" {
		_, err := m.ReplyText(
			c,
			"<b>Usage:</b> /playlistinfo [playlist id]",
			&td.SendTextMessageOpts{ParseMode: "HTML"},
		)
		return err
	}

	playlist, err := db.Instance.GetPlaylist(args)
	if err != nil {
		_, err = m.ReplyText(c, "❌ <b>Not Found</b>\n\nNo playlist found with that ID.", replyOpts)
		return err
	}

	var songs []string
	for i, song := range playlist.Songs {
		songs = append(songs, fmt.Sprintf("<code>%2d.</code> %s", i+1, song.Name))
	}

	owner, err := c.GetUser(playlist.UserID)
	if err != nil {
		return td.EndGroups
	}

	songList := "<i>Empty playlist</i>"
	if len(songs) > 0 {
		songList = strings.Join(songs, "\n")
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf(
			"🗂 <b>Playlist Info</b>\n"+
				"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
				"▸ <b>Name</b>   %s\n"+
				"▸ <b>Owner</b>  %s\n"+
				"▸ <b>Tracks</b> <code>%d</code>\n"+
				"▸ <b>ID</b>     <code>%s</code>\n\n"+
				"%s",
			playlist.Name,
			owner.FirstName,
			len(playlist.Songs),
			playlist.ID,
			songList,
		),
		replyOpts,
	)
	return td.EndGroups
}

func myPlaylistsHandler(c *td.Client, m *td.Message) error {

	userID := m.SenderID()

	playlists, err := db.Instance.GetUserPlaylists(userID)
	if err != nil {
		_, err := m.ReplyText(c, fmt.Sprintf("Error fetching playlists: %s", err.Error()), nil)
		return err
	}

	if len(playlists) == 0 {
		_, err := m.ReplyText(c, "📂 <b>No Playlists</b>\n\nYou haven't created any playlists yet.\n\n<i>Use /createplaylist [name] to get started.</i>", replyOpts)
		return err
	}

	var playlistInfo []string
	for i, playlist := range playlists {
		playlistInfo = append(
			playlistInfo,
			fmt.Sprintf("<code>%d.</code> <b>%s</b>  <code>%s</code>", i+1, playlist.Name, playlist.ID),
		)
	}

	_, err = m.ReplyText(
		c,
		fmt.Sprintf("🗂 <b>My Playlists</b>\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"%s", strings.Join(playlistInfo, "\n")),
		replyOpts,
	)

	return err
}
