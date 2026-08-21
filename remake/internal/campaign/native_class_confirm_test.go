package campaign

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestNativeBattleEndTurnFramesDoNotAliasAndRedrawPortrait(t *testing.T) {
	dialogue := make([]byte, 320*200)
	portrait := dato.Frame{Width: 3, Height: 1, Pixels: []byte{1, 2, 3}}
	strings, font := nativeClassListStrings(t), nativeClassListFont(t)
	question, err := ComposeNativeBattleEndTurnQuestion(dialogue, portrait, strings, font)
	if err != nil {
		t.Fatal(err)
	}
	questionBefore := append([]byte(nil), question...)
	accepted, err := ComposeNativeBattleEndTurnResponse(question, strings, font, true)
	if err != nil {
		t.Fatal(err)
	}
	canceled, err := ComposeNativeBattleEndTurnResponse(question, strings, font, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(question, questionBefore) {
		t.Fatal("預先合成YES／NO回覆污染了問句畫面")
	}
	accepted[0], canceled[1] = 7, 8
	if question[0] == 7 || question[1] == 8 || dialogue[0] == 7 || dialogue[1] == 8 {
		t.Fatal("question／accepted／canceled／dialogue仍共用底層像素")
	}
	if got := question[0x9017]; got != 1 {
		t.Fatalf("0x16559(0) right-edge pixel=%d want 1", got)
	}
	if got := question[0x9017-2 : 0x9017+1]; got[0] != 3 || got[1] != 2 || got[2] != 1 {
		t.Fatalf("0x4E8E1由右往左寫入=%v", got)
	}
	if got := dialogue[0x9017]; got != 0 {
		t.Fatalf("問句合成污染dialogue portrait pixel=%d", got)
	}
}

func TestNativeBattleEndTurnResponsePublishesOneFramePerGlyph(t *testing.T) {
	question := make([]byte, 320*200)
	strings, font := nativeClassListStrings(t), nativeClassListFont(t)
	for _, accepted := range []bool{true, false} {
		frames, err := NativeBattleEndTurnResponseFrames(question, strings, font, accepted)
		if err != nil {
			t.Fatal(err)
		}
		index := nativeBattleEndCanceledIndex
		if accepted {
			index = nativeBattleEndAcceptedIndex
		}
		words, err := strings.Words(index)
		if err != nil {
			t.Fatal(err)
		}
		visible := 0
		for _, word := range words {
			if word < fdtxt.ControlMin {
				visible++
			}
		}
		if len(frames) != visible {
			t.Fatalf("accepted=%v frames=%d visible glyphs=%d", accepted, len(frames), visible)
		}
		for i := 1; i < len(frames); i++ {
			if bytes.Equal(frames[i-1], frames[i]) {
				t.Fatalf("accepted=%v frame %d did not add a glyph", accepted, i)
			}
		}
		if len(frames) > 1 {
			before := append([]byte(nil), frames[0]...)
			frames[len(frames)-1][0] = 99
			if !bytes.Equal(frames[0], before) {
				t.Fatalf("accepted=%v progressive frames alias", accepted)
			}
		}
	}
}

func nativeClassConfirmCells() []fdother.RawCell {
	cells := make([]fdother.RawCell, 53)
	for _, index := range []int{16, 17, 48, 49, 51, 52} {
		height := 16
		if index == 16 || index == 17 {
			height = 20
		}
		cells[index] = fdother.RawCell{Width: 24, Height: height, Pixels: make([]byte, 24*height)}
		for i := range cells[index].Pixels {
			cells[index].Pixels[i] = byte(index)
		}
	}
	return cells
}

func TestNativeClassConfirmationOpenCloseCoordinates(t *testing.T) {
	background := make([]byte, 320*200)
	cells := nativeClassConfirmCells()
	open, err := NativeClassConfirmationOpeningFrames(background, cells)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 4 {
		t.Fatalf("open frames=%d", len(open))
	}
	for i, spread := range []int{4, 8, 12, 16} {
		if open[i][168*320+248-spread] != 16 || open[i][168*320+248+spread] != 17 {
			t.Fatalf("open frame%d spread%d mismatch", i, spread)
		}
	}
	closeFrames, err := NativeClassConfirmationClosingFrames(background, cells)
	if err != nil {
		t.Fatal(err)
	}
	for i, spread := range []int{12, 8, 4, 0} {
		if spread == 0 {
			if closeFrames[i][168*320+248] != 17 {
				t.Fatalf("close frame%d overlap did not preserve second-cell overwrite", i)
			}
			continue
		}
		if closeFrames[i][168*320+248-spread] != 16 || closeFrames[i][168*320+248+spread] != 17 {
			t.Fatalf("close frame%d spread%d mismatch", i, spread)
		}
	}
}

func TestComposeNativeClassConfirmationExpandsNameAndSelection(t *testing.T) {
	strings := nativeClassListStrings(t)
	frame, err := ComposeNativeClassConfirmationFrame(
		make([]byte, 320*200), nativeClassConfirmCells(),
		strings, nativeClassListFont(t), 1, 1, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[119*320+12]; got != 205 {
		t.Fatalf("dynamic name foreground=%d want 205", got)
	}
	if got := frame[168*320+232]; got != 48 {
		t.Fatalf("option0 cell=%d want 48", got)
	}
	if got := frame[168*320+264]; got != 52 {
		t.Fatalf("selected option1 pulse cell=%d want 52", got)
	}
}

func TestComposeNativePreparationConfirmationUses31D3CAnchor(t *testing.T) {
	background := make([]byte, 320*200)
	dialogue := make([]fdother.RawCell, 20)
	for index := 1; index <= 19; index++ {
		dialogue[index] = fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	frame, err := ComposeNativePreparationConfirmationFrame(
		background,
		nativeClassConfirmCells(),
		dialogue,
		dato.Frame{Width: 1, Height: 1, Pixels: []byte{99}},
		nativeClassListStrings(t),
		nativeClassListFont(t),
		1,
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[119*320+95]; got != 205 {
		t.Fatalf("question foreground=%d want 205", got)
	}
	if got := frame[120*320+94]; got != 76 {
		t.Fatalf("question shadow=%d want 76", got)
	}
	if got := frame[0x9017]; got != 99 {
		t.Fatalf("DATO#75 default portrait anchor=%d want 99", got)
	}
	if got := frame[168*320+232]; got != 48 {
		t.Fatalf("option0 cell=%d want 48", got)
	}
	if got := frame[168*320+264]; got != 52 {
		t.Fatalf("selected option1 pulse cell=%d want 52", got)
	}
	if got := background[119*320+95]; got != 0 {
		t.Fatalf("input background mutated: %d", got)
	}
	dialogueFrame, err := ComposeNativePreparationConfirmationDialogue(
		background, dialogue, dato.Frame{Width: 1, Height: 1, Pixels: []byte{99}},
	)
	if err != nil {
		t.Fatal(err)
	}
	question, err := ComposeNativePreparationConfirmationQuestion(
		background, dialogue,
		dato.Frame{Width: 1, Height: 1, Pixels: []byte{99}},
		nativeClassListStrings(t), nativeClassListFont(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	open, err := NativePreparationConfirmationOpeningFrames(
		background, dialogueFrame, question, nativeClassConfirmCells(),
	)
	if err != nil || len(open) != 10 {
		t.Fatalf("opening frames=%d err=%v", len(open), err)
	}
	closeFrames, err := NativePreparationConfirmationClosingFrames(
		background, dialogueFrame, question, nativeClassConfirmCells(),
	)
	if err != nil || len(closeFrames) != 9 {
		t.Fatalf("closing frames=%d err=%v", len(closeFrames), err)
	}
	if open[5][112*320+5] != dialogueFrame[112*320+5] {
		t.Fatal("sixth dialogue opening frame did not reach stable target")
	}
	if open[6][168*320+244] != 16 || open[9][168*320+232] != 16 {
		t.Fatal("choice opening did not follow the six dialogue frames")
	}
	if closeFrames[0][168*320+236] != 16 ||
		closeFrames[3][168*320+248] != 17 {
		t.Fatal("choice closing did not precede dialogue closing")
	}
}

func TestComposeNativePreparationPromptsUseCallerSpecificFDTXTAnchors(t *testing.T) {
	background := make([]byte, 320*200)
	dialogue := make([]fdother.RawCell, 20)
	for index := 1; index <= 19; index++ {
		dialogue[index] = fdother.RawCell{
			Width: 1, Height: 1, Pixels: []byte{byte(index)},
		}
	}
	portrait := dato.Frame{Width: 1, Height: 1, Pixels: []byte{99}}
	departure, err := ComposeNativePreparationDepartureQuestion(
		background, dialogue, portrait,
		nativeClassListStrings(t), nativeClassListFont(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	record, err := ComposeNativePreparationRecordQuestion(
		background, dialogue, portrait,
		nativeClassListStrings(t), nativeClassListFont(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	if departure[119*320+95] != 205 {
		t.Fatal("town departure question did not start at x=95")
	}
	if record[119*320+100] != 205 || record[119*320+95] == 205 {
		t.Fatal("standalone record question did not start at x=100")
	}
}
