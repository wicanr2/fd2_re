package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// 0x1E292 的經驗／升級對話：[0x53EC8]（最後一擊算出的經驗）非零就 0x1956B(unit+7) 開對話格，
// 0x15F84 在 0x951F（故事對白下框的文字座標）寫 FDTXT_000 0x1E8「得到經驗 FFFA 點！」；
// 經驗滿 100 就在第二列寫 0x1E9「等級上升了！！FFFD」，再由 0x1E529 逐欄擲成長、每欄非零
// 才寫 0x1EA..0x1EE「…上升 FFFA 點！FFFD」（第三列滿了先 0x16E24 捲一列），學到指令再寫
// 0x24B「學會了 FFFC！」；最後 0x1E5C0(11) 等 11 個 BIOS tick（有鍵就提前）→ 0x196CB 關框。
// FFFD 是 0x15F84 內的 0x16C57(1) 等鍵（箭頭閃爍），所以頁的切分就是 FFFD；這裡整份交給
// 既有的原生故事對白引擎（同一個框、同一個三列捲動、同一個箭頭），只多兩件事：最後一頁也
// 等鍵並畫箭頭、之後計時關框。第七章 r4 seq 968..973：五次 enter 走完 exp／升級／四欄。
const (
	nativeLevelUpExpTextIndex   = 0x1e8
	nativeLevelUpTextIndex      = 0x1e9
	nativeLevelUpGrowthTextBase = 0x1ea // AP、DP、DX、MaxHP、MaxMP 依序 0x1EA..0x1EE
	nativeLevelUpLearnTextIndex = 0x24b
	nativeLevelUpLearnNameBase  = 0x1b9 // FFFC = 0x1B9 + command id（0x1E389）
	// 0x1E504 push 0xb：11 個 BIOS tick（18.2 Hz）≈ 0.6 s；重製端以 60 Hz 幀近似。
	nativeLevelUpCloseFrames = 36
)

type pendingNativeLevelUp struct {
	unit *battle.Unit
	exp  int
	ups  []battle.LevelUpEvent
}

type nativeLevelUpDialogueState struct {
	unit *battle.Unit
	// awaitLastKey：最後一頁也要等鍵（原版最後一欄的 FFFD）；沒有升級時只有一頁一行、
	// 沒有 FFFD，直接進計時。
	awaitLastKey bool
	closeTimer   int
	closing      bool
	then         func()
}

// queueNativeLevelUpDialogue 在攻擊結算知道經驗時排一則；只有原版 `+6==2` 的單位會
// 拿到 [0x53EC8]（sub_29F72 以揮擊者 +6==2 覆寫）。
func (g *Game) queueNativeLevelUpDialogue(unit *battle.Unit, exp int, ups []battle.LevelUpEvent) {
	if g == nil || unit == nil || exp <= 0 || !unit.HasNativeRecordByte6 || unit.NativeRecordByte6 != 2 {
		return
	}
	g.pendingNativeLevelUps = append(g.pendingNativeLevelUps, pendingNativeLevelUp{unit: unit, exp: exp, ups: ups})
}

func (g *Game) runPendingNativeLevelUpDialogues(then func()) bool {
	if g == nil || len(g.pendingNativeLevelUps) == 0 || g.nativeLevelUpDialogue != nil || len(g.dialog) != 0 {
		return false
	}
	pending := g.pendingNativeLevelUps[0]
	g.pendingNativeLevelUps = g.pendingNativeLevelUps[1:]
	if err := g.beginNativeLevelUpDialogue(pending, then); err != nil {
		g.loadErr = "native level-up dialogue: " + err.Error()
	}
	return true
}

