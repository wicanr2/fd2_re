package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/ending"
)

func main() {
	gameRoot := flag.String("game-root", "", "player-provided original game directory")
	assetPack := flag.String("asset-pack", "", "separated asset pack root")
	output := flag.String("out", "", "output PNG path")
	segment := flag.Int("segment", 9, "approximate tail segment 0..19")
	contactSheet := flag.Bool("contact-sheet", false, "write all 20 approximate overlays as a 5x4 sheet")
	flag.Parse()
	if *gameRoot == "" || *assetPack == "" || *output == "" || *segment < 0 || *segment >= 20 {
		fatalf("usage: fd2-ending-tail-oracle -game-root DIR -asset-pack DIR -out FILE [-segment 0..19]")
	}
	tail, err := ending.LoadMontageTail(filepath.Join("assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		fatalf("load tail schedule: %v", err)
	}
	assets, err := ending.LoadMontageTailAssets(*tail, filepath.Join(*gameRoot, "FDOTHER.DAT"))
	if err != nil {
		fatalf("load tail FDOTHER assets: %v", err)
	}
	sets, err := ending.LoadMontageTailVisualSets(*tail, ending.MontageTailVisualPaths{
		SurfaceRoot:   filepath.Join(*assetPack, "surfaces"),
		AnimationRoot: filepath.Join(*assetPack, "animations"),
	})
	if err != nil {
		fatalf("load tail visual assets: %v", err)
	}
	player, err := ending.NewMontageTailPlayer(*tail, assets, sets, ending.NewIndexedCompositor())
	if err != nil {
		fatalf("create tail player: %v", err)
	}
	var captures []*image.RGBA
	for steps := 0; steps < 4096; steps++ {
		beforeSegment := player.Segment
		beforePhase := player.Phase
		if err := player.Step(); err != nil {
			fatalf("advance tail: %v", err)
		}
		if beforePhase == ending.MontageTailPhaseOverlay {
			captures = append(captures, cloneRGBA(player.Compositor.RGBA()))
			if !*contactSheet && beforeSegment == *segment {
				break
			}
			if *contactSheet && len(captures) == 20 {
				break
			}
		}
		if player.Ready() {
			fatalf("tail completed before segment %d overlay", *segment)
		}
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	file, err := os.Create(*output)
	if err != nil {
		fatalf("create output: %v", err)
	}
	outputImage := image.Image(player.Compositor.RGBA())
	if *contactSheet {
		if len(captures) != 20 {
			fatalf("tail produced %d contact-sheet captures, want 20", len(captures))
		}
		outputImage = makeContactSheet(captures)
	}
	if err := png.Encode(file, outputImage); err != nil {
		_ = file.Close()
		fatalf("encode output: %v", err)
	}
	if err := file.Close(); err != nil {
		fatalf("close output: %v", err)
	}
}

func cloneRGBA(source *image.RGBA) *image.RGBA {
	copyImage := image.NewRGBA(source.Bounds())
	copy(copyImage.Pix, source.Pix)
	return copyImage
}

func makeContactSheet(captures []*image.RGBA) *image.RGBA {
	const columns, rows, gap = 5, 4, 4
	width := columns*ending.Width + (columns+1)*gap
	height := rows*ending.Height + (rows+1)*gap
	sheet := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	for index, capture := range captures {
		x := gap + (index%columns)*(ending.Width+gap)
		y := gap + (index/columns)*(ending.Height+gap)
		draw.Draw(sheet, image.Rect(x, y, x+ending.Width, y+ending.Height), capture, image.Point{}, draw.Src)
	}
	return sheet
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
