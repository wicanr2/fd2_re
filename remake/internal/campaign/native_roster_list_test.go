package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestNativeTwoColumnWindowPreservesStatefulEvenScrollOrigin(t *testing.T) {
	start := 0
	steps := []struct {
		selected, wantStart, wantVisible int
	}{
		{0, 0, 6},
		{5, 0, 6},
		{6, 2, 6},
		{8, 4, 6},
		{7, 4, 6}, // still inside the current window: do not jump back
		{3, 2, 6}, // crossed above start: scroll up exactly one row
		{1, 0, 6},
	}
	for _, step := range steps {
		var visible int
		start, visible = NativeTwoColumnWindow(10, step.selected, start)
		if start != step.wantStart || visible != step.wantVisible {
			t.Fatalf(
				"selection %d window=(%d,%d), want (%d,%d)",
				step.selected, start, visible, step.wantStart, step.wantVisible,
			)
		}
	}
	if start, visible := NativeTwoColumnWindow(1, 0, 0); start != 0 || visible != 1 {
		t.Fatalf("single-entry window=(%d,%d), want (0,1)", start, visible)
	}
}

func TestRenderNativeRosterRowsMatches2EA90Geometry(t *testing.T) {
	sprite := fdicon.Sprite{
		Pixels: make([]byte, fdicon.NativeSize*fdicon.NativeSize),
		Mask:   make([]byte, fdicon.NativeSize*fdicon.NativeSize),
	}
	sprite.Pixels[0], sprite.Mask[0] = 99, 1
	rows := make([]NativeRosterRow, 6)
	for i := range rows {
		rows[i] = NativeRosterRow{Sprite: sprite, NameTextIndex: 1}
	}
	dst := make([]byte, NativeClassListStride*NativeClassListHeight)
	if err := RenderNativeRosterRows(
		dst, rows, 3, nativeClassListStrings(t), nativeClassListFont(t),
	); err != nil {
		t.Fatal(err)
	}
	positions := [][2]int{
		{14, 117}, {146, 117}, {14, 143}, {146, 143}, {14, 169}, {146, 169},
	}
	for i, position := range positions {
		if got := dst[position[1]*320+position[0]]; got != 99 {
			t.Fatalf("row%d sprite=%d want 99", i, got)
		}
	}
	if got := dst[121*320+40]; got != 205 {
		t.Fatalf("row0 foreground=%d want 205", got)
	}
	if got := dst[147*320+172]; got != 201 {
		t.Fatalf("selected row3 foreground=%d want 201", got)
	}
	if got := dst[148*320+171]; got != 76 {
		t.Fatalf("selected row3 shadow=%d want 76", got)
	}
	if got := dst[147*320+171]; got != 0 {
		t.Fatalf("selected row3 same-row shadow=%d want 0", got)
	}
}

func TestComposeNativeRosterFrameUsesEntry16Panel(t *testing.T) {
	panel := fdother.LMI1Entry{
		Width: nativeClassPanelW, Height: nativeClassPanelH,
		Pixels: make([]byte, nativeClassPanelW*nativeClassPanelH),
	}
	for i := range panel.Pixels {
		panel.Pixels[i] = 33
	}
	sprite := fdicon.Sprite{
		Pixels: make([]byte, fdicon.NativeSize*fdicon.NativeSize),
		Mask:   make([]byte, fdicon.NativeSize*fdicon.NativeSize),
	}
	frame, err := ComposeNativeRosterFrame(
		make([]byte, 320*200), panel,
		[]NativeRosterRow{{Sprite: sprite, NameTextIndex: 1}}, 0,
		nativeClassListStrings(t), nativeClassListFont(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[nativeClassPanelY*320+nativeClassPanelX]; got != 33 {
		t.Fatalf("panel origin=%d want 33", got)
	}
}

func TestComposeNativeRosterFrameWithoutNamesKeepsSpriteAndNameRectangleBlank(t *testing.T) {
	panel := fdother.LMI1Entry{
		Width: nativeClassPanelW, Height: nativeClassPanelH,
		Pixels: make([]byte, nativeClassPanelW*nativeClassPanelH),
	}
	for i := range panel.Pixels {
		panel.Pixels[i] = 33
	}
	sprite := fdicon.Sprite{
		Pixels: make([]byte, fdicon.NativeSize*fdicon.NativeSize),
		Mask:   make([]byte, fdicon.NativeSize*fdicon.NativeSize),
	}
	sprite.Pixels[0], sprite.Mask[0] = 99, 1
	frame, err := ComposeNativeRosterFrameWithoutNames(
		make([]byte, 320*200), panel,
		[]NativeRosterRow{{Sprite: sprite, NameTextIndex: 1}}, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[117*320+14]; got != 99 {
		t.Fatalf("sprite origin=%d want 99", got)
	}
	if got := frame[121*320+40]; got != 33 {
		t.Fatalf("blank name rectangle=%d want panel 33", got)
	}
}
