// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestNekoBTDefinition(t *testing.T) {
	t.Parallel()

	def := New()
	if def.Name() != "NEKOBT" {
		t.Fatalf("expected name NEKOBT, got %s", def.Name())
	}
}

func TestNekoBTMetadataMapping(t *testing.T) {
	t.Parallel()

	t.Run("resolves video types", func(t *testing.T) {
		cases := []struct {
			meta     api.PreparedMetadata
			expected int
		}{
			{meta: api.PreparedMetadata{Type: "REMUX", ReleaseName: "Show.S01.1080p.BDREMUX-GRP"}, expected: 14},
			{meta: api.PreparedMetadata{Type: "WEBDL", ReleaseName: "Show.S01.1080p.WEB-DL-GRP"}, expected: 9},
			{meta: api.PreparedMetadata{Type: "WEBRIP", ReleaseName: "Show.S01.1080p.WEBRip-GRP"}, expected: 8},
			{meta: api.PreparedMetadata{ReleaseName: "Show.S01.1080p.Hybrid-GRP"}, expected: 15},
			{meta: api.PreparedMetadata{ReleaseName: "Show.S01.BDRip-GRP"}, expected: 13},
			{meta: api.PreparedMetadata{DiscType: "BDMV"}, expected: 11},
			{meta: api.PreparedMetadata{DiscType: "DVD"}, expected: 16},
			{meta: api.PreparedMetadata{ReleaseName: "Show.S01.HDTV-GRP"}, expected: 3},
		}

		for _, tc := range cases {
			if got := resolveNekoBTVideoType(tc.meta); got != tc.expected {
				t.Errorf("for %s expected %d, got %d", tc.meta.ReleaseName, tc.expected, got)
			}
		}
	})

	t.Run("resolves video codecs", func(t *testing.T) {
		cases := []struct {
			meta     api.PreparedMetadata
			expected int
		}{
			{meta: api.PreparedMetadata{VideoCodec: "AVC"}, expected: 1},
			{meta: api.PreparedMetadata{VideoCodec: "HEVC"}, expected: 2},
			{meta: api.PreparedMetadata{VideoCodec: "AV1"}, expected: 3},
			{meta: api.PreparedMetadata{VideoCodec: "VP9"}, expected: 4},
			{meta: api.PreparedMetadata{VideoCodec: "MPEG-2"}, expected: 5},
		}

		for _, tc := range cases {
			if got := resolveNekoBTVideoCodec(tc.meta); got != tc.expected {
				t.Errorf("for %s expected %d, got %d", tc.meta.VideoCodec, tc.expected, got)
			}
		}
	})

	t.Run("resolves languages", func(t *testing.T) {
		langs := []string{"Japanese", "English", "Spanish", "Latin", "Chinese", "zh-hant"}
		got := resolveNekoBTLanguages(langs)
		expected := "ja,en,es-es,es-419,zh-hans,zh-hant"
		if got != expected {
			t.Fatalf("expected languages %q, got %q", expected, got)
		}
	})
}


