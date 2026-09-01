// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import (
	"os"
	"strings"

	"github.com/autobrr/upbrr/internal/trackers"
)

func buildDescription(req trackers.UploadRequest, assets trackers.DescriptionAssets) string {
	var sb strings.Builder

	if assets.Description != "" {
		sb.WriteString(assets.Description)
		sb.WriteString("\n\n")
	}

	if req.Meta.MediaInfoTextPath != "" {
		if data, err := os.ReadFile(req.Meta.MediaInfoTextPath); err == nil && len(data) > 0 {
			sb.WriteString("```\n")
			sb.WriteString(strings.TrimSpace(string(data)))
			sb.WriteString("\n```\n")
		}
	}

	return strings.TrimSpace(sb.String())
}
