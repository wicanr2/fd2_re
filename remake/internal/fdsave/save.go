// Package fdsave preserves the proven FD2.SAV storage ABI without assigning
// gameplay meaning to the still-opaque roster bytes.
package fdsave

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

const (
	FileSize     = 0x59cb
	ChecksumOff  = FileSize - 4
	SlotOffset   = 0x312b
	SlotSize     = 0xa28
	SlotCount    = 4
	RosterSize   = 0xa00
	UnitSize     = 0x50
	RosterUnits  = RosterSize / UnitSize
	MetadataSize = SlotSize - RosterSize

	CurrentPersistentRosterOffset = 0x08a3
	CurrentRuntimeRosterOffset    = 0x12a3
	CurrentFieldControlOffset     = 0x0000
	CurrentFieldControlSize       = 0x08a3
	CurrentFieldControlUnitOffset = 0x0083
	CurrentFieldControlUnitSize   = 0x001a
	CurrentFieldControlUnitCap    = (CurrentFieldControlSize - CurrentFieldControlUnitOffset) / CurrentFieldControlUnitSize
	CurrentNativeEventStateOffset = 0x30a3
	CurrentNativeEventStateSize   = 0x20
	CurrentRuntimeHeaderOffset    = 0x30c3
	CurrentRuntimeHeaderSize      = 18
)

var ErrEmptyChapterSlot = errors.New("fdsave: native chapter slot is empty")

// Slot is one native logical record. Roster and Metadata remain raw because
// only a subset of metadata fields has proven meaning; callers must not
// reinterpret opaque bytes as normalized campaign state.
type Slot struct {
	Roster   []byte
	Metadata []byte
}

func rol16(v uint16, n uint) uint16 { return (v << n) | (v >> (16 - n)) }

// XOREnvelope applies native 0x4dbd8. It is its own inverse.
func XOREnvelope(data []byte) []byte {
	out := append([]byte(nil), data...)
	state := uint16(0x00a5)
	for i, b := range out {
		state = rol16(state+0x9014, 3)
		out[i] = b ^ byte(state)
	}
	return out
}

func Checksum(plain []byte) (uint32, error) {
	if len(plain) != FileSize {
		return 0, fmt.Errorf("fdsave: want %#x bytes, got %#x", FileSize, len(plain))
	}
	var sum uint32
	for _, b := range plain[:ChecksumOff] {
		sum += uint32(b)
	}
	return sum, nil
}

// Decode validates size, reverses the envelope, and verifies the trailing
// little-endian checksum. It never mutates the caller's buffer.
func Decode(stored []byte) ([]byte, error) {
	if len(stored) != FileSize {
		return nil, fmt.Errorf("fdsave: want %#x bytes, got %#x", FileSize, len(stored))
	}
	plain := XOREnvelope(stored)
	want := binary.LittleEndian.Uint32(plain[ChecksumOff:])
	got, _ := Checksum(plain)
	if got != want {
		return nil, fmt.Errorf("fdsave: checksum mismatch: expected %#08x, got %#08x", want, got)
	}
	return plain, nil
}

// Encode writes the native checksum and applies the reversible envelope.
func Encode(plain []byte) ([]byte, error) {
	if len(plain) != FileSize {
		return nil, fmt.Errorf("fdsave: want %#x bytes, got %#x", FileSize, len(plain))
	}
	out := append([]byte(nil), plain...)
	sum, _ := Checksum(out)
	binary.LittleEndian.PutUint32(out[ChecksumOff:], sum)
	return XOREnvelope(out), nil
}

func SlotBounds(slot int) (start, end int, err error) {
	if slot < 0 || slot >= SlotCount {
		return 0, 0, fmt.Errorf("fdsave: slot %d outside 0..%d", slot, SlotCount-1)
	}
	start = SlotOffset + slot*SlotSize
	return start, start + SlotSize, nil
}

// ReadSlot returns copies of the exact roster/metadata regions.
func ReadSlot(plain []byte, slot int) (Slot, error) {
	if len(plain) != FileSize {
		return Slot{}, errors.New("fdsave: invalid plaintext size")
	}
	start, end, err := SlotBounds(slot)
	if err != nil {
		return Slot{}, err
	}
	return Slot{
		Roster:   append([]byte(nil), plain[start:start+RosterSize]...),
		Metadata: append([]byte(nil), plain[start+RosterSize:end]...),
	}, nil
}

