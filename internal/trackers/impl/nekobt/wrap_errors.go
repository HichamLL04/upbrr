// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nekobt

import "fmt"

func wrapError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("trackers: nekobt %s: %w", action, err)
}
