package parser

import (
	"fmt"
	"strings"
)

// Format identifies a plain-text accounting file format.
type Format int

const (
	FormatAuto      Format = iota // detect from filename extension
	FormatPTA                     // .pta (directional movements)
	FormatGoluca                  // .goluca (like PTA with quoted descriptions)
	FormatBeancount               // .beancount (postings with signed amounts)
)

// DetectFormat returns the Format for a filename based on extension.
func DetectFormat(filename string) Format {
	switch {
	case strings.HasSuffix(filename, ".goluca"):
		return FormatGoluca
	case strings.HasSuffix(filename, ".beancount"):
		return FormatBeancount
	default:
		return FormatPTA
	}
}

// FormatFromString converts a CLI flag value to a Format.
func FormatFromString(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "auto":
		return FormatAuto, nil
	case "pta":
		return FormatPTA, nil
	case "goluca":
		return FormatGoluca, nil
	case "beancount":
		return FormatBeancount, nil
	default:
		return 0, fmt.Errorf("unknown format %q (use auto, pta, goluca, or beancount)", s)
	}
}
