package campaign

import "errors"

// ComposeNativeTownDepartureFrame follows sub_2D093→sub_2FB9F. Coordinates
// remain in 1/128 source pixels; signed IDIV truncates before multiplication.
// Evidence: docs/data/ida/fd2_town_departure_20261005.json.
func ComposeNativeTownDepartureFrame(source []byte, variant, selection, step int) ([]byte, error) {
	if len(source) != NativeTownWidth*NativeTownHeight || variant < 0 || variant >= nativeTownVariantCount || selection < 0 || selection >= 6 || step < 1 || step > 10 {
		return nil, errors.New("campaign: native town departure source or step unavailable")
	}
	spacing := 128 - 9*step
	cx := 128 * (160 + (nativeTownSelectionX[variant][selection]-150)*step/10)
	cy := 128 * (100 + (nativeTownSelectionY[variant][selection]-100)*step/10)
	frame := make([]byte, NativeTownWidth*NativeTownHeight)
	for y := 0; y < NativeTownHeight; y++ {
		sy := cy - 100*spacing + y*spacing
		if sy < 0 || sy >= NativeTownHeight*128 {
			continue
		}
		for x := 0; x < NativeTownWidth; x++ {
			sx := cx - 160*spacing + x*spacing
			if sx >= 0 && sx < NativeTownWidth*128 {
				frame[y*NativeTownWidth+x] = source[(sy/128)*NativeTownWidth+sx/128]
			}
		}
	}
	return frame, nil
}
