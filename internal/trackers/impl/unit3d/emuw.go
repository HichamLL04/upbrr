// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func siteEMUWProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DEMUWTypeID,
		resolveResolutionID: resolveUnit3DEMUWResolutionID,
	}
}

func resolveUnit3DEMUWTypeID(meta api.PreparedMetadata) string {
	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"ENCODE": "3",
		"WEBDL":  "4",
		"WEBRIP": "5",
		"HDTV":   "6",
	}
	typeID := mapping[inferUnit3DType(meta)]
	if typeID == "" {
		return "3"
	}
	return typeID
}

func resolveUnit3DEMUWResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1080p": "3",
		"1080i": "4",
		"720p":  "5",
		"576p":  "6",
		"540p":  "7",
		"480p":  "8",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}

// BuildEMUWName dispatches release naming according to eMuwarez (EMUW) specifications.
// Formula Maestra:
// Movies: [Nombre] [Año] [Resolución] [Formato vídeo] [DV/HDR] [Códec video] [Idiomas audio] [Códec audio] [Canales audio] [SUBS] - [Tag]
// TV: [Nombre] [S## / S01E01] [Año] [Resolución] [Fuente/Servicio] [Formato] [Códec video] [Idiomas audio] [Códec audio] [Canales audio] [SUBS] - [Tag]
func BuildEMUWName(meta api.PreparedMetadata, customTag string) string {
	rawName := baseReleaseName(meta)
	if rawName == "" {
		return ""
	}

	title := resolveEMUWTitle(rawName, meta)
	yearStr := ""
	if meta.Release.Year > 0 {
		yearStr = strconv.Itoa(meta.Release.Year)
	}

	category := resolveUnit3DCategory(meta)
	isTV := category == "TV"

	seToken := formatSeasonEpisodeToken(meta, rawName)

	resolution := resolveEMUWResolution(rawName, meta)
	service := resolveLTService(rawName, meta)
	format := resolveEMUWFormat(rawName, meta)
	hdr := resolveEMUWHDR(rawName, meta)
	vcodec := resolveEMUWVCodec(rawName, meta)
	langStr := resolveEMUWAudioLangs(rawName, meta)
	acodec, channels := resolveEMUWAudioCodec(rawName, meta)
	hasSubs := resolveEMUWSubs(rawName, meta)

	tag := resolveEMUWGroupTag(rawName, meta, customTag)

	parts := make([]string, 0, 20)

	// Title
	if title != "" {
		parts = append(parts, title)
	}

	// TV Placement: Season/Episode before Year
	if isTV || seToken != "" {
		if seToken != "" {
			parts = append(parts, seToken)
		}
		if yearStr != "" {
			parts = append(parts, yearStr)
		}
	} else {
		// Movie Placement: Year
		if yearStr != "" {
			parts = append(parts, yearStr)
		}
	}

	// Resolution
	if resolution != "" {
		parts = append(parts, resolution)
	}

	edition := resolveEMUWEdition(rawName, meta)
	repack := detectLTRepackToken(rawName, meta)

	// Service & Format
	if service != "" {
		parts = append(parts, service)
	}
	if edition != "" {
		parts = append(parts, edition)
	}
	if format != "" {
		parts = append(parts, format)
	}
	if repack != "" {
		parts = append(parts, repack)
	}

	// DV/HDR before video codec
	if hdr != "" {
		parts = append(parts, hdr)
	}

	// Video Codec
	if vcodec != "" {
		parts = append(parts, vcodec)
	}

	// Audio Languages
	if langStr != "" {
		parts = append(parts, langStr)
	}

	// Audio Codec & Channels
	if acodec != "" {
		parts = append(parts, acodec)
	}
	if channels != "" {
		parts = append(parts, channels)
	}

	// Subtitles
	if hasSubs {
		parts = append(parts, "SUBS")
	}

	result := strings.Join(parts, " ")
	result = strings.TrimSpace(strings.Join(strings.Fields(result), " "))

	if tag != "" {
		result += "-" + tag
	}

	return result
}

func resolveEMUWTitle(rawName string, meta api.PreparedMetadata) string {
	return resolveLTTitle(meta)
}

func resolveEMUWResolution(rawName string, meta api.PreparedMetadata) string {
	res := resolveResolution(meta)
	switch res {
	case "4320p":
		return "4320p FUHD"
	case "2160p":
		return "2160p UHD"
	case "1080p":
		return "1080p"
	case "720p":
		return "720p"
	case "576p":
		return "576p SD"
	case "540p":
		return "540p SD"
	case "480p":
		return "480p SD"
	default:
		if strings.Contains(rawName, "2160p UHD") {
			return "2160p UHD"
		}
		if strings.Contains(rawName, "4320p FUHD") {
			return "4320p FUHD"
		}
		return res
	}
}

