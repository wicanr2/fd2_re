package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

type confirmationMouthObservation struct {
	Step  uint64                      `json:"step"`
	State dato.ConfirmationMouthState `json:"state"`
	Pulse int                         `json:"pulse"`
}

// Trace registers are pre-instruction. EAX at 19BFE is old EBP after DEC;
// EDX at 1998D / 19BF1 is the real IDIV remainder, not a guessed seed stream.
func confirmationMouthReadOracle(t *testing.T, run string) (int, []int, []confirmationMouthObservation) {
	t.Helper()
	metadata, err := os.ReadFile(filepath.Join(run, "runner.json"))
	var runner struct {
		Commit    string `json:"dosgolem_commit"`
		SHA       string `json:"original_fd2_exe_sha256"`
		Dirty     int    `json:"dosgolem_tracked_dirty_files"`
		Untracked int    `json:"dosgolem_untracked_files"`
		Locked    bool   `json:"lock_ally_hp"`
		Cleared   bool   `json:"force_enemy_clear_declared"`
	}
	if err != nil || json.Unmarshal(metadata, &runner) != nil || runner.Commit != "a01ff084f038ddd1d133480f08d32e3b2e6f98c0" || runner.SHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" || runner.Dirty != 0 || runner.Untracked != 0 || runner.Locked || runner.Cleared {
		t.Fatal("original runner source or injection policy differs")
	}
	f, err := os.Open(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	state := dato.ConfirmationMouthState{}
	initial := -1
	var samples []int
	var observations []confirmationMouthObservation
	pending, stepped := false, false
	pulse, openings, closings := 0, 0, 0
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	for scan.Scan() {
		var row struct {
			EIP, EAX, EDX string
			Step          uint64
		}
		if err := json.Unmarshal(scan.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		value := func(raw string) int {
			var n uint32
			if _, err := fmt.Sscanf(raw, "0x%x", &n); err != nil {
				t.Fatal(err)
			}
			return int(int32(n))
		}
		switch row.EIP {
		case "0x1998D":
			if initial != -1 {
				t.Fatal("duplicate mouth entry")
			}
			initial = value(row.EDX)
			state, err = dato.NewConfirmationMouthState(initial)
		case "0x19B96":
			if initial == -1 || pending || value(row.EAX) != boolInt(state.Open) {
				t.Fatal("native open flag or tick order differs", row.Step, state)
			}
			pending, stepped = true, false
		case "0x19BFE":
			if !pending || stepped || state.Open || value(row.EAX) != state.Countdown {
				t.Fatal("native old countdown differs", row.Step, row.EAX, state)
			}
			state, err = state.Tick(0)
			stepped = true
		case "0x19C46":
			if !pending || !stepped || !state.Open || state.Countdown != -1 {
				t.Fatal("native frame3 writer differs")
			}
			openings++
		case "0x19BF1":
			if !pending || stepped || !state.Open {
				t.Fatal("native re-close has no open owner")
			}
			sample := value(row.EDX)
			samples = append(samples, sample)
			state, err = state.Tick(sample)
			stepped = true
		case "0x19BF4":
			if !stepped || state.Open || state.Countdown != value(row.EDX)+10 {
				t.Fatal("native restart is not 10..39")
			}
			closings++
		case "0x19D23":
			pulse = value(row.EAX)
		case "0x19D59":
			if !pending || !stepped || pulse < 0 || pulse > 3 {
				t.Fatal("incomplete native present")
			}
			observations = append(observations, confirmationMouthObservation{row.Step, state, pulse})
			pending = false
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if scan.Err() != nil || initial != 5 || len(observations) != 1561 || openings != 58 || closings != 58 || !pending {
		t.Fatal("original probe coverage differs", initial, len(observations), openings, closings, pending, scan.Err())
	}
	// The last sampled tick is still in the loop at the plan stop; it is not
	// a completed present and is deliberately absent from observations.
	t.Log("1 initial state, 1561 complete mouth ticks, 58 opens/closes; final partial tick excluded")
	return initial, samples, observations
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestNativeConfirmationMouthCurrentOracle(t *testing.T) {
	run := os.Getenv("FD2_CONFIRMATION_MOUTH_ORIGINAL")
	if run == "" {
		t.Skip("requires #177 current original probe")
	}
	confirmationMouthReadOracle(t, run)
}

// Actual original remainders are explicit controlled cosmetic inputs. This
// source never alters the native gameplay RNG or claims global RNG parity.
type confirmationMouthRecordedSource struct {
	samples []int
	next    int
}

func (s *confirmationMouthRecordedSource) Seed(int64) { s.next = 0 }
func (s *confirmationMouthRecordedSource) Int63() int64 {
	if s.next >= len(s.samples) {
		panic("unexpected confirmation cosmetic random read")
	}
	value := s.samples[s.next]
	s.next++
	return int64(value) << 32 // rand.Int31n(30) consumes the recorded remainder.
}

func TestNativeConfirmationMouthNormalUpdateDrawOracle(t *testing.T) {
	run, out := os.Getenv("FD2_CONFIRMATION_MOUTH_ORIGINAL"), os.Getenv("FD2_CONFIRMATION_MOUTH_OUT")
	if run == "" || out == "" {
		t.Skip("requires #177 current original probe and output")
	}
	initial, samples, rows := confirmationMouthReadOracle(t, run)
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_NATIVE_SAVE", "../../../work/parity-slot-ch04/FD2.SAV")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
		t.Fatal("normal title LOAD failed", g.loadErr)
	}
	r := &parityReplay{t: t, g: g, out: out}
	r.settleTown()
	if g.camp.NodeID() != "town_ch04" {
		t.Fatal("original slot owner differs", g.camp.NodeID())
	}
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("normal town selection failed")
		}
	}
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.nativePreparationPromptActive() {
		t.Fatal("normal departure does not own confirmation input")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	frames := map[uint64]*image.Paletted{}
	frameHashes := map[uint64]string{}
	f, err := os.Open(filepath.Join(run, "frames", "frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	for scan.Scan() {
		var row struct {
			Step uint64
			File string
			Hash string `json:"indexed_sha256"`
		}
		if err := json.Unmarshal(scan.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		img := townDepartureReadPNG(t, filepath.Join(run, "frames", row.File))
		frames[row.Step], frameHashes[row.Step] = img, row.Hash
	}
	f.Close()
	if scan.Err() != nil || len(frames) != 88 {
		t.Fatal("original complete frame coverage differs", len(frames), scan.Err())
	}
	source := &confirmationMouthRecordedSource{samples: append([]int{initial}, samples...)}
	g.nativeConfirmationMouthRNG = rand.New(source)
	g.nativeConfirmationMouthOwner = nil
	g.nativeClassUIClock.Reset()
	g.nativeClassUIHasTick, g.nativeClassUIPulse = false, 0
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.nativeConfirmationMouth.Countdown != initial+2 || g.nativeConfirmationMouth.Open {
		t.Fatal("normal owner did not consume recorded initial input")
	}
	probe := &confirmationMouthGPUProbe{t: t, g: g, rows: rows, frames: frames, hashes: frameHashes, out: out}
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != "" || probe.index != len(rows) || len(probe.report) != 88 || source.next != len(source.samples) {
		t.Fatal("normal Update/Draw coverage", probe.err, probe.index, len(probe.report), source.next)
	}
	if !g.handleNativePreparationInput(nativePreparationInput{escape: true}) || !pump(t, g, 600, func() bool { return g.camp.NodeID() == "town_ch04" }) {
		t.Fatal("normal ESC did not restore town")
	}
	encoded, _ := json.MarshalIndent(map[string]any{"ticks": len(rows), "frames": probe.report, "random_reads": source.next, "slot_origin": "constructed chapter4 slot", "method": "normal Game.Update/Draw; explicit original RNG remainders and qualifying BIOS samples; no frame/countdown injection", "cancel_restored_town": true}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "validation.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

type confirmationMouthGPUProbe struct {
	t      *testing.T
	g      *Game
	rows   []confirmationMouthObservation
	frames map[uint64]*image.Paletted
	hashes map[uint64]string
	out    string
	index  int
	await  bool
	report []map[string]any
	err    string
}

func (p *confirmationMouthGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *confirmationMouthGPUProbe) Update() error {
	if p.err != "" || p.index == len(p.rows) {
		return ebiten.Termination
	}
	if p.await {
		return nil
	}
	row := p.rows[p.index]
	// Controlled timer input only. Update performs the real qualified tick,
	// countdown transition and random read; no mouth/pulse output is assigned.
	p.g.nativeClassUIClock.last = time.Now().Add(-2 * nativeBIOSTickPeriod)
	if err := p.g.Update(); err != nil {
		return err
	}
	if p.g.nativeConfirmationMouth != row.State || p.g.nativeClassUIPulse != row.Pulse {
		p.err = fmt.Sprintf("Update state differs at %d: %+v / %d, want %+v / %d", row.Step, p.g.nativeConfirmationMouth, p.g.nativeClassUIPulse, row.State, row.Pulse)
		return ebiten.Termination
	}
	if original := p.frames[row.Step]; original != nil {
		frame, ok := p.g.composeNativePreparationPromptFrame()
		hash := sha256.Sum256(frame)
		if !ok || hex.EncodeToString(hash[:]) != p.hashes[row.Step] || !bytes.Equal(frame, original.Pix) {
			p.err = fmt.Sprintf("complete indexed frame differs at %d", row.Step)
			return ebiten.Termination
		}
		p.await = true
	} else {
		p.index++
	}
	return nil
}

func (p *confirmationMouthGPUProbe) Draw(screen *ebiten.Image) {
	if !p.await || p.err != "" {
		return
	}
	row := p.rows[p.index]
	original := p.frames[row.Step]
	p.g.Draw(screen)
	rgba := make([]byte, 640*400*4)
	screen.ReadPixels(rgba)
	diff := 0
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, a := original.At(x/2, y/2).RGBA()
			offset := (y*640 + x) * 4
			if rgba[offset] != byte(r>>8) || rgba[offset+1] != byte(g>>8) || rgba[offset+2] != byte(b>>8) || rgba[offset+3] != byte(a>>8) {
				diff++
			}
		}
	}
	if diff != 0 {
		p.err = fmt.Sprintf("full GPU RGBA differs at %d: %d pixels", row.Step, diff)
		return
	}
	name := fmt.Sprintf("remake-%012d.png", row.Step)
	file, err := os.Create(filepath.Join(p.out, name))
	if err != nil {
		p.err = err.Error()
		return
	}
	img := &image.RGBA{Pix: rgba, Stride: 640 * 4, Rect: image.Rect(0, 0, 640, 400)}
	err = png.Encode(file, img)
	file.Close()
	if err != nil {
		p.err = err.Error()
		return
	}
	p.report = append(p.report, map[string]any{"step": row.Step, "mouth_frame": row.State.FrameIndex(), "countdown": row.State.Countdown, "pulse": row.Pulse, "indexed_sha256": p.hashes[row.Step], "gpu_rgba_diff_pixels": diff, "file": name})
	p.await, p.index = false, p.index+1
}
