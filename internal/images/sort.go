package images

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SortByCaptureDate orders paths by EXIF capture time. Files without a capture
// timestamp are moved to the end and ordered by file name, since names usually
// track capture order too.
func SortByCaptureDate(paths []string, descending bool) []string {
	type entry struct {
		path     string
		captured time.Time
		dated    bool
	}

	entries := make([]entry, 0, len(paths))
	for _, path := range paths {
		captured, ok := CaptureTime(path)
		entries = append(entries, entry{path: path, captured: captured, dated: ok})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.dated != right.dated {
			return left.dated
		}
		if !left.dated {
			return fileNameLess(left.path, right.path)
		}
		if left.captured.Equal(right.captured) {
			return fileNameLess(left.path, right.path)
		}
		if descending {
			return left.captured.After(right.captured)
		}
		return left.captured.Before(right.captured)
	})

	sorted := make([]string, 0, len(entries))
	for _, item := range entries {
		sorted = append(sorted, item.path)
	}
	return sorted
}

func fileNameLess(left, right string) bool {
	leftName, rightName := filepath.Base(left), filepath.Base(right)
	if leftName == rightName {
		return strings.Compare(left, right) < 0
	}
	return leftName < rightName
}
