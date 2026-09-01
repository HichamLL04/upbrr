// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func siteNOBSProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DNOBSTypeID,
		resolveResolutionID: resolveUnit3DNOBSResolutionID,
	}
}

func resolveUnit3DNOBSTypeID(meta api.PreparedMetadata) string {
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

func resolveUnit3DNOBSResolutionID(meta api.PreparedMetadata) string {
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

// BuildNOBSName builds release names matching NOBS specifications.
// Movies: [Nombre] [Año] [UHD] [Calidad] [Resolución] [REPACK] [Codec] [Bits] [HDR] [Idiomas] [Codec_Audio] [Canales] [SUBS]
// Series: [Nombre] [(Miniserie)] [Año] [SXX / SXXEXX] [Calidad] [Resolución] [REPACK] [Idiomas] [Codec_Audio] [Canales] [SUBS]
func BuildNOBSName(meta api.PreparedMetadata, customTag string) string {
	rawName := baseReleaseName(meta)
	if rawName == "" {
		return ""
	}

	title := resolveLTTitle(meta)
	yearStr := ""
	if meta.Release.Year > 0 {
		yearStr = strconv.Itoa(meta.Release.Year)
	}

	category := resolveUnit3DCategory(meta)
	isTV := category == "TV"
	seToken := formatSeasonEpisodeToken(meta, rawName)

	isMiniseries := strings.Contains(strings.ToLower(rawName), "miniserie") ||
		strings.Contains(strings.ToLower(meta.ReleaseNameClean), "miniserie") ||
		strings.Contains(strings.ToLower(meta.Edition), "miniserie")

	resolution := resolveEMUWResolution(rawName, meta)
	isUHD := strings.Contains(resolution, "2160p") || strings.Contains(resolution, "4320p") || strings.Contains(strings.ToUpper(rawName), "2160P") || strings.Contains(strings.ToUpper(rawName), "UHD")

	service := resolveLTService(rawName, meta)
	format := resolveNOBSFormat(rawName, meta)
	repack := detectLTRepackToken(rawName, meta)
	vcodec, bits := resolveNOBSVideoCodec(rawName, meta, isUHD)
	hdr := resolveNOBSHDR(rawName, meta)
	audioLangs, audioCodec, channels := resolveNOBSAudio(rawName, meta)
	hasSubs := resolveNOBSSubs(rawName, meta)

	parts := make([]string, 0, 20)

	// Title
	if title != "" {
		parts = append(parts, title)
	}

	// Miniseries indicator
	if isMiniseries {
		parts = append(parts, "(Miniserie)")
	}

	// Placement: TV vs Movie
	if isTV || seToken != "" {
		if yearStr != "" {
			parts = append(parts, yearStr)
		}
		if seToken != "" {
			parts = append(parts, seToken)
		}
	} else {
		if yearStr != "" {
			parts = append(parts, yearStr)
		}
	}

	if isUHD && !isTV && (format == "BDREMUX" || format == "BDRIP" || format == "BLURAY") {
		parts = append(parts, "UHD")
	}

	if service != "" {
		parts = append(parts, service)
	}
	if format != "" {
		parts = append(parts, format)
	}
	if resolution != "" {
		// Clean SD suffixes for standard display
		resClean := strings.Split(resolution, " ")[0]
		parts = append(parts, resClean)
	}
	if repack != "" {
		parts = append(parts, repack)
	}

	if vcodec != "" {
		parts = append(parts, vcodec)
	}
	if bits != "" {
		parts = append(parts, bits)
	}
	if hdr != "" {
		parts = append(parts, hdr)
	}

	if audioLangs != "" {
		parts = append(parts, audioLangs)
	}
	if audioCodec != "" {
		parts = append(parts, audioCodec)
	}
	if channels != "" {
		parts = append(parts, channels)
	}

	if hasSubs {
		if isTV {
			parts = append(parts, "SUB")
		} else {
			parts = append(parts, "SUBS")
		}
	}

	result := strings.Join(parts, " ")
	result = strings.TrimSpace(strings.Join(strings.Fields(result), " "))

	tag := resolveEMUWGroupTag(rawName, meta, customTag)
	if tag != "" {
		result += "-" + tag
	}

	return result
}

func resolveNOBSFormat(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	switch {
	case strings.Contains(upper, "BDREMUX") || strings.Contains(upper, "BD-REMUX"):
		return "BDREMUX"
	case strings.Contains(upper, "REMUX"):
		return "BDREMUX"
	case strings.Contains(upper, "BDRIP") || strings.Contains(upper, "BD-RIP"):
		return "BDRip"
	case strings.Contains(upper, "WEB-DL") || strings.Contains(upper, "WEBDL"):
		return "WEB-DL"
	case strings.Contains(upper, "WEBRIP") || strings.Contains(upper, "WEB-RIP"):
		return "WEBRip"
	case strings.Contains(upper, "HDTV"):
		return "HDTV"
	case strings.Contains(upper, "MHD") || strings.Contains(upper, "MICROHD"):
		return "MHD"
	case strings.Contains(upper, "DVD9"):
		return "DVD9"
	case strings.Contains(upper, "DVD5"):
		return "DVD5"
	case strings.Contains(upper, "DVDRIP"):
		return "DVDRip"
	case strings.Contains(upper, "BLURAY") || strings.Contains(upper, "BLU-RAY"):
		return "BluRay"
	}

	typeVal := inferUnit3DType(meta)
	switch typeVal {
	case "REMUX":
		return "BDREMUX"
	case "WEBDL":
		return "WEB-DL"
	case "WEBRIP":
		return "WEBRip"
	case "HDTV":
		return "HDTV"
	case "DVDRIP":
		return "DVDRip"
	case "BLURAY", "ENCODE":
		return "BDRip"
	}
	return typeVal
}

func resolveNOBSVideoCodec(rawName string, meta api.PreparedMetadata, isUHD bool) (codec string, bits string) {
	upper := strings.ToUpper(rawName)

	has10Bit := strings.Contains(upper, "10BIT") || strings.Contains(upper, "10-BIT") || meta.BitDepth == "10"
	has8Bit := strings.Contains(upper, "8BIT") || strings.Contains(upper, "8-BIT") || meta.BitDepth == "8"

	isHEVC := strings.Contains(upper, "X265") || strings.Contains(upper, "H265") || strings.Contains(upper, "HEVC") || strings.EqualFold(meta.VideoCodec, "HEVC") || strings.EqualFold(meta.VideoCodec, "H.265")
	isAVC := strings.Contains(upper, "X264") || strings.Contains(upper, "H264") || strings.Contains(upper, "AVC") || strings.EqualFold(meta.VideoCodec, "AVC") || strings.EqualFold(meta.VideoCodec, "H.264")
	isAV1 := strings.Contains(upper, "AV1") || strings.EqualFold(meta.VideoCodec, "AV1")

	if isAV1 {
		codec = "AV1"
	} else if isUHD {
		// UHD Standard is HEVC 10bit. Only specify if deviating.
		if isAVC {
			codec = "x264"
		}
		if has8Bit {
			bits = "8Bit"
		}
	} else {
		// HD Standard is AVC 8bit. Only specify if deviating.
		if isHEVC {
			codec = "x265"
		}
		if has10Bit {
			bits = "10Bit"
		}
	}

	return codec, bits
}

func resolveNOBSHDR(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	switch {
	case strings.Contains(upper, "DV") && (strings.Contains(upper, "HDR10+") || strings.Contains(upper, "HDR10PLUS")):
		return "DV HDR10+"
	case strings.Contains(upper, "DV") && strings.Contains(upper, "HDR"):
		return "DV HDR"
	case strings.Contains(upper, "DOVI") || strings.Contains(upper, "DOLBY VISION") || strings.Contains(upper, "DV"):
		return "DV"
	case strings.Contains(upper, "HDR10+"):
		return "HDR10+"
	case strings.Contains(upper, "HDR10") || strings.Contains(upper, "HDR"):
		return "HDR"
	case strings.Contains(upper, "SDR"):
		return "SDR"
	}
	if meta.HDR != "" {
		return meta.HDR
	}
	return ""
}

func resolveNOBSAudio(rawName string, meta api.PreparedMetadata) (audioLangs string, audioCodec string, channels string) {
	hasCast := false
	hasLat := false
	hasVO := false
	voCode := ""

	codeMap := map[string]string{
		"spanish":    "ESP",
		"castilian":  "ESP",
		"castellano": "ESP",
		"es":         "ESP",
		"es-es":      "ESP",
		"latino":     "LAT",
		"latin":      "LAT",
		"es-419":     "LAT",
		"english":    "ING",
		"en":         "ING",
		"japanese":   "JAP",
		"ja":         "JAP",
		"french":     "FRA",
		"fr":         "FRA",
		"german":     "ALE",
		"de":         "ALE",
		"portuguese": "POR",
		"pt":         "POR",
		"italian":    "ITA",
		"it":         "ITA",
	}

	seenLangs := make([]string, 0, len(meta.AudioLanguages))
	for _, lang := range meta.AudioLanguages {
		norm := strings.ToLower(languageutil.NormalizeLanguageDisplay(lang))
		if norm == "" {
			norm = strings.ToLower(strings.TrimSpace(lang))
		}
		c := codeMap[norm]
		if c == "" {
			for k, v := range codeMap {
				if strings.Contains(norm, k) {
					c = v
					break
				}
			}
		}
		if c == "" && len(norm) >= 3 {
			c = strings.ToUpper(norm[:3])
		}

		if c == "ESP" {
			hasCast = true
		} else if c == "LAT" {
			hasLat = true
		} else {
			hasVO = true
			if voCode == "" && c != "" {
				voCode = c
			}
		}

		if c != "" && !slicesContains(seenLangs, c) {
			seenLangs = append(seenLangs, c)
		}
	}

	// Audio Codec & Channels resolution
	rawUpper := strings.ToUpper(rawName)
	switch {
	case strings.Contains(rawUpper, "ATMOS"):
		audioCodec = "Dolby Atmos"
	case strings.Contains(rawUpper, "TRUEHD"):
		audioCodec = "TrueHD"
	case strings.Contains(rawUpper, "DTS:X") || strings.Contains(rawUpper, "DTS-X"):
		audioCodec = "DTS:X"
	case strings.Contains(rawUpper, "DTS-HD MA") || strings.Contains(rawUpper, "DTS-HD.MA") || strings.Contains(rawUpper, "DTS-HD"):
		audioCodec = "DTS-HD MA"
	case strings.Contains(rawUpper, "DTS-HD HRA"):
		audioCodec = "DTS-HD HRA"
	case strings.Contains(rawUpper, "EAC3") || strings.Contains(rawUpper, "DD+"):
		audioCodec = "EAC3"
	case strings.Contains(rawUpper, "AC3") || strings.Contains(rawUpper, "DD"):
		audioCodec = "AC3"
	case strings.Contains(rawUpper, "FLAC"):
		audioCodec = "FLAC"
	case strings.Contains(rawUpper, "AAC"):
		audioCodec = "AAC"
	case strings.Contains(rawUpper, "MP3"):
		audioCodec = "MP3"
	case strings.Contains(rawUpper, "DTS"):
		audioCodec = "DTS"
	default:
		audioCodec, channels = resolveEMUWAudioCodec(rawName, meta)
	}

	if channels == "" {
		if strings.Contains(rawUpper, "7.1") {
			channels = "7.1"
		} else if strings.Contains(rawUpper, "5.1") {
			channels = "5.1"
		} else if strings.Contains(rawUpper, "2.0") {
			channels = "2.0"
		}
	}

	// Mode Resolution: DUAL vs VOSE vs VO vs Lang List
	hasSubs := resolveNOBSSubs(rawName, meta)
	if hasCast && hasVO {
		if len(seenLangs) == 2 || strings.Contains(rawUpper, "DUAL") {
			audioLangs = "DUAL"
		} else {
			audioLangs = strings.Join(seenLangs, " ")
		}
	} else if hasCast && !hasVO {
		audioLangs = "ESP"
	} else if !hasCast && hasLat {
		audioLangs = "LAT"
	} else if !hasCast && !hasLat && hasVO {
		if hasSubs {
			if voCode != "" {
				audioLangs = "VOSE " + voCode
			} else {
				audioLangs = "VOSE"
			}
		} else {
			if voCode != "" {
				audioLangs = "VO " + voCode
			} else {
				audioLangs = "VO"
			}
		}
	} else if len(seenLangs) > 0 {
		audioLangs = strings.Join(seenLangs, " ")
	}

	return audioLangs, audioCodec, channels
}

func resolveNOBSSubs(rawName string, meta api.PreparedMetadata) bool {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "SUBS") || strings.Contains(upper, "SUB") || strings.Contains(upper, "VOSE") {
		return true
	}
	return len(meta.SubtitleLanguages) > 0
}

func slicesContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
