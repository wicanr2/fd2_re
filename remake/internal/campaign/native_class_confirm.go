package campaign

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	nativeConfirmBaseX             = 248
	nativeConfirmY                 = 168
	nativePreparationQuestionIndex = 658
	nativePreparationQuestionX     = 95
	nativePreparationQuestionY     = 119
	nativeDepartureQuestionIndex   = 513
	nativeDepartureQuestionX       = 95
	nativeRecordQuestionIndex      = 410
	nativeRecordQuestionX          = 100
	nativeBattleEndQuestionIndex   = 0x1a3
	nativeBattleEndAcceptedIndex   = 0x1a4
	nativeBattleEndCanceledIndex   = 0x19c
	nativeBattleExitQuestionIndex  = 0x19f
	nativeBattleExitAcceptedIndex  = 0x1a0
	nativeBattleMarchQuestionIndex = 0x1a1
	nativeBattleMarchAcceptedIndex = 0x1a2
	nativeBattleEndQuestionX       = 99
	nativeBattleEndQuestionY       = 127
	nativeBattleEndResponseY       = 146
)

// ComposeNativeBattleEndTurnQuestion 重現 END caller 透過0x15f84把
// FDTXT#0x1a3寫到畫面座標(99,127)；輸入必須先是0x1956b(0x4b)完成的
// DATO#75對話框畫面。
func ComposeNativeBattleEndTurnQuestion(
	dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font,
) ([]byte, error) {
	return composeNativeBattleSystemQuestion(
		dialogue, portrait, strings, font, nativeBattleEndQuestionIndex,
	)
}

// ComposeNativeBattleExitQuestion 重現 sub_19DF7 selector3 在相同原版對話座標
// 寫入的 FDTXT#0x19f 問句。
func ComposeNativeBattleExitQuestion(
	dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font,
) ([]byte, error) {
	return composeNativeBattleSystemQuestion(
		dialogue, portrait, strings, font, nativeBattleExitQuestionIndex,
	)
}

// ComposeNativeBattleGroupMarchQuestion 重現 sub_16F55 selector1 的
// FDTXT#0x1a1 問句。
func ComposeNativeBattleGroupMarchQuestion(
	dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font,
) ([]byte, error) {
	return composeNativeBattleSystemQuestion(
		dialogue, portrait, strings, font, nativeBattleMarchQuestionIndex,
	)
}

func composeNativeBattleSystemQuestion(
	dialogue []byte, portrait dato.Frame, strings *fdtxt.Strings, font *fdtxt.Font,
	index int,
) ([]byte, error) {
	frame, err := ComposeNativeChurchTextAt(
		append([]byte(nil), dialogue...), strings, font, index,
		nativeBattleEndQuestionY*320+nativeBattleEndQuestionX,
	)
	if err != nil {
		return nil, err
	}
	// 0x17204緊接著呼叫0x16559(0)，所以frame0在文字後再寫一次。
	if err := blitNativeDialoguePortraitAt(frame, portrait, nativeFacilityPortraitOffset(0x4b)); err != nil {
		return nil, err
	}
	return frame, nil
}

// ComposeNativeBattleEndTurnResponse 在(99,146)套用分支專屬回覆：
// YES使用FDTXT#0x1a4，NO／Escape使用FDTXT#0x19c。
func ComposeNativeBattleEndTurnResponse(
	question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool,
) ([]byte, error) {
	frames, err := NativeBattleEndTurnResponseFrames(question, strings, font, accepted)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), frames[len(frames)-1]...), nil
}

// NativeBattleEndTurnResponseFrames 保留0x15F84在END分支回覆期間直接寫入VGA的
// 可見過程：每個普通字形發布一個畫格；FFFE只把目的位置移到下一行。
func NativeBattleEndTurnResponseFrames(
	question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool,
) ([][]byte, error) {
	index := nativeBattleEndCanceledIndex
	if accepted {
		index = nativeBattleEndAcceptedIndex
	}
	return nativeBattleSystemResponseFrames(question, strings, font, index)
}

// NativeBattleExitResponseFrames 保留 selector3 逐字發布的 FDTXT#0x1a0
// 接受回覆；取消則與 END 共用 FDTXT#0x19c。
func NativeBattleExitResponseFrames(
	question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool,
) ([][]byte, error) {
	index := nativeBattleEndCanceledIndex
	if accepted {
		index = nativeBattleExitAcceptedIndex
	}
	return nativeBattleSystemResponseFrames(question, strings, font, index)
}

