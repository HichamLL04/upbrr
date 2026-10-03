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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/metadata/metautil"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/internal/trackers/impl/commonhttp"
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
	Torrent         string               `json:"torrent"`
	Title           string               `json:"title"`
	Movie           bool                 `json:"movie"`
	Category        int                  `json:"category"`
	VideoType       int                  `json:"video_type"`
	VideoCodec      int                  `json:"video_codec"`
	Level           int                  `json:"level"`
	MTL             bool                 `json:"mtl"`
	OTL             bool                 `json:"otl"`
	Hardsub         bool                 `json:"hardsub"`
	Batch           bool                 `json:"batch"`
	Hidden          bool                 `json:"hidden"`
	Complete        bool                 `json:"complete"`
	Anonymous       bool                 `json:"anonymous"`
	AudioLangs      string               `json:"audio_langs"`
	SubLangs        string               `json:"sub_langs"`
	FansubLangs     string               `json:"fansub_langs"`
	Description     string               `json:"description"`
	MediaInfo       string               `json:"mediainfo,omitempty"`
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

func resolveNekoBTGroupTag(meta api.PreparedMetadata, cfg config.TrackerConfig) string {
	tag := strings.TrimPrefix(strings.TrimSpace(meta.Tag), "-")
	if tag != "" {
		return tag
	}
	custom := strings.TrimPrefix(strings.TrimSpace(cfg.TagForCustomRelease), "-")
	if custom != "" {
		return custom
	}
	return "GapMoe"
}

