// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackers"
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
			expected: "Play Dead 2025 1080p WEBRip x264 ESP AAC LC 2.0 SUBS-JOARA",
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
		{
			name: "English audio uses ENG and not ING",
			meta: api.PreparedMetadata{
				ReleaseName: "The Movie 2024 1080p BluRay x264 ENG DD 5.1 SUBS-GRP",
				Release: api.ReleaseInfo{
					Title:      "The Movie",
					Year:       2024,
					Resolution: "1080p",
					Type:       "BLURAY",
					Group:      "GRP",
				},
				VideoCodec:        "x264",
				Audio:             "DD",
				Channels:          "5.1",
				AudioLanguages:    []string{"English"},
				SubtitleLanguages: []string{"Spanish"},
			},
			expected: "The Movie 2024 1080p BluRay x264 ENG DD 5.1 SUBS-GRP",
		},
		{
			name: "Year resolved from TMDB when Release.Year is 0",
			meta: api.PreparedMetadata{
				ReleaseName: "The.World.Is.Dancing.S01.1080p.WEB-DL.AAC2.0.H.264-kotopi",
				Release: api.ReleaseInfo{
					Title:      "The World Is Dancing",
					Resolution: "1080p",
					Type:       "WEBDL",
					Category:   "TV",
					Group:      "kotopi",
				},
				SeasonStr:  "S01",
				VideoCodec: "H.264",
				Audio:      "AAC",
				Channels:   "2.0",
				ExternalMetadata: api.ExternalMetadata{
					TMDB: &api.TMDBMetadata{
						Title: "The World Is Dancing",
						Year:  2025,
					},
				},
			},
			expected: "The World Is Dancing S01 2025 1080p WEB-DL H.264 AAC 2.0-kotopi",
		},
		{
			name: "Uses Castellano title from TMDB LocalizedTitles es-ES",
			meta: api.PreparedMetadata{
				ReleaseName: "Kusuriya.no.Hitorigoto.S03.1080p.CR.WEBRip.AV1.AAC.2.0-GapMoe",
				Release: api.ReleaseInfo{
					Title:      "The Apothecary Diaries",
					Resolution: "1080p",
					Type:       "WEBRIP",
					Category:   "TV",
					Group:      "GapMoe",
				},
				SeasonStr:      "S03",
				SeasonInt:      3,
				Service:        "CR",
				VideoCodec:     "AV1",
				Audio:          "AAC",
				Channels:       "2.0",
				AudioLanguages: []string{"Japanese", "Spanish"},
				ExternalMetadata: api.ExternalMetadata{
					TMDB: &api.TMDBMetadata{
						Title: "The Apothecary Diaries",
						Year:  2023,
						LocalizedTitles: map[string]string{
							"es-ES": "Los diarios de la boticaria",
							"es":    "Los diarios de la boticaria",
						},
					},
				},
			},
			expected: "Los diarios de la boticaria S03 2023 1080p CR WEBRip AV1 JAP-ESP AAC 2.0-GapMoe",
		},
		{
			name: "Kusuriya S03E02 episode matches exact user edit without year and with AAC LC",
			meta: api.PreparedMetadata{
				ReleaseName: "Kusuriya.no.Hitorigoto.S03E02.1080p.CR.WEBRip.AV1.AAC.2.0-GapMoe",
				Release: api.ReleaseInfo{
					Title:      "The Apothecary Diaries",
					Resolution: "1080p",
					Type:       "WEBRIP",
					Category:   "TV",
					Group:      "GapMoe",
				},
				EpisodeStr:        "S03E02",
				EpisodeInt:        2,
				SeasonInt:         3,
				Service:           "CR",
				VideoCodec:        "AV1",
				Audio:             "AAC LC",
				Channels:          "2.0",
				AudioLanguages:    []string{"Japanese"},
				SubtitleLanguages: []string{"Spanish"},
				ExternalMetadata: api.ExternalMetadata{
					IMDB: &api.IMDBMetadata{
						Akas: []api.IMDBAKA{
							{Country: "Spain", Title: "Los diarios de la boticaria"},
						},
					},
				},
			},
			expected: "Los diarios de la boticaria S03E02 1080p CR WEBRip AV1 JAP AAC LC 2.0 SUBS-GapMoe",
		},
		{
			name: "Kusuriya S03 season pack uses Season 1 premiere year (2023) instead of 2027",
			meta: api.PreparedMetadata{
				ReleaseName: "Kusuriya.no.Hitorigoto.S03.1080p.CR.WEBRip.AV1.AAC.2.0-GapMoe",
				Release: api.ReleaseInfo{
					Title:      "The Apothecary Diaries",
					Resolution: "1080p",
					Type:       "WEBRIP",
					Category:   "TV",
					Group:      "GapMoe",
				},
				SeasonStr:         "S03",
				SeasonInt:         3,
				Service:           "CR",
				VideoCodec:        "AV1",
				Audio:             "AAC LC",
				Channels:          "2.0",
				AudioLanguages:    []string{"Japanese"},
				SubtitleLanguages: []string{"Spanish"},
				ExternalMetadata: api.ExternalMetadata{
					AniList: &api.AniListMetadata{
						Season1Year: 2023,
						StartDate:   "2027-04-01",
						Synonyms:    []string{"Los diarios de la boticaria"},
					},
				},
			},
			expected: "Los diarios de la boticaria S03 2023 1080p CR WEBRip AV1 JAP AAC LC 2.0 SUBS-GapMoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := BuildEMUWName(tt.meta, tt.tag)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBuildUnit3DNameEMUZAlias(t *testing.T) {
	meta := api.PreparedMetadata{
		ReleaseName: "The Movie 2024 1080p BluRay x264 ENG DD 5.1 SUBS-GRP",
		Release: api.ReleaseInfo{
			Title:      "The Movie",
			Year:       2024,
			Resolution: "1080p",
			Type:       "BLURAY",
			Group:      "GRP",
		},
		VideoCodec:        "x264",
		Audio:             "DD",
		Channels:          "5.1",
		AudioLanguages:    []string{"English"},
		SubtitleLanguages: []string{"Spanish"},
	}

	gotEMUW := BuildUnit3DName("EMUW", meta, config.TrackerConfig{})
	gotEMUZ := BuildUnit3DName("EMUZ", meta, config.TrackerConfig{})
	gotEMU := BuildUnit3DName("EMU", meta, config.TrackerConfig{})

	assert.Equal(t, "The Movie 2024 1080p BluRay x264 ENG DD 5.1 SUBS-GRP", gotEMUW)
	assert.Equal(t, gotEMUW, gotEMUZ)
	assert.Equal(t, gotEMUW, gotEMU)
}