// NativeBattleGroupMarchResponseFrames 使用 selector1 接受文字 FDTXT#0x1a2；
// 取消仍共用 FDTXT#0x19c。
func NativeBattleGroupMarchResponseFrames(
	question []byte, strings *fdtxt.Strings, font *fdtxt.Font, accepted bool,
) ([][]byte, error) {
	index := nativeBattleEndCanceledIndex
	if accepted {
		index = nativeBattleMarchAcceptedIndex
	}
	return nativeBattleSystemResponseFrames(question, strings, font, index)
}

func nativeBattleSystemResponseFrames(
	question []byte, strings *fdtxt.Strings, font *fdtxt.Font, index int,
) ([][]byte, error) {
	if len(question) != 320*200 || strings == nil || font == nil {
		return nil, errors.New("campaign: 原版戰鬥結束回覆資產無效")
	}
	words, err := strings.Words(index)
	if err != nil {
		return nil, err
	}
	frame := append([]byte(nil), question...)
	frames := make([][]byte, 0, len(words))
	line, column := 0, 0
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76, Background: 74}
	for _, word := range words {
		if word == 0xfffe {
			line++
			column = 0
			continue
		}
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: 不支援的戰鬥結束回覆控制碼 %#x", word)
		}
		offset := nativeBattleEndResponseY*320 + nativeBattleEndQuestionX +
			line*19*320 + column*fdtxt.GlyphWidth
		if err := font.BlitNativeGlyph(frame, 320, offset, int(word), style); err != nil {
			return nil, err
		}
		frames = append(frames, append([]byte(nil), frame...))
		column++
	}
	if len(frames) == 0 {
		return nil, errors.New("campaign: 原版戰鬥結束回覆沒有可見字形")
	}
	return frames, nil
}

// NativeClassConfirmationOpeningFrames reproduces 0x19953's four opening
// presents with FDOTHER#2 raw cells 16/17.
func NativeClassConfirmationOpeningFrames(background []byte, cells []fdother.RawCell) ([][]byte, error) {
	if err := validateNativeClassConfirmationAssets(background, cells); err != nil {
		return nil, err
	}
	out := make([][]byte, 0, 4)
	for spread := 4; spread <= 16; spread += 4 {
		frame := append([]byte(nil), background...)
		if err := cells[16].BlitAt(frame, 320, nativeConfirmBaseX-spread, nativeConfirmY); err != nil {
			return nil, err
		}
		if err := cells[17].BlitAt(frame, 320, nativeConfirmBaseX+spread, nativeConfirmY); err != nil {
			return nil, err
		}
		out = append(out, frame)
	}
	return out, nil
}

// NativeClassConfirmationClosingFrames reproduces 0x197e5's four presented
// positions. The caller owns the post-close background restore.
func NativeClassConfirmationClosingFrames(background []byte, cells []fdother.RawCell) ([][]byte, error) {
	if err := validateNativeClassConfirmationAssets(background, cells); err != nil {
		return nil, err
	}
	out := make([][]byte, 0, 4)
	for spread := 12; spread >= 0; spread -= 4 {
		frame := append([]byte(nil), background...)
		if err := cells[16].BlitAt(frame, 320, nativeConfirmBaseX-spread, nativeConfirmY); err != nil {
			return nil, err
		}
		if err := cells[17].BlitAt(frame, 320, nativeConfirmBaseX+spread, nativeConfirmY); err != nil {
			return nil, err
		}
		out = append(out, frame)
	}
	return out, nil
}

// ComposeNativeClassConfirmationFrame draws FDTXT #594 with its verified
// FFFC dynamic-name substitution, then the normal/pulsed choice cells
// 48/49 and 51/52 used by 0x19953's input loop.
func ComposeNativeClassConfirmationFrame(
	background []byte,
	cells []fdother.RawCell,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	nameTextIndex, selected, pulse int,
) ([]byte, error) {
	if err := validateNativeClassConfirmationAssets(background, cells); err != nil {
		return nil, err
	}
	if selected < 0 || selected > 1 || pulse < 0 || pulse > 1 || strings == nil || font == nil {
		return nil, errors.New("campaign: invalid native class confirmation state")
	}
	frame, err := ComposeNativeClassConfirmationQuestion(background, strings, font, nameTextIndex)
	if err != nil {
		return nil, err
	}
	return ComposeNativeConfirmationChoices(frame, cells, selected, pulse)
}

