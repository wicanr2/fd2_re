package campaign

import (
	"bytes"
	"testing"
)

func TestNativeTownDepartureGeometryPreservesSourceSampling(t *testing.T) {
	source := make([]byte, 64000)
	for i := range source {
		source[i] = byte((i*17+i/320*31)%251 + 1)
	}
	before := append([]byte(nil), source...)
	// The exit coordinates are pinned by the canonical 18-byte EXE tables.
	for variant, xy := range [][2]int{{59, 163}, {53, 163}, {59, 163}} {
		for step := 1; step <= 10; step++ {
			got, err := ComposeNativeTownDepartureFrame(source, variant, 2, step)
			if err != nil {
				t.Fatal(err)
			}
			spacing := 128 - 9*step
			cx := 128 * (160 + (xy[0]-150)*step/10)
			cy := 128 * (100 + (xy[1]-100)*step/10)
			geometry, err := NativeTownDepartureGeometryFor(variant, 2, step)
			if err != nil || geometry.Spacing != spacing || geometry.Center.X != cx || geometry.Center.Y != cy {
				t.Fatal("changed fixed-point geometry", variant, step, geometry, err)
			}
			for y := 0; y < 200; y++ {
				for x := 0; x < 320; x++ {
					sx, sy := cx+(x-160)*spacing, cy+(y-100)*spacing
					want := byte(0)
					if sx >= 0 && sx < 320*128 && sy >= 0 && sy < 200*128 {
						want = source[sy/128*320+sx/128]
					}
					if got[y*320+x] != want {
						t.Fatalf("source sample changed v%d step%d (%d,%d)", variant, step, x, y)
					}
				}
			}
		}
	}
	if !bytes.Equal(source, before) {
		t.Fatal("source changed")
	}
	for _, state := range [][3]int{{-1, 2, 1}, {3, 2, 1}, {0, -1, 1}, {0, 6, 1}, {0, 2, 0}, {0, 2, 11}} {
		if _, err := NativeTownDepartureGeometryFor(state[0], state[1], state[2]); err == nil {
			t.Fatal("invalid geometry accepted", state)
		}
	}
}
