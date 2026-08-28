package ending

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
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

// MontageTailVisualResources records only the directly proven archive
// selectors consumed by the nonzero 0x28a6c branch. It is not a decoded
// animation, does not assign attacker/defender roles, and performs no runtime
// mutation or drawing.
type MontageTailVisualResources struct {
	Index             int
	TAI               int
	BG                int
	Record1FIGANIBase int
	Record1FIGANIAux  int
	Record0FIGANIBase int
	Record0FIGANIAux  int
}

const (
	// 0x2c435 passes this raw selector to 0x1088d before the party montage.
	// These constants deliberately retain archive arithmetic instead of naming
	// selector 0x1e as a chapter or a character list.
	nativeMontageTailLoadSelector       = 0x1e
	nativeMontageTailFieldMapResource   = nativeMontageTailLoadSelector * 3
	nativeMontageTailControlResource    = nativeMontageTailFieldMapResource + 1
	nativeMontageTailPositionsResource  = nativeMontageTailFieldMapResource + 2
	nativeMontageTailDeployRecordCount  = 0x1f
	nativeMontageTailControlGroupOffset = 21
)

// MontageTailLoaderPaths identifies the player-provided archives consumed by
// the record-construction portion of 0x1088d(0x1e).  The original archives
// remain read-only; this type never embeds their bytes in the remake.
type MontageTailLoaderPaths struct {
	FDFIELD string
	FDICON  string
}

// MontageTailVisualPaths identifies the separated visual roots needed by the
// source-bound bridge. It validates every selector and never falls back to
// player archives.
type MontageTailVisualPaths struct {
	SurfaceRoot   string
	AnimationRoot string
}

// MontageTailLoaderBaseline is the exact raw-runtime shape constructed by the
// 0x1088d(0x1e) party-slot loop: 31 deployment records, with the active
// persistent prefix copied at stride 0x50 and the remainder marked inactive.
//
// It is deliberately *not* an admitted 0x28a6c call-time input.  The later
// 0x2c548 montage can observe and potentially mutate the same runtime image;
// no original trace yet proves its final full-record state at 0x2c2a6.  A
// future tail renderer must require a separately proven call-time snapshot,
// rather than treating this loader baseline as sufficient evidence.
type MontageTailLoaderBaseline struct {
	runtimeRecords [][fdsave.UnitSize]byte
}

// RuntimeCount reports the native deployment-slot count, including inactive
// slots. It does not expose a mutable backing slice.
func (b MontageTailLoaderBaseline) RuntimeCount() int {
	return len(b.runtimeRecords)
}

// RuntimeRecord returns a value copy of one post-0x1088d deployment record.
func (b MontageTailLoaderBaseline) RuntimeRecord(index int) ([fdsave.UnitSize]byte, error) {
	if index < 0 || index >= len(b.runtimeRecords) {
		return [fdsave.UnitSize]byte{}, fmt.Errorf("ending: montage tail loader record %d is unavailable", index)
	}
	return b.runtimeRecords[index], nil
}

// FirstPair returns the first two post-loader records as value copies.  This
// convenience function does not upgrade the pair to a renderer admission;
// see MontageTailLoaderBaseline for the unresolved 0x2c548 boundary.
func (b MontageTailLoaderBaseline) FirstPair() ([2][fdsave.UnitSize]byte, error) {
	if len(b.runtimeRecords) < 2 {
		return [2][fdsave.UnitSize]byte{}, fmt.Errorf("ending: montage tail loader lacks the first two records")
	}
	return [2][fdsave.UnitSize]byte{b.runtimeRecords[0], b.runtimeRecords[1]}, nil
}