// nativeLevelUpDialogueLines 把 0x1E292 會寫的每一行展開成 FDTXT 原始字（FFFA／FFFC 已代入），
// 依 FFFD 切頁。回傳每頁的行（每行是原始 glyph word 序列），以及最後一行是否以 FFFD 收尾
// （是：最後一頁也等鍵；否：例如只有經驗一行、或「學會了」收尾，直接進 0x1E5C0 計時）。
func nativeLevelUpDialogueLines(strings interface {
	Words(int) ([]uint16, error)
}, pending pendingNativeLevelUp) ([][][]uint16, bool, error) {
	expand := func(index int, number int, name []uint16) ([]uint16, bool, error) {
		words, err := strings.Words(index)
		if err != nil {
			return nil, false, err
		}
		out := make([]uint16, 0, len(words)+8)
		wait := false
		for _, word := range words {
			switch word {
			case 0xfffa:
				for _, digit := range strconv.Itoa(number) {
					out = append(out, uint16(digit-'0'))
				}
			case 0xfffc:
				out = append(out, name...)
			case 0xfffd:
				wait = true
			case 0xfffe:
				return nil, false, fmt.Errorf("level-up text %#x has an unexpected line break", index)
			default:
				out = append(out, word)
			}
		}
		return out, wait, nil
	}
	pages := [][][]uint16{}
	page := [][]uint16{}
	lastWait := false
	push := func(line []uint16, wait bool) {
		page = append(page, line)
		lastWait = wait
		if wait {
			pages = append(pages, page)
			page = nil
		}
	}
	line, wait, err := expand(nativeLevelUpExpTextIndex, pending.exp, nil)
	if err != nil {
		return nil, false, err
	}
	push(line, wait)
	for _, up := range pending.ups {
		line, wait, err := expand(nativeLevelUpTextIndex, 0, nil)
		if err != nil {
			return nil, false, err
		}
		push(line, wait)
		for i, gain := range [5]int{up.ApGain, up.DpGain, up.DxGain, up.HpGain, up.MpGain} {
			if gain == 0 {
				continue // 0x1E560：欄位為零不寫那一行
			}
			line, wait, err := expand(nativeLevelUpGrowthTextBase+i, gain, nil)
			if err != nil {
				return nil, false, err
			}
			push(line, wait)
		}
		for _, command := range up.LearnedCommandIDs {
			name, err := strings.Words(nativeLevelUpLearnNameBase + command)
			if err != nil {
				return nil, false, err
			}
			line, wait, err := expand(nativeLevelUpLearnTextIndex, 0, name)
			if err != nil {
				return nil, false, err
			}
			push(line, wait)
		}
	}
	if len(page) != 0 {
		pages = append(pages, page)
	}
	return pages, lastWait, nil
}

func (g *Game) beginNativeLevelUpDialogue(pending pendingNativeLevelUp, then func()) error {
	unit := pending.unit
	if unit == nil || !unit.HasBattleFig || !unit.HasNativeMapPresentation {
		return errors.New("unit DATO selector or map presentation is unavailable")
	}
	if g.nativePreparationUI == nil || g.nativePreparationUI.status.Strings == nil {
		return errors.New("FDTXT_000 strings are unavailable")
	}
	if g.localeID == "zh-Hant" && (g.nativeBattleFont == nil || g.nativeBattleGlyphs == nil) {
		font, glyphs, err := loadNativeBattleNameAssets()
		if err != nil {
			return err
		}
		g.nativeBattleFont, g.nativeBattleGlyphs = font, glyphs
	}
	if g.nativeBattleGlyphs == nil {
		return errors.New("glyph index is unavailable")
	}
	pages, lastWait, err := nativeLevelUpDialogueLines(g.nativePreparationUI.status.Strings, pending)
	if err != nil {
		return err
	}
	// 原始 glyph word → 索引表的鍵（故事對白引擎用鍵查 glyph）。
	keyOf := make(map[int]string, len(g.nativeBattleGlyphs))
	for key, glyph := range g.nativeBattleGlyphs {
		if prev, ok := keyOf[glyph]; !ok || key < prev {
			keyOf[glyph] = key
		}
	}
	layout := &battle.NativeDialogueLayout{
		SourceDAT: "FDTXT_000", StringIndex: nativeLevelUpExpTextIndex, Utterance: 0,
		Control: "FFEC", Operand: 0, MotionTargetY: 0, HasMotionTargetY: true,
	}
	for _, page := range pages {
		texts, glyphRows := []string{}, [][]string{}
		for _, line := range page {
			tokens := make([]string, 0, len(line))
			text := ""
			for _, word := range line {
				key, ok := keyOf[int(word)]
				if !ok {
					return fmt.Errorf("glyph %d has no index key", word)
				}
				tokens = append(tokens, key)
				text += key
			}
			texts = append(texts, text)
			glyphRows = append(glyphRows, tokens)
		}
		layout.Pages = append(layout.Pages, texts)
		layout.GlyphPages = append(layout.GlyphPages, glyphRows)
	}
	// 0x11951 的 0x11CAC 整幀重繪：對話底圖是攻擊者已變灰、屍體已移除的地圖，不是攻擊
	// 演出前留下的那一張。
	g.nativeMapVGA = nil
	upper := false
	g.dialog = []battle.DialogLine{{Speaker: unit.BattleFig, Upper: &upper, NativeDialogue: layout}}
	g.dlgPage, g.dlgScrollT, g.dlgShown, g.dlgPhase, g.dlgT = 0, 0, dlgNone, 0, 0
	state := &nativeLevelUpDialogueState{unit: unit, awaitLastKey: lastWait, then: then}
	g.nativeLevelUpDialogue = state
	if os.Getenv("FD2_SHOT_AI") != "" {
		log.Printf("level-up dialogue: unit id=%d at (%d,%d) exp=%d ups=%d pages=%d lastWait=%v",
			unit.NativeIdentity, unit.X, unit.Y, pending.exp, len(pending.ups), len(pages), lastWait)
	}
	if err := g.startNativeDialogueFrames(); err != nil {
		g.dialog = nil
		g.nativeLevelUpDialogue = nil
		return err
	}
	return nil
}

