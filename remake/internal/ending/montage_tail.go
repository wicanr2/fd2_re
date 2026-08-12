package ending

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// MontageTail is the raw, editable schedule after the party montage.  It
// deliberately exposes byte offsets and native call shapes instead of
// assigning names such as unit state, animation, or ending outcome.
type MontageTail struct {
	SchemaVersion int                 `json:"schema_version"`
	NativeHandler string              `json:"native_handler"`
	Status        string              `json:"status"`
	Source        string              `json:"source"`
	Resources     []MontageTailAsset  `json:"resources"`
	RawTables     MontageTailRawTable `json:"raw_tables"`
	Loop          MontageTailLoop     `json:"loop"`
	Gate          MontageTailGate     `json:"gate"`
}

type MontageTailAsset struct {
	Archive string `json:"archive"`
	Index   int    `json:"index"`
	Source  string `json:"source"`
	Role    string `json:"role"`
}

type MontageTailRawTable struct {
	// Record0Byte7, Record1Byte7 and Global540FF retain the three raw
	// 20-byte tables at 0x525dc..0x52617. The byte-6 writes are not a fourth
	// table: native code derives each from its paired byte-7 value.
	Record0Byte7 []int `json:"record0_byte_7"`
	Record1Byte7 []int `json:"record1_byte_7"`
	Global540FF  []int `json:"global_540ff"`
}

type MontageTailLoop struct {
	Count                int    `json:"count"`
	RuntimeRecordsGlobal string `json:"runtime_records_global"`
	RuntimeRecordStride  int    `json:"runtime_record_stride"`
	Record0Byte6Formula  string `json:"record0_byte_6_formula"`
	Record0Byte7Source   string `json:"record0_byte_7_source"`
	Record1Byte6Formula  string `json:"record1_byte_6_formula"`
	Record1Byte7Source   string `json:"record1_byte_7_source"`
	Global540FFSource    string `json:"global_540ff_source"`
	RendererCall         string `json:"renderer_call"`
	PaletteCall          string `json:"palette_call"`
	FrameCall            string `json:"frame_call"`
	WaitBeforeFrameTicks int    `json:"wait_before_frame_ticks"`
	WaitAfterFrameTicks  int    `json:"wait_after_frame_ticks"`
	WaitHelper           string `json:"wait_helper"`
	PresentHelper        string `json:"present_helper"`
	RestoreHelper        string `json:"restore_helper"`
}

type MontageTailGate struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// MontageTailEntry keeps the three raw table bytes together for one native
// loop index. It is a plan only: no renderer, runtime mutation, or campaign
// transition is performed by this package.
type MontageTailEntry struct {
	Index        int
	Global540FF  byte
	Record0Byte6 byte
	Record0Byte7 byte
	Record1Byte6 byte
	Record1Byte7 byte
}

// MontageTailAssets preserves the directly loaded FDOTHER inputs of
// 0x2c194.  The 20-entry renderer still has an unresolved 0x28a6c stage, so
// merely loading these assets does not claim to reproduce that loop.  The
// final #59 frame is kept separately because the native code decodes it only
// after the loop has completed.
type MontageTailAssets struct {
	LoopPalette []byte
	Intro       fdother.Frame
	LoopFrames  []fdother.Frame
	Final       fdother.Frame
}

// LoadMontageTailAssets verifies the native resource shapes before the
// presentation adapter can use them.  It intentionally does not invent a
// meaning for the three per-entry raw bytes or the 0x28a6c rendering owner.
func LoadMontageTailAssets(tail MontageTail, fdotherPath string) (MontageTailAssets, error) {
	if fdotherPath == "" {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER path is unavailable")
	}
	if _, err := tail.Plan(); err != nil {
		return MontageTailAssets{}, err
	}
	palette, err := fdother.ReadResource(fdotherPath, 57)
	if err != nil {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#57: %w", err)
	}
	if len(palette) != 256*3 {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#57 palette length = %d", len(palette))
	}
	if _, err := fdother.ParseVGAPalette(palette); err != nil {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#57 palette: %w", err)
	}
	loopFrames, err := fdother.DecodeResource(fdotherPath, 58)
	if err != nil {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#58: %w", err)
	}
	if len(loopFrames) != tail.Loop.Count {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#58 frames = %d, want %d", len(loopFrames), tail.Loop.Count)
	}
	for i, frame := range loopFrames {
		if err := frame.Blit(make([]byte, Bytes), Width, -1); err != nil {
			return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#58 frame %d: %w", i, err)
		}
	}
	loadSingle := func(index int) (fdother.Frame, error) {
		frame, err := fdother.DecodeArchiveSingleFrame(fdotherPath, index)
		if err != nil {
			return fdother.Frame{}, err
		}
		if frame.Width != Width || frame.Height != Height {
			return fdother.Frame{}, fmt.Errorf("geometry %dx%d, want %dx%d", frame.Width, frame.Height, Width, Height)
		}
		if err := frame.Blit(make([]byte, Bytes), Width, -1); err != nil {
			return fdother.Frame{}, err
		}
		return frame, nil
	}
	intro, err := loadSingle(60)
	if err != nil {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#60: %w", err)
	}
	final, err := loadSingle(59)
	if err != nil {
		return MontageTailAssets{}, fmt.Errorf("ending: montage tail FDOTHER#59: %w", err)
	}
	return MontageTailAssets{
		LoopPalette: append([]byte(nil), palette...),
		Intro:       intro,
		LoopFrames:  append([]fdother.Frame(nil), loopFrames...),
		Final:       final,
	}, nil
}

