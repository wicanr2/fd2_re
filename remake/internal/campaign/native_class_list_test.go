package campaign

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func nativeClassListStrings(t *testing.T) *fdtxt.Strings {
	t.Helper()
	const count = 659
	offsetsBytes := count * 2
	raw := make([]byte, offsetsBytes)
	payload := make([]byte, 0, count*4)
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint16(raw[i*2:], uint16(offsetsBytes+len(payload)))
		words := []uint16{0}
		switch i {
		case 1:
			words = []uint16{1}
		case 150:
			words = []uint16{2}
		case 151:
			words = []uint16{3}
		case 0x19f:
			words = []uint16{1, 2}
		case 0x19a:
			words = []uint16{1, 2, 3}
		case 0x19b:
			words = []uint16{1, 2}
		case 0x1a0:
			words = []uint16{1, 2, 3}
		case 0x1a1:
			words = []uint16{1, 2, 3, 4}
		case 0x1a2:
			words = []uint16{1, 2}
		case 0x19c, 0x1a3, 0x1a4, 513:
			words = []uint16{4}
		case 593:
			words = []uint16{4}
		case 594:
			words = []uint16{0xfffc, 4}
		case 658:
			words = []uint16{4}
		}
		for _, word := range words {
			var pair [2]byte
			binary.LittleEndian.PutUint16(pair[:], word)
			payload = append(payload, pair[:]...)
		}
		var end [2]byte
		binary.LittleEndian.PutUint16(end[:], fdtxt.StringEnd)
		payload = append(payload, end[:]...)
	}
	strings, err := fdtxt.Parse(append(raw, payload...))
	if err != nil {
		t.Fatal(err)
	}
	return strings
}

func nativeClassListFont(t *testing.T) *fdtxt.Font {
	t.Helper()
	raw := make([]byte, 5*fdtxt.GlyphBytes)
	for glyph := 1; glyph <= 4; glyph++ {
		for y := 0; y < fdtxt.GlyphHeight; y++ {
			binary.BigEndian.PutUint16(raw[glyph*fdtxt.GlyphBytes+y*2:], 0x8000)
		}
	}
	font, err := fdtxt.ParseFont(raw)
	if err != nil {
		t.Fatal(err)
	}
	return font
}

func TestRenderNativeClassListRowsMatches31019CoordinatesAndColours(t *testing.T) {
	sprite := fdicon.Sprite{
		Pixels: make([]byte, fdicon.NativeSize*fdicon.NativeSize),
		Mask:   make([]byte, fdicon.NativeSize*fdicon.NativeSize),
	}
	sprite.Pixels[0], sprite.Mask[0] = 99, 1
	rows := []NativeClassListRow{
		{Sprite: sprite, NameTextIndex: 1, CurrentClassTextID: 0, TargetClassTextID: 1},
		{Sprite: sprite, NameTextIndex: 1, CurrentClassTextID: 0, TargetClassTextID: 1},
	}
	dst := make([]byte, NativeClassListStride*NativeClassListHeight)
	if err := RenderNativeClassListRows(dst, rows, 1, nativeClassListStrings(t), nativeClassListFont(t)); err != nil {
		t.Fatal(err)
	}
	if got := dst[117*320+14]; got != 99 {
		t.Fatalf("row0 sprite=%d want 99", got)
	}
	if got := dst[(117+26)*320+14]; got != 99 {
		t.Fatalf("row1 sprite=%d want 99", got)
	}
	if got := dst[121*320+40]; got != 205 {
		t.Fatalf("unselected foreground=%d want 205", got)
	}
	if got := dst[(121+26)*320+40]; got != 201 {
		t.Fatalf("selected foreground=%d want 201", got)
	}
	if got := dst[122*320+39]; got != 76 {
		t.Fatalf("native shadow=%d want 76", got)
	}
	if got := dst[121*320+39]; got != 0 {
		t.Fatalf("same-row shadow=%d want 0", got)
	}
	if got := dst[121*320+175]; got != 205 {
		t.Fatalf("FDTXT593 foreground=%d want 205", got)
	}
	if got := dst[121*320+239]; got != 205 {
		t.Fatalf("target class foreground=%d want 205", got)
	}
}

func TestRenderNativeClassListRowsFailsAtomically(t *testing.T) {
	dst := make([]byte, NativeClassListStride*NativeClassListHeight)
	for i := range dst {
		dst[i] = 7
	}
	before := append([]byte(nil), dst...)
	row := NativeClassListRow{
		Sprite:             fdicon.Sprite{Pixels: make([]byte, 1), Mask: make([]byte, 1)},
		NameTextIndex:      1,
		CurrentClassTextID: 0,
		TargetClassTextID:  1,
	}
	if err := RenderNativeClassListRows(dst, []NativeClassListRow{row}, 0, nativeClassListStrings(t), nativeClassListFont(t)); err == nil {
		t.Fatal("malformed sprite unexpectedly rendered")
	}
	for i := range dst {
		if dst[i] != before[i] {
			t.Fatalf("destination mutated at %d", i)
		}
	}
}

