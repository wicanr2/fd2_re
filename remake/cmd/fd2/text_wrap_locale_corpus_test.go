package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// TestLocaleDialogueCorpusNeverSplitsWords 把英文與日文的全部 1,564 句對白跑過
// 對話框的斷行，逐句檢查兩件事：接回去等於原文，以及沒有一行結束在單字中間。
//
// 單支例句證明不了全量——譯文裡有連字號專名、數字、片假名夾拉丁字這些邊界。
// 唯一允許斷在字母之間的情況是那個字本身就比一行寬，只能硬拆。
func TestLocaleDialogueCorpusNeverSplitsWords(t *testing.T) {
	displayFont := loadFont()
	if displayFont == nil {
		t.Fatal("official CJK font unavailable")
	}
	limit, ok := campaign.NativeDialogueLineGlyphLimit("FFEC")
	if !ok {
		t.Fatal("FFEC 沒有字數上限")
	}
	_, _, maxWidth, _, _, err := campaign.NativeStoryDialogueTextGeometry("FFEC")
	if err != nil {
		t.Fatal(err)
	}
	width := float64(maxWidth)

	for _, locale := range []string{"en", "ja"} {
		raw, err := os.ReadFile(filepath.Join("../../assets/locales", locale, "content.json"))
		if err != nil {
			t.Skipf("讀不到 %s 的語系內容：%v", locale, err)
		}
		var pack struct {
			Entries []struct {
				Role string `json:"role"`
				Text string `json:"text"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(raw, &pack); err != nil {
			t.Fatal(err)
		}
		checked := 0
		for _, entry := range pack.Entries {
			if entry.Role != "dialogue" || strings.TrimSpace(entry.Text) == "" {
				continue
			}
			rows, err := wrapLocalizedNativeDialogue(
				entry.Text, displayFont, localizedNativeDialogueFontScale, width, limit)
			if err != nil {
				t.Fatalf("%s %q 斷不了行：%v", locale, entry.Text, err)
			}
			if strings.Join(rows, "") != entry.Text {
				t.Fatalf("%s %q 斷行後文字改變：%q", locale, entry.Text, strings.Join(rows, ""))
			}
			for i := 0; i+1 < len(rows); i++ {
				tail := []rune(rows[i])
				head := []rune(rows[i+1])
				if len(tail) == 0 || len(head) == 0 {
					continue
				}
				last, first := tail[len(tail)-1], head[0]
				if !inseparable(last) || !inseparable(first) {
					continue
				}
				// 只有整個字本身放不下才准硬拆。
				word := trailingWord(rows[i]) + leadingWord(rows[i+1])
				if displayFont.Width(word, localizedNativeDialogueFontScale) <= width &&
					len([]rune(word)) <= limit {
					t.Fatalf("%s %q 把 %q 斷在單字中間（第 %d 行 %q → %q）",
						locale, entry.Text, word, i, rows[i], rows[i+1])
				}
			}
			checked++
		}
		if checked < 1000 {
			t.Fatalf("%s 只檢查到 %d 句對白，語系內容可能取錯", locale, checked)
		}
		t.Logf("%s 檢查 %d 句對白", locale, checked)
	}
}

// inseparable 是「不該被斷開的字元」：拉丁字母與數字。
func inseparable(r rune) bool {
	return !runeBreakable(r) && !unicode.IsSpace(r)
}

func trailingWord(line string) string {
	runes := []rune(line)
	i := len(runes)
	for i > 0 && inseparable(runes[i-1]) {
		i--
	}
	return string(runes[i:])
}

func leadingWord(line string) string {
	runes := []rune(line)
	i := 0
	for i < len(runes) && inseparable(runes[i]) {
		i++
	}
	return string(runes[:i])
}
