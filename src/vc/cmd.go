package vc

import (
        "github.com/zefronxd/TGMUSIC/src/vc/ntgcalls"
        "fmt"
        "regexp"
        "strings"
)

var isURLRegex = regexp.MustCompile(`^https?://`)

// defaultAudioFilters applies single-pass, real-time loudness normalization
// (dynaudnorm, safe for live streams unlike the two-pass loudnorm filter)
// followed by a peak limiter that stops clipping/distortion without
// noticeably coloring the sound. Shared with ChangeSpeed so tempo changes
// keep the same loudness/clipping safety net.
const defaultAudioFilters = "dynaudnorm=f=200:g=15:p=0.95:m=8:s=12:r=0.9,alimiter=limit=0.97:attack=5:release=50:level=disabled"

// reconnectFlags makes ffmpeg automatically re-establish dropped or stalled
// network connections (packet loss, transient HTTP errors, idle timeouts)
// instead of dying mid-stream, which is the main cause of a stream stopping
// unexpectedly on flaky networks.
const reconnectFlags = "-reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_on_network_error 1 -reconnect_on_http_error 4xx,5xx -reconnect_delay_max 5 -multiple_requests 1 -rw_timeout 15000000 "

// lowLatencyInputFlags trims demuxer probing/buffering so playback starts
// and recovers as fast as possible while still tolerating minor stream
// corruption instead of aborting.
const lowLatencyInputFlags = "-fflags +discardcorrupt+genpts+nobuffer -flags low_delay -avoid_negative_ts make_zero -err_detect ignore_err -thread_queue_size 4096 "

// getMediaDescription creates a media description for ntgcalls based on the provided file path, video status, and ffmpeg parameters.
func getMediaDescription(filePath string, isVideo bool, ffmpegParameters string) ntgcalls.MediaDescription {
        audioDescription := &ntgcalls.AudioDescription{
                MediaSource:  ntgcalls.MediaSourceShell,
                SampleRate:   48000,
                ChannelCount: 2,
        }

        quotedPath := fmt.Sprintf("\"%s\"", filePath)
        isURL := isURLRegex.MatchString(filePath)

        var seekFlags, filterFlags string
        if ffmpegParameters != "" {
                if strings.Contains(ffmpegParameters, "filter:") {
                        filterFlags = ffmpegParameters
                } else {
                        seekFlags = ffmpegParameters
                }
        }

        var audioCmd strings.Builder
        audioCmd.WriteString("ffmpeg -hide_banner -loglevel error -nostdin ")
        audioCmd.WriteString(lowLatencyInputFlags)
        if isURL {
                audioCmd.WriteString(reconnectFlags)
                audioCmd.WriteString("-analyzeduration 0 -probesize 32k ")
        }

        if seekFlags != "" {
                audioCmd.WriteString(seekFlags + " ")
        }

        audioCmd.WriteString("-i " + quotedPath + " ")
        if filterFlags != "" {
                // Custom filter graphs (e.g. speed change) already build their own
                // -filter:a chain including loudness/clipping safety, so pass it
                // through as-is instead of also injecting -af.
                audioCmd.WriteString(filterFlags + " ")
        } else {
                audioCmd.WriteString("-af " + defaultAudioFilters + " ")
        }

        audioCmd.WriteString(fmt.Sprintf("-f s16le -ac %d -ar %d pipe:1",
                audioDescription.ChannelCount,
                audioDescription.SampleRate,
        ))
        audioDescription.Input = audioCmd.String()

        if !isVideo {
                return ntgcalls.MediaDescription{
                        Microphone: audioDescription,
                }
        }

        originalWidth, originalHeight := getVideoDimensions(filePath)

        width := 1280
        height := 720

        if originalWidth > 0 && originalHeight > 0 {
                ratio := float64(originalWidth) / float64(originalHeight)
                newW := min(originalWidth, width)
                newH := int(float64(newW) / ratio)

                if newH > height {
                        newH = height
                        newW = int(float64(newH) * ratio)
                }

                if newW%2 != 0 {
                        newW--
                }
                if newH%2 != 0 {
                        newH--
                }

                width = newW
                height = newH
        }

        videoDescription := &ntgcalls.VideoDescription{
                MediaSource: ntgcalls.MediaSourceShell,
                Width:       int16(width),
                Height:      int16(height),
                Fps:         30,
        }

        var videoCmd strings.Builder
        videoCmd.WriteString("ffmpeg -hide_banner -loglevel error -nostdin ")
        videoCmd.WriteString(lowLatencyInputFlags)
        if isURL {
                videoCmd.WriteString(reconnectFlags)
                videoCmd.WriteString("-analyzeduration 0 -probesize 32k ")
        }

        if seekFlags != "" {
                videoCmd.WriteString(seekFlags + " ")
        }

        videoCmd.WriteString(fmt.Sprintf("-i %s ", quotedPath))

        scaleFilter := fmt.Sprintf("scale=%d:%d", videoDescription.Width, videoDescription.Height)
        if filterFlags != "" {
                // filterFlags may already define -filter:v (e.g. speed change's
                // setpts) and -filter:a; ffmpeg rejects mixing -filter:v with a
                // separate -vf on the same stream, so fold the scale into it.
                if strings.Contains(filterFlags, "-filter:v") {
                        filterFlags = strings.Replace(filterFlags, "-filter:v ", "", 1)
                        parts := strings.SplitN(filterFlags, " ", 2)
                        videoCmd.WriteString(fmt.Sprintf("-vf %s,%s ", parts[0], scaleFilter))
                        if len(parts) > 1 {
                                videoCmd.WriteString(parts[1] + " ")
                        }
                } else {
                        videoCmd.WriteString(fmt.Sprintf("-vf %s ", scaleFilter))
                        videoCmd.WriteString(filterFlags + " ")
                }
        } else {
                videoCmd.WriteString(fmt.Sprintf("-vf %s ", scaleFilter))
        }

        videoCmd.WriteString(fmt.Sprintf("-f rawvideo -r %d -pix_fmt yuv420p pipe:1",
                videoDescription.Fps,
        ))
        videoDescription.Input = videoCmd.String()

        return ntgcalls.MediaDescription{
                Microphone: audioDescription,
                Camera:     videoDescription,
        }
}
