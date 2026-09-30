// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"fmt"
	"strings"

	"github.com/autobrr/upbrr/internal/metadata/metautil"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

const maxDescriptionLength = 8192

func buildDescription(req trackers.UploadRequest, assets trackers.DescriptionAssets) string {
	var sections []string

	baseDesc := strings.TrimSpace(assets.Description)
	if baseDesc != "" {
		sections = append(sections, baseDesc)
	}

	// Format screenshots with centered 400px width images in pairs
	if len(assets.Screenshots) > 0 && !hasScreenshotMarkup(baseDesc) {
		if shotBlock := buildScreenshotMarkdown(assets.Screenshots); shotBlock != "" {
			sections = append(sections, shotBlock)
		}
	}

	// Append signature
	if !strings.Contains(baseDesc, "https://github.com/autobrr/upbrr") && !strings.Contains(baseDesc, "Uploaded by upbrr") {
		sections = append(sections, "<div align=\"right\">\n\n[Uploaded by upbrr](https://github.com/autobrr/upbrr)\n\n</div>")
	}

	desc := strings.TrimSpace(strings.Join(sections, "\n\n"))
	if len(desc) > maxDescriptionLength {
		desc = desc[:maxDescriptionLength]
	}
	return desc
}

func hasScreenshotMarkup(desc string) bool {
	return strings.Contains(desc, "width=\"400\"") || strings.Contains(desc, "<img ") || strings.Contains(desc, "![](")
}

func buildScreenshotMarkdown(images []api.ScreenshotImage) string {
	var formattedImages []string
	for _, shot := range images {
		imgURL := strings.TrimSpace(metautil.FirstNonEmptyTrimmed(shot.ImgURL, shot.RawURL))
		webURL := strings.TrimSpace(metautil.FirstNonEmptyTrimmed(shot.WebURL, imgURL))
		if imgURL == "" {
			continue
		}
		if webURL == "" {
			webURL = imgURL
		}
		formattedImages = append(formattedImages, fmt.Sprintf(`<a href="%s"><img src="%s" width="400"></a>`, webURL, imgURL))
	}

	if len(formattedImages) == 0 {
		return ""
	}

	var rows []string
	for i := 0; i < len(formattedImages); i += 2 {
		if i+1 < len(formattedImages) {
			rows = append(rows, formattedImages[i]+" "+formattedImages[i+1])
		} else {
			rows = append(rows, formattedImages[i])
		}
	}

	return fmt.Sprintf("<div align=\"center\">\n\n%s\n\n</div>", strings.Join(rows, "\n\n"))
}

