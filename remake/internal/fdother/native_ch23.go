package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// NativeCh23StageStride/Height are the raw 0x53AFF staging geometry used by
	// 0x24D22 and the dword_53C03==23 branch of 0x11EEE.
	NativeCh23StageStride = 312
	NativeCh23StageWidth  = 312
	NativeCh23StageHeight = 192
	// NativeCh23StageResource is the FDOTHER archive entry passed through the
	// raw 0x111ba call in 0x10652's chapter-23 branch.  It is an archive index,
	// not a semantic asset name.
	NativeCh23StageResource = 42
	// NativeCh23PaletteBase is the VGA DAC index written by 0x4DFCC.  The
	// original instruction is `mov ah,0xe0` before OUT 0x3c8, so this is not
	// the low 0x20 palette window used by an older, withdrawn interpretation.
	NativeCh23PaletteBase = 0xe0
)

// DecodeNativeCh23Stage mirrors the proven 0x10652 chapter-23 loader boundary:
// FDOTHER #42 is read without conversion, then validated as the exact
// 312×192 single-frame payload consumed by 0x4e63d at 0x10809.  The returned
// Frame still carries only raw geometry and RLE bytes; it does not claim a
// background, transition, or UI role.
func DecodeNativeCh23Stage(datPath string) (Frame, error) {
	frame, err := DecodeArchiveSingleFrame(datPath, NativeCh23StageResource)
	if err != nil {
		return Frame{}, err
	}
	if frame.X != 0 || frame.Y != 0 || frame.Width != NativeCh23StageWidth || frame.Height != NativeCh23StageHeight {
		return Frame{}, fmt.Errorf("fdother: ch23 resource #%d geometry is %dx%d at (%d,%d), want %dx%d at (0,0)", NativeCh23StageResource, frame.Width, frame.Height, frame.X, frame.Y, NativeCh23StageWidth, NativeCh23StageHeight)
	}
	return frame, nil
}

// LoadSeparatedNativeCh23Stage loads the standard indexed surface exported
// from FDOTHER #42. It validates the chapter-specific raw identity in addition
// to the generic surface contract and never falls back to the archive.
func LoadSeparatedNativeCh23Stage(surfaceRoot string) (Frame, error) {
	metadataPath := filepath.Join(surfaceRoot, "FDOTHER_042", "resource.json")
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return Frame{}, fmt.Errorf("fdother: separated ch23 metadata: %w", err)
	}
	var document separatedSurfaceDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return Frame{}, fmt.Errorf("fdother: separated ch23 metadata: %w", err)
	}
	if document.Source.RawSize != 59412 ||
		document.Source.RawMD5 != "b9cd7793d8eec9c80bad5a364029a7a8" ||
		document.Source.RawSHA256 != "4bef756aaf78b95cf97949785c71b4c3e04822a497741f468e6863d61da38d2d" {
		return Frame{}, errors.New("fdother: separated ch23 raw identity mismatch")
	}
	frame, err := LoadSeparatedSingleFrame(surfaceRoot, "FDOTHER.DAT", NativeCh23StageResource)
	if err != nil {
		return Frame{}, err
	}
	if frame.X != 0 || frame.Y != 0 || frame.Width != NativeCh23StageWidth || frame.Height != NativeCh23StageHeight {
		return Frame{}, errors.New("fdother: invalid separated ch23 stage geometry")
	}
	return frame, nil
}

// BlitNativeCh23Stage reproduces the raw 0x4e63d call at 0x10809.  It accepts
// only the exact 0xea00-byte staging surface allocated by 0x107dd and the
// verified transparent mode; invalid input is rejected before writing.
func BlitNativeCh23Stage(frame Frame, staging []byte) error {
	if len(staging) != NativeCh23StageStride*NativeCh23StageHeight {
		return errors.New("fdother: invalid ch23 staging surface")
	}
	if frame.X != 0 || frame.Y != 0 || frame.Width != NativeCh23StageWidth || frame.Height != NativeCh23StageHeight {
		return errors.New("fdother: invalid ch23 stage frame geometry")
	}
	// Decode into a private surface first.  A malformed RLE stream must not
	// leave a partially mutated native staging buffer behind.
	next := append([]byte(nil), staging...)
	if err := frame.Blit(next, NativeCh23StageStride, -1); err != nil {
		return err
	}
	copy(staging, next)
	return nil
}