// nativeLevelUpDialogueAtWait 回報對話正停在等鍵（頁尾 FFFD，或最後一頁字寫完）。
func (g *Game) nativeLevelUpDialogueAtWait() bool {
	return g != nil && g.nativeLevelUpDialogue != nil && !g.nativeLevelUpDialogue.closing &&
		g.nativeLevelUpDialogue.closeTimer == 0 && g.nativeStoryDialogueAtInputWait()
}

// stepNativeLevelUpDialogue 每幀：字寫完且不等鍵的頁（沒有升級的單頁）直接進 0x1E5C0 的
// 計時；計時到就 0x196CB 關框。
func (g *Game) stepNativeLevelUpDialogue(advance bool) {
	state := g.nativeLevelUpDialogue
	if state == nil || len(g.dialog) == 0 || g.nativeDialogueClosingLive {
		return
	}
	if state.closeTimer > 0 {
		state.closeTimer--
		if advance {
			state.closeTimer = 0
		}
		if state.closeTimer == 0 {
			state.closing = true
			if !g.beginNativeStoryDialogueClosing() {
				g.loadErr = "native level-up dialogue: closing frames unavailable"
			}
		}
		return
	}
	if !g.nativeStoryDialogueAtInputWait() {
		return
	}
	last := g.dlgPage+1 >= dlgPageCount(g.dialog[len(g.dialog)-1])
	if !last {
		if advance {
			g.dlgAdvance()
		}
		return
	}
	if !state.awaitLastKey || advance {
		state.closeTimer = nativeLevelUpCloseFrames
	}
}

func (g *Game) finishNativeLevelUpDialogue() {
	state := g.nativeLevelUpDialogue
	if state == nil {
		return
	}
	g.nativeLevelUpDialogue = nil
	if state.then != nil {
		state.then()
	}
}

// nativeStoryDialogueWaitVariants 回傳對白停在等鍵處時可能出現的整幀：閉嘴／張嘴（0x16C57
// 的 DATO 嘴型倒數吃 rand）× 箭頭兩格（FDOTHER#5 cell18／19 交替）。原版 checkpoint 沒記
// 這兩個相位，重播端各出一張給 verifier 取最小差。
func (g *Game) nativeStoryDialogueWaitVariants() ([][]byte, error) {
	if g == nil || !g.nativeStoryDialogueAtInputWait() || g.nativeClassUI == nil {
		return nil, errors.New("native story dialogue is not at an input wait")
	}
	frames := g.nativeDialogueProgressive[g.dlgPage]
	closed := frames[len(frames)-1]
	bases := [][]byte{closed}
	if g.dlgPage < len(g.nativeDialogueMouthOpen) && len(g.nativeDialogueMouthOpen[g.dlgPage]) == 320*200 {
		bases = append(bases, g.nativeDialogueMouthOpen[g.dlgPage])
	}
	arrow := g.dlgPage+1 < len(g.nativeDialogueProgressive) ||
		(g.nativeLevelUpDialogue != nil && g.nativeLevelUpDialogue.awaitLastKey && g.nativeLevelUpDialogue.closeTimer == 0)
	out := [][]byte{}
	for _, base := range bases {
		if !arrow {
			out = append(out, append([]byte(nil), base...))
			continue
		}
		for phase := 0; phase < 2; phase++ {
			frame, err := campaign.ComposeNativeStoryDialogueWaitArrow(base, g.nativeClassUI.dialogue, g.nativeDialogueLayout, phase)
			if err != nil {
				return nil, err
			}
			out = append(out, frame)
		}
	}
	return out, nil
}
