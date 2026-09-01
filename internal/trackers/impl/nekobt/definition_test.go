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
		langs := []string{"Japanese", "English", "Spanish", "Latin"}
		got := resolveNekoBTLanguages(langs)
		expected := "ja,en,es-es,es-419"
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
