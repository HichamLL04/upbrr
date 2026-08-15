// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildEMUWName(t *testing.T) {
	tests := []struct {
		name     string
		meta     api.PreparedMetadata
		tag      string
		expected string
	}{
		{
			name: "Play Dead 2025 1080p WEBRip",
			meta: api.PreparedMetadata{
				ReleaseName: "Play Dead 2025 1080p WEBRip x264 ESP AAC LC 2.0-JOARA",
				Release: api.ReleaseInfo{
					Title:      "Play Dead",
					Year:       2025,
					Resolution: "1080p",
					Type:       "WEBRIP",
					Group:      "JOARA",
				},
				VideoCodec:        "x264",
				Audio:             "AAC LC",
				Channels:          "2.0",
				AudioLanguages:    []string{"Spanish"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "Play Dead 2025 1080p WEBRip x264 ESP AAC LC SUBS-JOARA",
		},
		{
			name: "Steel Rain 2017 1080p WEB-DL without Atmos",
			meta: api.PreparedMetadata{
				ReleaseName: "Steel Rain 2017 1080p WEB-DL HEVC LAT-COR DD+ 5.1 SUBS-MATTS",
				Release: api.ReleaseInfo{
					Title:      "Steel Rain",
					Year:       2017,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "MATTS",
				},
				VideoCodec:        "HEVC",
				Audio:             "DD+",
				Channels:          "5.1",
				AudioLanguages:    []string{"Spanish", "Korean"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "Steel Rain 2017 1080p WEB-DL HEVC ESP-COR DD+ 5.1 SUBS-MATTS",
		},
		{
			name: "El maestro del crimen 2024 1080p BluRay DUAL",
			meta: api.PreparedMetadata{
				ReleaseName: "El maestro del crimen 2024 1080p BluRay x264 DUAL DD 5.1 SUBS-EMUWAREZ",
				Release: api.ReleaseInfo{
					Title:      "El maestro del crimen",
					Year:       2024,
					Resolution: "1080p",
					Type:       "BLURAY",
				},
				VideoCodec:        "x264",
				Audio:             "DD",
				Channels:          "5.1",
				AudioLanguages:    []string{"Spanish", "English"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "El maestro del crimen 2024 1080p BluRay x264 DUAL DD 5.1 SUBS-EMUWAREZ",
		},
		{
			name: "Upload S01 TV Series",
			meta: api.PreparedMetadata{
				ReleaseName: "Upload S01 2020 1080p AMZN WEB-DL AVC DUAL DD+ 5.1 SUBS-EMUWAREZ",
				Release: api.ReleaseInfo{
					Title:      "Upload",
					Year:       2020,
					Resolution: "1080p",
					Type:       "WEBDL",
					Category:   "TV",
				},
				SeasonStr:         "S01",
				Service:           "AMZN",
				VideoCodec:        "AVC",
				Audio:             "DD+",
				Channels:          "5.1",
				AudioLanguages:    []string{"Spanish", "English"},
				SubtitleLanguages: []string{"Spanish"},
				ExternalMetadata: api.ExternalMetadata{
					TMDB: &api.TMDBMetadata{
						Title: "Upload",
					},
				},
			},
			expected: "Upload S01 2020 1080p AMZN WEB-DL AVC DUAL DD+ 5.1 SUBS-EMUWAREZ",
		},
		{
			name: "It: Bienvenidos a Derry S01E01 TV Episode",
			meta: api.PreparedMetadata{
				ReleaseName: "It: Bienvenidos a Derry S01E01 2025 1080p WEB-DL HBO AVC DUAL DD+ 5.1 SUBS-EMUWAREZ",
				Release: api.ReleaseInfo{
					Title:      "It: Bienvenidos a Derry",
					Year:       2025,
					Resolution: "1080p",
					Type:       "WEBDL",
				},
				EpisodeStr:        "S01E01",
				Service:           "HBO",
				VideoCodec:        "AVC",
				Audio:             "DD+",
				Channels:          "5.1",
				AudioLanguages:    []string{"Spanish", "English"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "It: Bienvenidos a Derry S01E01 2025 1080p HBO WEB-DL AVC DUAL DD+ 5.1 SUBS-EMUWAREZ",
		},
		{
			name: "Charlotte E02 Special Episode",
			meta: api.PreparedMetadata{
				ReleaseName: "Charlotte E02 1080p BluRay Dual-Audio Opus 2.0 x265-GapMoe",
				Release: api.ReleaseInfo{
					Title:      "Charlotte",
					Category:   "TV",
					Resolution: "1080p",
					Type:       "ENCODE",
				},
				ExternalIDs: api.ExternalIDs{
					Category: "TV",
				},
				VideoCodec:        "x265",
				Audio:             "Opus",
				Channels:          "2.0",
				AudioLanguages:    []string{"Japanese", "Spanish"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "Charlotte S00E02 1080p BluRay x265 DUAL Opus 2.0 SUBS-GapMoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := BuildEMUWName(tt.meta, tt.tag)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
