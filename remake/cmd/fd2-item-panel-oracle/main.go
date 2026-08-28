// Command fd2-item-panel-oracle renders the recovered 0x17eef+0x17fc0
// indexed item panel from player-provided original archives and the separated
// portrait pack.
package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func main() {
	if len(os.Args) != 8 {
		fmt.Fprintln(os.Stderr, "usage: fd2-item-panel-oracle FDOTHER.DAT FDTXT.DAT PORTRAIT_ROOT native_item_effect_rows.json spells.json item.png command.png")
		os.Exit(2)
	}
	fdotherPath, fdtxtPath, portraitRoot := os.Args[1], os.Args[2], os.Args[3]
	itemRowsPath, spellsPath := os.Args[4], os.Args[5]
	itemOutputPath, commandOutputPath := os.Args[6], os.Args[7]

	record := make([]byte, 80)
	record[7] = 0
	record[8] = 0
	record[31] = 0
	record[32] = 0
	record[33] = 12
	record[37] = 1
	for slot := 0; slot < 8; slot++ {
		record[0x0a+slot*2] = 0x80
	}
	record[0x0a], record[0x0b] = 0x40, 0
	record[0x0c], record[0x0d] = 0, 79
	binary.LittleEndian.PutUint16(record[62:], 345)
	binary.LittleEndian.PutUint16(record[64:], 80)
	binary.LittleEndian.PutUint16(record[66:], 100)
	binary.LittleEndian.PutUint16(record[68:], 20)
	binary.LittleEndian.PutUint16(record[70:], 40)
	binary.LittleEndian.PutUint16(record[72:], 123)
	binary.LittleEndian.PutUint16(record[74:], 98)
	binary.LittleEndian.PutUint16(record[76:], 76)
	binary.LittleEndian.PutUint16(record[78:], 54)

	base := make([]byte, 320*200)
	if err := battle.RenderNativeItemPanelResourcesArchive(fdotherPath, fdtxtPath, portraitRoot, record, base); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	assets, err := battle.LoadNativeItemPanelDataAssetsArchive(fdotherPath, fdtxtPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(itemRowsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	itemPixels := append([]byte(nil), base...)
	if err := battle.RenderNativeItemPanelRows(assets, record, 0, itemRows, itemPixels); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	book, err := battle.LoadNativeCommandRecords(spellsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	commandPixels := append([]byte(nil), base...)
	if err := battle.RenderNativeCommandOverlay(
		assets, []int{0, 13, 20, 24, 26}, book, -1, commandPixels,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	paletteData, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	palette, err := fdother.ParseVGAPalette(paletteData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// The completed native framebuffer is opaque; palette index zero is no
	// longer a sprite transparency marker at this presentation boundary.
	palette[0] = color.NRGBA{A: 0xff}
	writePNG(itemOutputPath, itemPixels, palette)
	writePNG(commandOutputPath, commandPixels, palette)
}

func writePNG(path string, pixels []byte, palette color.Palette) {
	frame := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(frame.Pix, pixels)
	output, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := png.Encode(output, frame); err != nil {
		output.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := output.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
