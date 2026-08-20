package fdother

import "fmt"

// NativeCommandHealPresentationFrame is one 0x21EB1 call into the shared
// 0x22046 indexed compositor. LUTIndex stays raw because FDOTHER #3 is a
// palette-index remap bank, not a named spell-animation archive.
type NativeCommandHealPresentationFrame struct {
	LUTIndex     int
	Radius       int
	FrameDelayMs int
}

// NativeCommandHealPresentationSchedule preserves the two directly observed
// 0x21EB1 loops. MidDelayMs follows descriptors 9..1; TailDelayMs follows the
// final descriptor 9 and the 0x11CAC(0) present.
type NativeCommandHealPresentationSchedule struct {
	Frames      []NativeCommandHealPresentationFrame
	MidFrame    int
	MidDelayMs  int
	TailDelayMs int
	SampleIndex int
}

// BuildNativeCommandHealPresentationSchedule reproduces the wrapper literals
// and loop bounds for commands 13..16. The first loop advances radius while
// visiting FDOTHER #3 LUTs 9..1. The second loop keeps the final radius while
// visiting LUTs 3..9.
func BuildNativeCommandHealPresentationSchedule(commandID int) (NativeCommandHealPresentationSchedule, error) {
	start, step := 0, 0
	switch commandID {
	case 13:
		start, step = 1, 2
	case 14:
		start, step = 2, 4
	case 15:
		start, step = 8, 4
	case 16:
		start, step = 6, 6
	default:
		return NativeCommandHealPresentationSchedule{}, fmt.Errorf("native command heal presentation: unsupported id %d", commandID)
	}
	frames := make([]NativeCommandHealPresentationFrame, 0, 16)
	radius := start
	for descriptor := 9; descriptor > 0; descriptor-- {
		frames = append(frames, NativeCommandHealPresentationFrame{
			LUTIndex: descriptor, Radius: radius, FrameDelayMs: 5,
		})
		radius += step
	}
	radius -= step
	for descriptor := 3; descriptor < 10; descriptor++ {
		frames = append(frames, NativeCommandHealPresentationFrame{
			LUTIndex: descriptor, Radius: radius, FrameDelayMs: 5,
		})
	}
	return NativeCommandHealPresentationSchedule{
		Frames: frames, MidFrame: 9, MidDelayMs: 200, TailDelayMs: 200,
		SampleIndex: 11,
	}, nil
}
