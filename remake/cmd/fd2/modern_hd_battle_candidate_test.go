package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func hdBattleCandidatePalette() color.Palette {
	p := make(color.Palette, 256)
	for i := range p {
		p[i] = color.NRGBA{R: byte(i), G: byte(255 - i), B: byte(i ^ 0x55), A: 255}
	}
	return p
}

func TestModernHDBattleCandidateNativeGeometry(t *testing.T) {
	palette := hdBattleCandidatePalette()
	base := make([]byte, 320*200)
	for i := range base {
		base[i] = byte(i % 251)
	}
	attack := figani.Frame{Width: 4, Height: 3, Pixels: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, Mask: []byte{1, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1}}
	idle := figani.Frame{Width: 3, Height: 2, Pixels: []byte{0, 81, 82, 83, 84, 85}, Mask: []byte{1, 1, 1, 1, 0, 1}}
	for _, pos := range []image.Point{image.Pt(-2, -1), image.Pt(318, 198), image.Pt(35, 75)} {
		attack.X, attack.Y, idle.X, idle.Y = pos.X, pos.Y, pos.X+1, pos.Y+1
		for _, side := range []byte{0, 1, 255} {
			for _, flag := range []byte{0, 1} {
				attack.RawByte7 = flag
				for _, fill := range []int{-1, 33} {
					for _, shift := range []image.Point{{}, {X: -3, Y: 2}} {
						present := battlepresent.NativePhysicalBodyPresent{DX: shift.X, DY: shift.Y, OpaqueFill: fill}
						want, err := battlepresent.ComposeNativePhysicalBodyPresent(base, attack, idle, side, present)
						if err != nil {
							t.Fatal(err)
						}
						got, err := composeModernHDBattleBodyCandidate(base, attack, idle, side, present, palette, nil, nil)
						if err != nil {
							t.Fatal(err)
						}
						for y := 0; y < 200; y++ {
							for x := 0; x < 320; x++ {
								value := color.RGBAModel.Convert(palette[want[y*320+x]]).(color.RGBA)
								for _, point := range []image.Point{{X: x * 6, Y: y * 6}, {X: x*6 + 5, Y: y*6 + 5}} {
									if got.RGBAAt(point.X, point.Y) != value {
										t.Fatalf("pos=%v side=%d flag=%d fill=%d shift=%v pixel=%v", pos, side, flag, fill, shift, point)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestModernHDBattleCandidateHDDetailsAlphaAndFill(t *testing.T) {
	palette := hdBattleCandidatePalette()
	base := make([]byte, 64000)
	frame := figani.Frame{X: 10, Y: 20, Width: 1, Height: 1, Pixels: []byte{55}, Mask: []byte{1}}
	blank := figani.Frame{Width: 1, Height: 1, Pixels: []byte{0}, Mask: []byte{0}}
	hd := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	hd.SetNRGBA(1, 2, color.NRGBA{R: 230, G: 70, B: 10, A: 255})
	hd.SetNRGBA(2, 2, color.NRGBA{R: 230, G: 70, B: 10, A: 128})
	present := battlepresent.NativePhysicalBodyPresent{DX: 2, DY: -1, OpaqueFill: -1}
	for _, fill := range []int{-1, 33} {
		present.OpaqueFill = fill
		got, err := composeModernHDBattleBodyCandidate(base, blank, frame, 0, present, palette, nil, hd)
		if err != nil {
			t.Fatal(err)
		}
		x, y := 12*6, 19*6
		want := color.RGBA{R: 230, G: 70, B: 10, A: 255}
		if fill == 33 {
			want = color.RGBAModel.Convert(palette[33]).(color.RGBA)
		}
		if got.RGBAAt(x+1, y+2) != want {
			t.Fatalf("HD detail or fill lost: %v", got.RGBAAt(x+1, y+2))
		}
		if got.RGBAAt(x, y) != color.RGBAModel.Convert(palette[0]).(color.RGBA) {
			t.Fatal("transparent HD pixel overwrote the background")
		}
		mixed := got.RGBAAt(x+2, y+2)
		if mixed == want || mixed == got.RGBAAt(x, y) || mixed.A != 255 {
			t.Fatal("coverage alpha was discarded")
		}
	}
}

func TestModernHDBattleCandidateRejectsInvalidSources(t *testing.T) {
	frame := figani.Frame{Width: 1, Height: 1, Pixels: []byte{0}, Mask: []byte{1}}
	present := battlepresent.NativePhysicalBodyPresent{OpaqueFill: -1}
	if _, err := composeModernHDBattleBodyCandidate(make([]byte, 64000), frame, frame, 0, present,
		hdBattleCandidatePalette(), image.NewRGBA(image.Rect(0, 0, 5, 6)), nil); err == nil {
		t.Fatal("malformed HD image silently fell back")
	}
	present.OpaqueFill = 0
	if _, err := composeModernHDBattleBodyCandidate(make([]byte, 64000), frame, frame, 0, present,
		hdBattleCandidatePalette(), nil, nil); err == nil {
		t.Fatal("unknown fill accepted")
	}
}

// 需要合法原版分離包及候選圖。只產生離線 pose 預覽，不建立 Game 或改存檔。
func TestModernHDBattleCandidateGaiaPosePreview(t *testing.T) {
	pack, pose, out := os.Getenv("FD2_TEST_ORIGINAL_PACK"), os.Getenv("FD2_TEST_BATTLE_POSE"), os.Getenv("FD2_TEST_BATTLE_OUT")
	if pack == "" || pose == "" || out == "" {
		t.Skip("requires explicit original pack, pose and output paths")
	}
	animation, err := figani.LoadSeparatedResource(filepath.Join(pack, "animations"), 91)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(pose)
	if err != nil {
		t.Fatal(err)
	}
	hd, err := png.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	_, palette, err := fdother.LoadSeparatedFDOTHERPalette(filepath.Join(pack, "palette"), 0)
	if err != nil {
		t.Fatal(err)
	}
	frame := animation.Frames[7]
	blank := figani.Frame{Width: 1, Height: 1, Pixels: []byte{0}, Mask: []byte{0}}
	present := battlepresent.NativePhysicalBodyPresent{OpaqueFill: -1}
	native, err := composeModernHDBattleBodyCandidate(make([]byte, 64000), frame, blank, 0, present, palette, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := composeModernHDBattleBodyCandidate(make([]byte, 64000), frame, blank, 0, present, palette, hd, nil)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for y := hd.Bounds().Min.Y; y < hd.Bounds().Max.Y; y++ {
		for x := hd.Bounds().Min.X; x < hd.Bounds().Max.X; x++ {
			r, g, b, a := hd.At(x, y).RGBA()
			if a == 0xffff {
				want := color.RGBA{R: byte(r >> 8), G: byte(g >> 8), B: byte(b >> 8), A: 255}
				if candidate.RGBAAt(frame.X*6+x, frame.Y*6+y) != want {
					t.Fatal("candidate art moved or changed")
				}
				checked++
			}
		}
	}
	if checked < 1000 {
		t.Fatal("candidate lacks an opaque core")
	}
	for name, picture := range map[string]image.Image{"gaia-battle-native-layout.png": native, "gaia-battle-hd-layout.png": candidate} {
		path := filepath.Join(out, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, picture); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("offline resource91/frame7: %d opaque HD pixels retain descriptor position; no GUI/parity claim", checked)
}
