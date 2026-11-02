package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestChurchServiceSourceExport(t *testing.T) {
	source := "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	raw, err := fdother.ReadResource(source, 14)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != "0249104999fcb533c848f831860997da8c2d6baeb1971d171066bff7ce23c60c" {
		t.Fatal("wrong fixed resource")
	}
	out := os.Getenv("FD2_CHURCH_EXPORT_OUT")
	if out == "" {
		out = t.TempDir()
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	if err := exportChurchUI(source, out); err != nil {
		t.Fatal(err)
	}
	got, err := fdother.LoadSeparatedChurchUIAssets(filepath.Join(out, "ui"))
	if err != nil {
		t.Fatal(err)
	}
	expected := map[int]string{
		3:  "3bfa1877cda0f3c3701c97fb02d18a6f764bcd0848992f5e94f15fc31ccb9b99",
		4:  "0766b683e1d4b9ab159e19ca0f5d360cb8d53f230b4b5f44df13e135b50b3f77",
		5:  "4115296fe45fb99533b73a069ef8cb4f901f4dd59cdcf1a49f82b43284e0f836",
		6:  "87cd44a2e40c6c27c4e3b16d9be72a2fab97a43d9a0764fa670b8e0726d088ad",
		7:  "a4e461165c3b39fdca6aa53bde7a75af12709923ac46a18a4f5b3e7275f793e5",
		8:  "34212988c525a267bd6c3b305dc46e08746f486142d8f9ba6de782b5d20ebb8b",
		9:  "a006557542152c73672ff9f79a2e406ff3a86cab22069f323ee6300320f5c1bb",
		10: "63cf53b7f21159d38ffd04961d474ea4c4c96adcf7e872267356368ecfecdd9b",
	}
	checks := []map[string]any{}
	for index := 3; index <= 10; index++ {
		offset := int(binary.LittleEndian.Uint32(raw[6+index*4:]))
		end := int(binary.LittleEndian.Uint32(raw[6+(index+1)*4:]))
		if end-offset != 484 || binary.LittleEndian.Uint16(raw[offset:]) != 24 || binary.LittleEndian.Uint16(raw[offset+2:]) != 20 {
			t.Fatal("raw entry header/bounds", index)
		}
		want := raw[offset+4 : end]
		sum := sha256.Sum256(want)
		if hex.EncodeToString(sum[:]) != expected[index] || !bytes.Equal(got.Entries[index].Pixels, want) {
			t.Fatal("exact raw payload differs", index)
		}
		wrong, e := fdother.ParseLMI1OpaqueEntry(raw, index)
		if e == nil && bytes.Equal(wrong.Pixels, want) {
			t.Fatal("did not distinguish old wrong codec", index)
		}
		// The original raw primitive skips zero without changing adjacent pixels.
		frame := bytes.Repeat([]byte{23}, 64000)
		if err := got.Entries[index].BlitAt(frame, 320, 201, 169, false); err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 20; y++ {
			for x := 0; x < 24; x++ {
				pixel := want[y*24+x]
				if pixel == 0 {
					pixel = 23
				}
				if frame[(169+y)*320+201+x] != pixel {
					t.Fatal("raw/zero-transparent pixel", index, x, y)
				}
			}
		}
		if frame[168*320+201] != 23 || frame[189*320+201] != 23 {
			t.Fatal("write outside cell")
		}
		checks = append(checks, map[string]any{"entry": index, "indexed_sha256": expected[index], "raw_offset": offset, "bytes": end - offset})
	}
	for _, index := range []int{1, 16} {
		want, e := fdother.ParseLMI1OpaqueEntry(raw, index)
		if e != nil || !bytes.Equal(want.Pixels, got.Entries[index].Pixels) {
			t.Fatal("unchanged opaque entry", index, e)
		}
	}
	for _, index := range []int{0, 23, 24, 25, 26, 27, 28, 29, 30, 31} {
		want, e := fdother.ParseLMI1FrameEntry(raw, index)
		if e != nil {
			t.Fatal(e)
		}
		pixels, mask, e := want.IndexedLayers()
		if e != nil {
			t.Fatal(e)
		}
		actual := got.Background
		if index != 0 {
			actual = got.ReviveFX[index-23]
		}
		if !bytes.Equal(pixels, actual.Indexed) || !bytes.Equal(mask, actual.Mask) {
			t.Fatal("unchanged four-mode entry", index)
		}
	}
	if os.Getenv("FD2_CHURCH_EXPORT_OUT") != "" {
		data, _ := json.MarshalIndent(map[string]any{"status": "PASS-ORIGINAL-CHURCH-RAW-SERVICE-EXPORT", "raw_sha256": hex.EncodeToString(digest[:]), "checks": checks, "unchanged_opaque": 2, "unchanged_four_mode": 10, "pixels_per_service": 480, "zero_transparency": true}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "codec-receipt.json"), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
