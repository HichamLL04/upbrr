// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func resolveNekoBTVideoType(meta api.PreparedMetadata) int {
	typeCandidate := strings.ToUpper(strings.TrimSpace(meta.Type))
	releaseName := strings.ToUpper(strings.TrimSpace(meta.ReleaseName))
	source := strings.ToUpper(strings.TrimSpace(meta.Source))

	switch {
	case strings.Contains(releaseName, "HYBRID") || meta.WebDV:
		return 15 // Hybrid
	case strings.Contains(releaseName, "BDREMUX") || strings.Contains(releaseName, "REMUX") || typeCandidate == "REMUX":
		return 14 // BD - Remux
	case strings.EqualFold(meta.DiscType, "BDMV") || strings.Contains(releaseName, "BLURAY DISC") || strings.Contains(releaseName, "COMPLETE BLURAY"):
		return 11 // BD - Disc
	case strings.Contains(releaseName, "MINI") && (strings.Contains(releaseName, "BDRIP") || strings.Contains(releaseName, "BLURAY")):
		return 12 // BD - Mini
	case strings.Contains(releaseName, "BDRIP") || strings.Contains(releaseName, "BLURAY") || strings.Contains(source, "BLURAY"):
		return 13 // BD - Encode
	case strings.Contains(releaseName, "WEB-DL") || strings.Contains(releaseName, "WEBDL") || typeCandidate == "WEBDL":
		return 9 // WEB-DL
	case strings.Contains(releaseName, "MINI") && (strings.Contains(releaseName, "WEBRIP") || strings.Contains(releaseName, "WEB")):
		return 7 // WEB - Mini
	case strings.Contains(releaseName, "WEBRIP") || typeCandidate == "WEBRIP":
		return 8 // WEB - Encode
	case strings.EqualFold(meta.DiscType, "DVD") || strings.Contains(releaseName, "DVD9") || strings.Contains(releaseName, "DVD5"):
		return 16 // DVD - Disc
	case strings.Contains(releaseName, "DVDREMUX") || strings.Contains(releaseName, "DVD-REMUX"):
		return 5 // DVD - Remux
	case strings.Contains(releaseName, "DVDRIP") || typeCandidate == "DVDRIP":
		return 6 // DVD - Encode
	case strings.Contains(releaseName, "HDTV"):
		return 3 // TV - Encode
	case strings.Contains(releaseName, "LASERDISC"):
		return 2 // LaserDisc
	case strings.Contains(releaseName, "VHS"):
		return 1 // VHS
	}

	return 0 // Other
}

func resolveNekoBTVideoCodec(meta api.PreparedMetadata) int {
	codec := strings.ToUpper(strings.TrimSpace(meta.VideoCodec))
	releaseName := strings.ToUpper(strings.TrimSpace(meta.ReleaseName))

	switch {
	case strings.Contains(codec, "AV1") || strings.Contains(releaseName, "AV1"):
		return 3 // AV1
	case strings.Contains(codec, "HEVC") || strings.Contains(codec, "H265") || strings.Contains(codec, "H.265") || strings.Contains(releaseName, "X265") || strings.Contains(releaseName, "HEVC"):
		return 2 // H265
	case strings.Contains(codec, "AVC") || strings.Contains(codec, "H264") || strings.Contains(codec, "H.264") || strings.Contains(releaseName, "X264") || strings.Contains(releaseName, "AVC"):
		return 1 // H264
	case strings.Contains(codec, "VP9") || strings.Contains(releaseName, "VP9"):
		return 4 // VP9
	case strings.Contains(codec, "MPEG-2") || strings.Contains(codec, "MPEG2"):
		return 5 // MPEG-2
	case strings.Contains(codec, "MPEG-4") || strings.Contains(codec, "MPEG4"):
		return 6 // MPEG-4
	case strings.Contains(codec, "WMV"):
		return 7 // WMV
	case strings.Contains(codec, "VC-1") || strings.Contains(codec, "VC1"):
		return 8 // VC1
	}

	return 0 // Other
}

func resolveNekoBTLevel(meta api.PreparedMetadata) int {
	if len(meta.SubtitleLanguages) == 0 {
		return -1 // No subs
	}
	// Default level for official sources (BD, WEB) is Level 0
	return 0
}

