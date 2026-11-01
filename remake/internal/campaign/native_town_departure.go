package campaign

import (
	"errors"
	"image"
)

// NativeTownDepartureGeometry retains the original 1/128-pixel source space.
type NativeTownDepartureGeometry struct {
	Center  image.Point
	Spacing int
}

func NativeTownDepartureGeometryFor(variant, selection, step int) (NativeTownDepartureGeometry, error) {
	if variant < 0 || variant >= nativeTownVariantCount || selection < 0 || selection >= 6 || step < 1 || step > 10 {
		return NativeTownDepartureGeometry{}, errors.New("campaign: native town departure geometry unavailable")
	}
	return NativeTownDepartureGeometry{
		Center: image.Pt(128*(160+(nativeTownSelectionX[variant][selection]-150)*step/10),
			128*(100+(nativeTownSelectionY[variant][selection]-100)*step/10)),
		Spacing: 128 - 9*step,
	}, nil
}

// ComposeNativeTownDepartureFrame follows sub_2D093→sub_2FB9F. Coordinates
// remain in 1/128 source pixels; signed IDIV truncates before multiplication.
// Evidence: docs/data/ida/fd2_town_departure_20261005.json.
func ComposeNativeTownDepartureFrame(source []byte, variant, selection, step int) ([]byte, error) {
	geometry, err := NativeTownDepartureGeometryFor(variant, selection, step)
	if len(source) != NativeTownWidth*NativeTownHeight || err != nil {
		return nil, errors.New("campaign: native town departure source or step unavailable")
	}
	spacing, cx, cy := geometry.Spacing, geometry.Center.X, geometry.Center.Y
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
