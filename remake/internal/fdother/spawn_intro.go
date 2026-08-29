package fdother

import "fmt"

const (
	NativeSpawnIntroFrameResource = 9
	NativeSpawnIntroSoundResource = 95
	NativeSpawnIntroPassCount     = 12
)

// NativeSpawnIntroVisible is the exact 0x32A3D..0x32A6C camera gate. Unlike
// the steady unit layer, its lower Y bound is cameraY rather than cameraY-1.
func NativeSpawnIntroVisible(x, y, cameraX, cameraY, visibleXMax, visibleYMax int) bool {
	return x >= cameraX-1 && x <= cameraX+visibleXMax &&
		y >= cameraY && y <= cameraY+visibleYMax+1
}

// NativeSpawnIntroLMIOrigin reproduces 0x32A6E..0x32AA9. The returned offset
// is relative to the unchanged [0x53A49] work-buffer base.
func NativeSpawnIntroLMIOrigin(x, y, cameraX, cameraY int) int {
	return 0x8088 + 24*(x-cameraX-1) + 24*0x1c8*(y-cameraY) - 0xab0
}

// BlitNativeSpawnIntroLMI applies 0x4E85B's transparent-zero LMI write using
// the original 456-byte stride. Camera clipping remains a caller decision.
func BlitNativeSpawnIntroLMI(entry LMI1Entry, work []byte, x, y, cameraX, cameraY int) error {
	origin := NativeSpawnIntroLMIOrigin(x, y, cameraX, cameraY)
	if origin < 0 || origin >= len(work) {
		return fmt.Errorf("fdother: spawn-intro LMI origin %#x outside work buffer", origin)
	}
	return entry.BlitAt(work, 0x1c8, origin%0x1c8, origin/0x1c8, false)
}

// NativeSpawnIntroSnapshotMode records the post-present snapshot rebuild in
// sub_32999. The semantic label is additive: original pass indices and raw
// pointer shifts remain explicit in NativeSpawnIntroStep.
type NativeSpawnIntroSnapshotMode uint8

const (
	NativeSpawnIntroKeepSnapshot NativeSpawnIntroSnapshotMode = iota
	NativeSpawnIntroSplitUnits
	NativeSpawnIntroFullFrame
)

// NativeSpawnIntroStep is one exact loop iteration in 0x32999. Presentation
// happens before SnapshotMode. SoundResource is -1 except pass 1, where the
// raw FDOTHER #95 payload is passed to 0x25A96; no audio semantic is inferred.
type NativeSpawnIntroStep struct {
	Pass             int
	LMIEntry         int
	SoundResource    int
	DelayTicks       int
	SnapshotMode     NativeSpawnIntroSnapshotMode
	RedrawTerrain    bool
	NewUnitPixelLift int
}

// NativeSpawnIntroSchedule preserves all twelve loop iterations and the three
// post-present snapshot rebuilds. A negative lift mirrors the temporary
// [0x53A49] pointer subtraction: -0xE40/456=-8 and -0x8E8/456=-5 rows.
func NativeSpawnIntroSchedule() []NativeSpawnIntroStep {
	steps := make([]NativeSpawnIntroStep, NativeSpawnIntroPassCount)
	for pass := range steps {
		steps[pass] = NativeSpawnIntroStep{
			Pass: pass, LMIEntry: pass, SoundResource: -1, DelayTicks: 1,
		}
	}
	steps[1].SoundResource = NativeSpawnIntroSoundResource
	steps[6].SnapshotMode = NativeSpawnIntroSplitUnits
	steps[6].NewUnitPixelLift = -8
	steps[7].SnapshotMode = NativeSpawnIntroSplitUnits
	steps[7].RedrawTerrain = true
	steps[7].NewUnitPixelLift = -5
	steps[8].SnapshotMode = NativeSpawnIntroFullFrame
	steps[8].RedrawTerrain = true
	return steps
}

func ValidateNativeSpawnIntroSchedule(steps []NativeSpawnIntroStep) error {
	want := NativeSpawnIntroSchedule()
	if len(steps) != len(want) {
		return fmt.Errorf("fdother: spawn-intro schedule has %d passes, want %d", len(steps), len(want))
	}
	for i := range want {
		if steps[i] != want[i] {
			return fmt.Errorf("fdother: spawn-intro pass %d differs from 0x32999", i)
		}
	}
	return nil
}

// DecodeNativeSpawnIntroFrames reads FDOTHER #9 and rejects any resource which
// does not match the twelve entry geometries consumed by 0x32999. The archive
// hash/version gate remains owned by the project reference manifest.
func DecodeNativeSpawnIntroFrames(datPath string) ([]LMI1Entry, error) {
	entries, err := DecodeLMI1Resource(datPath, NativeSpawnIntroFrameResource)
	if err != nil {
		return nil, err
	}
	if err := validateNativeSpawnIntroFrames(entries); err != nil {
		return nil, err
	}
	return entries, nil
}