// WriteSlot replaces one native logical record in a plaintext save image.
// Roster and Metadata remain opaque byte regions; callers must use Encode
// afterward to rebuild the native checksum/envelope. Validation happens before
// copying so malformed editable input cannot partially mutate the image.
func WriteSlot(plain []byte, slot int, replacement Slot) ([]byte, error) {
	if len(plain) != FileSize {
		return nil, errors.New("fdsave: invalid plaintext size")
	}
	if len(replacement.Roster) != RosterSize {
		return nil, fmt.Errorf("fdsave: roster size=%#x, want %#x", len(replacement.Roster), RosterSize)
	}
	if len(replacement.Metadata) != MetadataSize {
		return nil, fmt.Errorf("fdsave: metadata size=%#x, want %#x", len(replacement.Metadata), MetadataSize)
	}
	start, end, err := SlotBounds(slot)
	if err != nil {
		return nil, err
	}
	out := append([]byte(nil), plain...)
	copy(out[start:start+RosterSize], replacement.Roster)
	copy(out[start+RosterSize:end], replacement.Metadata)
	return out, nil
}

// VerifiedMetadata exposes only fields whose writer and reader are both
// closed. Native 0x30012 writes and 0x2602c restores bytes +0..+9. The final
// three bytes keep address-derived names until their gameplay consumers are
// independently closed; metadata +10..+39 remains opaque.
type VerifiedMetadata struct {
	Chapter     byte
	RosterCount byte
	Currency    uint32
	HUDGateA    byte
	Raw53AF9    byte
	Raw51E61    byte
	Raw51E62    byte
}

func ReadVerifiedMetadata(plain []byte, slot int) (VerifiedMetadata, error) {
	s, err := ReadSlot(plain, slot)
	if err != nil {
		return VerifiedMetadata{}, err
	}
	if len(s.Metadata) != MetadataSize {
		return VerifiedMetadata{}, errors.New("fdsave: invalid metadata size")
	}
	return VerifiedMetadata{
		Chapter:     s.Metadata[0],
		RosterCount: s.Metadata[1],
		Currency:    binary.LittleEndian.Uint32(s.Metadata[2:6]),
		HUDGateA:    s.Metadata[6],
		Raw53AF9:    s.Metadata[7],
		Raw51E61:    s.Metadata[8],
		Raw51E62:    s.Metadata[9],
	}, nil
}

// PersistentRecord preserves one exact native 0x50-byte roster record.
//
// Individual offsets must be decoded only after their writer and consumer are
// proven. Keeping the complete record is required because native 0x2604a copies
// all 32 records before loading the selected slot's metadata.
type PersistentRecord struct {
	Raw [UnitSize]byte
}

// PersistentInventoryCell is one exact two-byte native item cell. Flags remain
// raw because only individual consumers, such as the equipped 0x40 bit, have
// closed meaning.
type PersistentInventoryCell struct {
	Flags  byte
	ItemID byte
}

// PersistentRecordView exposes only offsets with direct constructor and UI
// consumers. RawPresentationKey is deliberately not named portrait, map slot,
// or character identity; native code uses it as a mutable resource key while
// RawIdentity has the separate +8 roster lookup contract.
type PersistentRecordView struct {
	RawByte5           byte
	RawCamp            byte
	RawPresentationKey byte
	RawIdentity        byte
	Inventory          [8]PersistentInventoryCell
	CommandMask        [5]byte
	Race               byte
	Class              byte
	Level              byte
	Transient          [6]byte
	RawByte34          byte
	RawByte35          byte
	RawByte36          byte
	BaseAP             int16
	BaseDP             int16
	Movement           byte
	Experience         byte
	DX                 int16
	HP                 int16
	MaxHP              int16
	MP                 int16
	MaxMP              int16
	AP                 int16
	DP                 int16
	HIT                int16
	EV                 int16
}

