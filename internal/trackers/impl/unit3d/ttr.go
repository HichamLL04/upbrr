// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"os"
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func siteTTRProfile() unit3DSiteProfile {
	return unit3DSiteProfile{}
}

func BuildTTRName(meta api.PreparedMetadata, customTag string) string {
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

	resolution := resolveEMUWResolution(rawName, meta)
	service := resolveLTService(rawName, meta)
	format := resolveEMUWFormat(rawName, meta)
	hdr := resolveEMUWHDR(rawName, meta)
	vcodec := resolveEMUWVCodec(rawName, meta)
	acodec, channels := resolveEMUWAudioCodec(rawName, meta)
	suffix := resolveTTRAudioSuffix(meta)

	tag := resolveEMUWGroupTag(rawName, meta, customTag)

	parts := make([]string, 0, 20)

	// Title
	if title != "" {
		parts = append(parts, title)
	}

	// TV Placement: Season/Episode before Year
	if isTV {
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

	rawUpper := strings.ToUpper(rawName)
	audioMode := ""
	if len(meta.AudioLanguages) >= 3 || strings.Contains(rawUpper, "MULTI") {
		audioMode = "Multi-Audio"
	} else if len(meta.AudioLanguages) == 2 || strings.Contains(rawUpper, "DUAL") || strings.Contains(suffix, "Latino Castellano") {
		audioMode = "Dual-Audio"
	}

	if resolution != "" {
		parts = append(parts, resolution)
	}
	if service != "" {
		parts = append(parts, service)
	}
	if format != "" {
		parts = append(parts, format)
	}
	if audioMode != "" {
		parts = append(parts, audioMode)
	}
	if acodec != "" {
		parts = append(parts, acodec)
	}
	if channels != "" {
		parts = append(parts, channels)
	}
	if hdr != "" {
		parts = append(parts, hdr)
	}
	if vcodec != "" {
		parts = append(parts, vcodec)
	}
	if suffix != "" {
		parts = append(parts, suffix)
	}

	result := strings.Join(parts, " ")
	result = strings.TrimSpace(strings.Join(strings.Fields(result), " "))

	if tag != "" {
		result += "-" + tag
	}

	return result
}

func resolveTTRAudioSuffix(meta api.PreparedMetadata) string {
	hasCast := false
	hasLat := false
	hasLatSub := false

	for _, lang := range meta.AudioLanguages {
		normalized := strings.ToLower(languageutil.NormalizeLanguageDisplay(lang))
		if normalized == "" {
			normalized = strings.ToLower(strings.TrimSpace(lang))
		}
		if strings.Contains(normalized, "castellano") || strings.Contains(normalized, "castilian") {
			hasCast = true
		} else if strings.Contains(normalized, "latin") || strings.Contains(normalized, "latino") || normalized == "es-mx" || normalized == "es-ar" || normalized == "es-cl" {
			hasLat = true
		} else if isSpanishLanguageToken(normalized) {
			hasCast = true
		}
	}

	if meta.MediaInfoJSONPath != "" {
		if data, err := os.ReadFile(meta.MediaInfoJSONPath); err == nil {
			text := strings.ToLower(string(data))
			audioSegment := text
			if subIdx := strings.Index(text, "\"text\""); subIdx != -1 {
				audioSegment = text[:subIdx]
				subSegment := text[subIdx:]
				if strings.Contains(subSegment, "es-ar") || strings.Contains(subSegment, "es-mx") || strings.Contains(subSegment, "latino") {
					hasLatSub = true
				}
			}
			if strings.Contains(audioSegment, "castellano") {
				hasCast = true
			} else if strings.Contains(audioSegment, "es-mx") || strings.Contains(audioSegment, "es-ar") || strings.Contains(audioSegment, "es-cl") || strings.Contains(audioSegment, "latino") {
				hasLat = true
			} else if strings.Contains(audioSegment, "\"language\": \"es\"") || strings.Contains(audioSegment, "\"language\": \"spanish\"") {
				hasCast = true
			}
		}
	}

	for _, sub := range meta.SubtitleLanguages {
		normalized := strings.ToLower(languageutil.NormalizeLanguageDisplay(sub))
		if normalized == "" {
			normalized = strings.ToLower(strings.TrimSpace(sub))
		}
		if strings.Contains(normalized, "latin") || strings.Contains(normalized, "latino") || normalized == "es-mx" || normalized == "es-ar" {
			hasLatSub = true
		}
	}

	if hasCast && hasLat {
		return "Latino Castellano"
	}
	if hasCast {
		return "Castellano"
	}
	if hasLat {
		return "Latino"
	}
	if hasLatSub {
		return "Latino Subs"
	}
	return ""
}