// BuildMontageTailLoaderBaseline reproduces only the proven raw record writes
// in 0x1088d's selector-0x1e own-deployment loop.  It validates the complete
// FDFIELD resource triplet needed by that fixed selector, validates FDICON
// keys against the player archive, copies the complete persistent records,
// performs the documented overwrites, and reuses the independently verified
// 0x1b750 equipment tail.  It deliberately stops before 0x2c548 and does not
// create a tail renderer or campaign transition.
func BuildMontageTailLoaderBaseline(
	tail MontageTail,
	persistent []fdsave.PersistentRecord,
	paths MontageTailLoaderPaths,
	itemTable []byte,
) (MontageTailLoaderBaseline, error) {
	if err := tail.validateNativeContract(); err != nil {
		return MontageTailLoaderBaseline{}, err
	}
	// The fixed FDFIELD selector supplies 31 deployment slots.  0x2c405 later
	// indexes using the persistent count, so a 32nd persistent record lacks a
	// proven corresponding deployment record and must remain fail-closed.
	if len(persistent) < 2 || len(persistent) > nativeMontageTailDeployRecordCount {
		return MontageTailLoaderBaseline{}, fmt.Errorf(
			"ending: montage tail loader persistent count %d is outside proven 2..%d",
			len(persistent), nativeMontageTailDeployRecordCount,
		)
	}
	if paths.FDFIELD == "" || paths.FDICON == "" {
		return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail loader archive path is unavailable")
	}

	fieldMap, err := fdother.ReadResource(paths.FDFIELD, nativeMontageTailFieldMapResource)
	if err != nil {
		return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail FDFIELD#%d: %w", nativeMontageTailFieldMapResource, err)
	}
	control, err := fdother.ReadResource(paths.FDFIELD, nativeMontageTailControlResource)
	if err != nil {
		return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail FDFIELD#%d: %w", nativeMontageTailControlResource, err)
	}
	positions, err := fdother.ReadResource(paths.FDFIELD, nativeMontageTailPositionsResource)
	if err != nil {
		return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail FDFIELD#%d: %w", nativeMontageTailPositionsResource, err)
	}
	unitRows, err := validateMontageTailLoaderField(fieldMap, control, positions)
	if err != nil {
		return MontageTailLoaderBaseline{}, err
	}

	bank, err := fdicon.DecodeFile(paths.FDICON)
	if err != nil || bank == nil || len(bank.Sprites) == 0 || len(bank.Sprites)%12 != 0 {
		if err != nil {
			return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail FDICON: %w", err)
		}
		return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail FDICON has invalid selector groups")
	}
	selectorGroups := len(bank.Sprites) / 12

	runtime := make([][fdsave.UnitSize]byte, nativeMontageTailDeployRecordCount)
	cache := &fdicon.NativeSelectorCache{}
	for index := range runtime {
		if index >= len(persistent) {
			runtime[index][5] = 1
			continue
		}
		record := persistent[index].Raw
		if int(record[7]) >= selectorGroups {
			return MontageTailLoaderBaseline{}, fmt.Errorf(
				"ending: montage tail persistent record %d has FDICON key %d outside %d groups",
				index, record[7], selectorGroups,
			)
		}
		slot, err := cache.SlotFor(int(record[7]))
		if err != nil || slot > 0xff {
			if err != nil {
				return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail selector cache: %w", err)
			}
			return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail selector slot %d is not a byte", slot)
		}
		positionOffset := 2 + (unitRows+index)*6
		record[0] = positions[positionOffset]
		record[1] = positions[positionOffset+2]
		record[2] = byte(slot)
		record[3] = 0
		record[4] = 0
		record[6] = 2
		record[0x31] = 0xff
		clear(record[0x22:0x28])
		if err := battle.ApplyNativeRuntimeEquipmentRecalc(record[:], itemTable); err != nil {
			return MontageTailLoaderBaseline{}, fmt.Errorf("ending: montage tail runtime record %d: %w", index, err)
		}
		runtime[index] = record
	}
	return MontageTailLoaderBaseline{runtimeRecords: runtime}, nil
}