func resolveEMUWFormat(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "BDREMUX") || strings.Contains(upper, "BD REMUX") || strings.Contains(upper, "BLURAY REMUX") || strings.Contains(upper, "BLU-RAY REMUX") {
		return "BDRemux"
	}
	if strings.Contains(upper, "REMUX") {
		return "BDRemux"
	}
	if strings.Contains(upper, "FBD") || strings.Contains(upper, "FULL BD") {
		return "FBD"
	}
	if strings.Contains(upper, "FDVD") || strings.Contains(upper, "FULL DVD") {
		return "FDVD"
	}
	if strings.Contains(upper, "MHD") || strings.Contains(upper, "MICRO HD") {
		return "MHD"
	}
	if strings.Contains(upper, "WEB-DL") {
		return "WEB-DL"
	}
	if strings.Contains(upper, "WEBRIP") {
		return "WEBRip"
	}
	if strings.Contains(upper, "BLURAY") || strings.Contains(upper, "BLU-RAY") {
		return "BluRay"
	}
	if strings.Contains(upper, "HDTV") {
		return "HDTV"
	}

	typeVal := inferUnit3DType(meta)
	switch typeVal {
	case "WEBDL":
		return "WEB-DL"
	case "WEBRIP":
		return "WEBRip"
	case "REMUX":
		return "BDRemux"
	case "ENCODE", "BLURAY":
		return "BluRay"
	default:
		return typeVal
	}
}

func resolveEMUWEdition(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	sourceUpper := strings.ToUpper(meta.SourcePath)
	cleanUpper := strings.ToUpper(meta.ReleaseNameClean)
	editionUpper := strings.ToUpper(meta.Edition)
	if strings.Contains(upper, "HYBRID") || strings.Contains(sourceUpper, "HYBRID") || strings.Contains(cleanUpper, "HYBRID") || meta.WebDV || strings.Contains(editionUpper, "HYBRID") || strings.Contains(upper, "CUSTOM") || strings.Contains(sourceUpper, "CUSTOM") || strings.Contains(cleanUpper, "CUSTOM") || strings.Contains(editionUpper, "CUSTOM") {
		return "CUSTOM"
	}
	return ""
}

func resolveEMUWHDR(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "DV HDR") {
		return "DV HDR"
	}
	if strings.Contains(upper, "DV") {
		return "DV"
	}
	if strings.Contains(upper, "HDR") {
		return "HDR"
	}
	return strings.TrimSpace(meta.HDR)
}

func resolveEMUWVCodec(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "X264") {
		return "x264"
	}
	if strings.Contains(upper, "X265") {
		return "x265"
	}
	if strings.Contains(upper, "HEVC") {
		return "HEVC"
	}
	if strings.Contains(upper, "AVC") {
		return "AVC"
	}
	if strings.Contains(upper, "AV1") {
		return "AV1"
	}
	if strings.Contains(upper, "VC-1") {
		return "VC-1"
	}
	if strings.Contains(upper, "MPEG") {
		return "MPEG"
	}
	if meta.VideoCodec != "" {
		return meta.VideoCodec
	}
	return ""
}

