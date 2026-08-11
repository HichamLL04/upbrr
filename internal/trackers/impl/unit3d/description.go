// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	descriptionunit3d "github.com/autobrr/upbrr/internal/services/description/unit3d"
	"github.com/autobrr/upbrr/pkg/api"
)

func buildUnit3DDescription(ctx context.Context, tracker string, meta api.PreparedMetadata, appConfig config.Config, trackerConfig config.TrackerConfig, logger api.Logger, keptDescription string, menuImages []api.ScreenshotImage, screenshots []api.ScreenshotImage) (string, error) {
	if strings.EqualFold(strings.TrimSpace(tracker), "ACM") {
		return buildACMDescription(ctx, meta, appConfig, trackerConfig, logger, keptDescription, menuImages, screenshots)
	}
	description, err := descriptionunit3d.BuildDescription(ctx, meta, appConfig, trackerConfig, logger, keptDescription, menuImages, screenshots)
	if err != nil {
		return "", fmt.Errorf("trackers: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(tracker), "SHRI") {
		return applySHRIDescriptionNotes(description, meta), nil
	}
	if strings.EqualFold(strings.TrimSpace(tracker), "LT") {
		if hasIAToken(meta) {
			group := resolveLTGroupTag(meta.ReleaseName, meta, trackerConfig.TagForCustomRelease)
			groupName := "GapMoe"
			if group != "" {
				cleanGroup := strings.TrimSuffix(strings.TrimSuffix(group, "-IA"), "-ia")
				cleanGroup = strings.TrimSuffix(strings.TrimSuffix(cleanGroup, "-Ia"), "-iA")
				cleanGroup = strings.TrimSpace(cleanGroup)
				if cleanGroup != "" {
					groupName = cleanGroup
				}
			}
			note := fmt.Sprintf("[center][note]Este aporte de %s uso IA para la traducción[/note][/center]", groupName)
			description = note + "\n\n" + description
		}
	}
	if strings.EqualFold(strings.TrimSpace(tracker), "EMUW") {
		var youtubeURL string
		if meta.ExternalMetadata.TMDB != nil {
			youtubeURL = meta.ExternalMetadata.TMDB.YouTube
		}
		if youtubeURL != "" {
			if id := extractYouTubeID(youtubeURL); id != "" {
				description = fmt.Sprintf("[center][youtube]%s[/youtube][/center]\n\n%s", id, description)
			}
		}
	}
	return description, nil
}

func extractYouTubeID(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return ""
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		if idx := strings.Index(urlStr, "v="); idx != -1 {
			id := urlStr[idx+2:]
			if ampersandIdx := strings.Index(id, "&"); ampersandIdx != -1 {
				id = id[:ampersandIdx]
			}
			return id
		}
		return ""
	}
	if u.Host == "youtu.be" {
		return strings.TrimPrefix(u.Path, "/")
	}
	if strings.Contains(u.Path, "/embed/") {
		return strings.TrimPrefix(u.Path, "/embed/")
	}
	if strings.Contains(u.Path, "/v/") {
		return strings.TrimPrefix(u.Path, "/v/")
	}
	return u.Query().Get("v")
}

func hasIAToken(meta api.PreparedMetadata) bool {
	if meta.HasIA {
		return true
	}
	tag := strings.ToLower(strings.TrimSpace(meta.Tag))
	if strings.Contains(tag, "ia") {
		return true
	}
	group := strings.ToLower(strings.TrimSpace(meta.Release.Group))
	if strings.Contains(group, "ia") {
		return true
	}
	rawUpper := strings.ToUpper(meta.ReleaseName)
	if strings.Contains(rawUpper, "-IA") || strings.HasSuffix(rawUpper, " IA") || strings.Contains(rawUpper, " IA-") {
		return true
	}
	return false
}