// ComposeNativeConfirmationChoices applies 0x19953's stable choice cells over
// a caller-owned question frame.
func ComposeNativeConfirmationChoices(
	question []byte,
	cells []fdother.RawCell,
	selected, pulse int,
) ([]byte, error) {
	if err := validateNativeClassConfirmationAssets(question, cells); err != nil {
		return nil, err
	}
	if selected < 0 || selected > 1 || pulse < 0 || pulse > 1 {
		return nil, errors.New("campaign: invalid native confirmation choice state")
	}
	frame := append([]byte(nil), question...)
	for option, base := range []int{48, 51} {
		index := base
		if option == selected {
			index += pulse
		}
		x := nativeConfirmBaseX + []int{-16, 16}[option]
		if err := cells[index].BlitAt(frame, 320, x, nativeConfirmY); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

// ComposeNativePreparationConfirmationQuestion reproduces 0x31d3c's
// 0x15f84 call: FDTXT index 0x292 is drawn at framebuffer address 0xa951f,
// which is screen coordinate (95,119), before 0x19953 owns the two choices.
func ComposeNativePreparationConfirmationQuestion(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
) ([]byte, error) {
	return composeNativePreparationQuestion(
		background, dialogueCells, portrait, strings, font,
		nativePreparationQuestionIndex, nativePreparationQuestionX,
	)
}

// ComposeNativePreparationDepartureQuestion reproduces 0x2d0d1's
// town-backed FDTXT index 0x201 prompt at 0xa951f.
func ComposeNativePreparationDepartureQuestion(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
) ([]byte, error) {
	return composeNativePreparationQuestion(
		background, dialogueCells, portrait, strings, font,
		nativeDepartureQuestionIndex, nativeDepartureQuestionX,
	)
}

// ComposeNativePreparationRecordQuestion reproduces 0x2cc29's standalone
// FDTXT index 0x19a prompt at 0xa9524.
func ComposeNativePreparationRecordQuestion(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
) ([]byte, error) {
	return composeNativePreparationQuestion(
		background, dialogueCells, portrait, strings, font,
		nativeRecordQuestionIndex, nativeRecordQuestionX,
	)
}

func composeNativePreparationQuestion(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	textIndex, textX int,
) ([]byte, error) {
	if len(background) != 320*200 || strings == nil || font == nil {
		return nil, errors.New("campaign: native preparation confirmation assets are unavailable")
	}
	frame, err := ComposeNativePreparationConfirmationDialogue(
		background, dialogueCells, portrait,
	)
	if err != nil {
		return nil, err
	}
	question, err := strings.Words(textIndex)
	if err != nil {
		return nil, err
	}
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76}
	for i, word := range question {
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: unsupported preparation confirmation control %#x", word)
		}
		if err := font.BlitNativeGlyph(
			frame, 320,
			nativePreparationQuestionY*320+textX+i*fdtxt.GlyphWidth,
			int(word), style,
		); err != nil {
			return nil, err
		}
	}
	// 0x31d70 immediately calls 0x16559(0), so DATO frame zero is written
	// once more after the text layout and owns every overlapping pixel.
	if err := blitNativeDialoguePortraitAt(
		frame, portrait, nativeFacilityPortraitOffset(0x4b),
	); err != nil {
		return nil, err
	}
	return frame, nil
}

// ComposeNativePreparationConfirmationDialogue returns 0x1956b(0x4b)'s
// stable dialogue target before 0x15f84 writes FDTXT index 0x292.
func ComposeNativePreparationConfirmationDialogue(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
) ([]byte, error) {
	return ComposeNativeChurchDialogueOverlayAt(
		background, dialogueCells, portrait, nativeFacilityPortraitOffset(0x4b),
	)
}

// ComposeNativePreparationConfirmationFrame adds the stable 0x19953 choice
// state to the caller-owned preparation screen. Opening/closing presentation
// remains caller-owned and is not implied by this compositor.
func ComposeNativePreparationConfirmationFrame(
	background []byte,
	cells []fdother.RawCell,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	selected, pulse int,
) ([]byte, error) {
	question, err := ComposeNativePreparationConfirmationQuestion(
		background, dialogueCells, portrait, strings, font,
	)
	if err != nil {
		return nil, err
	}
	return ComposeNativeConfirmationChoices(question, cells, selected, pulse)
}

// NativePreparationConfirmationOpeningFrames preserves the caller sequence:
// 0x1956b reveals the dialogue in six bands, then 0x19953 presents four
// widening choice frames over the completed question.
func NativePreparationConfirmationOpeningFrames(
	source, dialogue, question []byte,
	cells []fdother.RawCell,
) ([][]byte, error) {
	dialogueFrames, err := NativeClassListOpeningFrames(source, dialogue)
	if err != nil {
		return nil, err
	}
	choiceFrames, err := NativeClassConfirmationOpeningFrames(question, cells)
	if err != nil {
		return nil, err
	}
	return append(dialogueFrames, choiceFrames...), nil
}