func resolveEMUWAudioLangs(rawName string, meta api.PreparedMetadata) string {
	langs := []string{}
	seen := map[string]bool{}

	for _, lang := range meta.AudioLanguages {
		normalized := strings.ToLower(languageutil.NormalizeLanguageDisplay(lang))
		if normalized == "" {
			normalized = strings.ToLower(strings.TrimSpace(lang))
		}
		var code string
		if isSpanishLanguageToken(normalized) {
			if strings.Contains(normalized, "castellano") || strings.Contains(normalized, "castilian") {
				code = "ESP"
			} else {
				code = "ESP"
			}
		} else if strings.Contains(normalized, "latin") || strings.Contains(normalized, "latino") {
			code = "LAT"
		} else if strings.Contains(normalized, "english") || strings.Contains(normalized, "eng") {
			code = "ING"
		} else if strings.Contains(normalized, "french") || strings.Contains(normalized, "fra") {
			code = "FRA"
		} else if strings.Contains(normalized, "german") || strings.Contains(normalized, "ale") || strings.Contains(normalized, "deu") {
			code = "ALE"
		} else if strings.Contains(normalized, "japanese") || strings.Contains(normalized, "jap") {
			code = "JAP"
		} else if strings.Contains(normalized, "korean") || strings.Contains(normalized, "cor") || strings.Contains(normalized, "kor") {
			code = "COR"
		} else if strings.Contains(normalized, "catalan") || strings.Contains(normalized, "cat") {
			code = "CA"
		} else if strings.Contains(normalized, "basque") || strings.Contains(normalized, "eus") || strings.Contains(normalized, "baq") {
			code = "EUS"
		} else if strings.Contains(normalized, "galician") || strings.Contains(normalized, "glg") {
			code = "GLG"
		} else if normalized != "" {
			code = strings.ToUpper(normalized)
			if len(code) > 3 {
				code = code[:3]
			}
		}
		if code != "" && !seen[code] {
			seen[code] = true
			langs = append(langs, code)
		}
	}

	// 4 or more unique audio languages = MULTI
	if len(langs) >= 4 {
		return "MULTI"
	}

	upper := strings.ToUpper(rawName)

	if len(langs) == 3 {
		return langs[0] + "-" + langs[1] + "-" + langs[2]
	}
	if len(langs) == 2 {
		if strings.Contains(upper, "DUAL") {
			return "DUAL"
		}
		return langs[0] + "-" + langs[1]
	}
	if len(langs) == 1 {
		return langs[0]
	}

	if strings.Contains(upper, "MULTI") {
		return "MULTI"
	}
	if strings.Contains(upper, "DUAL") {
		return "DUAL"
	}
	if strings.Contains(upper, "ESP") {
		return "ESP"
	}
	return ""
}

func resolveEMUWAudioCodec(rawName string, meta api.PreparedMetadata) (acodec string, channels string) {
	upper := strings.ToUpper(rawName)

	// Lossless codecs take priority over lossy.
	if strings.Contains(upper, "FLAC") {
		acodec = "FLAC"
	} else if strings.Contains(upper, "PCM") {
		acodec = "PCM"
	} else if strings.Contains(upper, "MLP") {
		acodec = "MLP"
	} else if strings.Contains(upper, "TRUEHD") {
		acodec = "TrueHD"
	} else if strings.Contains(upper, "DTS-HD MA") {
		acodec = "DTS-HD MA"
	} else if strings.Contains(upper, "DTS-HD HRA") {
		acodec = "DTS-HD HRA"
	} else if strings.Contains(upper, "DTS:X") {
		acodec = "DTS:X"
	} else if strings.Contains(upper, "AAC LC") {
		acodec = "AAC LC"
	} else if strings.Contains(upper, "AAC LD") {
		acodec = "AAC LD"
	} else if strings.Contains(upper, "AAC HE") {
		acodec = "AAC HE"
	} else if strings.Contains(upper, "AAC") {
		acodec = "AAC"
	} else if strings.Contains(upper, "DD+ ATMOS") || strings.Contains(upper, "DD+ 5.1 ATMOS") {
		// Check meta.Audio to confirm Atmos before using it from rawName
		if strings.Contains(strings.ToUpper(meta.Audio), "ATMOS") {
			acodec = "DD+ Atmos"
		} else {
			acodec = "DD+"
		}
	} else if strings.Contains(upper, "DD+") || strings.Contains(upper, "DDP") {
		acodec = "DD+"
	} else if strings.Contains(upper, "DD") {
		acodec = "DD"
	} else if strings.Contains(upper, "OPUS") {
		acodec = "Opus"
	}

	chans := []string{"7.1", "6.1", "5.1", "5.0", "3.1", "3.0", "2.1", "2.0", "1.0"}
	for _, c := range chans {
		for _, field := range strings.Fields(upper) {
			if field == c {
				channels = c
				break
			}
		}
		if channels != "" {
			break
		}
	}

	return acodec, channels
}

func resolveEMUWSubs(rawName string, meta api.PreparedMetadata) bool {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "SUBS") {
		return true
	}
	return len(meta.SubtitleLanguages) > 0
}

func resolveEMUWGroupTag(rawName string, meta api.PreparedMetadata, customTag string) string {
	tag := strings.TrimPrefix(strings.TrimSpace(meta.Tag), "-")
	if tag == "" {
		tag = strings.TrimPrefix(strings.TrimSpace(customTag), "-")
	}
	if tag == "" {
		if idx := strings.LastIndex(rawName, "-"); idx != -1 && idx < len(rawName)-1 {
			possible := strings.TrimSpace(rawName[idx+1:])
			if !isNoGroupTag(possible) && !strings.Contains(possible, " ") {
				tag = possible
			}
		}
	}
	if tag == "" || isNoGroupTag(tag) {
		return "EMUWAREZ"
	}
	return tag
}