// View decodes the proven read-only field projection of one native persistent
// record. The complete byte array remains authoritative and is never mutated.
func (r PersistentRecord) View() PersistentRecordView {
	var view PersistentRecordView
	view.RawByte5 = r.Raw[5]
	view.RawCamp = r.Raw[6]
	view.RawPresentationKey = r.Raw[7]
	view.RawIdentity = r.Raw[8]
	for slot := range view.Inventory {
		offset := 0x0a + slot*2
		view.Inventory[slot] = PersistentInventoryCell{
			Flags:  r.Raw[offset],
			ItemID: r.Raw[offset+1],
		}
	}
	copy(view.CommandMask[:], r.Raw[0x1a:0x1f])
	view.Race = r.Raw[0x1f]
	view.Class = r.Raw[0x20]
	view.Level = r.Raw[0x21]
	copy(view.Transient[:], r.Raw[0x22:0x28])
	view.RawByte34 = r.Raw[0x34]
	view.RawByte35 = r.Raw[0x35]
	view.RawByte36 = r.Raw[0x36]
	view.BaseAP = persistentRecordI16(r.Raw[:], 0x37)
	view.BaseDP = persistentRecordI16(r.Raw[:], 0x39)
	view.Movement = r.Raw[0x3b]
	view.Experience = r.Raw[0x3c]
	view.DX = persistentRecordI16(r.Raw[:], 0x3e)
	view.HP = persistentRecordI16(r.Raw[:], 0x40)
	view.MaxHP = persistentRecordI16(r.Raw[:], 0x42)
	view.MP = persistentRecordI16(r.Raw[:], 0x44)
	view.MaxMP = persistentRecordI16(r.Raw[:], 0x46)
	view.AP = persistentRecordI16(r.Raw[:], 0x48)
	view.DP = persistentRecordI16(r.Raw[:], 0x4a)
	view.HIT = persistentRecordI16(r.Raw[:], 0x4c)
	view.EV = persistentRecordI16(r.Raw[:], 0x4e)
	return view
}

func persistentRecordI16(record []byte, offset int) int16 {
	return int16(binary.LittleEndian.Uint16(record[offset:]))
}

// ChapterSlotSnapshot is the safe import boundary for one native chapter slot.
// It mirrors the selected record copied by 0x2602c..0x26098, while retaining
// opaque bytes instead of guessing normalized campaign or battle semantics.
//
// The RosterCount capacity check is a remake safety invariant. The native load
// path copies the fixed 0xa00-byte roster and does not establish that check.
type ChapterSlotSnapshot struct {
	Slot     int
	Verified VerifiedMetadata
	Metadata [MetadataSize]byte
	Records  [RosterUnits]PersistentRecord
}

// InspectChapterSlot validates and copies a non-empty native chapter slot.
// It does not convert persistent records to battle.Unit and therefore cannot
// by itself be used to claim successful native campaign restore.
func InspectChapterSlot(plain []byte, slot int) (ChapterSlotSnapshot, error) {
	raw, err := ReadSlot(plain, slot)
	if err != nil {
		return ChapterSlotSnapshot{}, err
	}
	verified, err := ReadVerifiedMetadata(plain, slot)
	if err != nil {
		return ChapterSlotSnapshot{}, err
	}
	if verified.Chapter == 0xff {
		return ChapterSlotSnapshot{}, ErrEmptyChapterSlot
	}
	if int(verified.RosterCount) > RosterUnits {
		return ChapterSlotSnapshot{}, fmt.Errorf(
			"fdsave: roster count %d exceeds native capacity %d",
			verified.RosterCount, RosterUnits,
		)
	}
	var snapshot ChapterSlotSnapshot
	snapshot.Slot = slot
	snapshot.Verified = verified
	copy(snapshot.Metadata[:], raw.Metadata)
	for index := range snapshot.Records {
		start := index * UnitSize
		copy(snapshot.Records[index].Raw[:], raw.Roster[start:start+UnitSize])
	}
	return snapshot, nil
}

