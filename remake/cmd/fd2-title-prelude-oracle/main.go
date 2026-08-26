// fd2-title-prelude-oracle 從玩家自備的 FDOTHER.DAT 重建原版開場前導畫面。
// 它不夾帶原版素材；輸出只供本機原版／重製證據比較。
package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "用法: fd2-title-prelude-oracle <FDOTHER.DAT> <輸出.png>")
		os.Exit(2)
	}
	frameRaw, err := fdother.ReadResource(os.Args[1], 74)
	if err != nil {
		fatal(err)
	}
	paletteRaw, err := fdother.ReadResource(os.Args[1], 76)
	if err != nil {
		fatal(err)
	}
	frame, err := fdother.ParseSingleFrame(frameRaw)
	if err != nil {
		fatal(err)
	}
	if frame.Width != 320 || frame.Height != 200 {
		fatal(fmt.Errorf("FDOTHER #74 幾何=%dx%d，預期 320x200", frame.Width, frame.Height))
	}
	pixels := make([]byte, 320*200)
	if err := frame.Blit(pixels, 320, -1); err != nil {
		fatal(err)
	}
	palette, err := fdother.ParseVGAPalette(paletteRaw)
	if err != nil {
		fatal(err)
	}
	indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(indexed.Pix, pixels)
	out, err := os.Create(os.Args[2])
	if err != nil {
		fatal(err)
	}
	if err := png.Encode(out, indexed); err != nil {
		_ = out.Close()
		fatal(err)
	}
	if err := out.Close(); err != nil {
		fatal(err)
	}
	fmt.Printf("FDOTHER #74 + #76: 320x200 indexed SHA-256=%x\n", sha256.Sum256(pixels))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