func buildTitle(meta api.PreparedMetadata, cfg config.TrackerConfig) string {
	tag := resolveNekoBTGroupTag(meta, cfg)

	title := strings.TrimSpace(meta.Release.Title)
	if title == "" {
		title = strings.TrimSpace(meta.ReleaseNameClean)
	}
	if title == "" {
		title = strings.TrimSpace(meta.ReleaseName)
	}

	// Strip " AKA ..." suffix
	if idx := strings.Index(title, " AKA "); idx != -1 {
		title = strings.TrimSpace(title[:idx])
	}
	// Strip existing [Group] prefix if present
	if strings.HasPrefix(title, "[") && strings.Contains(title, "]") {
		idx := strings.Index(title, "]")
		title = strings.TrimSpace(title[idx+1:])
	}
	// Strip trailing " -Tag" or "-Tag"
	for _, suffix := range []string{"-" + tag, " -" + tag} {
		if strings.HasSuffix(title, suffix) {
			title = strings.TrimSuffix(title, suffix)
			title = strings.TrimSpace(title)
			break
		}
	}

	year := meta.Release.Year
	if year == 0 && meta.ExternalMetadata.TMDB != nil && meta.ExternalMetadata.TMDB.Year > 0 {
		year = meta.ExternalMetadata.TMDB.Year
	}
	if year == 0 && meta.ExternalMetadata.IMDB != nil && meta.ExternalMetadata.IMDB.Year > 0 {
		year = meta.ExternalMetadata.IMDB.Year
	}

	var parts []string
	if res := strings.TrimSpace(meta.Release.Resolution); res != "" {
		parts = append(parts, res)
	}

	source := strings.TrimSpace(meta.Release.Source)
	relType := strings.TrimSpace(meta.Type)
	if strings.EqualFold(relType, "REMUX") {
		if source != "" {
			parts = append(parts, source+" REMUX")
		} else {
			parts = append(parts, "BluRay REMUX")
		}
	} else if source != "" {
		parts = append(parts, source)
	}

	if meta.Edition != "" {
		parts = append(parts, meta.Edition)
	}

	if vc := strings.TrimSpace(meta.VideoCodec); vc != "" {
		parts = append(parts, vc)
	}

	audio := strings.TrimSpace(meta.Audio)
	if audio != "" {
		channels := strings.TrimSpace(meta.Channels)
		// Only append channels if they are not already embedded in the audio string
		if channels != "" && !strings.Contains(audio, channels) {
			audio = audio + " " + channels
		}
		parts = append(parts, audio)
	}

	var sb strings.Builder
	sb.WriteString("[" + tag + "] " + title)
	if year > 0 && !strings.Contains(title, fmt.Sprintf("(%d)", year)) && !strings.Contains(title, fmt.Sprintf("%d", year)) {
		sb.WriteString(fmt.Sprintf(" (%d)", year))
	}
	// Append season/episode identifier for TV content
	seasonStr := meta.SeasonStr
	if seasonStr == "" && meta.Release.Season > 0 {
		seasonStr = fmt.Sprintf("S%02d", meta.Release.Season)
	}
	episodeStr := meta.EpisodeStr
	if episodeStr == "" && meta.Release.Episode > 0 && !meta.TVPack {
		episodeStr = fmt.Sprintf("E%02d", meta.Release.Episode)
	}
	if meta.TVPack {
		episodeStr = ""
	}
	seasonEp := strings.TrimSpace(seasonStr + episodeStr)
	if seasonEp != "" && !strings.Contains(title, seasonEp) {
		sb.WriteString(" " + seasonEp)
	}
	if len(parts) > 0 {
		sb.WriteString(" (" + strings.Join(parts, " ") + ")")
	}

	return strings.TrimSpace(sb.String())
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
	isBatch := (meta.SeasonInt > 0 || meta.Release.Season > 0) && (meta.EpisodeInt == 0 && meta.Release.Episode == 0 || meta.TVPack)

	assets, err := trackers.ResolveDescriptionAssets(ctx, req.Tracker, req.Meta, req.Repo, req.Logger)
	if err != nil {
		assets = trackers.DescriptionAssets{}
	}
	description := buildDescription(req, assets)

	logger := req.Logger
	if logger == nil {
		logger = api.NopLogger{}
	}

	title := buildTitle(meta, req.TrackerConfig)
	isOfficial := isOfficialNekoBTRelease(meta, req.TrackerConfig)
	overrides := meta.TrackerSiteOverrides.NekoBT

	audioLangs := resolveNekoBTTrackLanguages(meta, "audio", meta.AudioLanguages)
	allSubLangs := resolveNekoBTTrackLanguages(meta, "text", meta.SubtitleLanguages)

	var subLangs, fansubLangs string
	if isOfficial {
		subLangs = allSubLangs
		fansubLangs = ""
	} else {
		subLangs = ""
		fansubLangs = allSubLangs
	}
	if overrides.SubLangs != nil && strings.TrimSpace(*overrides.SubLangs) != "" {
		subLangs = strings.TrimSpace(*overrides.SubLangs)
	}
	if overrides.FansubLangs != nil && strings.TrimSpace(*overrides.FansubLangs) != "" {
		fansubLangs = strings.TrimSpace(*overrides.FansubLangs)
	}

	videoType := resolveNekoBTVideoType(meta)
	if overrides.VideoType != nil {
		videoType = *overrides.VideoType
	}

	videoCodec := resolveNekoBTVideoCodec(meta)
	if overrides.VideoCodec != nil {
		videoCodec = *overrides.VideoCodec
	}

	level := resolveNekoBTLevel(meta, isOfficial)
	if overrides.Level != nil {
		level = *overrides.Level
	} else if req.TrackerConfig.NekoBTLevel != nil {
		level = *req.TrackerConfig.NekoBTLevel
	}

	hardsub := false
	if overrides.Hardsub != nil {
		hardsub = *overrides.Hardsub
	} else if req.TrackerConfig.NekoBTHardsub != nil {
		hardsub = *req.TrackerConfig.NekoBTHardsub
	}

	mtl := false
	if overrides.MTL != nil {
		mtl = *overrides.MTL
	} else if req.TrackerConfig.NekoBTMTL != nil {
		mtl = *req.TrackerConfig.NekoBTMTL
	}

	otl := false
	if overrides.OTL != nil {
		otl = *overrides.OTL
	} else if req.TrackerConfig.NekoBTOTL != nil {
		otl = *req.TrackerConfig.NekoBTOTL
	}

	hidden := false
	if overrides.Hidden != nil {
		hidden = *overrides.Hidden
	} else if req.TrackerConfig.NekoBTHidden != nil {
		hidden = *req.TrackerConfig.NekoBTHidden
	}

	secGroups := make([]any, 0)
	rawSecGroups := overrides.SecondaryGroups
	if len(rawSecGroups) == 0 {
		rawSecGroups = req.TrackerConfig.NekoBTSecondaryGroups
	}
	for _, sg := range rawSecGroups {
		trimmedSG := strings.TrimSpace(sg)
		if trimmedSG != "" {
			parts := strings.SplitN(trimmedSG, ":", 2)
			groupID := strings.TrimSpace(parts[0])
			var role any = "Encode"
			if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
				role = strings.TrimSpace(parts[1])
			}
			secGroups = append(secGroups, map[string]any{
				"id":   groupID,
				"role": role,
			})
		}
	}

	logger.Debugf("trackers: nekobt payload title=%q tag=%q official=%t level=%d audio_langs=%q sub_langs=%q fansub_langs=%q", title, meta.Tag, isOfficial, level, audioLangs, subLangs, fansubLangs)

	miText := strings.TrimSpace(commonhttp.ReadOptionalFile(metautil.FirstNonEmptyTrimmed(meta.MediaInfoTextPath, meta.DVDVOBMediaInfoText)))
	if len(miText) > 65536 {
		miText = miText[:65536]
	}
	if len(miText) > 0 {
		logger.Debugf("trackers: nekobt mediainfo loaded length=%d bytes", len(miText))
	} else {
		logger.Warnf("trackers: nekobt mediainfo empty or not found (text_path=%q)", meta.MediaInfoTextPath)
	}

	payload := uploadPayload{
		Torrent:         base64.StdEncoding.EncodeToString(torrentBytes),
		Title:           title,
		Movie:           isMovie,
		Category:        1,
		VideoType:       videoType,
		VideoCodec:      videoCodec,
		Level:           level,
		MTL:             mtl,
		OTL:             otl,
		Hardsub:         hardsub,
		Batch:           isBatch,
		Hidden:          hidden,
		Complete:        false,
		Anonymous:       req.TrackerConfig.Anon,
		AudioLangs:      audioLangs,
		SubLangs:        subLangs,
		FansubLangs:     fansubLangs,
		Description:     description,
		MediaInfo:       miText,
		SecondaryGroups: secGroups,
		IgnoreWarnings:  true,
	}

	releaseTag := strings.TrimPrefix(strings.TrimSpace(meta.Tag), "-")
	primaryGroupID := ""
	if overrides.PrimaryGroup != nil {
		primaryGroupID = strings.TrimSpace(*overrides.PrimaryGroup)
	}
	if primaryGroupID == "" {
		primaryGroupID = resolveNekoBTGroupID(releaseTag, req.TrackerConfig)
	}
	if primaryGroupID != "" {
		payload.PrimaryGroup = &primaryGroupPayload{
			ID:      primaryGroupID,
			Members: []any{},
		}
		logger.Debugf("trackers: nekobt attached primary_group ID=%s for release group %q", primaryGroupID, releaseTag)
	} else if req.TrackerConfig.GroupID != "" {
		logger.Infof("trackers: nekobt omitting primary_group for release group %q (no matching group ID configured)", releaseTag)
	}

	return payload, nil
}