// ActiveRecords returns a copy of the count-delimited prefix selected by the
// native metadata. The remaining fixed-capacity records stay available in
// Records for exact preservation and future evidence work.
func (s ChapterSlotSnapshot) ActiveRecords() []PersistentRecord {
	count := int(s.Verified.RosterCount)
	if count < 0 || count > len(s.Records) {
		return nil
	}
	out := make([]PersistentRecord, count)
	copy(out, s.Records[:count])
	return out
}

// CurrentRuntimeHeader is the 18-byte header restored by native 0x10010.
// TurnCounter and both count fields are fixed by direct writes to
// [0x53bef], [0x53beb], and [0x53bfb]. The four trailing option bytes remain
// raw except for HUDGateA, whose system-menu writer and renderer consumer are
// independently closed.
type CurrentRuntimeHeader struct {
	TurnCounter     byte
	RuntimeCount    byte
	Chapter         byte
	CameraX         byte
	CameraY         byte
	CursorX         byte
	CursorY         byte
	VisibleCursorX  byte
	VisibleCursorY  byte
	PersistentCount byte
	Currency        uint32
	Raw53AF9        byte
	HUDGateA        byte
	Raw51E61        byte
	Raw51E62        byte
}

// CurrentSnapshot preserves every directly copied FD2.SAV source region
// consumed by 0x10010. RuntimeRecords is count-delimited; PersistentRecords
// retains the full native capacity because the loader copies a fixed 0xa00
// bytes. NativeEventState is the exact 32-byte battle-local table pointed to
// by [0x53ad5]; individual indexes remain raw until their callers are closed.
// NativeFieldControl is the fixed-capacity runtime image at [0x53a55]. The
// chapter loader fills it from FDFIELD.DAT control resource 3N+1, while
// current-save and CONTINUE copy all 0x8a3 bytes symmetrically.
type CurrentSnapshot struct {
	NativeFieldControl [CurrentFieldControlSize]byte
	Header             CurrentRuntimeHeader
	PersistentRecords  [RosterUnits]PersistentRecord
	RuntimeRecords     []PersistentRecord
	NativeEventState   [CurrentNativeEventStateSize]byte
}

// ContinueRuntimeContext is the external resource identity required to
// validate a current-battle snapshot without guessing from the save alone.
// SelectorGroupCount is the exact FDICON.B24 sprite count divided by twelve.
type ContinueRuntimeContext struct {
	Chapter            int
	FieldWidth         int
	FieldHeight        int
	SelectorGroupCount int
	// TitleTimerTick is the signed low BIOS timer word captured by native main
	// at 0x25d83..0x25d8b before the title selector. It is not stored in
	// FD2.SAV, so a title CONTINUE caller must supply it explicitly.
	TitleTimerTick    int
	HasTitleTimerTick bool
}

// ContinueRuntimeRecord preserves one exact runtime record together with the
// cache slot which 0x1036a..0x10395 rebuilds from record +7. The saved +2 byte
// is deliberately not trusted: native CONTINUE overwrites it after the copy.
type ContinueRuntimeRecord struct {
	Raw          PersistentRecord
	SelectorKey  byte
	SelectorSlot byte
}

type ContinueTurnEventControl struct {
	Turn, EventID, RawCamp byte
}

type ContinueFieldEventControl struct {
	EventID, Selector byte
}

type ContinueChestControl struct {
	RawType byte
	Value   uint16
}

type ContinueUnitControl struct {
	Raw [CurrentFieldControlUnitSize]byte
}

// ContinueFieldControlView exposes only the fixed FDFIELD control-resource
// layout consumed by native code. RawUnitCount is an exclusive count:
// 0x10bcc compares index >= [0x53be3] before reading a row. Resource padding
// beyond that count remains available in NativeFieldControl but is not exposed
// as a live unit. RawMapSelector and RawOwnDeployCount remain raw; individual
// unit bytes remain in their exact 26-byte rows.
type ContinueFieldControlView struct {
	RawMapSelector    byte
	RawOwnDeployCount byte
	RawUnitCount      byte
	TurnEvents        [16]ContinueTurnEventControl
	FieldEvents       [16]ContinueFieldEventControl
	Chests            [16]ContinueChestControl
	Units             []ContinueUnitControl
}

type ContinueRuntimeOwner string

