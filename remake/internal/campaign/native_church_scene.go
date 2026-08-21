package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	nativeChurchPortraitOffset   = 4*320 + 118
	nativeLowerPortraitRightEdge = 0x9017
	nativeChurchDecorationOffset = 95*320 + 5
	nativeChurchGoldOffset       = 99*320 + 16
	nativeChurchTextOffset       = 119*320 + 12
)

// ComposeNativeChurchScene reproduces the stable 0x3072f indexed scene after
// sub_1956b's six-frame reveal and before 0x2d669 opens the service cells.
// Callers must provide the exact mixed-codec resources used by each callee.
func ComposeNativeChurchScene(
	background []byte,
	decoration fdother.LMI1Entry,
	dialogueCells []fdother.RawCell,
	digitFrames []fdother.Frame,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	gold, textIndex int,
) ([]byte, error) {
	frame, err := ComposeNativeChurchSceneBase(
		background, decoration, dialogueCells, digitFrames, portrait, gold,
	)
	if err != nil {
		return nil, err
	}
	return renderNativeChurchText(frame, strings, font, textIndex)
}

// ComposeNativeChurchSceneBase builds 0x3072f's stable scene before FDTXT585
// or FDTXT586 is written.
func ComposeNativeChurchSceneBase(
	background []byte,
	decoration fdother.LMI1Entry,
	dialogueCells []fdother.RawCell,
	digitFrames []fdother.Frame,
	portrait dato.Frame,
	gold int,
) ([]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 ||
		len(digitFrames) != 10 || gold < 0 || gold > 99_999_999 {
		return nil, errors.New("campaign: native church scene assets/state are invalid")
	}
	frame, err := ComposeNativeChurchDialogueOverlay(background, dialogueCells, portrait)
	if err != nil {
		return nil, err
	}
	if err := decoration.BlitOpaqueAt(
		frame, 320,
		nativeChurchDecorationOffset%320,
		nativeChurchDecorationOffset/320,
		false,
	); err != nil {
		return nil, err
	}
	digits := fmt.Sprintf("%08d", gold)
	for i := 0; i < 8; i++ {
		index := int(digits[i] - '0')
		if err := digitFrames[index].BlitAt(frame, 320, nativeChurchGoldOffset+6*i, -1); err != nil {
			return nil, err
		}
	}
	return frame, nil
}

// ComposeNativeChurchDialogueOverlay reproduces sub_1956b's stable target:
// the FDOTHER#5 19×5 dialogue grid and DATO#131 frame zero over a caller-owned
// source snapshot. Text is intentionally excluded because 0x2d31b closes
// against this buffer after the text was drawn only to VGA.
func ComposeNativeChurchDialogueOverlay(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
) ([]byte, error) {
	return ComposeNativeChurchDialogueOverlayAt(
		background, dialogueCells, portrait, nativeChurchPortraitOffset,
	)
}

// ComposeNativeChurchDialogueOverlayAt is the recovered stable target of
// 0x1956b shared by shop, church, and other facility variants. The portrait
// destination is selected by the native resource-ID switch at 0x195fc.
func ComposeNativeChurchDialogueOverlayAt(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	portraitOffset int,
) ([]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 {
		return nil, errors.New("campaign: native church dialogue overlay assets are invalid")
	}
	frame := append([]byte(nil), background...)
	placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, 112, 19, 5)
	if err != nil {
		return nil, err
	}
	for _, placement := range placements {
		if err := dialogueCells[placement.ResourceIndex].BlitOpaqueAtOffset(
			frame, 320, placement.DestinationByte,
		); err != nil {
			return nil, err
		}
	}
	if err := blitNativeDialoguePortraitAt(frame, portrait, portraitOffset); err != nil {
		return nil, err
	}
	return frame, nil
}

func blitNativeDialoguePortraitAt(frame []byte, portrait dato.Frame, offset int) error {
	if offset == nativeLowerPortraitRightEdge {
		return portrait.BlitRightToLeftAtOffset(frame, 320, offset)
	}
	return portrait.BlitAtOffset(frame, 320, offset)
}

func renderNativeChurchText(
	frame []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	textIndex int,
) ([]byte, error) {
	return ComposeNativeChurchTextAt(
		frame, strings, font, textIndex, nativeChurchTextOffset,
	)
}

// ComposeNativeChurchTextAt applies the proven 0x15f84 church text style at
// a caller-owned VGA offset. It supports only the verified FFFE soft newline.
func ComposeNativeChurchTextAt(
	frame []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	textIndex, baseOffset int,
) ([]byte, error) {
	words, err := strings.Words(textIndex)
	if err != nil {
		return nil, err
	}
	return composeNativeChurchWordsAt(frame, font, words, baseOffset)
}

// ComposeNativeChurchTextWithNameAt expands the proven FFFC actor-name
// control used by facility feedback strings such as FDTXT506.
func ComposeNativeChurchTextWithNameAt(
	frame []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	textIndex, nameTextIndex, baseOffset int,
) ([]byte, error) {
	words, err := strings.Words(textIndex)
	if err != nil {
		return nil, err
	}
	name, err := strings.Words(nameTextIndex)
	if err != nil {
		return nil, err
	}
	expanded := make([]uint16, 0, len(words)+len(name))
	for _, word := range words {
		if word == 0xfffc {
			expanded = append(expanded, name...)
			continue
		}
		expanded = append(expanded, word)
	}
	return composeNativeChurchWordsAt(frame, font, expanded, baseOffset)
}

func composeNativeChurchWordsAt(
	frame []byte,
	font *fdtxt.Font,
	words []uint16,
	baseOffset int,
) ([]byte, error) {
	line, column := 0, 0
	style := fdtxt.NativeGlyphStyle{Foreground: 205, Shadow: 76, Background: 74}
	for _, word := range words {
		if word == 0xfffe {
			line++
			column = 0
			continue
		}
		if word >= fdtxt.ControlMin {
			return nil, fmt.Errorf("campaign: unsupported church text control %#x", word)
		}
		offset := baseOffset + line*19*320 + column*fdtxt.GlyphWidth
		if err := font.BlitNativeGlyph(frame, 320, offset, int(word), style); err != nil {
			return nil, err
		}
		column++
	}
	return frame, nil
}
