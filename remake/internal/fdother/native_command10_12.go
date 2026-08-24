package fdother

import "fmt"

const (
	NativeCommand10To12SoundResource = 80
	NativeCommand10To12PreludeSample = 2
	NativeCommand10To12MainSample    = 13
	NativeCommand10To12Frames        = 60
	NativeCommand10To12DelayMS       = 10
)

type NativeCommand10To12Prelude struct {
	Enabled       bool
	InitialRadius int
	RadiusStep    int
	Repeat        int
	Sample        int
}

// NativeCommand10To12Schedule preserves only the raw tables and call schedule
// proven for 0x21527/0x2185F/0x21A9E -> 0x21548. The renderer still owns the
// 0x1F558-equivalent pointer-grid sampling and atomic state publication.
type NativeCommand10To12Schedule struct {
	CommandID             int
	SoundResource         int
	Prelude               NativeCommand10To12Prelude
	XOrigins              [3]int
	YOrigins              [3]int
	SamplingIncrements    [3]int
	SurfaceCycle          [4]int
	Frames                int
	DelayMS               int
	MainSample            int
	MainSampleFrames      [8]int
	ResultResource        int
	ResultDigitBias       int
	ResultMissDescriptors [4]int
	ResultFrames          int
	ResultHoldMS          int
	ResultVertical        [25]int
}

func BuildNativeCommand10To12Schedule(commandID int) (NativeCommand10To12Schedule, error) {
	if commandID < 10 || commandID > 12 {
		return NativeCommand10To12Schedule{}, fmt.Errorf("native command10-12 schedule unavailable id=%d", commandID)
	}
	prelude := NativeCommand10To12Prelude{}
	if commandID == 11 {
		prelude = NativeCommand10To12Prelude{Enabled: true, InitialRadius: 15, RadiusStep: 10, Repeat: 10, Sample: NativeCommand10To12PreludeSample}
	}
	if commandID == 12 {
		prelude = NativeCommand10To12Prelude{Enabled: true, InitialRadius: 30, RadiusStep: 16, Repeat: 10, Sample: NativeCommand10To12PreludeSample}
	}
	return NativeCommand10To12Schedule{
		CommandID: commandID, SoundResource: NativeCommand10To12SoundResource, Prelude: prelude,
		XOrigins: [3]int{128, 0, -128}, YOrigins: [3]int{128, 0, 128},
		SamplingIncrements: [3]int{131, 128, 125}, SurfaceCycle: [4]int{0, 1, 2, 3},
		Frames: NativeCommand10To12Frames, DelayMS: NativeCommand10To12DelayMS,
		MainSample:       NativeCommand10To12MainSample,
		MainSampleFrames: [8]int{0, 6, 12, 18, 24, 30, 36, 42},
		ResultResource:   5, ResultDigitBias: 94, ResultMissDescriptors: [4]int{116, 117, 118, 118},
		ResultFrames: 22, ResultHoldMS: 500,
		ResultVertical: [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15},
	}, nil
}