// PresentFinal renders the final native frame and restores the immutable
// palette baseline that was captured by the preceding indexed ending.  The
// original reaches this image after a separate fade helper; this small
// adapter deliberately exposes only the stable terminal frame, not a claim
// of timing parity for that helper.
func (assets MontageTailAssets) PresentFinal(compositor *IndexedCompositor) error {
	if compositor == nil || !compositor.BaselineKnown() {
		return fmt.Errorf("ending: montage tail final frame lacks palette baseline")
	}
	if assets.Final.Width != Width || assets.Final.Height != Height {
		return fmt.Errorf("ending: montage tail final frame geometry = %dx%d", assets.Final.Width, assets.Final.Height)
	}
	clear(compositor.VGA)
	copy(compositor.Palette[:], compositor.Baseline[:])
	if err := assets.Final.Blit(compositor.VGA, Width, -1); err != nil {
		return fmt.Errorf("ending: montage tail final frame: %w", err)
	}
	return nil
}

func LoadMontageTail(path string) (*MontageTail, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tail MontageTail
	if err := json.Unmarshal(raw, &tail); err != nil {
		return nil, err
	}
	if tail.SchemaVersion != 2 || tail.NativeHandler != "0x2c194" || tail.Status != "mapped_post_montage_tail_fail_closed" ||
		tail.Source != "0x2c194..0x2c39a" || len(tail.Resources) != 4 || tail.Loop.Count != 20 ||
		tail.Loop.RuntimeRecordsGlobal != "0x53a45" || tail.Loop.RuntimeRecordStride != 0x50 ||
		tail.Loop.Record0Byte6Formula != "record0_byte_7 < 0x4c ? 2 : 0" ||
		tail.Loop.Record0Byte7Source != "record0_byte_7" ||
		tail.Loop.Record1Byte6Formula != "record1_byte_7 < 0x4c ? 2 : 0" ||
		tail.Loop.Record1Byte7Source != "record1_byte_7" ||
		tail.Loop.Global540FFSource != "global_540ff" ||
		tail.Loop.RendererCall != "0x28a6c(0,1)" ||
		tail.Loop.PaletteCall != "0x11d40(0,255,0)" ||
		tail.Loop.FrameCall != "0x2935b(resource_58,loop_index,staging,0x140,-1)" ||
		tail.Loop.WaitBeforeFrameTicks != 20 || tail.Loop.WaitAfterFrameTicks != 78 ||
		tail.Loop.WaitHelper != "0x17aa9" || tail.Loop.PresentHelper != "0x1f882" ||
		tail.Loop.RestoreHelper != "0x375c0" || tail.Gate.Source == "" || tail.Gate.Reason == "" {
		return nil, fmt.Errorf("ending montage tail %q has incomplete native contract", path)
	}
	for i, resource := range tail.Resources {
		wantIndex := []int{60, 58, 57, 59}[i]
		if resource.Archive != "FDOTHER.DAT" || resource.Index != wantIndex || resource.Source == "" || resource.Role == "" {
			return nil, fmt.Errorf("ending montage tail %q resource %d is incomplete", path, i)
		}
	}
	for name, table := range map[string][]int{
		"record0_byte_7": tail.RawTables.Record0Byte7,
		"record1_byte_7": tail.RawTables.Record1Byte7,
		"global_540ff":   tail.RawTables.Global540FF,
	} {
		if len(table) != tail.Loop.Count {
			return nil, fmt.Errorf("ending montage tail %q table %s has %d entries", path, name, len(table))
		}
		for i, value := range table {
			if value < 0 || value > 0xff {
				return nil, fmt.Errorf("ending montage tail %q table %s entry %d is not a byte", path, name, i)
			}
		}
	}
	return &tail, nil
}

// Plan returns the raw 20-entry schedule and the exact derived byte-6 values.
// It does not write a runtime record or present a frame.
func (t MontageTail) Plan() ([]MontageTailEntry, error) {
	if t.Loop.Count <= 0 || len(t.RawTables.Record0Byte7) != t.Loop.Count || len(t.RawTables.Record1Byte7) != t.Loop.Count || len(t.RawTables.Global540FF) != t.Loop.Count {
		return nil, fmt.Errorf("ending montage tail has incomplete raw tables")
	}
	entries := make([]MontageTailEntry, t.Loop.Count)
	for i := range entries {
		if t.RawTables.Record0Byte7[i] < 0 || t.RawTables.Record0Byte7[i] > 0xff || t.RawTables.Record1Byte7[i] < 0 || t.RawTables.Record1Byte7[i] > 0xff || t.RawTables.Global540FF[i] < 0 || t.RawTables.Global540FF[i] > 0xff {
			return nil, fmt.Errorf("ending montage tail raw table entry %d is not a byte", i)
		}
		entries[i] = MontageTailEntry{
			Index:        i,
			Global540FF:  byte(t.RawTables.Global540FF[i]),
			Record0Byte6: tailRecordByte6(byte(t.RawTables.Record0Byte7[i])),
			Record0Byte7: byte(t.RawTables.Record0Byte7[i]),
			Record1Byte6: tailRecordByte6(byte(t.RawTables.Record1Byte7[i])),
			Record1Byte7: byte(t.RawTables.Record1Byte7[i]),
		}
	}
	return entries, nil
}

func tailRecordByte6(recordByte7 byte) byte {
	if recordByte7 < 0x4c {
		return 2
	}
	return 0
}
