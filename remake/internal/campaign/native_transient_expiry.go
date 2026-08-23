package campaign

import (
	"errors"

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
