package ansi

import (
	"fmt"
	"strings"
)

const (
	black = iota
	red
	green
	yellow
	blue
	magenta
	cyan
	white

	normalIntensityFG = 30
	highIntensityFG   = 90
	normalIntensityBG = 40
	highIntensityBG   = 100

	start     = "\033["
	bold      = "1;"
	blink     = "5;"
	underline = "4;"
	inverse   = "7;"
	Reset     = "\033[0m"
)

var (
	plain  = false
	colors = map[string]int{
		"black":   black,
		"red":     red,
		"green":   green,
		"yellow":  yellow,
		"blue":    blue,
		"magenta": magenta,
		"cyan":    cyan,
		"white":   white,
	}
)

// Gets the ANSI escape code for a color style.
func ColorCode(style string) string {
	if plain || style == "" {
		return ""
	}
	if style == "reset" {
		return Reset
	}

	foreground_background := strings.Split(style, ":")
	foreground := strings.Split(foreground_background[0], "+")
	fg := colors[foreground[0]]
	fgStyle := ""
	if len(foreground) > 1 {
		fgStyle = foreground[1]
	}

	bg, bgStyle := "", ""

	if len(foreground_background) > 1 {
		background := strings.Split(foreground_background[1], "+")
		bg = background[0]
		if len(background) > 1 {
			bgStyle = background[1]
		}
	}

	code := start
	base := normalIntensityFG
	if len(fgStyle) > 0 {
		if strings.Contains(fgStyle, "b") {
			code += bold
		}
		if strings.Contains(fgStyle, "B") {
			code += blink
		}
		if strings.Contains(fgStyle, "u") {
			code += underline
		}
		if strings.Contains(fgStyle, "i") {
			code += inverse
		}
		if strings.Contains(fgStyle, "h") {
			base = highIntensityFG
		}
	}
	code += fmt.Sprintf("%d;", base+fg)

	base = normalIntensityBG
	if len(bg) > 0 {
		if strings.Contains(bgStyle, "h") {
			base = highIntensityBG
		}
		code += fmt.Sprintf("%d;", base+colors[bg])
	}

	// remove last ";"
	return code[:len(code)-1] + "m"
}

// Surrounds `s` with ANSI color and reset code.
func Color(s, style string) string {
	if plain || len(style) < 1 {
		return s
	}
	return ColorCode(style) + s + Reset
}

// Creates a fast closure.
//
// Prefer ColorFunc over Color as it does not recompute ANSI codes.
func ColorFunc(style string) func(string) string {
	if style == "" {
		return func(s string) string {
			return s
		}
	} else {
		code := ColorCode(style)
		return func(s string) string {
			if plain || len(s) < 1 {
				return s
			}
			return code + s + Reset
		}
	}
}

// Disables ANSI color codes. On by default.
func DisableColors(disable bool) {
	plain = disable
}