// nativeDACPaletteCycleE0EF is the fixed 31×RGB byte window at linear 0x60003
// used by 0x4DFCC.  The helper selects a byte offset of 3*byte_60002 and
// writes the next 16 RGB triples to DAC indexes 0xe0..0xef. Values are six-bit
// VGA components copied from the fixed FD2.EXE, not inferred from a screenshot.
var nativeDACPaletteCycleE0EF = [31][3]byte{
	{0x0e, 0x15, 0x26}, {0x0d, 0x14, 0x25}, {0x0d, 0x14, 0x25}, {0x0d, 0x14, 0x25},
	{0x0c, 0x13, 0x24}, {0x0c, 0x13, 0x24}, {0x0b, 0x12, 0x23}, {0x0b, 0x12, 0x23},
	{0x0b, 0x12, 0x23}, {0x0b, 0x12, 0x23}, {0x0c, 0x13, 0x24}, {0x0c, 0x13, 0x24},
	{0x0d, 0x14, 0x25}, {0x0e, 0x15, 0x26}, {0x0e, 0x15, 0x26},
	{0x0e, 0x15, 0x26}, {0x0d, 0x14, 0x25}, {0x0d, 0x14, 0x25}, {0x0d, 0x14, 0x25},
	{0x0c, 0x13, 0x24}, {0x0c, 0x13, 0x24}, {0x0b, 0x12, 0x23}, {0x0b, 0x12, 0x23},
	{0x0b, 0x12, 0x23}, {0x0b, 0x12, 0x23}, {0x0c, 0x13, 0x24}, {0x0c, 0x13, 0x24},
	{0x0d, 0x14, 0x25}, {0x0e, 0x15, 0x26}, {0x0e, 0x15, 0x26}, {0x0e, 0x15, 0x26},
}

// RotateNativeCh23Rows reproduces the arg==0 row-copy branch of 0x24D22.
// The last latch rows wrap to the top; all other rows move down by latch.
// Invalid input is rejected before the destination changes, preserving the
// raw helper's bounded gate.
func RotateNativeCh23Rows(buffer []byte, latch int) error {
	if len(buffer) < NativeCh23StageStride*NativeCh23StageHeight ||
		latch < 0 || latch > NativeCh23StageHeight {
		return errors.New("fdother: invalid ch23 staging latch")
	}
	if latch == 0 {
		return nil
	}
	top := append([]byte(nil), buffer[(NativeCh23StageHeight-latch)*NativeCh23StageStride:NativeCh23StageHeight*NativeCh23StageStride]...)
	for row := NativeCh23StageHeight - latch - 1; row >= 0; row-- {
		src := row * NativeCh23StageStride
		dst := (row + latch) * NativeCh23StageStride
		copy(buffer[dst:dst+NativeCh23StageWidth], buffer[src:src+NativeCh23StageWidth])
	}
	copy(buffer[:latch*NativeCh23StageStride], top)
	return nil
}

// ApplyNativeDACPaletteCycleE0EF reproduces 0x4DFCC's 16-entry DAC write.  It
// changes only palette indexes 0xe0..0xef and leaves every other entry intact.
// The phase is the raw byte_60002 value and must be in 0..15.
func ApplyNativeDACPaletteCycleE0EF(dac []byte, phase int) error {
	if len(dac) != 256*3 || phase < 0 || phase > 15 {
		return errors.New("fdother: invalid native DAC E0..EF palette-cycle input")
	}
	next := append([]byte(nil), dac...)
	for i := 0; i < 16*3; i++ {
		rgb := nativeDACPaletteCycleE0EF[(phase*3+i)/3][(phase*3+i)%3]
		next[NativeCh23PaletteBase*3+i] = rgb
	}
	copy(dac, next)
	return nil
}

// AdvanceNativeDACPaletteCycleE0EF reproduces 0x4DFCC's unsigned BIOS-word
// gate and process-lifetime phase update. The DAC changes only when at least
// two low-word ticks elapsed; invalid state leaves every input byte untouched.
func AdvanceNativeDACPaletteCycleE0EF(dac []byte, phase, lastTimerTick, rawTimerTick int) (nextPhase, nextTimerTick int, advanced bool, err error) {
	if len(dac) != 256*3 || phase < 0 || phase > 15 ||
		lastTimerTick < -0x8000 || lastTimerTick > 0x7fff ||
		rawTimerTick < -0x8000 || rawTimerTick > 0x7fff {
		return phase, lastTimerTick, false, errors.New("fdother: invalid native DAC cycle timing state")
	}
	if uint16(rawTimerTick)-uint16(lastTimerTick) < 2 {
		return phase, lastTimerTick, false, nil
	}
	nextPhase = (phase + 1) & 15
	if err := ApplyNativeDACPaletteCycleE0EF(dac, nextPhase); err != nil {
		return phase, lastTimerTick, false, err
	}
	return nextPhase, rawTimerTick, true, nil
}