// NativePreparationConfirmationClosingFrames preserves 0x197e5's four
// choice frames followed by 0x2d31b's five dialogue bands. The caller still
// owns the final untouched source presentation and continuation.
func NativePreparationConfirmationClosingFrames(
	source, dialogue, question []byte,
	cells []fdother.RawCell,
) ([][]byte, error) {
	choiceFrames, err := NativeClassConfirmationClosingFrames(question, cells)
	if err != nil {
		return nil, err
	}
	dialogueFrames, err := NativeClassListClosingFrames(source, dialogue)
	if err != nil {
		return nil, err
	}
	return append(choiceFrames, dialogueFrames...), nil
}

// ComposeNativeReviveConfirmationQuestion reproduces FDTXT590 with the
// selected actor (-4/FFFC) and fee (-6/FFFA) dynamic substitutions.
func ComposeNativeReviveConfirmationQuestion(
	background []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	nameTextIndex, fee int,
) ([]byte, error) {
	if len(background) != 320*200 || strings == nil || font == nil || fee < 0 {
		return nil, errors.New("campaign: native revive confirmation assets/state are invalid")
	}
	question, err := strings.Words(590)
	if err != nil {
		return nil, err
	}
	name, err := strings.Words(nameTextIndex)
	if err != nil {
		return nil, err
	}
	feeWords := make([]uint16, 0, 5)
	for _, digit := range strconv.Itoa(fee) {
		feeWords = append(feeWords, uint16(digit-'0'))
	}
	expanded := make([]uint16, 0, len(question)+len(name)+len(feeWords))
	for _, word := range question {
		switch word {
		case 0xfffc:
			expanded = append(expanded, name...)
		case 0xfffa:
			expanded = append(expanded, feeWords...)
		case 0xfffe:
			expanded = append(expanded, word)
		default:
			if word >= fdtxt.ControlMin {
				return nil, fmt.Errorf("campaign: unsupported revive confirmation control %#x", word)
			}
			expanded = append(expanded, word)
		}
	}
	frame := append([]byte(nil), background...)
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76}
	line, column := 0, 0
	for _, word := range expanded {
		if word == 0xfffe {
			line++
			column = 0
			continue
		}
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: dynamic revive text contains control %#x", word)
		}
		if err := font.BlitNativeGlyph(
			frame, 320,
			(119+line*19)*320+12+column*fdtxt.GlyphWidth,
			int(word), style,
		); err != nil {
			return nil, err
		}
		column++
	}
	return frame, nil
}

// ComposeNativeClassConfirmationQuestion reproduces the class caller's
// FDTXT#594 draw before 0x19953 starts its choice-cell opening.
func ComposeNativeClassConfirmationQuestion(
	background []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	nameTextIndex int,
) ([]byte, error) {
	if len(background) != 320*200 || strings == nil || font == nil {
		return nil, errors.New("campaign: native class confirmation question assets are unavailable")
	}
	question, err := strings.Words(594)
	if err != nil {
		return nil, err
	}
	name, err := strings.Words(nameTextIndex)
	if err != nil {
		return nil, err
	}
	expanded := make([]uint16, 0, len(name)+len(question)-1)
	for _, word := range question {
		if word == 0xfffc {
			expanded = append(expanded, name...)
			continue
		}
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: unsupported class confirmation control %#x", word)
		}
		expanded = append(expanded, word)
	}
	frame := append([]byte(nil), background...)
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76}
	for i, word := range expanded {
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: dynamic class name contains control %#x", word)
		}
		if err := font.BlitNativeGlyph(frame, 320, 119*320+12+i*fdtxt.GlyphWidth, int(word), style); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

func validateNativeClassConfirmationAssets(background []byte, cells []fdother.RawCell) error {
	if len(background) != 320*200 || len(cells) <= 52 {
		return errors.New("campaign: native class confirmation assets are unavailable")
	}
	for _, index := range []int{16, 17, 48, 49, 51, 52} {
		wantHeight := 16
		if index == 16 || index == 17 {
			wantHeight = 20
		}
		if cells[index].Width != 24 || cells[index].Height != wantHeight {
			return fmt.Errorf("campaign: native class confirmation cell %d is %dx%d", index, cells[index].Width, cells[index].Height)
		}
	}
	return nil
}
