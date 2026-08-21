// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildLTName(t *testing.T) {
	tests := []struct {
		name     string
		meta     api.PreparedMetadata
		tag      string
		expected string
	}{
		{
			name: "Jackass 2.5 WEB-DL",
			meta: api.PreparedMetadata{
				ReleaseName: "Jackass 2.5 2007 480p PMTP WEB-DL AAC 2.0 H.264-Muffin",
				Release: api.ReleaseInfo{
					Title:      "Jackass 2.5",
					Year:       2007,
					Resolution: "480p",
					Type:       "WEBDL",
					Group:      "Muffin",
				},
				Service:        "PMTP",
				VideoCodec:     "H.264",
				Audio:          "AAC",
				Channels:       "2.0",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Jackass 2.5 2007 480p PMTP WEB-DL AAC 2.0 H.264 [CAST]-Muffin",
		},
		{
			name: "Violet Evergarden Recollections NF WEB-DL",
			meta: api.PreparedMetadata{
				ReleaseName: "Violet Evergarden: Recollections 2021 1080p NF WEB-DL DD+ 2.0 H.264-NAN0",
				Release: api.ReleaseInfo{
					Title:      "Violet Evergarden: Recollections",
					Year:       2021,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "NAN0",
				},
				Service:        "NF",
				VideoCodec:     "H.264",
				Audio:          "DD+",
				Channels:       "2.0",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Violet Evergarden: Recollections 2021 1080p NF WEB-DL DD+ 2.0 H.264 [CAST]-NAN0",
		},
		{
			name: "Leave the World Behind NF WEB-DL Atmos DV HDR",
			meta: api.PreparedMetadata{
				ReleaseName: "Leave the World Behind 2023 1080p NF WEB-DL DD+ 5.1 Atmos DV HDR H.265-Kitsune",
				Release: api.ReleaseInfo{
					Title:      "Leave the World Behind",
					Year:       2023,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "Kitsune",
				},
				Service:        "NF",
				HDR:            "DV HDR",
				VideoCodec:     "H.265",
				Audio:          "DD+ Atmos",
				Channels:       "5.1",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Leave the World Behind 2023 1080p NF WEB-DL DD+ 5.1 Atmos DV HDR H.265 [CAST]-Kitsune",
		},
		{
			name: "Bird WEB-DL [CAST]",
			meta: api.PreparedMetadata{
				ReleaseName: "Bird 2024 1080p WEB-DL AAC 2.0 H.264 [CAST]-LatTeam",
				Release: api.ReleaseInfo{
					Title:      "Bird",
					Year:       2024,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "LatTeam",
				},
				VideoCodec:     "H.264",
				Audio:          "AAC",
				Channels:       "2.0",
				AudioLanguages: []string{"Castellano"},
			},
			expected: "Bird 2024 1080p WEB-DL AAC 2.0 H.264 [CAST]-LatTeam",
		},
		{
			name: "The Effects of Lying AMZN WEB-DL [SUBS]",
			meta: api.PreparedMetadata{
				ReleaseName: "The Effects of Lying 2023 1080p AMZN WEB-DL DD+ 5.1 H.264 [SUBS]-BiOMA",
				Release: api.ReleaseInfo{
					Title:      "The Effects of Lying",
					Year:       2023,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "BiOMA",
				},
				Service:           "AMZN",
				VideoCodec:        "H.264",
				Audio:             "DD+",
				Channels:          "5.1",
				AudioLanguages:    []string{"English"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "The Effects of Lying 2023 1080p AMZN WEB-DL DD+ 5.1 H.264 [SUBS]-BiOMA",
		},
		{
			name: "Dragon Ball Super Super Hero AMZN WEB-DL REPACK without Atmos",
			meta: api.PreparedMetadata{
				ReleaseName: "Dragon Ball Super: Super Hero 2022 REPACK 1080p AMZN WEB-DL DD+ 5.1 H.264-Kitsune",
				Release: api.ReleaseInfo{
					Title:      "Dragon Ball Super: Super Hero",
					Year:       2022,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "Kitsune",
				},
				Service:        "AMZN",
				VideoCodec:     "H.264",
				Audio:          "DD+",
				Channels:       "5.1",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Dragon Ball Super: Super Hero 2022 REPACK 1080p AMZN WEB-DL DD+ 5.1 H.264 [CAST]-Kitsune",
		},
		{
			name: "Wednesday TV Season S01",
			meta: api.PreparedMetadata{
				ReleaseName: "Wednesday 2022 S01 1080p NF WEB-DL DD+ 5.1 Atmos H.264-Muffin",
				Release: api.ReleaseInfo{
					Title:      "Wednesday",
					Year:       2022,
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "Muffin",
				},
				SeasonStr:      "S01",
				Service:        "NF",
				VideoCodec:     "H.264",
				Audio:          "DD+ Atmos",
				Channels:       "5.1",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Wednesday S01 1080p NF WEB-DL DD+ 5.1 Atmos H.264 [CAST]-Muffin",
		},
		{
			name: "Wednesday TV Episode S01E01",
			meta: api.PreparedMetadata{
				ReleaseName: "Wednesday S01E01 1080p NF WEB-DL DD+ 5.1 Atmos H.264-Muffin",
				Release: api.ReleaseInfo{
					Title:      "Wednesday",
					Resolution: "1080p",
					Type:       "WEBDL",
					Group:      "Muffin",
				},
				EpisodeStr:     "S01E01",
				Service:        "NF",
				VideoCodec:     "H.264",
				Audio:          "DD+ Atmos",
				Channels:       "5.1",
				AudioLanguages: []string{"Spanish"},
			},
			expected: "Wednesday S01E01 1080p NF WEB-DL DD+ 5.1 Atmos H.264 [CAST]-Muffin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := BuildLTName(tt.meta, tt.tag)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestLTDescriptionIA(t *testing.T) {
	t.Run("by tag", func(t *testing.T) {
		meta := api.PreparedMetadata{
			ReleaseName: "Example Release 2026 1080p WEB-DL DD+ 5.1 H.264-GapMoe-ia",
			Tag:         "GapMoe-ia",
		}
		assert.True(t, hasIAToken(meta))
	})

	t.Run("by CLI argument flag", func(t *testing.T) {
		meta := api.PreparedMetadata{
			ReleaseName: "Example Release 2026 1080p WEB-DL DD+ 5.1 H.264-GapMoe",
			Tag:         "GapMoe",
			HasIA:       true,
		}
		assert.True(t, hasIAToken(meta))
	})
}