const (
	// The 0x10C50→0x1B750 constructor transaction and an exact static-schedule
	// slice are closed. CONTINUE still needs an owner for handler-mutated turn
	// slots and group formulas before every legal save can bind future rows.
	ContinueOwnerPendingGroupBinding ContinueRuntimeOwner = "pending_group_binding"
	// Original main enters 0x117E7 after sub_10010 returns. The remaining
	// owner is the remake's atomic Game publication, not a missing EXE driver.
	ContinueOwnerBattleControllerHandoff ContinueRuntimeOwner = "battle_controller_handoff"
)

// ContinueMapPresentation is the raw map-presentation state established by
// the title-menu CONTINUE caller at 0x26130 and sub_10010 before its final
// 0x4e031 BIOS keyboard-buffer word copy. Copying the head word at 0x41a to
// the tail word at 0x41c strongly implies pending-input discard, but the
// direct instructions prove only the word copy.
// OpeningRangeMode is used by the opening redraw; InteractiveRangeMode is the
// value installed before returning to main's 0x117e7 battle controller.
// HUDAnchorX is the
// persistent data-image seed after 0x1acf3 applies the restored visible cursor.
// This contract does not describe the separate in-battle caller at 0x1a251.
type ContinueMapPresentation struct {
	OpeningRangeMode     int
	InteractiveRangeMode int
	HUDGateB             byte
	HUDAnchorX           int
}

// ContinueMapTimingSeed preserves the process-global timing state immediately
// before sub_10010's first 0x11cac redraw. All fields except
// SpriteLastTimerTick come from the zeroed executable data image;
// SpriteLastTimerTick comes from ContinueRuntimeContext.TitleTimerTick.
// The subsequent redraw samples and delays remain a separate runtime owner.
type ContinueMapTimingSeed struct {
	SpriteIdleCycle             int
	SpriteMovingCycle           int
	SpriteLastTimerTick         int
	TerrainPhase                int
	TerrainPhaseLastTimerTick   int
	TerrainPhaseOverride        int
	TerrainFlip                 int
	TerrainFlipLastTimerTick    int
	UnitPixelShift              int
	UnitPixelShiftLastTimerTick int
}

// ContinueRuntimeInput is a deep-copied, read-only preflight result. It proves
// the save/resource boundary and selector rebuild only; UnresolvedOwners keeps
// the production CONTINUE gate closed until every remaining native owner has
// independent evidence and a strict adapter.
type ContinueRuntimeInput struct {
	Context            ContinueRuntimeContext
	Header             CurrentRuntimeHeader
	NativeFieldControl [CurrentFieldControlSize]byte
	FieldControl       ContinueFieldControlView
	PersistentRecords  [RosterUnits]PersistentRecord
	RuntimeRecords     []ContinueRuntimeRecord
	NativeEventState   [CurrentNativeEventStateSize]byte
	MapPresentation    ContinueMapPresentation
	MapTimingSeed      ContinueMapTimingSeed
	UnresolvedOwners   []ContinueRuntimeOwner
	validated          bool
}

func (i ContinueRuntimeInput) ReadyForContinue() bool {
	return len(i.UnresolvedOwners) == 0
}

// ValidatedForRuntimeBridge rejects both manually assembled values and public
// fields modified after BuildContinueRuntimeInput returned. It reconstructs
// the source snapshot, reruns the complete preflight, and requires an exact
// typed result match. Downstream adapters still validate their own chapter
// asset identity and topology.
func (i ContinueRuntimeInput) ValidatedForRuntimeBridge() bool {
	if !i.validated {
		return false
	}
	snapshot := CurrentSnapshot{
		NativeFieldControl: i.NativeFieldControl,
		Header:             i.Header,
		PersistentRecords:  i.PersistentRecords,
		NativeEventState:   i.NativeEventState,
		RuntimeRecords:     make([]PersistentRecord, len(i.RuntimeRecords)),
	}
	for index, record := range i.RuntimeRecords {
		snapshot.RuntimeRecords[index] = record.Raw
	}
	rebuilt, err := BuildContinueRuntimeInput(snapshot, i.Context)
	return err == nil && reflect.DeepEqual(rebuilt, i)
}