// NativeCh23LoopSpec 是 raw ch23 handler 的一段固定排程。它只描述
// 0x24c61..0x24cf2 的呼叫次數、0x24d22 stage 值與原始 ESI 參數；不把
// 0x11cac 的畫面內容或 0x11d40 的第三參數命名成遊戲語意。
type NativeCh23LoopSpec struct {
	Phase       string
	Repeat      int
	StageValues []int
	Palette     bool
}

// NativeCh23LoopHooks 是尚未命名的原始呼叫端。呼叫者必須自行提供
// 0x24d22 的 raw latch setter、indexed draw、tick 與（palette 段）
// 0x11d40 的 raw ESI 消費端；缺任何 callback 就拒絕執行，避免把純資料
// 排程誤接成 generic renderer。latch setter 只代表非零 byte 寫入；它不會
// 在此排程器內偷偷執行 arg==0 的列旋轉，因為那是 BIOS tick 閘門下的共享
// 消費端副作用，必須由有原始 state provenance 的 adapter 自行決定。
type NativeCh23LoopHooks struct {
	Latch   func(rawStage int) error
	Palette func(rawESI int) error
	Draw    func() error
	Tick    func() error
}

func nativeCh23LoopSpecValid(spec NativeCh23LoopSpec) bool {
	if spec.Phase == "initial" {
		return !spec.Palette && spec.Repeat == 30 && equalNativeCh23Stages(spec.StageValues, 2, 9)
	}
	if spec.Phase == "palette" {
		return spec.Palette && spec.Repeat == 12 && equalNativeCh23Stages(spec.StageValues, 10, 14)
	}
	return false
}

// IsRecoveredContract reports whether the typed loop is one of the two exact
// schedules admitted by the raw ch23 handler. Runtime adapters use this same
// gate instead of duplicating the stage/repeat contract.
func (spec NativeCh23LoopSpec) IsRecoveredContract() bool {
	return nativeCh23LoopSpecValid(spec)
}

func equalNativeCh23Stages(values []int, first, last int) bool {
	if len(values) != last-first+1 {
		return false
	}
	for i, value := range values {
		if value != first+i {
			return false
		}
	}
	return true
}

// RunNativeCh23Loop 執行已證實的 raw staging 排程，但不發布畫面、不改
// battle/campaign state，也不自行解釋 palette。每次呼叫前先複製 staging
// 與 DAC；任何 callback 或 shape 錯誤都回復兩個 buffer，保留原子拒絕。
func RunNativeCh23Loop(spec NativeCh23LoopSpec, staging, dac []byte, hooks NativeCh23LoopHooks) error {
	if !nativeCh23LoopSpecValid(spec) {
		return errors.New("fdother: invalid ch23 raw loop spec")
	}
	if len(staging) != NativeCh23StageStride*NativeCh23StageHeight || hooks.Latch == nil || hooks.Draw == nil || hooks.Tick == nil {
		return errors.New("fdother: ch23 loop requires exact latch/draw/tick callbacks")
	}
	if spec.Palette && (hooks.Palette == nil || len(dac) != 256*3) {
		return errors.New("fdother: ch23 palette loop requires raw ESI callback and DAC")
	}
	beforeStage := append([]byte(nil), staging...)
	beforeDAC := append([]byte(nil), dac...)
	rollback := func(err error) error {
		copy(staging, beforeStage)
		copy(dac, beforeDAC)
		return err
	}
	rawESI := 0
	for _, stage := range spec.StageValues {
		// Native 0x24c81/0x24cf2 only calls 0x24d22(stage), whose non-zero
		// branch writes byte_0x51a10. Do not infer that the staging rows have
		// already rotated here: 0x11eee's BIOS-tick consumer may call
		// 0x24d22(0) later, and that timing/state belongs to the adapter.
		if err := hooks.Latch(stage); err != nil {
			return rollback(err)
		}
		for i := 0; i < spec.Repeat; i++ {
			if spec.Palette {
				if err := hooks.Palette(rawESI); err != nil {
					return rollback(err)
				}
				rawESI++
			}
			if err := hooks.Draw(); err != nil {
				return rollback(err)
			}
			if err := hooks.Tick(); err != nil {
				return rollback(err)
			}
		}
	}
	return nil
}
