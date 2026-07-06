/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package dl

import (
	"time"

	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/zefronxd/TGMUSIC/config"
	"github.com/zefronxd/TGMUSIC/src/utils"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	urlpkg "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// youTubeData provides an interface for fetching track and playlist information from YouTube.
type youTubeData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var youtubePatterns = map[string]*regexp.Regexp{
	"youtube":   regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/.*`),
	"youtu_be":  regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtu\.be/.*`),
	"yt_music":  regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/.*`),
	"yt_shorts": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/shorts/.*`),
}

// newYouTubeData initializes a youTubeData instance with pre-compiled regex patterns and a cleaned query.
func newYouTubeData(query string) *youTubeData {
	return &youTubeData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.ApiUrl, "/"),
		APIKey:   config.ApiKey,
		Patterns: youtubePatterns,
	}
}

func (y *youTubeData) isValid() bool {
	if y.Query == "" {
		slog.Info("The query or patterns are empty.")
		return false
	}

	for _, pattern := range y.Patterns {
		if pattern.MatchString(y.Query) {
			return true
		}
	}
	return false
}

func (y *youTubeData) getInfo() (utils.PlatformTracks, error) {
	if !y.isValid() {
		return utils.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	y.Query = normalizeYouTubeURL(y.Query)
	videoID := extractVideoID(y.Query)
	playlistID := extractPlaylistID(y.Query)

	switch {
	case playlistID != "":
		if strings.HasPrefix(playlistID, "RD") {
			return getYouTubeMixPlaylist(ctx, playlistID)
		}
		return getYouTubePlaylist(ctx, playlistID)

	case videoID != "":
		for _, query := range []string{videoID, y.Query} {
			tracks, err := searchYouTube(query, 10)
			if err != nil {
				continue
			}

			for _, track := range tracks {
				if track.Id == videoID {
					return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
				}
			}
		}

		if title, err := getYouTubeTitleFromOEmbed(videoID); err == nil && title != "" {
			tracks, err := searchYouTube(title, 10)
			if err == nil {
				for _, track := range tracks {
					if track.Id == videoID {
						return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
					}
				}
			}
		}

		slog.Warn("Video ID was extracted but no matching track was found in search results", "video_id", videoID)
		return getYouTubeVideo(ctx, videoID)
	}

	return utils.PlatformTracks{}, errors.New("no video or playlist results were found")
}

func (y *youTubeData) search() (utils.PlatformTracks, error) {
	tracks, err := searchYouTube(y.Query, 5)
	if err != nil {
		return utils.PlatformTracks{}, err
	}

	if len(tracks) == 0 {
		return utils.PlatformTracks{}, errors.New("no video results were found")
	}

	return utils.PlatformTracks{Results: tracks}, nil
}

func (y *youTubeData) getTrack() (utils.TrackInfo, error) {
	if y.Query == "" {
		return utils.TrackInfo{}, errors.New("the query is empty")
	}

	if !y.isValid() {
		return utils.TrackInfo{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	if y.ApiUrl != "" && y.APIKey != "" {
		if trackInfo, err := newApiData(y.Query).getTrack(); err == nil {
			return trackInfo, nil
		}
	}

	getInfo, err := y.getInfo()
	if err != nil {
		return utils.TrackInfo{}, err
	}
	if len(getInfo.Results) == 0 {
		return utils.TrackInfo{}, errors.New("no video results were found")
	}

	track := getInfo.Results[0]
	trackInfo := utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		Platform: utils.YouTube,
	}

	return trackInfo, nil
}

// downloadTrack handles the download of a track from YouTube.
func (y *youTubeData) downloadTrack(info utils.TrackInfo, video bool) (string, error) {
	if y.ApiUrl != "" && y.APIKey != "" {
		if filePath, err := y.downloadWithApi(info.Id, video); err == nil {
			return filePath, nil
		} else {
			slog.Warn("API download failed, falling back to yt-dlp", "video_id", info.Id, "error", err)
		}
	}

	return y.downloadWithYtDlp(info.Id, video)
}

// buildYtdlpParams constructs the command-line parameters for yt-dlp to download media.
func (y *youTubeData) buildYtdlpParams(videoID string, video bool) ([]string, string) {
	outputTemplate := filepath.Join(config.DownloadsDir, "%(id)s.%(ext)s")
	var cookieFile string

	params := []string{
		"yt-dlp",
		"--no-warnings",
		"--quiet",
		"--geo-bypass",
		"--retries", "2",
		"--continue",
		"--no-part",
		"--concurrent-fragments", "3",
		"--socket-timeout", "10",
		"--throttled-rate", "100K",
		"--retry-sleep", "1",
		"--no-write-thumbnail",
		"--no-write-info-json",
		"--no-embed-metadata",
		"--no-embed-chapters",
		"--no-embed-subs",
		"--extractor-args", "youtube:player_js_version=actual",
		"-o", outputTemplate,
	}

	if video {
		formatSelector := "bestvideo[height<=720]+bestaudio/best[height<=720]"
		params = append(params, "-f", formatSelector, "--merge-output-format", "mp4")
	} else {
		params = append(params, "-f", "bestaudio[ext=m4a]/bestaudio")
	}

	cookieFile = y.getCookieFile()
	if cookieFile != "" {
		params = append(params, "--cookies", cookieFile)
	} else if config.Proxy != "" {
		params = append(params, "--proxy", config.Proxy)
	}

	videoURL := "https://www.youtube.com/watch?v=" + videoID
	params = append(params, videoURL, "--print", "after_move:filepath")

	return params, cookieFile
}

// downloadWithYtDlp downloads media from YouTube using the yt-dlp command-line tool.
func (y *youTubeData) downloadWithYtDlp(videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	ytdlpParams, cookieFile := y.buildYtdlpParams(videoID, video)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlpParams[0], ytdlpParams[1:]...)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			if cookieFile != "" && strings.Contains(stderr, "Sign in to confirm you're not a bot") {
				_ = os.Remove(cookieFile)
			}
			return "", fmt.Errorf("yt-dlp failed with exit code %d: %s", exitErr.ExitCode(), stderr)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("yt-dlp timed out for video ID: %s", videoID)
		}

		return "", fmt.Errorf("an unexpected error occurred while downloading %s: %w", videoID, err)
	}

	downloadedPathStr := strings.TrimSpace(string(output))
	if downloadedPathStr == "" {
		return "", fmt.Errorf("no output path was returned for %s", videoID)
	}

	if _, err := os.Stat(downloadedPathStr); os.IsNotExist(err) {
		return "", fmt.Errorf("the file was not found at the reported path: %s", downloadedPathStr)
	}

	return downloadedPathStr, nil
}