func resolveNekoBTGroupID(tag string, cfg config.TrackerConfig) string {
	tagTrimmed := strings.TrimPrefix(strings.TrimSpace(tag), "-")
	tagLower := strings.ToLower(tagTrimmed)

	// 1. Check groups mapping if configured
	if len(cfg.Groups) > 0 {
		for groupName, id := range cfg.Groups {
			if strings.EqualFold(strings.TrimSpace(groupName), tagLower) && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}

	// 2. Fall back to group_id if tag matches configured custom release tag or GapMoe
	configuredGroupTag := strings.TrimPrefix(strings.TrimSpace(cfg.TagForCustomRelease), "-")
	if configuredGroupTag == "" {
		configuredGroupTag = "GapMoe"
	}
	if tagTrimmed == "" || strings.EqualFold(tagLower, strings.ToLower(configuredGroupTag)) {
		return strings.TrimSpace(cfg.GroupID)
	}

	return ""
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return api.UploadSummary{}, wrapError("read response body", err)
	}

	var res uploadResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return api.UploadSummary{}, wrapError("parse response JSON: "+string(bodyBytes), err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || res.Error {
		errMsg := res.Message
		if errMsg == "" {
			errMsg = string(bodyBytes)
		}
		if len(res.Fails) > 0 {
			errMsg += " - " + strings.Join(res.Fails, ", ")
		}
		return api.UploadSummary{}, fmt.Errorf("trackers: nekobt upload error: %s", errMsg)
	}

	torrentID := res.Data.ID
	downloadURL := fmt.Sprintf("%s/api/v1/torrents/%s/download", baseURL, torrentID)
	torrentURL := fmt.Sprintf("%s/torrents/%s", baseURL, torrentID)

	artifactPath := ""
	if torrentID != "" {
		if artPath, err := trackers.ResolveTrackerTorrentArtifactPath(req.Meta, req.AppConfig.MainSettings.DBPath, "NEKOBT"); err == nil && artPath != "" {
			if dlErr := downloadNekoBTTorrent(ctx, downloadURL, apiKey, artPath); dlErr != nil {
				if req.Logger != nil {
					req.Logger.Warnf("trackers: nekobt download torrent failed: %v", dlErr)
				}
			} else {
				artifactPath = artPath
			}
		}
	}
	if artifactPath == "" {
		artifactPath = req.Meta.TorrentPath
	}

	return api.UploadSummary{
		Uploaded: 1,
		UploadedTorrents: []api.UploadedTorrent{{
			Tracker:     "NEKOBT",
			TorrentID:   torrentID,
			DownloadURL: downloadURL,
			TorrentURL:  torrentURL,
			TorrentPath: artifactPath,
		}},
	}, nil
}

func downloadNekoBTTorrent(ctx context.Context, urlValue, apiKey, output string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlValue, nil)
	if err != nil {
		return fmt.Errorf("trackers: nekobt build torrent download request: %w", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		httpReq.Header.Set("Cookie", "ssid="+strings.TrimSpace(apiKey))
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		return fmt.Errorf("trackers: nekobt torrent download request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("trackers: nekobt torrent download HTTP %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("trackers: nekobt read torrent response: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return fmt.Errorf("trackers: nekobt create torrent output dir: %w", err)
	}
	if err := os.WriteFile(output, body, 0o600); err != nil {
		return fmt.Errorf("trackers: nekobt write torrent output: %w", err)
	}
	return nil
}

func buildUploadDryRun(ctx context.Context, req trackers.UploadRequest) (api.TrackerDryRunEntry, error) {
	payload, err := preparePayload(ctx, req)
	if err != nil {
		return api.TrackerDryRunEntry{}, err
	}

	return api.TrackerDryRunEntry{
		Tracker:     "NEKOBT",
		Status:      "ready",
		ReleaseName: payload.Title,
		Payload: func() map[string]string {
			p := map[string]string{
				"title":       payload.Title,
				"movie":       strconv.FormatBool(payload.Movie),
				"video_type":  strconv.Itoa(payload.VideoType),
				"video_codec": strconv.Itoa(payload.VideoCodec),
				"level":       strconv.Itoa(payload.Level),
				"mtl":         strconv.FormatBool(payload.MTL),
				"batch":       strconv.FormatBool(payload.Batch),
				"audio_langs": payload.AudioLangs,
				"sub_langs":   payload.SubLangs,
			}
			if payload.PrimaryGroup != nil {
				p["primary_group"] = payload.PrimaryGroup.ID
			}
			return p
		}(),
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
