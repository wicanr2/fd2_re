package main

import (
	"encoding/json"
	"image/color"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestParityPaletteCycleBeforeNextIndex(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/data/ida/fd2_ch17_palette_writer_20261002.json")
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		SHA        string `json:"sha256"`
		Checkpoint struct {
			EIP       string   `json:"eip"`
			Registers []uint32 `json:"registers"`
			RGB       []byte   `json:"observed_e0ef_rgb"`
		} `json:"original_checkpoint"`
	}
	if err := json.Unmarshal(raw, &evidence); err != nil || len(evidence.Checkpoint.RGB) != 48 {
		t.Fatalf("固定原版窗口無效：%v", err)
	}
	pal := make(color.Palette, 256)
	for i := 0; i < 16; i++ {
		v := evidence.Checkpoint.RGB[i*3 : i*3+3]
		pal[0xe0+i] = color.NRGBA{R: v[0], G: v[1], B: v[2], A: 255}
	}
	doc := map[string]interface{}{"eip": evidence.Checkpoint.EIP, "exe_sha256": evidence.SHA, "registers": evidence.Checkpoint.Registers}
	checkpoint, _ := json.Marshal(doc)
	if _, state, ok := parityPaletteCycleWriteWindow(pal, checkpoint); !ok || state.Phase != 0 || state.CompletedEntries != 7 {
		t.Fatalf("固定原版4E014未承接：ok=%v state=%+v", ok, state)
	}
	for _, name := range []string{"半triplet", "port", "首筆前", "ECX零", "AH", "AL", "ESI", "未知色槽"} {
		t.Run(name, func(t *testing.T) {
			regs := append([]uint32(nil), evidence.Checkpoint.Registers...)
			candidate := append(color.Palette(nil), pal...)
			input := map[string]interface{}{"eip": evidence.Checkpoint.EIP, "exe_sha256": evidence.SHA, "registers": regs}
			switch name {
			case "半triplet":
				input["eip"] = "0x4E01C"
			case "port":
				regs[2] = 0x3c9
			case "首筆前":
				regs[1] = 16
			case "ECX零":
				regs[1] = 0
			case "AH":
				regs[0] ^= 1 << 8
			case "AL":
				regs[0] ^= 1
			case "ESI":
				regs[6]++
			case "未知色槽":
				candidate[0xef] = color.NRGBA{R: 255, A: 255}
			}
			b, _ := json.Marshal(input)
			if _, _, ok := parityPaletteCycleWriteWindow(candidate, b); ok {
				t.Fatal("未知窗口被承接")
			}
		})
	}
	for _, phase := range []int{0, 1, 15} {
		for _, completed := range []int{1, 7, 15} {
			old, next := make([]byte, 768), make([]byte, 768)
			if fdother.ApplyNativeDACPaletteCycleE0EF(old, (phase+15)&15) != nil || fdother.ApplyNativeDACPaletteCycleE0EF(next, phase) != nil {
				t.Fatal("raw窗口無效")
			}
			copy(old[0xe0*3:(0xe0+completed)*3], next[0xe0*3:(0xe0+completed)*3])
			pal, err := fdother.VGAPaletteFromDAC(old)
			if err != nil {
				t.Fatal(err)
			}
			regs := []uint32{uint32(0xe0+completed)<<8 | uint32(next[(0xe0+completed-1)*3+2]), uint32(16 - completed), 0x3c8, 0, 0, 0, uint32(0x60003 + 3*phase + 3*completed), 0}
			doc, _ := json.Marshal(map[string]interface{}{"eip": "0x4E014", "exe_sha256": evidence.SHA, "registers": regs})
			if _, state, ok := parityPaletteCycleWriteWindow(pal, doc); !ok || state.Phase != phase || state.CompletedEntries != completed {
				t.Fatalf("phase%d completed%d 未承接：%+v", phase, completed, state)
			}
		}
	}
}