// getCookieFile retrieves the path to a cookie file from the configured list.
func (y *youTubeData) getCookieFile() string {
	cookiesPath := config.CookiesPath
	if len(cookiesPath) == 0 {
		return ""
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cookiesPath))))
	if err != nil {
		slog.Info("Could not generate a random number", "error", err)
		return cookiesPath[0]
	}

	return cookiesPath[n.Int64()]
}

// downloadWithApi downloads a track directly from the external download
// gateway (e.g. "{ApiUrl}/download?url={videoID}&type=audio|video&api_key={APIKey}"),
// which streams the media file directly rather than returning JSON metadata.
func (y *youTubeData) downloadWithApi(videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	mediaType, ext := "audio", "m4a"
	if video {
		mediaType, ext = "video", "mp4"
	}

	if err := os.MkdirAll(config.DownloadsDir, defaultDownloadDirPerm); err != nil {
		return "", fmt.Errorf("failed to create the downloads directory: %w", err)
	}

	filePath := filepath.Join(config.DownloadsDir, videoID+"."+ext)
	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		return filePath, nil
	}

	fullURL := fmt.Sprintf("%s/download?%s", y.ApiUrl, urlpkg.Values{
		"url":     {videoID},
		"type":    {mediaType},
		"api_key": {y.APIKey},
	}.Encode())

	resp, err := sendRequest(http.MethodGet, fullURL, nil, nil)
	if err != nil {
		return "", fmt.Errorf("the API download request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code from the API: %s", resp.Status)
	}

	tempPath := filePath + ".part"
	if err := appendToFile(tempPath, resp.Body, false); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to write the downloaded file: %w", err)
	}

	if stat, err := os.Stat(tempPath); err != nil || stat.Size() == 0 {
		_ = os.Remove(tempPath)
		return "", errors.New("the downloaded file is empty")
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to finalize the downloaded file: %w", err)
	}

	return filePath, nil
}