func TestNekoBTUploadAndDryRun(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	torrentFile := filepath.Join(tmpDir, "test.torrent")
	if err := os.WriteFile(torrentFile, []byte("d8:announce41:https://tracker.nekobt.to/api/tracker...e"), 0644); err != nil {
		t.Fatalf("failed to write test torrent: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/upload" && r.Method == http.MethodPost:
			cookie := r.Header.Get("Cookie")
			if !strings.Contains(cookie, "ssid=test-token") {
				http.Error(w, `{"error":true,"message":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"error":false,"data":{"id":"9876543210"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	req := trackers.UploadRequest{
		Tracker: "NEKOBT",
		Meta: api.PreparedMetadata{
			ReleaseName:       "Show.S01.1080p.BluRay.REMUX.AVC.FLAC-GRP",
			ReleaseNameClean:  "Show S01 1080p BluRay REMUX AVC FLAC-GRP",
			TorrentPath:       torrentFile,
			Tag:               "GRP",
			Type:              "REMUX",
			VideoCodec:        "AVC",
			AudioLanguages:    []string{"ja"},
			SubtitleLanguages: []string{"en"},
			SeasonInt:         1,
			SeasonStr:         "S01",
			EpisodeInt:        0,
		},
		TrackerConfig: config.TrackerConfig{
			URL:     server.URL,
			APIKey:  "test-token",
			GroupID: "12345",
		},
	}

	def := New()

	// Dry run test
	dryRunner, ok := def.(trackers.UploadDryRunBuilder)
	if !ok {
		t.Fatalf("expected def to implement UploadDryRunBuilder")
	}
	dryRun, err := dryRunner.BuildUploadDryRun(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no dry run error, got %v", err)
	}
	if dryRun.Tracker != "NEKOBT" {
		t.Fatalf("expected dry run tracker NEKOBT, got %s", dryRun.Tracker)
	}

	// Upload test
	summary, err := def.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no upload error, got %v", err)
	}
	if len(summary.UploadedTorrents) == 0 || summary.UploadedTorrents[0].TorrentID != "9876543210" {
		t.Fatalf("expected torrent ID 9876543210, got %#v", summary)
	}
}

func TestNekoBTBuildTitleIncludesSeason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		meta     api.PreparedMetadata
		contains string
	}{
		{
			name: "series with season only",
			meta: api.PreparedMetadata{
				Release:   api.ReleaseInfo{Title: "Example Show", Year: 2026},
				SeasonInt: 2,
				SeasonStr: "S02",
			},
			contains: "S02",
		},
		{
			name: "series with season and episode",
			meta: api.PreparedMetadata{
				Release:    api.ReleaseInfo{Title: "Example Show", Year: 2026},
				SeasonInt:  1,
				SeasonStr:  "S01",
				EpisodeInt: 5,
				EpisodeStr: "E05",
			},
			contains: "S01E05",
		},
		{
			name: "movie has no season",
			meta: api.PreparedMetadata{
				Release: api.ReleaseInfo{Title: "Example Movie", Year: 2026},
			},
			contains: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title := buildTitle(tc.meta, config.TrackerConfig{})
			if tc.contains == "" {
				if strings.Contains(title, "S0") || strings.Contains(title, "E0") {
					t.Errorf("expected no season/episode in title, got %q", title)
				}
			} else {
				if !strings.Contains(title, tc.contains) {
					t.Errorf("expected %q in title, got %q", tc.contains, title)
				}
			}
		})
	}
}

func TestNekoBTBuildDescription(t *testing.T) {
	t.Parallel()

	req := trackers.UploadRequest{}

	assets := trackers.DescriptionAssets{
		Description: "**Repack definitivo.**\n- Se solucionó el problema con las fuentes.",
		Screenshots: []api.ScreenshotImage{
			{WebURL: "https://ibb.co/shot1", ImgURL: "https://i.ibb.co/shot1.png"},
			{WebURL: "https://ibb.co/shot2", ImgURL: "https://i.ibb.co/shot2.png"},
			{WebURL: "https://ibb.co/shot3", ImgURL: "https://i.ibb.co/shot3.png"},
		},
	}

	desc := buildDescription(req, assets)

	// Check base description preserved
	if !strings.Contains(desc, "**Repack definitivo.**") {
		t.Error("expected base description in result")
	}

	// Check centered screenshots with width="400"
	if !strings.Contains(desc, `<div align="center">`) {
		t.Error("expected <div align=\"center\"> in result")
	}
	if !strings.Contains(desc, `<a href="https://ibb.co/shot1"><img src="https://i.ibb.co/shot1.png" width="400"></a>`) {
		t.Error("expected formatted screenshot 1")
	}
	if !strings.Contains(desc, `<a href="https://ibb.co/shot2"><img src="https://i.ibb.co/shot2.png" width="400"></a>`) {
		t.Error("expected formatted screenshot 2")
	}
	if !strings.Contains(desc, `<a href="https://ibb.co/shot3"><img src="https://i.ibb.co/shot3.png" width="400"></a>`) {
		t.Error("expected formatted screenshot 3")
	}

	// Check signature
	if !strings.Contains(desc, `<div align="right">`) || !strings.Contains(desc, "[Uploaded by upbrr](https://github.com/autobrr/upbrr)") {
		t.Error("expected upbrr signature in result")
	}

	// Check idempotence - building again should not duplicate
	assets.Description = desc
	desc2 := buildDescription(req, assets)
	if strings.Count(desc2, "<div align=\"center\">") != 1 {
		t.Errorf("expected exactly 1 center div, got %d", strings.Count(desc2, "<div align=\"center\">"))
	}
	if strings.Count(desc2, "<div align=\"right\">") != 1 {
		t.Errorf("expected exactly 1 signature div, got %d", strings.Count(desc2, "<div align=\"right\">"))
	}
}

func TestNekoBTPayloadIncludesMediaInfo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	torrentFile := filepath.Join(tmpDir, "test.torrent")
	if err := os.WriteFile(torrentFile, []byte("d8:announce41:https://tracker.nekobt.to/api/tracker...e"), 0644); err != nil {
		t.Fatalf("failed to write test torrent: %v", err)
	}
	miPath := filepath.Join(tmpDir, "mediainfo.txt")
	if err := os.WriteFile(miPath, []byte("General\nFormat : Matroska\nDuration : 24 min"), 0600); err != nil {
		t.Fatalf("failed to write dummy mediainfo: %v", err)
	}

	req := trackers.UploadRequest{
		Tracker: "NEKOBT",
		Meta: api.PreparedMetadata{
			ReleaseName:       "Show.S01.1080p.BluRay.REMUX.AVC.FLAC-GRP",
			ReleaseNameClean:  "Show S01 1080p BluRay REMUX AVC FLAC-GRP",
			TorrentPath:       torrentFile,
			MediaInfoTextPath: miPath,
			Tag:               "GRP",
			Type:              "REMUX",
			VideoCodec:        "AVC",
		},
		TrackerConfig: config.TrackerConfig{
			APIKey: "test-token",
		},
	}

	payload, err := preparePayload(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected preparePayload error: %v", err)
	}
	if !strings.Contains(payload.MediaInfo, "Format : Matroska") {
		t.Errorf("expected payload.MediaInfo to contain mediainfo, got %q", payload.MediaInfo)
	}
}

func TestNekoBTSpanishDialectResolution(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// 1. MediaInfo JSON with both Castilian and Latino tracks
	bothJSON := `{
		"media": {
			"track": [
				{"@type": "Audio", "Language": "ja", "Title": "Japanese"},
				{"@type": "Audio", "Language": "es", "Title": "Español Castellano"},
				{"@type": "Audio", "Language": "es-419", "Title": "Español Latino"},
				{"@type": "Text", "Language": "es", "Title": "Subtítulos Castellano"},
				{"@type": "Text", "Language": "spa-419", "Title": "Subtítulos Latino"}
			]
		}
	}`
	bothPath := filepath.Join(tmpDir, "both.json")
	if err := os.WriteFile(bothPath, []byte(bothJSON), 0600); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}

	metaBoth := api.PreparedMetadata{
		MediaInfoJSONPath: bothPath,
		AudioLanguages:    []string{"Japanese", "Spanish"},
	}

	audioLangs := resolveNekoBTTrackLanguages(metaBoth, "audio", metaBoth.AudioLanguages)
	if !strings.Contains(audioLangs, "es-es") || !strings.Contains(audioLangs, "es-419") || !strings.Contains(audioLangs, "ja") {
		t.Errorf("expected ja, es-es, and es-419 for audio, got %q", audioLangs)
	}

	subLangs := resolveNekoBTTrackLanguages(metaBoth, "text", metaBoth.SubtitleLanguages)
	if !strings.Contains(subLangs, "es-es") || !strings.Contains(subLangs, "es-419") {
		t.Errorf("expected es-es and es-419 for subs, got %q", subLangs)
	}

	// 2. MediaInfo JSON with only Castilian
	castJSON := `{
		"media": {
			"track": [
				{"@type": "Audio", "Language": "ja"},
				{"@type": "Audio", "Language": "es", "Title": "Castellano"}
			]
		}
	}`
	castPath := filepath.Join(tmpDir, "cast.json")
	if err := os.WriteFile(castPath, []byte(castJSON), 0600); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}
	metaCast := api.PreparedMetadata{
		MediaInfoJSONPath: castPath,
		AudioLanguages:    []string{"Japanese", "Spanish"},
	}
	audioCast := resolveNekoBTTrackLanguages(metaCast, "audio", metaCast.AudioLanguages)
	if !strings.Contains(audioCast, "es-es") || strings.Contains(audioCast, "es-419") {
		t.Errorf("expected es-es only without es-419, got %q", audioCast)
	}

	// 3. MediaInfo JSON with only Latino
	latJSON := `{
		"media": {
			"track": [
				{"@type": "Audio", "Language": "ja"},
				{"@type": "Audio", "Language": "es", "Title": "Español Latino"}
			]
		}
	}`
	latPath := filepath.Join(tmpDir, "lat.json")
	if err := os.WriteFile(latPath, []byte(latJSON), 0600); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}
	metaLat := api.PreparedMetadata{
		MediaInfoJSONPath: latPath,
		AudioLanguages:    []string{"Japanese", "Spanish"},
	}
	audioLat := resolveNekoBTTrackLanguages(metaLat, "audio", metaLat.AudioLanguages)
	if !strings.Contains(audioLat, "es-419") || strings.Contains(audioLat, "es-es") {
		t.Errorf("expected es-419 only without es-es, got %q", audioLat)
	}
}



