package fdother

import "fmt"

const (
	NativeCh28PostLMIResource = 5
	NativeCh28PostLMIFirst    = 0x44
	NativeCh28PostLMILast     = 0x4f
	NativeCh28PostSFXResource = 31
	NativeCh28PostSFXIndex    = 3
	NativeCh28PostSFXArg      = 1

	NativeCh28PostPoseFrames = 13
	NativeCh28PostWorkSize   = 0x25680
	NativeCh28PostStride     = 0x1c8
	NativeCh28PostViewBase   = 0x8088
)

// NativeCh28PostOverlayFrame preserves one 0x1DB65 present boundary. Phase
// remains a raw control-flow label; the LMI cells do not yet have proven
// player-facing names.
type NativeCh28PostOverlayFrame struct {
	Phase string
	Entry int
}

// NativeCh28PostOverlayFrames returns the exact two six-frame loops at
// 0x1DDB2..0x1DE9C. The second loop restores the staged snapshot before each
// frame; that buffer transaction belongs to the presentation owner.
func NativeCh28PostOverlayFrames() []NativeCh28PostOverlayFrame {
	frames := make([]NativeCh28PostOverlayFrame, 0, 12)
	for entry := NativeCh28PostLMIFirst; entry < NativeCh28PostLMIFirst+6; entry++ {
		frames = append(frames, NativeCh28PostOverlayFrame{Phase: "overlay_a", Entry: entry})
	}
	for entry := NativeCh28PostLMIFirst + 6; entry <= NativeCh28PostLMILast; entry++ {
		frames = append(frames, NativeCh28PostOverlayFrame{Phase: "overlay_b", Entry: entry})
	}
	return frames
}

// NativeCh28PostOverlayOrigin reproduces 0x1DBE7..0x1DC23 and returns a byte
// offset into the original 0x25680-byte, 456-stride work surface. Coordinates
// and camera values are the raw unit/global integers, not normalized pixels.
func NativeCh28PostOverlayOrigin(x, y, cameraX, cameraY int) (int, error) {
	offset := NativeCh28PostViewBase +
		24*(x-1-cameraX) +
		24*NativeCh28PostStride*(y-1-cameraY) -
		6*NativeCh28PostStride
	if offset < 0 || offset >= NativeCh28PostWorkSize {
		return 0, fmt.Errorf("fdother: ch28 post overlay origin %#x is outside work surface", offset)
	}
	return offset, nil
}

// BlitNativeCh28PostOverlay applies the exact transparent 0x4E85B cell write
// at a precomputed native work-buffer address. Presentation and timing remain
// mandatory responsibilities of the caller.
func BlitNativeCh28PostOverlay(entry LMI1Entry, work []byte, origin int) error {
	if len(work) != NativeCh28PostWorkSize || origin < 0 || origin >= len(work) {
		return fmt.Errorf("fdother: invalid ch28 post work surface or origin")
	}
	return entry.BlitAt(
		work, NativeCh28PostStride,
		origin%NativeCh28PostStride, origin/NativeCh28PostStride, false,
	)
}

// ValidateNativeCh28PostLMI rejects partial or caller-renamed resource banks.
// Only entries 0x44..0x4F are required by this presenter.
func ValidateNativeCh28PostLMI(entries []LMI1Entry) error {
	if len(entries) <= NativeCh28PostLMILast {
		return fmt.Errorf("fdother: FDOTHER#5 lacks ch28 post entries %#x..%#x", NativeCh28PostLMIFirst, NativeCh28PostLMILast)
	}
	for index := NativeCh28PostLMIFirst; index <= NativeCh28PostLMILast; index++ {
		entry := entries[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return fmt.Errorf("fdother: invalid FDOTHER#5 entry %#x", index)
		}
	}
	return nil
}
