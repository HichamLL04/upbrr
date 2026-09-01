// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"fmt"
	"strings"

	"github.com/autobrr/upbrr/internal/trackers"
)

const maxDescriptionLength = 8192

func buildDescription(req trackers.UploadRequest, assets trackers.DescriptionAssets) string {
	var sb strings.Builder

	if strings.TrimSpace(assets.Description) != "" {
		sb.WriteString(strings.TrimSpace(assets.Description))
		sb.WriteString("\n\n")
	}

	if len(assets.Screenshots) > 0 && !strings.Contains(assets.Description, "http") {
		for _, shot := range assets.Screenshots {
			rawURL := strings.TrimSpace(shot.RawURL)
			if rawURL == "" {
				rawURL = strings.TrimSpace(shot.ImgURL)
			}
			if rawURL != "" {
				sb.WriteString(fmt.Sprintf("![](%s)\n", rawURL))
			}
		}
	}

	desc := strings.TrimSpace(sb.String())
	if len(desc) > maxDescriptionLength {
		desc = desc[:maxDescriptionLength]
	}
	return desc
}