// BuildContinueRuntimeInput validates every current-snapshot field whose
// resource-side contract is already closed. It performs no battle.State
// mutation and does not invent the still-unresolved runtime owners.
func BuildContinueRuntimeInput(
	snapshot CurrentSnapshot,
	context ContinueRuntimeContext,
) (ContinueRuntimeInput, error) {
	if context.Chapter < 0 || context.Chapter >= 30 ||
		int(snapshot.Header.Chapter) != context.Chapter {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE chapter/resource mismatch")
	}
	if context.FieldWidth < 13 || context.FieldHeight < 8 ||
		context.FieldWidth > 0x100 || context.FieldHeight > 0x100 {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE field dimensions are invalid")
	}
	if context.SelectorGroupCount <= 0 || context.SelectorGroupCount > 0x100 {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE FDICON group count is invalid")
	}
	if !context.HasTitleTimerTick ||
		context.TitleTimerTick < -0x8000 || context.TitleTimerTick > 0x7fff {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE title timer seed is invalid")
	}
	runtimeCount := int(snapshot.Header.RuntimeCount)
	if runtimeCount > RosterUnits*3 || len(snapshot.RuntimeRecords) != runtimeCount {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE runtime record count mismatch")
	}
	if int(snapshot.Header.PersistentCount) > RosterUnits {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE persistent record count exceeds capacity")
	}
	if int(snapshot.NativeFieldControl[2]) > CurrentFieldControlUnitCap {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE FDFIELD unit count exceeds control capacity")
	}
	header := snapshot.Header
	if int(header.CameraX) > context.FieldWidth-13 ||
		int(header.CameraY) > context.FieldHeight-8 ||
		int(header.CursorX) >= context.FieldWidth ||
		int(header.CursorY) >= context.FieldHeight ||
		int(header.VisibleCursorX) != int(header.CursorX)-int(header.CameraX) ||
		int(header.VisibleCursorY) != int(header.CursorY)-int(header.CameraY) {
		return ContinueRuntimeInput{}, errors.New("fdsave: CONTINUE map view violates field bounds or cursor identity")
	}

	runtime := make([]ContinueRuntimeRecord, runtimeCount)
	slots := make(map[byte]byte)
	nextSlot := 0
	for index, record := range snapshot.RuntimeRecords {
		key := record.Raw[7]
		if int(key) >= context.SelectorGroupCount {
			return ContinueRuntimeInput{}, fmt.Errorf(
				"fdsave: CONTINUE runtime record %d has invalid FDICON key %d",
				index, key,
			)
		}
		slot, ok := slots[key]
		if !ok {
			slot = byte(nextSlot)
			slots[key] = slot
			nextSlot++
		}
		if record.Raw[5]&1 == 0 {
			if int(record.Raw[0]) >= context.FieldWidth ||
				int(record.Raw[1]) >= context.FieldHeight ||
				record.Raw[3] > 3 || record.Raw[4] > 6 {
				return ContinueRuntimeInput{}, fmt.Errorf(
					"fdsave: CONTINUE active runtime record %d has invalid map presentation",
					index,
				)
			}
		}
		runtime[index] = ContinueRuntimeRecord{
			Raw: record, SelectorKey: key, SelectorSlot: slot,
		}
	}

	input := ContinueRuntimeInput{
		Context:        context,
		Header:         snapshot.Header,
		RuntimeRecords: runtime,
		FieldControl:   decodeContinueFieldControl(snapshot.NativeFieldControl),
		MapPresentation: ContinueMapPresentation{
			OpeningRangeMode:     0,
			InteractiveRangeMode: 1,
			HUDGateB:             1,
			HUDAnchorX: continueTitleHUDAnchor(
				int(snapshot.Header.VisibleCursorX),
				int(snapshot.Header.VisibleCursorY),
			),
		},
		MapTimingSeed: ContinueMapTimingSeed{
			SpriteLastTimerTick:  context.TitleTimerTick,
			TerrainPhaseOverride: -1,
		},
		UnresolvedOwners: []ContinueRuntimeOwner{
			ContinueOwnerPendingGroupBinding,
			ContinueOwnerBattleControllerHandoff,
		},
		validated: true,
	}
	copy(input.NativeFieldControl[:], snapshot.NativeFieldControl[:])
	copy(input.PersistentRecords[:], snapshot.PersistentRecords[:])
	copy(input.NativeEventState[:], snapshot.NativeEventState[:])
	return input, nil
}

