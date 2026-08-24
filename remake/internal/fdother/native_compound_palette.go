package fdother

import "fmt"

// InterpolateNativeCompoundDAC reproduces sub_286BD's caller-owned
// [start,end) DAC interpolation. Division is signed and truncates toward zero.
func InterpolateNativeCompoundDAC(baseline []byte, start, end, delta int, raw [3]byte) ([]byte, error) {
	if len(baseline) != 256*3 || start < 0 || end > 256 || start >= end || delta < 0 || delta > 40 {
		return nil, fmt.Errorf("fdother: native compound DAC interpolation unavailable")
	}
	out := append([]byte(nil), baseline...)
	for index := start; index < end; index++ {
		for component := 0; component < 3; component++ {
			base := int(baseline[3*index+component])
			origin := int(raw[component])
			out[3*index+component] = byte(origin + delta*(base-origin)/40)
		}
	}
	return out, nil
}