func TestComposeNativeClassListFrameUsesFDOTHER14Entry16Geometry(t *testing.T) {
	panel := fdother.LMI1Entry{
		Width:  nativeClassPanelW,
		Height: nativeClassPanelH,
		Pixels: make([]byte, nativeClassPanelW*nativeClassPanelH),
	}
	for i := range panel.Pixels {
		panel.Pixels[i] = 33
	}
	sprite := fdicon.Sprite{
		Pixels: make([]byte, fdicon.NativeSize*fdicon.NativeSize),
		Mask:   make([]byte, fdicon.NativeSize*fdicon.NativeSize),
	}
	frame, err := ComposeNativeClassListFrame(
		make([]byte, 320*200), panel,
		[]NativeClassListRow{{Sprite: sprite, NameTextIndex: 1, CurrentClassTextID: 0, TargetClassTextID: 1}},
		0, nativeClassListStrings(t), nativeClassListFont(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[nativeClassPanelY*320+nativeClassPanelX]; got != 33 {
		t.Fatalf("panel origin=%d want 33", got)
	}
	if _, err := ComposeNativeClassListFrame(
		make([]byte, 320*200),
		fdother.LMI1Entry{Width: 309, Height: 86, Pixels: make([]byte, 309*86)},
		[]NativeClassListRow{{Sprite: sprite, NameTextIndex: 1, CurrentClassTextID: 0, TargetClassTextID: 1}},
		0, nativeClassListStrings(t), nativeClassListFont(t),
	); err == nil {
		t.Fatal("wrong panel geometry unexpectedly accepted")
	}
}

func TestNativeClassListOpeningFramesMatch1974CCropSchedule(t *testing.T) {
	background := make([]byte, 320*200)
	composed := make([]byte, 320*200)
	for y := 112; y < 198; y++ {
		for x := 5; x < 315; x++ {
			composed[y*320+x] = byte(y - 111)
		}
	}
	frames, err := NativeClassListOpeningFrames(background, composed)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 6 {
		t.Fatalf("frames=%d want 6", len(frames))
	}
	destinations := []int{177, 164, 151, 138, 125, 112}
	for i, y := range destinations {
		if got := frames[i][y*320+5]; got != 1 {
			t.Fatalf("frame%d origin=%d want source row1", i, got)
		}
		if y > 0 && frames[i][(y-1)*320+5] != 0 {
			t.Fatalf("frame%d wrote before destination y=%d", i, y)
		}
	}
	if got := frames[0][199*320+5]; got != 23 {
		t.Fatalf("first clipped tail=%d want 23", got)
	}
	if got := frames[5][197*320+5]; got != 86 {
		t.Fatalf("final full tail=%d want 86", got)
	}
}

func TestNativeClassListClosingFramesMatch2D31BCropSchedule(t *testing.T) {
	background := make([]byte, 320*200)
	composed := make([]byte, 320*200)
	for y := 112; y < 198; y++ {
		for x := 5; x < 315; x++ {
			composed[y*320+x] = byte(y - 111)
		}
	}
	frames, err := NativeClassListClosingFrames(background, composed)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 5 {
		t.Fatalf("frames=%d want 5", len(frames))
	}
	destinations := []int{125, 138, 151, 164, 177}
	for i, y := range destinations {
		if got := frames[i][y*320+5]; got != 1 {
			t.Fatalf("frame%d origin=%d want source row1", i, got)
		}
		if y > 0 && frames[i][(y-1)*320+5] != 0 {
			t.Fatalf("frame%d wrote before destination y=%d", i, y)
		}
	}
	if got := frames[0][197*320+5]; got != 73 {
		t.Fatalf("first clipped tail=%d want 73", got)
	}
	if got := frames[4][199*320+5]; got != 23 {
		t.Fatalf("last clipped tail=%d want 23", got)
	}
}

func TestNativeClassListComposesPlayerOriginalResources(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	resource14, err := fdother.ReadResource(fdotherPath, 14)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fdother.ParseLMI1(resource14)
	if err != nil {
		t.Fatal(err)
	}
	backgroundFrame, err := fdother.ParseLMI1FrameEntry(resource14, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) <= 16 || backgroundFrame.Width != 320 || backgroundFrame.Height != 200 ||
		entries[16].Width != 310 || entries[16].Height != 86 {
		t.Fatalf("FDOTHER#14 shape count=%d entry0=%dx%d entry16=%dx%d",
			len(entries), backgroundFrame.Width, backgroundFrame.Height, entries[16].Width, entries[16].Height)
	}
	background := make([]byte, 320*200)
	if err := backgroundFrame.BlitAt(background, 320, 0, -1); err != nil {
		t.Fatal(err)
	}
	textRaw, err := fdother.ReadResource(filepath.Join(base, "FDTXT.DAT"), 0)
	if err != nil {
		t.Fatal(err)
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		t.Fatal(err)
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	units, err := fdicon.DecodeFile(filepath.Join(base, "FDICON.B24"))
	if err != nil {
		t.Fatal(err)
	}
	sprite, err := units.SpriteFor(9, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := ComposeNativeClassListFrame(background, entries[16], []NativeClassListRow{{
		Sprite: sprite, NameTextIndex: 10, CurrentClassTextID: 5, TargetClassTextID: 21,
	}}, 0, strings, font)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := NativeClassListOpeningFrames(background, frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 6 || frame[117*320+14] == background[117*320+14] {
		t.Fatal("player original class-list composition produced no visible native row")
	}
	cells, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		t.Fatal(err)
	}
	question, err := ComposeNativeClassConfirmationFrame(background, cells, strings, font, 10, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	openConfirm, err := NativeClassConfirmationOpeningFrames(question, cells)
	if err != nil {
		t.Fatal(err)
	}
	closeConfirm, err := NativeClassConfirmationClosingFrames(question, cells)
	if err != nil {
		t.Fatal(err)
	}
	questionChanged := false
	for y := 119; y < 136 && !questionChanged; y++ {
		for x := 12; x < 220; x++ {
			if question[y*320+x] != background[y*320+x] {
				questionChanged = true
				break
			}
		}
	}
	if len(openConfirm) != 4 || len(closeConfirm) != 4 || !questionChanged {
		t.Fatal("player original confirmation composition produced no visible native question")
	}
}
