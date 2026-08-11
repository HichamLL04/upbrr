// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func siteLTProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DLTTypeID,
		resolveResolutionID: resolveUnit3DLTResolutionID,
		resolveCategoryID:   resolveUnit3DLTCategoryID,
	}
}

func resolveUnit3DLTCategoryID(meta api.PreparedMetadata) string {
	category := resolveUnit3DCategory(meta)
	if category == "TV" {
		if meta.Anime {
			return "5"
		}
		if meta.ExternalMetadata.TMDB != nil {
			kw := strings.ToLower(meta.ExternalMetadata.TMDB.Keywords)
			if strings.Contains(kw, "novela") || strings.Contains(kw, "telenovela") {
				return "8"
			}
			for _, country := range meta.ExternalMetadata.TMDB.OriginCountry {
				c := strings.ToUpper(strings.TrimSpace(country))
				if c == "TR" || c == "CN" || c == "KR" || c == "JP" {
					return "20"
				}
			}
		}
		return "2"
	}
	return "1"
}

func resolveUnit3DLTTypeID(meta api.PreparedMetadata) string {
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

func resolveUnit3DLTResolutionID(meta api.PreparedMetadata) string {
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

// BuildLTName builds the release name according to LatTeam (LT) naming rules.
// Formula:
// Movies: Titulo Año [Edicion] [Ratio] [REPACK] Resolucion [Pais] [Servicio] [Fuente/Disc/Tipo] [Hybrid] [HDR] [VCodec] [ACodec] [Canales] [AFeatures] [SUBS] [CAST]-Tag
// TV: Titulo [Año] S## [S##E##] [Episodio] [Edicion] [Ratio] [REPACK] Resolucion [Servicio] [Fuente/Disc/Tipo] [ACodec] [Canales] [AFeatures] [HDR] [VCodec] [SUBS] [CAST]-Tag
func BuildLTName(meta api.PreparedMetadata, customTag string) string {
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

	editions := resolveLTEditions(rawName, meta)
	repack := detectLTRepackToken(rawName)
	resolution := resolveResolution(meta)
	if isDiscType(meta.DiscType) && strings.EqualFold(meta.DiscType, "DVD") {
		resolution = ""
	}

	country := ""
	if isDiscType(meta.DiscType) && meta.Region != "" {
		country = strings.ToUpper(strings.TrimSpace(meta.Region))
	}

	service := resolveLTService(rawName, meta)
	sourceType := resolveLTSourceType(rawName, meta)

	isHybrid := strings.Contains(strings.ToLower(rawName), "hybrid") || meta.WebDV
	hdr := resolveLTHDR(rawName, meta)
	vcodec := resolveLTVCodec(rawName, meta, sourceType)
	acodec, channels, afeatures := resolveLTAudio(rawName, meta)
	langTags := resolveLTLangTags(meta)

	tag := resolveLTGroupTag(rawName, meta, customTag)

	parts := make([]string, 0, 20)

	// Title
	if title != "" {
		parts = append(parts, title)
	}

	// Year & Season/Episode placement
	if isTV {
		if seToken != "" {
			parts = append(parts, seToken)
		} else if yearStr != "" {
			parts = append(parts, yearStr)
		}
	} else {
		if yearStr != "" {
			parts = append(parts, yearStr)
		}
	}

	// Editions / Cut / Ratio
	for _, ed := range editions {
		if ed != "" {
			parts = append(parts, ed)
		}
	}

	// Hybrid (if before resolution in some templates or after)
	if isHybrid {
		parts = append(parts, "Hybrid")
	}

	// Repack
	if repack != "" {
		parts = append(parts, repack)
	}

	// Resolution
	if resolution != "" {
		parts = append(parts, resolution)
	}

	// Country (for full disc)
	if country != "" {
		parts = append(parts, country)
	}

	// Service (for WEB-DL/WEBRip)
	if service != "" {
		parts = append(parts, service)
	}

	// Source/Disc/Type
	if sourceType != "" {
		parts = append(parts, sourceType)
	}

	// ACodec, Canales, AFeatures placement depending on type order
	// Per LT guide:
	// WEB-DL Movies: Service WEB-DL ACodec Canales AFeatures HDR VCodec
	// Encode Movies: Source ACodec Canales AFeatures HDR VCodec
	// Remux Movies: Source REMUX HDR VCodec ACodec Canales AFeatures
	// Full Disc Movies: Pais Blu-ray HDR VCodec ACodec Canales AFeatures
	isRemuxOrDisc := strings.Contains(strings.ToUpper(sourceType), "REMUX") || isDiscType(meta.DiscType)

	if isRemuxOrDisc {
		if hdr != "" {
			parts = append(parts, hdr)
		}
		if vcodec != "" {
			parts = append(parts, vcodec)
		}
		if acodec != "" {
			parts = append(parts, acodec)
		}
		if channels != "" {
			parts = append(parts, channels)
		}
		if afeatures != "" {
			parts = append(parts, afeatures)
		}
	} else {
		if acodec != "" {
			parts = append(parts, acodec)
		}
		if channels != "" {
			parts = append(parts, channels)
		}
		if afeatures != "" {
			parts = append(parts, afeatures)
		}
		if hdr != "" {
			parts = append(parts, hdr)
		}
		if vcodec != "" {
			parts = append(parts, vcodec)
		}
	}

	// Language Tags [SUBS] [CAST]
	for _, lt := range langTags {
		if lt != "" {
			parts = append(parts, lt)
		}
	}

	result := strings.Join(parts, " ")
	result = strings.TrimSpace(strings.Join(strings.Fields(result), " "))

	if tag != "" {
		result += "-" + tag
	}

	return result
}

func resolveLTTitle(meta api.PreparedMetadata) string {
	if meta.ExternalMetadata.TMDB != nil {
		tmdb := meta.ExternalMetadata.TMDB
		origLang := strings.ToLower(strings.TrimSpace(tmdb.OriginalLanguage))
		isSpanish := isSpanishLanguageToken(origLang)
		if isSpanish {
			if tmdb.RetrievedAKA != "" && !isNonLatinText(tmdb.RetrievedAKA) {
				return cleanLTTitlePunctuation(tmdb.RetrievedAKA)
			}
			if tmdb.OriginalTitle != "" && !isNonLatinText(tmdb.OriginalTitle) {
				return cleanLTTitlePunctuation(tmdb.OriginalTitle)
			}
			if tmdb.Title != "" && !isNonLatinText(tmdb.Title) {
				return cleanLTTitlePunctuation(tmdb.Title)
			}
		} else {
			if tmdb.Title != "" && !isNonLatinText(tmdb.Title) {
				return cleanLTTitlePunctuation(tmdb.Title)
			}
			if tmdb.OriginalTitle != "" && !isNonLatinText(tmdb.OriginalTitle) {
				return cleanLTTitlePunctuation(tmdb.OriginalTitle)
			}
		}
	}
	if meta.ExternalMetadata.TVDB != nil {
		tvdb := meta.ExternalMetadata.TVDB
		if tvdb.NameEnglish != "" && !isNonLatinText(tvdb.NameEnglish) {
			return cleanLTTitlePunctuation(tvdb.NameEnglish)
		}
		if tvdb.Name != "" && !isNonLatinText(tvdb.Name) {
			return cleanLTTitlePunctuation(tvdb.Name)
		}
	}
	if meta.Release.Title != "" && !isNonLatinText(meta.Release.Title) {
		return cleanLTTitlePunctuation(meta.Release.Title)
	}
	return cleanLTTitlePunctuation(baseReleaseName(meta))
}

func isNonLatinText(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

func cleanLTTitlePunctuation(title string) string {
	title = strings.ReplaceAll(title, "(", "")
	title = strings.ReplaceAll(title, ")", "")
	if idx := strings.Index(strings.ToUpper(title), " AKA "); idx != -1 {
		title = title[:idx]
	}
	if strings.HasPrefix(strings.ToUpper(title), "AKA ") {
		title = title[4:]
	}
	return strings.TrimSpace(strings.Join(strings.Fields(title), " "))
}

func isSpanishLanguageToken(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "es", "spanish", "spa", "es-es", "es-mx", "es-ar", "es-co", "es-cl":
		return true
	default:
		return false
	}
}

func isEnglishText(text string) bool {
	// Simple check for English titles
	return true
}

func shouldIncludeYearInLTEpisode(rawName string, meta api.PreparedMetadata) bool {
	if meta.Release.Year > 0 {
		return strings.Contains(rawName, strconv.Itoa(meta.Release.Year))
	}
	return false
}

func detectLTSeasonToken(name string) string {
	for _, field := range strings.Fields(name) {
		upper := strings.ToUpper(field)
		if strings.HasPrefix(upper, "S") && len(upper) >= 3 && !strings.Contains(upper, "E") {
			return upper
		}
	}
	return ""
}

func detectLTEpisodeToken(name string) string {
	for _, field := range strings.Fields(name) {
		upper := strings.ToUpper(field)
		if strings.HasPrefix(upper, "S") && strings.Contains(upper, "E") {
			return upper
		}
	}
	return ""
}

func resolveLTEditions(rawName string, meta api.PreparedMetadata) []string {
	editions := []string{}
	upper := strings.ToUpper(rawName)

	if strings.Contains(upper, "IMAX") {
		editions = append(editions, "IMAX")
	} else if strings.Contains(upper, "OPEN MATTE") {
		editions = append(editions, "Open Matte")
	}

	if strings.Contains(upper, "DIRECTORS CUT") || strings.Contains(upper, "DIRECTOR'S CUT") {
		editions = append(editions, "Directors Cut")
	} else if strings.Contains(upper, "EXTENDED COLLECTOR'S EDITION") {
		editions = append(editions, "Extended Collector's Edition")
	} else if strings.Contains(upper, "EXTENDED") {
		editions = append(editions, "Extended")
	} else if strings.Contains(upper, "SPECIAL EDITION") {
		editions = append(editions, "Special Edition")
	} else if strings.Contains(upper, "UNRATED") {
		editions = append(editions, "UNRATED")
	} else if strings.Contains(upper, "UNCUT") {
		editions = append(editions, "Uncut")
	}

	if strings.Contains(upper, "CRITERION") {
		editions = append(editions, "Criterion")
	} else if strings.Contains(upper, "REMASTERED") {
		editions = append(editions, "Remastered")
	} else if strings.Contains(upper, "LIMITED") {
		editions = append(editions, "Limited")
	}

	return editions
}

func detectLTRepackToken(name string) string {
	upper := strings.ToUpper(name)
	if strings.Contains(upper, "REPACK2") {
		return "REPACK2"
	}
	if strings.Contains(upper, "REPACK") {
		return "REPACK"
	}
	return ""
}

func resolveLTService(rawName string, meta api.PreparedMetadata) string {
	service := strings.ToUpper(strings.TrimSpace(meta.Service))
	if service != "" {
		return service
	}
	upper := strings.ToUpper(rawName)
	services := []string{"NF", "AMZN", "HMAX", "PMTP", "CR", "DSNP", "ATVP", "PCOK"}
	for _, s := range services {
		for _, field := range strings.Fields(upper) {
			if field == s {
				return s
			}
		}
	}
	return ""
}

func resolveLTSourceType(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "UHD BLURAY REMUX") || strings.Contains(upper, "UHD BLU-RAY REMUX") {
		return "UHD BluRay REMUX"
	}
	if strings.Contains(upper, "BLURAY REMUX") || strings.Contains(upper, "BLU-RAY REMUX") {
		return "BluRay REMUX"
	}
	if strings.Contains(upper, "HDDVD REMUX") {
		return "HDDVD REMUX"
	}
	if strings.Contains(upper, "DVD REMUX") {
		return "DVD REMUX"
	}
	if strings.Contains(upper, "WEB-DL") {
		return "WEB-DL"
	}
	if strings.Contains(upper, "WEBRIP") {
		return "WEBRip"
	}
	if strings.Contains(upper, "UHD BLURAY") || strings.Contains(upper, "UHD BLU-RAY") {
		return "UHD BluRay"
	}
	if strings.Contains(upper, "BLURAY") || strings.Contains(upper, "BLU-RAY") {
		return "BluRay"
	}
	if strings.Contains(upper, "DVDRIP") {
		return "DVDRip"
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
		return "REMUX"
	case "ENCODE", "BLURAY":
		return "BluRay"
	case "DVDRIP":
		return "DVDRip"
	default:
		return typeVal
	}
}

func resolveLTHDR(rawName string, meta api.PreparedMetadata) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "DV HDR10+") {
		return "DV HDR10+"
	}
	if strings.Contains(upper, "DV HDR") {
		return "DV HDR"
	}
	if strings.Contains(upper, "HDR10+") {
		return "HDR10+"
	}
	if strings.Contains(upper, "HDR10P") {
		return "HDR10P"
	}
	if strings.Contains(upper, "DV") {
		return "DV"
	}
	if strings.Contains(upper, "HDR") {
		return "HDR"
	}
	hdr := strings.TrimSpace(meta.HDR)
	if hdr != "" {
		return hdr
	}
	return ""
}

