package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFontWrapPreservesCJKTextWithinWidth(t *testing.T) {
	const text = "未能以天空之鑰開啟通往空中大陸的道路，炎龍騎士團的遠征在巨塔前告終。"
	width := func(line string) float64 { return float64(utf8.RuneCountInString(line) * 18) }
	lines := wrapTextByWidth(text, 180, width)
	if len(lines) < 2 || strings.Join(lines, "") != text {
		t.Fatalf("wrapped ending lost text: %#v", lines)
	}
	for _, line := range lines {
		if got := width(line); got > 180 {
			t.Fatalf("wrapped line width %.1f exceeds limit: %q", got, line)
		}
	}
}
