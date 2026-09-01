// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/autobrr/upbrr/internal/redaction"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

var nekoBTExtraAnnounces = []string{
	"https://tracker.nekobt.to/api/tracker/public/announce",
	"https://tracker.zhuqiy.com:443/announce",
	"https://tracker.pmman.tech:443/announce",
	"https://tracker.nekomi.cn:443/announce",
	"https://tracker.leechshield.link:443/announce",
	"https://tracker.openbittorrent.com:80/announce",
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://explodie.org:6969/announce",
}

const (
	defaultBaseURL = "https://nekobt.to"
	defaultTimeout = 60 * time.Second
)

type primaryGroupPayload struct {
	ID      string `json:"id"`
	Members []any  `json:"members"`
}

type uploadPayload struct {
	Torrent        string               `json:"torrent"`
	Title          string               `json:"title"`
	Movie          bool                 `json:"movie"`
	Category       int                  `json:"category"`
	VideoType      int                  `json:"video_type"`
	VideoCodec     int                  `json:"video_codec"`
	Level          int                  `json:"level"`
	MTL            bool                 `json:"mtl"`
	OTL            bool                 `json:"otl"`
	Hardsub        bool                 `json:"hardsub"`
	Batch          bool                 `json:"batch"`
	Hidden         bool                 `json:"hidden"`
	Complete       bool                 `json:"complete"`
	Anonymous      bool                 `json:"anonymous"`
	AudioLangs     string               `json:"audio_langs"`
	SubLangs       string               `json:"sub_langs"`
	FansubLangs    string               `json:"fansub_langs"`
	Description    string               `json:"description"`
	MediaInfo      string               `json:"mediainfo,omitempty"`
	PrimaryGroup    *primaryGroupPayload `json:"primary_group,omitempty"`
	SecondaryGroups []any                `json:"secondary_groups"`
	IgnoreWarnings  bool                 `json:"ignore_warnings,omitempty"`
}

type uploadResponse struct {
	Error   bool     `json:"error"`
	Message string   `json:"message,omitempty"`
	Fails   []string `json:"fails,omitempty"`
	Warns   []string `json:"warns,omitempty"`
	Data    struct {
		ID string `json:"id"`
	} `json:"data"`
}

func resolveBaseURL(cfg trackers.UploadRequest) string {
	if strings.TrimSpace(cfg.TrackerConfig.URL) != "" {
		return strings.TrimRight(strings.TrimSpace(cfg.TrackerConfig.URL), "/")
	}
	return defaultBaseURL
}

func buildTitle(meta api.PreparedMetadata) string {
	tag := strings.TrimSpace(meta.Tag)
	cleanName := strings.TrimSpace(meta.ReleaseNameClean)
	if cleanName == "" {
		cleanName = strings.TrimSpace(meta.ReleaseName)
	}

	if tag != "" && !strings.HasPrefix(cleanName, "["+tag+"]") {
		return "[" + tag + "] " + cleanName
	}
	return cleanName
}

func preparePayload(ctx context.Context, req trackers.UploadRequest) (uploadPayload, error) {
	meta := req.Meta
	torrentPath := strings.TrimSpace(meta.TorrentPath)
	if torrentPath == "" {
		var err error
		torrentPath, err = trackers.ResolveUploadTorrentPath(meta, req.AppConfig.MainSettings.DBPath)
		if err != nil {
			return uploadPayload{}, fmt.Errorf("trackers: %w", err)
		}
	}

	torrentBytes, err := prepareNekoBTTorrentBytes(torrentPath, req.TrackerConfig.AnnounceURL)
	if err != nil {
		torrentBytes, err = os.ReadFile(torrentPath)
		if err != nil {
			return uploadPayload{}, wrapError("read torrent file", err)
		}
	}

	isMovie := meta.SeasonInt == 0 && meta.EpisodeInt == 0 && !meta.HasTVSeasonEpisodeSignal()
	isBatch := meta.SeasonInt > 0 && meta.EpisodeInt == 0

	assets, err := trackers.ResolveDescriptionAssets(ctx, req.Tracker, req.Meta, req.Repo, req.Logger)
	if err != nil {
		assets = trackers.DescriptionAssets{}
	}
	description := buildDescription(req, assets)

	var mediainfo string
	if meta.MediaInfoTextPath != "" {
		if b, err := os.ReadFile(meta.MediaInfoTextPath); err == nil {
			mediainfo = string(b)
		}
	}

	payload := uploadPayload{
		Torrent:        base64.StdEncoding.EncodeToString(torrentBytes),
		Title:          buildTitle(meta),
		Movie:          isMovie,
		Category:       1,
		VideoType:      resolveNekoBTVideoType(meta),
		VideoCodec:     resolveNekoBTVideoCodec(meta),
		Level:          resolveNekoBTLevel(meta),
		MTL:            meta.HasIA,
		OTL:            false,
		Hardsub:        false,
		Batch:          isBatch,
		Hidden:         false,
		Complete:       false,
		Anonymous:      req.TrackerConfig.Anon,
		AudioLangs:     resolveNekoBTLanguages(meta.AudioLanguages),
		SubLangs:       resolveNekoBTLanguages(meta.SubtitleLanguages),
		FansubLangs:    "",
		Description:     description,
		MediaInfo:       mediainfo,
		SecondaryGroups: []any{},
		IgnoreWarnings:  true,
	}

	if groupID := strings.TrimSpace(req.TrackerConfig.GroupID); groupID != "" {
		payload.PrimaryGroup = &primaryGroupPayload{
			ID:      groupID,
			Members: []any{},
		}
	}

	return payload, nil
}