func validateMontageTailLoaderField(fieldMap, control, positions []byte) (int, error) {
	if len(fieldMap) < 4 {
		return 0, fmt.Errorf("ending: montage tail FDFIELD map is too short")
	}
	width := int(binary.LittleEndian.Uint16(fieldMap))
	height := int(binary.LittleEndian.Uint16(fieldMap[2:]))
	if width != 35 || height != 45 || len(fieldMap) != 4+width*height*4 {
		return 0, fmt.Errorf("ending: montage tail FDFIELD#%d geometry is not the proven 35x45 selector image", nativeMontageTailFieldMapResource)
	}
	if len(control) < 3 || control[0] != nativeMontageTailLoadSelector ||
		int(control[1]) != nativeMontageTailDeployRecordCount || control[2] != 1 {
		return 0, fmt.Errorf("ending: montage tail FDFIELD control header does not match selector %#x", nativeMontageTailLoadSelector)
	}
	unitRows := int(control[2])
	wantControlLength := fdsave.CurrentFieldControlUnitOffset + unitRows*fdsave.CurrentFieldControlUnitSize
	if len(control) != wantControlLength {
		return 0, fmt.Errorf("ending: montage tail FDFIELD control length %d, want %d", len(control), wantControlLength)
	}
	unitOffset := fdsave.CurrentFieldControlUnitOffset
	if control[unitOffset+nativeMontageTailControlGroupOffset] != 0xff {
		return 0, fmt.Errorf("ending: montage tail FDFIELD control row may append an unproven group-0 record")
	}
	if len(positions) < 2 {
		return 0, fmt.Errorf("ending: montage tail FDFIELD positions are too short")
	}
	positionCount := int(binary.LittleEndian.Uint16(positions))
	if positionCount != unitRows+nativeMontageTailDeployRecordCount || len(positions) != 2+positionCount*6 {
		return 0, fmt.Errorf("ending: montage tail FDFIELD positions do not match the proven 32-row layout")
	}
	if binary.LittleEndian.Uint16(positions[6:]) != nativeMontageTailLoadSelector {
		return 0, fmt.Errorf("ending: montage tail FDFIELD first position key is not selector %#x", nativeMontageTailLoadSelector)
	}
	for index := 0; index < nativeMontageTailDeployRecordCount; index++ {
		offset := 2 + (unitRows+index)*6
		if binary.LittleEndian.Uint16(positions[offset+4:]) != 0 {
			return 0, fmt.Errorf("ending: montage tail FDFIELD deployment position %d has unexpected raw key", index)
		}
	}
	return unitRows, nil
}

// MontageTailAssets preserves the directly loaded FDOTHER inputs of
// 0x2c194. The original nonzero 0x28a6c branch is proven to load and compose
// visual resources, but the remake lacks its admitted full call-time runtime
// records and exact renderer adapter. Merely loading these assets therefore
// does not claim to reproduce that loop. The final #59 frame is kept separately
// because the native code decodes it only after the loop has completed.
type MontageTailAssets struct {
	LoopPalette []byte
	Intro       fdother.Frame
	LoopFrames  []fdother.Frame
	Final       fdother.Frame
}

// MontageTailVisualSet is one preflighted 0x28a6c resource transaction. The
// field names retain raw record ownership rather than assigning character or
// attacker/defender identities to finale records 0 and 1.
type MontageTailVisualSet struct {
	Plan              MontageTailVisualResources
	TAI               fdother.Frame
	BG                fdother.Frame
	Record1FIGANIBase *figani.Animation
	Record1FIGANIAux  *figani.Animation
	Record0FIGANIBase *figani.Animation
	Record0FIGANIAux  *figani.Animation
}

// LoadMontageTailVisualSets performs an all-or-nothing preflight of the 20
// original TAI/BG/FIGANI selector transactions. The player additionally
// validates the actual header branch before admitting 0x2939d composition;
// archive availability alone still does not prove DOS timing or sound parity.
func LoadMontageTailVisualSets(tail MontageTail, paths MontageTailVisualPaths) ([]MontageTailVisualSet, error) {
	if paths.SurfaceRoot == "" || paths.AnimationRoot == "" {
		return nil, fmt.Errorf("ending: montage tail separated visual root is unavailable")
	}
	plans, err := tail.PlanVisualResources()
	if err != nil {
		return nil, err
	}
	sets := make([]MontageTailVisualSet, len(plans))
	for index, plan := range plans {
		tai, err := fdother.LoadSeparatedSingleFrame(paths.SurfaceRoot, "TAI.DAT", plan.TAI)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d TAI#%d: %w", index, plan.TAI, err)
		}
		taiAt := tai
		taiAt.X, taiAt.Y = 164, 157
		if err := validateMontageTailVisualFrame(taiAt); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d TAI#%d: %w", index, plan.TAI, err)
		}
		bg, err := fdother.LoadSeparatedSingleFrame(paths.SurfaceRoot, "BG.DAT", plan.BG)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d BG#%d: %w", index, plan.BG, err)
		}
		bgAt := bg
		bgAt.Y = 50
		if err := validateMontageTailVisualFrame(bgAt); err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d BG#%d: %w", index, plan.BG, err)
		}
		loadAnimation := func(resource int) (*figani.Animation, error) {
			animation, err := figani.LoadSeparatedResource(paths.AnimationRoot, resource)
			if err != nil {
				return nil, err
			}
			for frameIndex, frame := range animation.Frames {
				if frame.X < 0 || frame.Y < 0 || frame.X+frame.Width > Width || frame.Y+frame.Height > Height {
					return nil, fmt.Errorf("frame %d is outside the proven indexed viewport", frameIndex)
				}
			}
			return animation, nil
		}
		record1Base, err := loadAnimation(plan.Record1FIGANIBase)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d FIGANI#%d: %w", index, plan.Record1FIGANIBase, err)
		}
		record1Aux, err := loadAnimation(plan.Record1FIGANIAux)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d FIGANI#%d: %w", index, plan.Record1FIGANIAux, err)
		}
		record0Base, err := loadAnimation(plan.Record0FIGANIBase)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d FIGANI#%d: %w", index, plan.Record0FIGANIBase, err)
		}
		record0Aux, err := loadAnimation(plan.Record0FIGANIAux)
		if err != nil {
			return nil, fmt.Errorf("ending: montage tail entry %d FIGANI#%d: %w", index, plan.Record0FIGANIAux, err)
		}
		sets[index] = MontageTailVisualSet{
			Plan: plan, TAI: tai, BG: bg, Record1FIGANIBase: record1Base,
			Record1FIGANIAux: record1Aux, Record0FIGANIBase: record0Base, Record0FIGANIAux: record0Aux,
		}
	}
	return sets, nil
}

