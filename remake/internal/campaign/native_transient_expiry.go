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
	NativeTransientExpiryTextBase   = 0x1e1
	NativeTransientExpiryTextCount  = 6
	NativeTransientExpiryTextOffset = 0x9f23
)

// ComposeNativeTransientExpiryFrame consumes the exact sub_1A866 expiry
// inputs without assigning gameplay names to raw +0x22..+0x27. The original
// DATO +7 selector is resolved by the caller and supplied as a decoded frame.
func ComposeNativeTransientExpiryFrame(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	counterIndex int,
) ([]byte, error) {
	if counterIndex < 0 || counterIndex >= NativeTransientExpiryTextCount {
		return nil, errors.New("campaign: native transient counter index is invalid")
	}
	frame, err := ComposeNativeChurchDialogueOverlayAt(
		background, dialogueCells, portrait, nativeLowerPortraitRightEdge,
	)
	if err != nil {
		return nil, err
	}
	return ComposeNativeChurchTextAt(
		frame, strings, font,
		NativeTransientExpiryTextBase+counterIndex,
		NativeTransientExpiryTextOffset,
	)
}

const (
	// 0x1AA1D 物品型態 0／1 的成功訊息：FDTXT_000 0x1B0「從敵人身上，得到 FFFC！」
	// （FFFC = [0x53AD9] = item+0xB5 的物品名）與 0x1B3「從敵人身上，得到 FFFA 元！」，
	// 都以 0x15F84(buffer, index, 0xA9F23, 320, 205, 76, 74, 19, 1) 寫在對話格
	// 文字座標 (99,127)，和 sub_1A866 到期訊息同一個 offset。
	NativeDeathRewardItemTextIndex   = 0x1b0
	NativeDeathRewardGoldTextIndex   = 0x1b3
	NativeDeathRewardItemNameBase    = 0xb5
	NativeDeathRewardMessageTextOffs = NativeTransientExpiryTextOffset
)

// ComposeNativeDeathRewardMessageFrame 在已由 0x1956B(killer+7) 開好的對話格上寫
// 0x1AA1D 成功訊息。kind 0 是物品（value 為物品 byte），kind 1 是金錢（value 為金額）。
func ComposeNativeDeathRewardMessageFrame(
	dialogue []byte,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	kind, value int,
) ([]byte, error) {
	if len(dialogue) != 320*200 || strings == nil || font == nil || value < 0 {
		return nil, errors.New("campaign: native death reward message assets/state are invalid")
	}
	var words, dynamic []uint16
	var err error
	switch kind {
	case 0:
		if value > 0xff {
			return nil, errors.New("campaign: native death reward item byte is out of range")
		}
		if words, err = strings.Words(NativeDeathRewardItemTextIndex); err != nil {
			return nil, err
		}
		if dynamic, err = strings.Words(value + NativeDeathRewardItemNameBase); err != nil {
			return nil, err
		}
	case 1:
		if words, err = strings.Words(NativeDeathRewardGoldTextIndex); err != nil {
			return nil, err
		}
		for _, digit := range strconv.Itoa(value) {
			dynamic = append(dynamic, uint16(digit-'0'))
		}
	default:
		return nil, fmt.Errorf("campaign: native death reward kind %d has no message", kind)
	}
	expanded := make([]uint16, 0, len(words)+len(dynamic))
	for _, word := range words {
		switch word {
		case 0xfffc, 0xfffa:
			expanded = append(expanded, dynamic...)
		default:
			expanded = append(expanded, word)
		}
	}
	return composeNativeChurchWordsAt(append([]byte(nil), dialogue...), font, expanded, NativeDeathRewardMessageTextOffs)
}