func upload(ctx context.Context, req trackers.UploadRequest) (api.UploadSummary, error) {
	payload, err := preparePayload(ctx, req)
	if err != nil {
		return api.UploadSummary{}, err
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return api.UploadSummary{}, wrapError("marshal payload", err)
	}

	baseURL := resolveBaseURL(req)
	uploadURL := baseURL + "/api/v1/upload"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return api.UploadSummary{}, wrapError("create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	apiKey := strings.TrimSpace(req.TrackerConfig.APIKey)
	if apiKey != "" {
		httpReq.Header.Set("Cookie", "ssid="+apiKey)
	}

	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return api.UploadSummary{}, wrapError("execute request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return api.UploadSummary{}, wrapError("read response", err)
	}

	if resp.StatusCode >= 400 {
		return api.UploadSummary{}, fmt.Errorf("trackers: nekobt upload failed status=%d: %s", resp.StatusCode, redaction.RedactValue(string(body), nil))
	}

	var res uploadResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return api.UploadSummary{}, wrapError("unmarshal response", err)
	}

	if res.Error {
		errMsg := res.Message
		if len(res.Fails) > 0 {
			errMsg += " - " + strings.Join(res.Fails, ", ")
		}
		return api.UploadSummary{}, fmt.Errorf("trackers: nekobt upload error: %s", errMsg)
	}

	torrentID := res.Data.ID
	downloadURL := fmt.Sprintf("%s/api/v1/torrents/%s/download", baseURL, torrentID)
	torrentURL := fmt.Sprintf("%s/torrents/%s", baseURL, torrentID)

	return api.UploadSummary{
		Uploaded: 1,
		UploadedTorrents: []api.UploadedTorrent{{
			Tracker:     "NEKOBT",
			TorrentID:   torrentID,
			DownloadURL: downloadURL,
			TorrentURL:  torrentURL,
		}},
	}, nil
}

func buildUploadDryRun(ctx context.Context, req trackers.UploadRequest) (api.TrackerDryRunEntry, error) {
	payload, err := preparePayload(ctx, req)
	if err != nil {
		return api.TrackerDryRunEntry{}, err
	}

	return api.TrackerDryRunEntry{
		Tracker: "NEKOBT",
		Status:  "ready",
		Payload: map[string]string{
			"title":       payload.Title,
			"movie":       strconv.FormatBool(payload.Movie),
			"video_type":  strconv.Itoa(payload.VideoType),
			"video_codec": strconv.Itoa(payload.VideoCodec),
			"level":       strconv.Itoa(payload.Level),
			"mtl":         strconv.FormatBool(payload.MTL),
			"batch":       strconv.FormatBool(payload.Batch),
			"audio_langs": payload.AudioLangs,
			"sub_langs":   payload.SubLangs,
		},
	}, nil
}

func prepareNekoBTTorrentBytes(torrentPath string, customAnnounce string) ([]byte, error) {
	torrentMeta, err := metainfo.LoadFromFile(torrentPath)
	if err != nil {
		return nil, err
	}
	announce := strings.TrimSpace(customAnnounce)
	if announce == "" {
		announce = "https://tracker.nekobt.to/api/tracker/public/announce"
	}
	torrentMeta.Announce = announce

	var announceList metainfo.AnnounceList
	announceList = append(announceList, []string{announce})
	for _, extra := range nekoBTExtraAnnounces {
		if !strings.EqualFold(extra, announce) {
			announceList = append(announceList, []string{extra})
		}
	}
	torrentMeta.AnnounceList = announceList

	var buf bytes.Buffer
	if err := bencode.NewEncoder(&buf).Encode(torrentMeta); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