func decodeContinueFieldControl(
	raw [CurrentFieldControlSize]byte,
) ContinueFieldControlView {
	view := ContinueFieldControlView{
		RawMapSelector:    raw[0],
		RawOwnDeployCount: raw[1],
		RawUnitCount:      raw[2],
		Units: make(
			[]ContinueUnitControl,
			int(raw[2]),
		),
	}
	const turnOffset = 3
	for index := range view.TurnEvents {
		offset := turnOffset + index*3
		view.TurnEvents[index] = ContinueTurnEventControl{
			Turn: raw[offset], EventID: raw[offset+1], RawCamp: raw[offset+2],
		}
	}
	const fieldOffset = turnOffset + 16*3
	for index := range view.FieldEvents {
		offset := fieldOffset + index*2
		view.FieldEvents[index] = ContinueFieldEventControl{
			EventID: raw[offset], Selector: raw[offset+1],
		}
	}
	const chestOffset = fieldOffset + 16*2
	for index := range view.Chests {
		offset := chestOffset + index*3
		view.Chests[index] = ContinueChestControl{
			RawType: raw[offset],
			Value:   binary.LittleEndian.Uint16(raw[offset+1 : offset+3]),
		}
	}
	for index := range view.Units {
		offset := CurrentFieldControlUnitOffset +
			index*CurrentFieldControlUnitSize
		copy(
			view.Units[index].Raw[:],
			raw[offset:offset+CurrentFieldControlUnitSize],
		)
	}
	return view
}

// continueTitleHUDAnchor starts from the executable data-image value one and
// applies the only two writes in 0x1acf3. The argument order follows the save
// header (X, Y), while the native comparisons are [0x53abd] Y then [0x53ab9] X.
func continueTitleHUDAnchor(visibleX, visibleY int) int {
	if visibleY > 5 {
		if visibleX < 3 {
			return 0xf2
		}
		if visibleX > 9 {
			return 1
		}
	}
	return 1
}

// InspectCurrentSnapshot decodes only the verified plaintext layout used by
// the title CONTINUE branch. It is separate from the four chapter slots.
func InspectCurrentSnapshot(plain []byte) (CurrentSnapshot, error) {
	if len(plain) != FileSize {
		return CurrentSnapshot{}, errors.New("fdsave: invalid plaintext size")
	}
	header := plain[CurrentRuntimeHeaderOffset : CurrentRuntimeHeaderOffset+CurrentRuntimeHeaderSize]
	if int(header[1]) > RosterUnits*3 {
		return CurrentSnapshot{}, fmt.Errorf(
			"fdsave: runtime count %d exceeds native capacity %d",
			header[1], RosterUnits*3,
		)
	}
	if int(header[9]) > RosterUnits {
		return CurrentSnapshot{}, fmt.Errorf(
			"fdsave: persistent count %d exceeds native capacity %d",
			header[9], RosterUnits,
		)
	}
	snapshot := CurrentSnapshot{
		Header: CurrentRuntimeHeader{
			TurnCounter:     header[0],
			RuntimeCount:    header[1],
			Chapter:         header[2],
			CameraX:         header[3],
			CameraY:         header[4],
			CursorX:         header[5],
			CursorY:         header[6],
			VisibleCursorX:  header[7],
			VisibleCursorY:  header[8],
			PersistentCount: header[9],
			Currency:        binary.LittleEndian.Uint32(header[10:14]),
			Raw53AF9:        header[14],
			HUDGateA:        header[15],
			Raw51E61:        header[16],
			Raw51E62:        header[17],
		},
	}
	copy(
		snapshot.NativeFieldControl[:],
		plain[CurrentFieldControlOffset:CurrentFieldControlOffset+CurrentFieldControlSize],
	)
	for index := range snapshot.PersistentRecords {
		start := CurrentPersistentRosterOffset + index*UnitSize
		copy(
			snapshot.PersistentRecords[index].Raw[:],
			plain[start:start+UnitSize],
		)
	}
	snapshot.RuntimeRecords = make(
		[]PersistentRecord, int(snapshot.Header.RuntimeCount),
	)
	for index := range snapshot.RuntimeRecords {
		start := CurrentRuntimeRosterOffset + index*UnitSize
		copy(snapshot.RuntimeRecords[index].Raw[:], plain[start:start+UnitSize])
	}
	copy(
		snapshot.NativeEventState[:],
		plain[CurrentNativeEventStateOffset:CurrentNativeEventStateOffset+CurrentNativeEventStateSize],
	)
	return snapshot, nil
}