func normalizeNekoBTLanguage(code string) string {
	norm := strings.ToLower(strings.TrimSpace(code))
	switch norm {
	case "jp", "jpn", "japanese":
		return "ja"
	case "eng", "english":
		return "en"
	case "spa-419", "es-419", "latin", "latino", "latam":
		return "es-419"
	case "spa", "es", "spanish", "castilian", "castellano", "es-es":
		return "es-es"
	case "por", "pt", "portuguese":
		return "pt-pt"
	case "fra", "fre", "fr", "french":
		return "fr-fr"
	case "ger", "deu", "de", "german":
		return "de"
	case "ita", "it", "italian":
		return "it"
	case "rus", "ru", "russian":
		return "ru"
	case "zh-hant", "hant", "traditional chinese", "chi-hant", "zho-hant", "zh-tw", "zh-hk":
		return "zh-hant"
	case "zh-hans", "hans", "simplified chinese", "chi-hans", "zho-hans", "zh-cn", "zh-sg", "chi", "zho", "zh", "chinese":
		return "zh-hans"
	case "kor", "ko", "korean":
		return "ko"
	}
	return norm
}

func resolveNekoBTLanguages(languages []string) string {
	res := make([]string, 0, len(languages))
	seen := make(map[string]bool)
	for _, l := range languages {
		norm := normalizeNekoBTLanguage(l)
		if norm != "" && !seen[norm] {
			seen[norm] = true
			res = append(res, norm)
		}
	}
	return strings.Join(res, ",")
}

type nekoBTTrackDoc struct {
	Media struct {
		Track []map[string]any `json:"track"`
	} `json:"media"`
}

func parseNekoBTMediaInfoTracks(jsonPath string) []map[string]any {
	if strings.TrimSpace(jsonPath) == "" {
		return nil
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var doc nekoBTTrackDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	return doc.Media.Track
}

func nekoBTTrackString(track map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := track[key]; ok && val != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", val))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func detectNekoBTSpanishDialect(lang, title string) (isLatino bool, isCastilian bool) {
	lowerLang := strings.ToLower(strings.TrimSpace(lang))
	lowerTitle := strings.ToLower(strings.TrimSpace(title))

	if lowerLang == "es-419" || lowerLang == "spa-419" || strings.Contains(lowerLang, "latin") || strings.Contains(lowerLang, "latino") {
		isLatino = true
	}
	if strings.Contains(lowerTitle, "latino") || strings.Contains(lowerTitle, "latam") || strings.Contains(lowerTitle, "latin") || strings.Contains(lowerTitle, "es-419") {
		isLatino = true
	}

	if lowerLang == "es-es" || strings.Contains(lowerLang, "castellano") || strings.Contains(lowerLang, "castilian") {
		isCastilian = true
	}
	if strings.Contains(lowerTitle, "castellano") || strings.Contains(lowerTitle, "castilian") || strings.Contains(lowerTitle, "españa") || strings.Contains(lowerTitle, "espana") || strings.Contains(lowerTitle, "es-es") {
		isCastilian = true
	}

	return isLatino, isCastilian
}

func resolveNekoBTTrackLanguages(meta api.PreparedMetadata, trackType string, fallbackLangs []string) string {
	tracks := parseNekoBTMediaInfoTracks(meta.MediaInfoJSONPath)
	res := make([]string, 0)
	seen := make(map[string]bool)

	addLang := func(code string) {
		norm := normalizeNekoBTLanguage(code)
		if norm != "" && !seen[norm] {
			seen[norm] = true
			res = append(res, norm)
		}
	}

	if len(tracks) > 0 {
		for _, track := range tracks {
			tType := strings.ToLower(strings.TrimSpace(nekoBTTrackString(track, "@type")))
			if tType != trackType {
				continue
			}
			title := nekoBTTrackString(track, "Title", "Title_String", "Title_String2", "Title_String3")
			if trackType == "audio" && strings.Contains(strings.ToLower(title), "commentary") {
				continue
			}
			lang := nekoBTTrackString(track, "Language", "Language_String", "Language_String2", "Language_String3")
			lowerLang := strings.ToLower(lang)

			// Check if Spanish track with specific dialect
			if strings.HasPrefix(lowerLang, "es") || strings.HasPrefix(lowerLang, "spa") || strings.Contains(lowerLang, "spanish") || strings.Contains(strings.ToLower(title), "español") || strings.Contains(strings.ToLower(title), "spanish") {
				isLat, isCast := detectNekoBTSpanishDialect(lang, title)
				if isLat {
					addLang("es-419")
				}
				if isCast {
					addLang("es-es")
				}
				if !isLat && !isCast {
					rawName := strings.ToLower(meta.ReleaseName + " " + meta.Audio)
					if strings.Contains(rawName, "latino") || strings.Contains(rawName, "latam") || strings.Contains(rawName, "es-419") {
						addLang("es-419")
					} else {
						addLang("es-es")
					}
				}
				continue
			}

			if lang != "" {
				addLang(lang)
			}
		}
	}

	// Fall back to fallbackLangs if no languages detected from tracks
	if len(res) == 0 {
		for _, l := range fallbackLangs {
			addLang(l)
		}
	}

	return strings.Join(res, ",")
}

