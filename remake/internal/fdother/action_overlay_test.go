package fdother

import "testing"

func TestActionOverlayCellIndex(t *testing.T) {
	state := ActionOverlayState{
		Availability:   [4]int{0, 1, 0, 1},
		DirectionState: [4]int{0x12, 0x14, 0x16, 0x18},
	}
	want := [4]int{54, 62, 66, 74}
	for direction := range want {
		got, err := state.CellIndex(direction)
		if err != nil || got != want[direction] {
			t.Fatalf("direction %d: index=%d err=%v, want %d", direction, got, err, want[direction])
		}
	}
	if _, err := state.CellIndex(4); err == nil {
		t.Fatal("invalid direction was accepted")
	}
}

func TestBattleActionOverlayStateMatches18D8C(t *testing.T) {
	state := BattleActionOverlayState([4]int{0, 1, 0, 1})
	if state.DirectionState != [4]int{0, 1, 2, 3} {
		t.Fatalf("direction states=%v", state.DirectionState)
	}
	want := [4]int{0, 5, 6, 11}
	for direction := range want {
		got, err := state.CellIndex(direction)
		if err != nil || got != want[direction] {
			t.Fatalf("direction %d: index=%d err=%v, want %d", direction, got, err, want[direction])
		}
	}
}

func TestNativeContinueActionOverlayStateMatches16F55InitialTables(t *testing.T) {
	state := NativeContinueActionOverlayState()
	if state.Availability != [4]int{} || state.DirectionState != [4]int{7, 5, 6, 4} {
		t.Fatalf("current-runtime state=%+v", state)
	}
	want := [4]int{21, 15, 18, 12}
	for direction := range want {
		got, err := state.CellIndex(direction)
		if err != nil || got != want[direction] {
			t.Fatalf("direction %d: index=%d err=%v, want %d", direction, got, err, want[direction])
		}
	}
}

func TestActionOverlayOriginMatchesNativeAddressExpression(t *testing.T) {
	got, err := ActionOverlayOrigin(6, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := 0x8088 + 6*0x18 + 5*0x18*0x1c8
	if got != want {
		t.Fatalf("origin=%#x, want %#x", got, want)
	}
	if _, err := ActionOverlayOrigin(-1, 0); err == nil {
		t.Fatal("negative origin was accepted")
	}
}

func TestActionOverlaySnapshotOriginMatchesNativePredecessorCell(t *testing.T) {
	got, err := ActionOverlaySnapshotOrigin(6, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := 0x8088 + 5*0x18 + 4*0x18*0x1c8
	if got != want {
		t.Fatalf("snapshot origin=%#x, want %#x", got, want)
	}
	for _, cursor := range [][2]int{{0, 1}, {1, 0}, {-1, 1}} {
		if _, err := ActionOverlaySnapshotOrigin(cursor[0], cursor[1]); err == nil {
			t.Fatalf("invalid snapshot cursor %v was accepted", cursor)
		}
	}
}

func TestActionOverlaySnapshotCapturesAndRestoresExplicit72By72Region(t *testing.T) {
	const stride, height = 100, 100
	src := make([]byte, stride*height)
	for y := 0; y < height; y++ {
		for x := 0; x < stride; x++ {
			src[y*stride+x] = byte((x + 3*y) & 0xff)
		}
	}
	original := append([]byte(nil), src...)
	snapshot, err := CaptureActionOverlaySnapshot(src, stride, 11, 13)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot) != NativeActionOverlaySnapshotBytes {
		t.Fatalf("snapshot bytes=%d, want %d", len(snapshot), NativeActionOverlaySnapshotBytes)
	}
	for y := 13; y < 13+NativeActionOverlaySnapshotHeight; y++ {
		for x := 11; x < 11+NativeActionOverlaySnapshotWidth; x++ {
			src[y*stride+x] = 0xee
		}
	}
	if err := RestoreActionOverlaySnapshot(src, snapshot, stride, 11, 13); err != nil {
		t.Fatal(err)
	}
	if string(src) != string(original) {
		t.Fatal("snapshot restore did not recover the original rectangle")
	}
	if _, err := CaptureActionOverlaySnapshot(src, stride, 29, 29); err == nil {
		t.Fatal("out-of-bounds 72x72 snapshot was accepted")
	}
	if err := RestoreActionOverlaySnapshot(src, snapshot[:len(snapshot)-1], stride, 11, 13); err == nil {
		t.Fatal("malformed snapshot was accepted")
	}
}

func TestActionOverlayFrameOffsets(t *testing.T) {
	got, err := ActionOverlayFrameOffsets(1, false)
	if err != nil {
		t.Fatal(err)
	}
	want := [4]int{0x390 - 0x8e8, 0x390 - 6, 0x390 + 6, 0x390 + 0x8e8}
	if got != want {
		t.Fatalf("opening offsets=%#x, want %#x", got, want)
	}
	got, err = ActionOverlayFrameOffsets(0, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != [4]int{-0x23a0, 0x378, 0x3a8, 0x2ac0} {
		t.Fatalf("closing offsets=%#x", got)
	}
	if _, err := ActionOverlayFrameOffsets(4, false); err == nil {
		t.Fatal("invalid frame was accepted")
	}
}

func TestBlitActionOverlayFrameUsesNativeCellsAndTransparency(t *testing.T) {
	cells := make([]RawCell, 75)
	for i := range cells {
		cells[i] = RawCell{Width: 1, Height: 1, Pixels: []byte{byte(i + 1)}}
	}
	// Direction 1 intentionally has a transparent pixel, which must preserve dst.
	cells[62].Pixels[0] = 0
	state := ActionOverlayState{
		Availability:   [4]int{0, 1, 0, 1},
		DirectionState: [4]int{0x12, 0x14, 0x16, 0x18},
	}
	const stride = 500
	dst := make([]byte, stride*20)
	for i := range dst {
		dst[i] = 0xee
	}
	const origin = stride*10 + 100
	if err := BlitActionOverlayFrame(cells, state, dst, stride, origin, 1, false); err != nil {
		t.Fatal(err)
	}
	offsets, _ := ActionOverlayFrameOffsets(1, false)
	if got := dst[origin+offsets[0]]; got != 55 { // cell 54
		t.Fatalf("up pixel=%d, want 55", got)
	}
	if got := dst[origin+offsets[1]]; got != 0xee { // transparent cell 62
		t.Fatalf("left transparent pixel=%d, want preserved", got)
	}
	if got := dst[origin+offsets[2]]; got != 67 { // cell 66
		t.Fatalf("right pixel=%d, want 67", got)
	}
	if got := dst[origin+offsets[3]]; got != 75 { // cell 74
		t.Fatalf("down pixel=%d, want 75", got)
	}
}