func resolveLTVCodec(rawName string, meta api.PreparedMetadata, sourceType string) string {
	upper := strings.ToUpper(rawName)
	if strings.Contains(upper, "X264") {
		return "x264"
	}
	if strings.Contains(upper, "X265") {
		return "x265"
	}
	if strings.Contains(upper, "H.264") || strings.Contains(upper, "H264") {
		return "H.264"
	}
	if strings.Contains(upper, "H.265") || strings.Contains(upper, "H265") {
		return "H.265"
	}
	if strings.Contains(upper, "HEVC") {
		return "HEVC"
	}
	if strings.Contains(upper, "AVC") {
		return "AVC"
	}
	if strings.Contains(upper, "MPEG-2") || strings.Contains(upper, "MPEG2") {
		return "MPEG-2"
	}
	if strings.Contains(upper, "VC-1") || strings.Contains(upper, "VC1") {
		return "VC-1"
	}
	if strings.Contains(upper, "AV1") {
		return "AV1"
	}
	if meta.VideoCodec != "" {
		return meta.VideoCodec
	}
	return ""
}

func resolveLTAudio(rawName string, meta api.PreparedMetadata) (acodec string, channels string, afeatures string) {
	upper := strings.ToUpper(rawName)

	if strings.Contains(upper, "ATMOS") {
		afeatures = "Atmos"
	} else if strings.Contains(upper, "AURO3D") {
		afeatures = "Auro3D"
	}

	if strings.Contains(upper, "DTS-HD MA") {
		acodec = "DTS-HD MA"
	} else if strings.Contains(upper, "DTS-HD HRA") {
		acodec = "DTS-HD HRA"
	} else if strings.Contains(upper, "DTS:X") {
		acodec = "DTS:X"
	} else if strings.Contains(upper, "TRUEHD") {
		acodec = "TrueHD"
	} else if strings.Contains(upper, "DD+ EX") {
		acodec = "DD+ EX"
	} else if strings.Contains(upper, "DD+") || strings.Contains(upper, "DDP") {
		acodec = "DD+"
	} else if strings.Contains(upper, "DD EX") {
		acodec = "DD EX"
	} else if strings.Contains(upper, "DD") {
		acodec = "DD"
	} else if strings.Contains(upper, "FLAC") {
		acodec = "FLAC"
	} else if strings.Contains(upper, "LPCM") || strings.Contains(upper, "PCM") {
		acodec = "LPCM"
	} else if strings.Contains(upper, "AAC") {
		acodec = "AAC"
	} else if strings.Contains(upper, "OPUS") {
		acodec = "Opus"
	} else if meta.Audio != "" {
		acodec = meta.Audio
	}

	channels = resolveLTChannels(upper, meta)

	return acodec, channels, afeatures
}

