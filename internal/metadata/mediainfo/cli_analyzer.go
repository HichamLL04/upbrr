// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package mediainfo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CLIAnalyzer implements Analyzer by shelling out to the official MediaInfo binary.
// It produces output equivalent to "mediainfo --Output=TEXT" and "mediainfo --Output=JSON",
// which includes fields like Encoded_Library_Settings that the pure-Go module does not emit.
type CLIAnalyzer struct {
	bin string
}

// NewCLIAnalyzer returns a CLIAnalyzer using the resolved binary path.
// Use ResolveBin to find the binary before calling this.
func NewCLIAnalyzer(bin string) *CLIAnalyzer {
	return &CLIAnalyzer{bin: bin}
}

// Analyze runs the MediaInfo CLI against target and returns text + JSON output.
func (a *CLIAnalyzer) Analyze(ctx context.Context, target string) (string, []byte, error) {
	if strings.TrimSpace(a.bin) == "" {
		return "", nil, fmt.Errorf("mediainfo: no CLI binary configured — set mediainfo_path in config.yaml or ensure the mediainfo binary is in PATH")
	}
	text, err := runMediaInfoCLI(ctx, a.bin, target, "TEXT")
	if err != nil {
		return "", nil, fmt.Errorf("mediainfo: cli text: %w", err)
	}
	jsonStr, err := runMediaInfoCLI(ctx, a.bin, target, "JSON")
	if err != nil {
		return "", nil, fmt.Errorf("mediainfo: cli json: %w", err)
	}
	return text, []byte(jsonStr), nil
}

func toCLIPath(target string) string {
	if runtime.GOOS == "windows" {
		cleaned := filepath.Clean(target)
		if len(cleaned) >= 240 && filepath.IsAbs(cleaned) && !strings.HasPrefix(cleaned, `\\?\`) {
			return `\\?\` + cleaned
		}
		return cleaned
	}
	return target
}

func runMediaInfoCLI(ctx context.Context, bin, target, outputFmt string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "--Output="+outputFmt, toCLIPath(target))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mediainfo binary: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// ResolveBin looks up the MediaInfo binary path. It tries override first, then
// platform-appropriate candidates in PATH. Returns (path, true) when found.
func ResolveBin(override string) (string, bool) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		if path, err := exec.LookPath(trimmed); err == nil {
			return path, true
		}
		// User provided an explicit path - trust it even if LookPath cannot verify it
		// (e.g. absolute path on a mounted volume not yet in the exec cache).
		return trimmed, true
	}

	for _, name := range platformCandidates() {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	return "", false
}

// platformCandidates returns binary name candidates for the current OS.
func platformCandidates() []string {
	if runtime.GOOS == "windows" {
		return []string{"MediaInfo.exe", "mediainfo.exe", "mediainfo"}
	}
	return []string{"mediainfo", "MediaInfo"}
}