func validateMontageTailVisualFrame(frame fdother.Frame) error {
	if frame.Width <= 0 || frame.Height <= 0 || frame.Width > Width || frame.Height > Height {
		return fmt.Errorf("geometry %dx%d exceeds the indexed viewport", frame.Width, frame.Height)
	}
	dst := make([]byte, Bytes)
	if err := frame.Blit(dst, Width, -1); err != nil {
		return err
	}
	return nil
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
	if err := tail.validateNativeContract(); err != nil {
		return nil, fmt.Errorf("ending montage tail %q: %w", path, err)
	}
	return &tail, nil
}

func (tail MontageTail) validateNativeContract() error {
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
		return fmt.Errorf("has incomplete native contract")
	}
	for i, resource := range tail.Resources {
		wantIndex := []int{60, 58, 57, 59}[i]
		if resource.Archive != "FDOTHER.DAT" || resource.Index != wantIndex || resource.Source == "" || resource.Role == "" {
			return fmt.Errorf("resource %d is incomplete", i)
		}
	}
	for name, table := range map[string][]int{
		"record0_byte_7": tail.RawTables.Record0Byte7,
		"record1_byte_7": tail.RawTables.Record1Byte7,
		"global_540ff":   tail.RawTables.Global540FF,
	} {
		if len(table) != tail.Loop.Count {
			return fmt.Errorf("table %s has %d entries", name, len(table))
		}
		for i, value := range table {
			if value < 0 || value > 0xff {
				return fmt.Errorf("table %s entry %d is not a byte", name, i)
			}
		}
	}
	return nil
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

// PlanVisualResources preserves the resource-index arithmetic directly used
// by the original nonzero 0x28a6c branch. The TAI#3 substitutions are raw
// selector comparisons, not named character exceptions. The source-bound E1
// player consumes this plan without mutating runtime records; an original E2
// claim still needs a separately admitted call-time record snapshot.
func (t MontageTail) PlanVisualResources() ([]MontageTailVisualResources, error) {
	entries, err := t.Plan()
	if err != nil {
		return nil, err
	}
	resources := make([]MontageTailVisualResources, len(entries))
	for index, entry := range entries {
		tai := int(entry.Global540FF)
		if entry.Record0Byte7 == 0x1a || entry.Record0Byte7 == 0x36 || entry.Record1Byte7 == 0x37 {
			tai = 3
		}
		resources[index] = MontageTailVisualResources{
			Index:             index,
			TAI:               tai,
			BG:                int(entry.Global540FF),
			Record1FIGANIBase: int(entry.Record1Byte7) * 3,
			Record1FIGANIAux:  int(entry.Record1Byte7)*3 + 1,
			Record0FIGANIBase: int(entry.Record0Byte7) * 3,
			Record0FIGANIAux:  int(entry.Record0Byte7)*3 + 1,
		}
	}
	return resources, nil
}

func tailRecordByte6(recordByte7 byte) byte {
	if recordByte7 < 0x4c {
		return 2
	}
	return 0
}