func TestEMUWUploadPayloadIDsAndMetadata(t *testing.T) {
	meta := api.PreparedMetadata{
		ReleaseName: "Kusuriya.no.Hitorigoto.S03E02.1080p.CR.WEBRip.AV1.AAC.2.0-GapMoe",
		Release: api.ReleaseInfo{
			Title:      "The Apothecary Diaries",
			Resolution: "1080p",
			Type:       "WEBRIP",
			Category:   "TV",
			Group:      "GapMoe",
		},
		EpisodeStr: "S03E02",
		EpisodeInt: 2,
		SeasonInt:  3,
		VideoCodec: "AV1",
		Audio:      "AAC LC",
		Channels:   "2.0",
		ExternalIDs: api.ExternalIDs{
			Category: "TV",
			TMDBID:   220542,
		},
		ExternalMetadata: api.ExternalMetadata{
			TMDB: &api.TMDBMetadata{
				Title:   "The Apothecary Diaries",
				YouTube: "dQw4w9WgXcQ",
			},
			AniList: &api.AniListMetadata{
				AniListID:   161645,
				Season1Year: 2023,
			},
		},
	}

	req := trackers.UploadRequest{
		Tracker: "EMUZ",
		Meta:    meta,
	}

	data, err := buildUnit3DData(req, "test_name", "test_desc", "test_mi", "")
	assert.NoError(t, err)

	assert.Equal(t, "2", data["category_id"])
	assert.Equal(t, "5", data["type_id"])
	assert.Equal(t, "3", data["resolution_id"])
	assert.Equal(t, "161645", data["anilist"])
	assert.Equal(t, "161645", data["anilist_id"])
	assert.Equal(t, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", data["youtube"])
}