// WriteCurrentSnapshot replaces only the current-runtime regions written by
// native sub_19DF7 selector 1. The chapter-slot area and every other opaque
// plaintext byte are preserved from the checksum-valid caller image. Callers
// must use Encode after this function to rebuild the checksum and envelope.
//
// The function accepts exact raw records only. It deliberately does not
// serialize normalized campaign or battle values into the native ABI.
func WriteCurrentSnapshot(plain []byte, snapshot CurrentSnapshot) ([]byte, error) {
	if len(plain) != FileSize {
		return nil, errors.New("fdsave: invalid plaintext size")
	}
	runtimeCount := int(snapshot.Header.RuntimeCount)
	if runtimeCount > RosterUnits*3 || len(snapshot.RuntimeRecords) != runtimeCount {
		return nil, errors.New("fdsave: current snapshot runtime record count mismatch")
	}
	if int(snapshot.Header.PersistentCount) > RosterUnits {
		return nil, errors.New("fdsave: current snapshot persistent record count exceeds capacity")
	}
	if int(snapshot.NativeFieldControl[2]) > CurrentFieldControlUnitCap {
		return nil, errors.New("fdsave: current snapshot field unit count exceeds capacity")
	}

	out := append([]byte(nil), plain...)
	copy(
		out[CurrentFieldControlOffset:CurrentFieldControlOffset+CurrentFieldControlSize],
		snapshot.NativeFieldControl[:],
	)
	for index := range snapshot.PersistentRecords {
		start := CurrentPersistentRosterOffset + index*UnitSize
		copy(out[start:start+UnitSize], snapshot.PersistentRecords[index].Raw[:])
	}
	for index := range snapshot.RuntimeRecords {
		start := CurrentRuntimeRosterOffset + index*UnitSize
		copy(out[start:start+UnitSize], snapshot.RuntimeRecords[index].Raw[:])
	}
	copy(
		out[CurrentNativeEventStateOffset:CurrentNativeEventStateOffset+CurrentNativeEventStateSize],
		snapshot.NativeEventState[:],
	)
	header := out[CurrentRuntimeHeaderOffset : CurrentRuntimeHeaderOffset+CurrentRuntimeHeaderSize]
	header[0] = snapshot.Header.TurnCounter
	header[1] = snapshot.Header.RuntimeCount
	header[2] = snapshot.Header.Chapter
	header[3] = snapshot.Header.CameraX
	header[4] = snapshot.Header.CameraY
	header[5] = snapshot.Header.CursorX
	header[6] = snapshot.Header.CursorY
	header[7] = snapshot.Header.VisibleCursorX
	header[8] = snapshot.Header.VisibleCursorY
	header[9] = snapshot.Header.PersistentCount
	binary.LittleEndian.PutUint32(header[10:14], snapshot.Header.Currency)
	header[14] = snapshot.Header.Raw53AF9
	header[15] = snapshot.Header.HUDGateA
	header[16] = snapshot.Header.Raw51E61
	header[17] = snapshot.Header.Raw51E62
	return out, nil
}

// ActivePersistentRecords returns a copy of the header-delimited current
// persistent roster.
func (s CurrentSnapshot) ActivePersistentRecords() []PersistentRecord {
	count := int(s.Header.PersistentCount)
	if count < 0 || count > len(s.PersistentRecords) {
		return nil
	}
	out := make([]PersistentRecord, count)
	copy(out, s.PersistentRecords[:count])
	return out
}