func resolveLTChannels(upper string, meta api.PreparedMetadata) string {
	chans := []string{"11.1", "9.1", "7.1", "6.1", "5.1", "4.0", "2.0", "1.0"}
	for _, c := range chans {
		for _, field := range strings.Fields(upper) {
			if field == c {
				return c
			}
		}
	}
	if meta.Channels != "" {
		return meta.Channels
	}
	return ""
}

func resolveLTLangTags(meta api.PreparedMetadata) []string {
	tags := []string{}
	hasLatinDub := false
	hasCastDub := false
	hasSpanishSub := false

	origLang := resolveOriginalLanguage(meta)
	origIsSpanish := isSpanishLanguageToken(origLang)

	audioLangs := meta.AudioLanguages
	subLangs := meta.SubtitleLanguages

	if meta.MediaInfoJSONPath != "" {
		if data, err := os.ReadFile(meta.MediaInfoJSONPath); err == nil {
			text := strings.ToLower(string(data))
			if strings.Contains(text, "castellano") {
				hasCastDub = true
			}
			if strings.Contains(text, "sub") || strings.Contains(text, "\"text\"") {
				hasSpanishSub = true
			}
		}
	}

	for _, lang := range audioLangs {
		normalized := strings.ToLower(languageutil.NormalizeLanguageDisplay(lang))
		if normalized == "" {
			normalized = strings.ToLower(strings.TrimSpace(lang))
		}
		if strings.Contains(normalized, "latin") || strings.Contains(normalized, "latino") || normalized == "lat" {
			hasLatinDub = true
		} else if strings.Contains(normalized, "castellano") || strings.Contains(normalized, "castilian") || normalized == "cast" {
			hasCastDub = true
		} else if isSpanishLanguageToken(normalized) {
			hasCastDub = true
		}
	}

	for _, sub := range subLangs {
		normalized := strings.ToLower(languageutil.NormalizeLanguageDisplay(sub))
		if normalized == "" {
			normalized = strings.ToLower(strings.TrimSpace(sub))
		}
		if isSpanishLanguageToken(normalized) || strings.Contains(normalized, "spanish") || strings.Contains(normalized, "spa") {
			hasSpanishSub = true
		}
	}

	if !hasLatinDub && !hasCastDub && hasSpanishSub {
		tags = append(tags, "[SUBS]")
	} else if hasCastDub && !origIsSpanish {
		tags = append(tags, "[CAST]")
	}

	return tags
}

func resolveLTGroupTag(rawName string, meta api.PreparedMetadata, customTag string) string {
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
		return ""
	}
	return tag
}
