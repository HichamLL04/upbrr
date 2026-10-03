// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package mediainfo

import (
	"context"
	"fmt"

	gomediainfo "github.com/autobrr/go-mediainfo"
)

type ModuleAnalyzer struct{}

func NewModuleAnalyzer() *ModuleAnalyzer {
	return &ModuleAnalyzer{}
}

func (ModuleAnalyzer) Analyze(_ context.Context, target string) (string, []byte, error) {
	report, err := gomediainfo.AnalyzeFile(target)
	if err != nil {
		return "", nil, fmt.Errorf("mediainfo: analyze file: %w", err)
	}
	reports := []gomediainfo.Report{report}
	text, err := gomediainfo.Render(reports, gomediainfo.OutputText)
	if err != nil {
		return "", nil, fmt.Errorf("mediainfo: render text: %w", err)
	}
	jsonBytes, err := gomediainfo.Render(reports, gomediainfo.OutputJSON)
	if err != nil {
		return "", nil, fmt.Errorf("mediainfo: render json: %w", err)
	}
	return text, []byte(jsonBytes), nil
}
