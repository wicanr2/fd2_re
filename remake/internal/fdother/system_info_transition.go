package fdother

import "errors"

const (
	NativeSystemInfoWidth  = 320
	NativeSystemInfoHeight = 200
	NativeSystemInfoBytes  = NativeSystemInfoWidth * NativeSystemInfoHeight
	NativeSystemInfoSteps  = 12
)

type nativeSystemInfoRect struct {
	srcX, srcY int
	dstX, dstY int
	width      int
	height     int
}

func blitNativeSystemInfoRect(dst, src []byte, rect nativeSystemInfoRect) error {
	if rect.width <= 0 || rect.height <= 0 {
		return nil
	}
	if rect.srcX < 0 || rect.srcY < 0 || rect.dstX < 0 || rect.dstY < 0 ||
		rect.srcX+rect.width > NativeSystemInfoWidth ||
		rect.dstX+rect.width > NativeSystemInfoWidth ||
		rect.srcY+rect.height > NativeSystemInfoHeight ||
		rect.dstY+rect.height > NativeSystemInfoHeight {
		return errors.New("fdother: native system info transition rectangle is invalid")
	}
	for row := 0; row < rect.height; row++ {
		srcStart := (rect.srcY+row)*NativeSystemInfoWidth + rect.srcX
		dstStart := (rect.dstY+row)*NativeSystemInfoWidth + rect.dstX
		copy(dst[dstStart:dstStart+rect.width], src[srcStart:srcStart+rect.width])
	}
	return nil
}

func nativeSystemInfoOpenRects(step int) []nativeSystemInfoRect {
	rects := make([]nativeSystemInfoRect, 0, 4)
	dstX, width := 75, 170
	if step <= 4 {
		dstX += (4 - step) * 50
		if dstX+width > NativeSystemInfoWidth {
			width = NativeSystemInfoWidth - dstX
		}
	}
	rects = append(rects, nativeSystemInfoRect{75, 37, dstX, 37, width, 117})
	if step >= 3 {
		dstY := 19
		if step <= 7 {
			dstY += (7 - step) * 6
		}
		rects = append(rects, nativeSystemInfoRect{109, 19, 109, dstY, 102, 17})
	}
	if step >= 6 {
		dstY := 155
		if step <= 9 {
			dstY += (9 - step) * 9
		}
		rects = append(rects, nativeSystemInfoRect{75, 155, 75, dstY, 170, 16})
	}
	if step >= 9 {
		dstY := 172
		if step <= 12 {
			dstY += (12 - step) * 4
		}
		rects = append(rects, nativeSystemInfoRect{129, 172, 129, dstY, 63, 15})
	}
	return rects
}

func nativeSystemInfoCloseRects(step int) []nativeSystemInfoRect {
	rects := nativeSystemInfoOpenRects(step)
	if step >= 5 {
		return rects
	}
	shift := (4 - step) * 50
	srcX, dstX, width := 75, 75-shift, 170
	if dstX < 0 {
		srcX -= dstX
		width += dstX
		dstX = 0
	}
	rects[0] = nativeSystemInfoRect{srcX, 37, dstX, 37, width, 117}
	return rects
}

// NativeSystemInfoTransitionFrames reproduces sub_1B1E7's twelve opening
// presents and independent eleven-to-zero closing pass. Each frame starts
// from the caller's unchanged battlefield baseline; the information surface
// is then copied through the four direct clipped regions recovered from
// sub_1AF1E/sub_1AF99/sub_1B019/sub_1B0AD/sub_1B14B.
func NativeSystemInfoTransitionFrames(baseline, information []byte) (opening, closing [][]byte, err error) {
	if len(baseline) != NativeSystemInfoBytes || len(information) != NativeSystemInfoBytes {
		return nil, nil, errors.New("fdother: native system info surfaces must be 320x200")
	}
	compose := func(rects []nativeSystemInfoRect) ([]byte, error) {
		frame := append([]byte(nil), baseline...)
		for _, rect := range rects {
			if err := blitNativeSystemInfoRect(frame, information, rect); err != nil {
				return nil, err
			}
		}
		return frame, nil
	}
	opening = make([][]byte, 0, NativeSystemInfoSteps)
	for step := 0; step < NativeSystemInfoSteps; step++ {
		frame, err := compose(nativeSystemInfoOpenRects(step))
		if err != nil {
			return nil, nil, err
		}
		opening = append(opening, frame)
	}
	closing = make([][]byte, 0, NativeSystemInfoSteps)
	for step := NativeSystemInfoSteps - 1; step >= 0; step-- {
		frame, err := compose(nativeSystemInfoCloseRects(step))
		if err != nil {
			return nil, nil, err
		}
		closing = append(closing, frame)
	}
	return opening, closing, nil
}
