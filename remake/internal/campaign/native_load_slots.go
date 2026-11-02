package campaign

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	NativeLoadSlotWidth  = 320
	NativeLoadSlotHeight = 200

	nativeLoadFrameX    = 5
	nativeLoadFrameY    = 112
	nativeLoadTextY     = 119
	nativeLoadRowStride = 19
	nativeLoadNumberX   = 10
	nativeLoadEmptyX    = 88
	nativeLoadChapterX  = 40
	nativeLoadTitleX    = 130
)

// NativeLoadSlot preserves only metadata consumed by the proven 0x30437
// renderer. Chapter is the raw saved byte; it is not normalized to a remake
// campaign node.
type NativeLoadSlot struct {
	Empty   bool
	Chapter byte
}

// ComposeNativeLoadSlotsFrame reproduces the stable 0x30437 frame used by
// 0x30550. It deliberately accepts a caller-owned source snapshot because
// 0x30550 copies the current VGA page before opening FDOTHER#13 LMI entry16.
func ComposeNativeLoadSlotsFrame(
	source []byte,
	dialogueBox fdother.LMI1Entry,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	slots [4]NativeLoadSlot,
	selection int,
) ([]byte, error) {
	if len(source) != NativeLoadSlotWidth*NativeLoadSlotHeight ||
		dialogueBox.Width != 310 || dialogueBox.Height != 86 ||
		len(dialogueBox.Pixels) != dialogueBox.Width*dialogueBox.Height ||
		strings == nil || font == nil ||
		selection < 0 || selection >= len(slots) {
		return nil, errors.New("campaign: native load-slot assets/state are invalid")
	}
	frame := append([]byte(nil), source...)
	for row := 0; row < dialogueBox.Height; row++ {
		y := nativeLoadFrameY + row
		if y >= NativeLoadSlotHeight {
			break
		}
		sourceStart := row * dialogueBox.Width
		destinationStart := y*NativeLoadSlotWidth + nativeLoadFrameX
		copy(
			frame[destinationStart:destinationStart+dialogueBox.Width],
			dialogueBox.Pixels[sourceStart:sourceStart+dialogueBox.Width],
		)
	}
	for row, slot := range slots {
		foreground := byte(0xcd)
		if row == selection {
			foreground = 0xc9
		}
		style := fdtxt.NativeGlyphStyle{
			Foreground: foreground,
			Shadow:     0x4c,
		}
		y := nativeLoadTextY + row*nativeLoadRowStride
		if err := renderNativeLoadText(
			frame, strings, font, 0x225,
			nativeLoadNumberX, y, style, row+1,
		); err != nil {
			return nil, err
		}
		if slot.Empty {
			if err := renderNativeLoadText(
				frame, strings, font, 0x202,
				nativeLoadEmptyX, y, style, 0,
			); err != nil {
				return nil, err
			}
			continue
		}
		if err := renderNativeLoadText(
			frame, strings, font, 0x202+int(slot.Chapter),
			nativeLoadChapterX, y, style, 0,
		); err != nil {
			return nil, err
		}
		if err := renderNativeLoadText(
			frame, strings, font, 0x226+int(slot.Chapter),
			nativeLoadTitleX, y, style, 0,
		); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

func renderNativeLoadText(
	frame []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	textIndex, x, y int,
	style fdtxt.NativeGlyphStyle,
	number int,
) error {
	words, err := strings.Words(textIndex)
	if err != nil {
		return err
	}
	expanded := make([]uint16, 0, len(words)+2)
	for _, word := range words {
		if word == 0xfffa {
			for _, digit := range strconv.Itoa(number) {
				expanded = append(expanded, uint16(digit-'0'))
			}
			continue
		}
		if word >= fdtxt.ControlMin {
			return fmt.Errorf(
				"campaign: unsupported native load-slot control %#x",
				word,
			)
		}
		expanded = append(expanded, word)
	}
	for column, word := range expanded {
		offset := y*NativeLoadSlotWidth + x + column*fdtxt.GlyphWidth
		if err := font.BlitNativeGlyph(
			frame, NativeLoadSlotWidth, offset, int(word), style,
		); err != nil {
			return err
		}
	}
	return nil
}
