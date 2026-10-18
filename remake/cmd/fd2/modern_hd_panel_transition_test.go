package main

import (
	"bytes"
	"image"
	"testing"
	"time"
)

func TestModernHDPanelRegionsMoveBothTextAndClip(t *testing.T) {
	for _, tc := range []struct {
		frame        int
		clip, want   image.Rectangle
		x, y         float64
		wantX, wantY float64
	}{
		{7, image.Rect(40, 20, 60, 36), image.Rect(8, 20, 28, 36), 40, 20, 8, 20},
		{7, image.Rect(300, 134, 312, 150), image.Rect(300, 6, 312, 22), 300, 134, 300, 6},
		{3, image.Rect(20, 250, 60, 280), image.Rect(20, 346, 60, 376), 20, 250, 20, 346},
	} {
		regions, ok := nativeChurchPanelRegions(tc.frame)
		if !ok {
			t.Fatal("region plan unavailable")
		}
		runs := modernHDPanelRegionText([]modernHDTextRun{{text: "moving", x: tc.x, y: tc.y, clip: tc.clip}}, regions)
		if len(runs) != 1 || runs[0].clip != tc.want || runs[0].x != tc.wantX || runs[0].y != tc.wantY {
			t.Fatalf("frame%d translation = %+v, want clip%v at %v,%v", tc.frame, runs, tc.want, tc.wantX, tc.wantY)
		}
	}
}

func TestModernHDStatusTransitionsKeepSnapshotAndIndexedCache(t *testing.T) {
	for _, tc := range []struct {
		kind     nativeStatusTransitionKind
		commands bool
		count    int
	}{
		{nativeStatusOpening, false, 12}, {nativeStatusClosing, false, 12},
		{nativeStatusClosing, true, 12}, {nativeStatusToCommands, false, 14},
	} {
		g, screen, unit := newModernHDStatusGame(t)
		defer prepareHDChurchText(t, g, screen)()
		data, status, commands, ok := g.prepareNativeUnitStatusData(unit)
		if !ok {
			t.Fatal("status source unavailable")
		}
		source := make([]byte, 64000)
		source[0] = 19
		d := snapshotNativeStatusTransition(source, data, tc.kind, tc.commands)
		var frames [][]byte
		if tc.kind == nativeStatusToCommands {
			for i := 0; i < 14; i++ {
				panel, step := status, i
				if i >= 7 {
					panel, step = commands, 13-i
				}
				frame, err := nativeChurchBottomPanelFrame(source, panel, step)
				if err != nil {
					t.Fatal(err)
				}
				frames = append(frames, frame)
			}
		} else {
			panel := status
			if tc.commands {
				panel = commands
			}
			var err error
			frames, err = nativeChurchPanelFrames(source, panel, tc.kind == nativeStatusOpening)
			if err != nil {
				t.Fatal(err)
			}
		}
		original := make([][]byte, len(frames))
		for i, want := range frames {
			original[i] = append([]byte(nil), want...)
			regions, command, valid := nativeStatusTransitionPass(tc.kind, i, tc.commands)
			panel, ok := d.data.compose(d.data.assets, command)
			if !valid || !ok {
				t.Fatal("recipe unavailable")
			}
			frame := append([]byte(nil), d.source...)
			for _, region := range regions {
				copyNativeChurchPanelRegion(frame, panel, region)
			}
			if !bytes.Equal(frame, want) {
				t.Fatalf("native recipe differs at %d", i)
			}
		}
		// The job survives replacement or modification of the live panel owner.
		source[0] = 77
		data.record[0] ^= 1
		data.base[0] ^= 1
		data.commandBook[0].MPCost = 99
		called := 0
		job := &nativeClassUIJob{frames: frames, statusText: d, after: func() { called++ }}
		g.nativeClassUIJob = job
		withText := false
		for i := 0; i < tc.count; i++ {
			for _, hd := range []bool{true, false, true} {
				g.modernHDText = nil
				if hd {
					g.modernHDText = &modernHDTextFrame{}
				}
				if !g.drawNativeClassUIJob(screen) || !job.drawn || job.frame != i {
					t.Fatal("draw changed acknowledgement schedule")
				}
				if hd {
					withText = withText || len(g.modernHDText.runs) > 0
					regions, _, _ := nativeStatusTransitionPass(tc.kind, i, tc.commands)
					for _, r := range g.modernHDText.runs {
						contained := false
						for _, region := range regions {
							box := image.Rect(region.DestX*2, region.DestY*2, (region.DestX+region.Width)*2, (region.DestY+region.Height)*2)
							contained = contained || (region.Enabled && r.clip.In(box))
						}
						if !contained {
							t.Fatal("text escaped destination region")
						}
					}
				}
				if !bytes.Equal(job.frames[i], original[i]) || g.font.indexedHD != nil {
					t.Fatal("indexed cache or font leaked")
				}
			}
			g.stepNativeClassUILifecycle(time.Time{})
		}
		if !withText || called != 1 || g.nativeClassUIJob != nil {
			t.Fatal("missing transition text or wrong continuation")
		}
	}
}

func TestModernHDItemEveryTransitionPassHasTranslatedText(t *testing.T) {
	g, screen, unit := newModernHDStatusGame(t)
	if !g.prepareNativeItemPanel(unit) {
		t.Fatal("item panel unavailable")
	}
	defer prepareHDChurchText(t, g, screen)()
	textFrames := 0
	for _, closing := range []bool{false, true} {
		g.itemClosing = closing
		for i := 0; i < 12; i++ {
			g.itemAnimStep = i
			g.modernHDText.runs = nil
			if !g.drawNativeItemPanel(screen) {
				t.Fatal("item frame failed")
			}
			if len(g.modernHDText.runs) > 0 {
				textFrames++
			}
		}
	}
	if textFrames < 12 {
		t.Fatal("item transition text was limited to stable frame")
	}
}
