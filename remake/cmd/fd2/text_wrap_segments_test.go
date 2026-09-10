package main

import (
	"strings"
	"testing"
)

// TestTextSegmentsKeepsLatinWordsWhole 釘住斷點單位：拉丁單字整串一段，中日韓
// 逐字一段，空白黏在前一段尾巴。這是英文與日文譯文不會斷在單字中間的根據。
func TestTextSegmentsKeepsLatinWordsWhole(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"Indexed text", []string{"Indexed ", "text"}},
		{"翻訳された会話", []string{"翻", "訳", "さ", "れ", "た", "会", "話"}},
		{"Indexed 日本語 text", []string{"Indexed ", "日", "本", "語 ", "text"}},
		{"HP 120/240", []string{"HP ", "120/240"}},
		{"やあ、Zephyr。", []string{"や", "あ", "、", "Zephyr", "。"}},
	} {
		got := textSegments(tc.text)
		if strings.Join(got, "") != tc.text {
			t.Fatalf("%q 切完接不回原文：%q", tc.text, strings.Join(got, ""))
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%q 切成 %d 段（%q），應為 %d 段", tc.text, len(got), got, len(tc.want))
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q 第 %d 段是 %q，應為 %q", tc.text, i, got[i], tc.want[i])
			}
		}
	}
}

// TestWrapTextByWidthDoesNotSplitLatinWords 用固定寬度（每個 rune 算 1）驗三件
// 事：英文在空白處斷、中文逐字斷、單字本身超過一行時才硬拆。
func TestWrapTextByWidthDoesNotSplitLatinWords(t *testing.T) {
	width := func(s string) float64 { return float64(len([]rune(s))) }

	got := wrapTextByWidth("alpha beta gamma delta", 12, width)
	for _, line := range got {
		for _, word := range strings.Fields(line) {
			if !strings.Contains("alpha beta gamma delta", word) {
				t.Fatalf("行 %q 裡的 %q 不是完整單字", line, word)
			}
		}
	}
	if strings.Join(got, "") != "alpha beta gamma delta" {
		t.Fatalf("斷行後文字改變：%q", strings.Join(got, ""))
	}

	if got := wrapTextByWidth("一二三四五六", 2, width); len(got) != 3 {
		t.Fatalf("中文兩字一行應切成 3 行，得到 %d 行：%q", len(got), got)
	}

	// 沒有空白可斷的長串：硬拆，但每一行都不超過寬度。
	got = wrapTextByWidth("supercalifragilistic", 6, width)
	for _, line := range got {
		if width(line) > 6 {
			t.Fatalf("硬拆之後 %q 仍超過寬度", line)
		}
	}
	if strings.Join(got, "") != "supercalifragilistic" {
		t.Fatalf("硬拆後文字改變：%q", strings.Join(got, ""))
	}
}

// TestLocalizedDialogueWrapKeepsEnglishWordsWhole 對真字型驗對話框那一條路：
// 英文譯文的每一行都以完整單字結尾，而且接回去等於原文。
func TestLocalizedDialogueWrapKeepsEnglishWordsWhole(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	const text = "A translated indexed dialogue keeps its frame, portrait, camera, and progressive text."
	rows, err := wrapLocalizedNativeDialogue(text, displayFont, localizedNativeDialogueFontScale, 224, 28)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("這句話應該要斷成多行，得到 %d 行", len(rows))
	}
	if strings.Join(rows, "") != text {
		t.Fatalf("斷行後文字改變：%q", strings.Join(rows, ""))
	}
	words := strings.Fields(text)
	seen := map[string]bool{}
	for _, w := range words {
		seen[w] = true
	}
	for _, row := range rows {
		for _, w := range strings.Fields(row) {
			if !seen[w] {
				t.Fatalf("行 %q 含被切斷的 %q", row, w)
			}
		}
	}
}
