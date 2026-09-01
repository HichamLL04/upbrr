// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteMilnueveProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DMilnueveTypeID,
		resolveResolutionID: resolveUnit3DMilnueveResolutionID,
	}
}

func resolveUnit3DMilnueveTypeID(meta api.PreparedMetadata) string {
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

func resolveUnit3DMilnueveResolutionID(meta api.PreparedMetadata) string {
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

// BuildMilnueveName builds release names matching Milnueve specifications.
// Movies: [Título] [Año] [Fuente] [Resolución] [REPACK] [Codec] [Bits] [HDR] [Idiomas] [Codec_Audio] [Canales] [SUBS]
// Series: [Título] [(Miniserie)] [Año] [SXX / SXXEXX] [Fuente] [Resolución] [REPACK] [Idiomas] [Codec_Audio] [Canales] [SUB]
func BuildMilnueveName(meta api.PreparedMetadata, customTag string) string {
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
	format := resolveMilnueveFormat(rawName, meta)
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

	// TV vs Movie placement
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

	if service != "" {
		parts = append(parts, service)
	}
	if format != "" {
		parts = append(parts, format)
	}
	if resolution != "" {
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

func resolveMilnueveFormat(rawName string, meta api.PreparedMetadata) string {
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
