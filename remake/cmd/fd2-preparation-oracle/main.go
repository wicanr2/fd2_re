// Command fd2-preparation-oracle renders the evidence-closed subpasses of the
// original 0x318ad preparation roster. It is a resource/compositor oracle, not
// an original runtime capture or a fabricated FD2.SAV.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func main() {
	base := flag.String("base", "", "player-owned FLAME2 directory")
	scenario := flag.String("scenario", "", "editable scenario supplying one proven native party record")
	out := flag.String("out", "preparation-roster-oracle.png", "output PNG")
	confirm := flag.Bool("confirm", false, "overlay the evidence-closed stable 0x31d3c final confirmation state")
	prompt := flag.String("prompt", "", "optional native pre-selection prompt: record or departure-ch02")
	lifecycleOut := flag.String("lifecycle-out", "", "optional 7×3 contact sheet: 10 open, stable, 9 close, restore")
	flag.Parse()
	if *base == "" {
		fmt.Fprintln(os.Stderr, "缺少 -base：請指定玩家持有的 FLAME2 目錄")
		os.Exit(2)
	}
	fdotherPath := filepath.Join(*base, "FDOTHER.DAT")
	assets, err := fdother.DecodeNativePreparationAssetsArchive(
		fdotherPath, filepath.Join(*base, "FDICON.B24"),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	keys := make([]int, 20)
	selected := make([]bool, len(keys))
	for i := range keys {
		keys[i] = i
		selected[i] = i < 5
	}
	frame, err := fdother.ComposeNativePreparationFrame(
		assets, keys, selected, 0, 0, 15,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *scenario != "" {
		dataAssets, err := battle.LoadNativeItemPanelDataAssetsArchive(
			fdotherPath, filepath.Join(*base, "FDTXT.DAT"),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		battleScenario, err := battle.LoadScenario(*scenario)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		units := battleScenario.PartyUnits(nil)
		if len(units) == 0 {
			fmt.Fprintln(os.Stderr, "指定情境沒有可用的原始玩家角色記錄")
			os.Exit(1)
		}
		record, err := battle.NativeItemPanelRecordForUnit(units[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := battle.RenderNativeItemPanelData(dataAssets, record, frame); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var lifecycleFrames [][]byte
	if *prompt != "" && *prompt != "record" && *prompt != "departure-ch02" {
		fmt.Fprintln(os.Stderr, "-prompt 只接受 record 或 departure-ch02")
		os.Exit(2)
	}
	if *confirm || *prompt != "" || *lifecycleOut != "" {
		source := append([]byte(nil), frame...)
		if *prompt == "record" {
			// 0x2cc04 clears all 64000 VGA bytes before FDTXT 0x19a.
			source = make([]byte, 320*200)
		}
		choices, err := fdother.DecodeRawCellResource(fdotherPath, 2)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		textRaw, err := fdother.ReadResource(filepath.Join(*base, "FDTXT.DAT"), 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		strings, err := fdtxt.Parse(textRaw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fontRaw, err := fdother.ReadResource(fdotherPath, 4)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		font, err := fdtxt.ParseFont(fontRaw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *prompt == "departure-ch02" {
			townAssets, err := campaign.DecodeNativeTownAssetsArchive(
				fdotherPath, filepath.Join(*base, "FDICON.B24"),
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			source, err = campaign.ComposeNativeTownFrame(
				townAssets, strings, font, 0, 2, 0,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		resource5, err := fdother.ReadResource(fdotherPath, 5)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dialogue := make([]fdother.RawCell, 20)
		for index := 1; index <= 19; index++ {
			dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		portraits, err := dato.DecodeResource(filepath.Join(*base, "DATO.DAT"), 0x4b)
		if err != nil || len(portraits) == 0 {
			if err == nil {
				err = fmt.Errorf("DATO#75 has no frames")
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dialogueFrame, err := campaign.ComposeNativePreparationConfirmationDialogue(
			source, dialogue, portraits[0],
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var question []byte
		switch *prompt {
		case "record":
			question, err = campaign.ComposeNativePreparationRecordQuestion(
				source, dialogue, portraits[0], strings, font,
			)
		case "departure-ch02":
			question, err = campaign.ComposeNativePreparationDepartureQuestion(
				source, dialogue, portraits[0], strings, font,
			)
		default:
			question, err = campaign.ComposeNativePreparationConfirmationQuestion(
				source, dialogue, portraits[0], strings, font,
			)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		frame, err = campaign.ComposeNativeConfirmationChoices(
			question, choices, 0, 0,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *lifecycleOut != "" {
			open, err := campaign.NativePreparationConfirmationOpeningFrames(
				source, dialogueFrame, question, choices,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			closeFrames, err := campaign.NativePreparationConfirmationClosingFrames(
				source, dialogueFrame, question, choices,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			lifecycleFrames = append(lifecycleFrames, open...)
			lifecycleFrames = append(lifecycleFrames, frame)
			lifecycleFrames = append(lifecycleFrames, closeFrames...)
			lifecycleFrames = append(lifecycleFrames, source)
		}
	}
	paletteRaw, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	palette, err := fdother.ParseVGAPalette(paletteRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(indexed.Pix, frame)
	scaled := image.NewPaletted(image.Rect(0, 0, 640, 400), palette)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			scaled.SetColorIndex(x, y, indexed.ColorIndexAt(x/2, y/2))
		}
	}
	file, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	if err := png.Encode(file, scaled); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *lifecycleOut != "" {
		const columns = 7
		rows := (len(lifecycleFrames) + columns - 1) / columns
		sheet := image.NewPaletted(
			image.Rect(0, 0, columns*320, rows*200), palette,
		)
		for index, candidate := range lifecycleFrames {
			x0 := (index % columns) * 320
			y0 := (index / columns) * 200
			for y := 0; y < 200; y++ {
				dst := (y0+y)*sheet.Stride + x0
				copy(sheet.Pix[dst:dst+320], candidate[y*320:(y+1)*320])
			}
		}
		lifecycleFile, err := os.Create(*lifecycleOut)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer lifecycleFile.Close()
		if err := png.Encode(lifecycleFile, sheet); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
