// Command fd2-class-list-oracle renders the recovered
// 0x31385 class-change scene from player-provided original archives. The
// fixture is 悠妮 (native identity/portrait 9), current class 5, special
// target class 21.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func main() {
	if len(os.Args) != 16 {
		fmt.Fprintln(os.Stderr, "usage: fd2-class-list-oracle FDOTHER.DAT FDTXT.DAT FDICON.B24 DATO.DAT native_item_effect_rows.json list.png confirm.png transfer.png transfer-full.png revive-list.png revive-confirm.png revive-empty.png revive-insufficient.png revive-success.png revive-success-flash.png")
		os.Exit(2)
	}
	fdotherPath, fdtxtPath, fdiconPath, datoPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	itemRowsPath := os.Args[5]
	listOutputPath, confirmOutputPath, transferOutputPath := os.Args[6], os.Args[7], os.Args[8]
	transferFullOutputPath := os.Args[9]
	reviveListOutputPath, reviveConfirmOutputPath := os.Args[10], os.Args[11]
	reviveEmptyOutputPath, reviveInsufficientOutputPath := os.Args[12], os.Args[13]
	reviveSuccessOutputPath, reviveSuccessFlashOutputPath := os.Args[14], os.Args[15]

	resource14, err := fdother.ReadResource(fdotherPath, 14)
	if err != nil {
		fail(err)
	}
	entries, err := fdother.ParseLMI1(resource14)
	if err != nil {
		fail(err)
	}
	backgroundFrame, err := fdother.ParseLMI1FrameEntry(resource14, 0)
	if err != nil {
		fail(err)
	}
	if len(entries) <= 16 || backgroundFrame.Width != 320 || backgroundFrame.Height != 200 {
		fail(fmt.Errorf("FDOTHER#14 lacks native background/panel entries"))
	}
	background := make([]byte, 320*200)
	if err := backgroundFrame.BlitAt(background, 320, 0, -1); err != nil {
		fail(err)
	}
	textRaw, err := fdother.ReadResource(fdtxtPath, 0)
	if err != nil {
		fail(err)
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		fail(err)
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		fail(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		fail(err)
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		fail(err)
	}
	dialogue := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			fail(err)
		}
	}
	digits := make([]fdother.Frame, 10)
	for digit := 0; digit < 10; digit++ {
		digits[digit], err = fdother.ParseLMI1FrameEntry(resource5, 31+digit)
		if err != nil {
			fail(err)
		}
	}
	portraits, err := dato.DecodeResource(datoPath, 131)
	if err != nil || len(portraits) == 0 {
		if err != nil {
			fail(err)
		}
		fail(fmt.Errorf("DATO#131 has no frames"))
	}
	scene, err := campaign.ComposeNativeChurchScene(
		background, entries[1], dialogue, digits, portraits[0], strings, font, 1000, 585,
	)
	if err != nil {
		fail(err)
	}
	source, err := campaign.NativeChurchMenuBase(scene)
	if err != nil {
		fail(err)
	}
	units, err := fdicon.DecodeFile(fdiconPath)
	if err != nil {
		fail(err)
	}
	sprite, err := units.SpriteFor(9, 0, 0)
	if err != nil {
		fail(err)
	}
	frame, err := campaign.ComposeNativeClassListFrame(source, entries[16], []campaign.NativeClassListRow{{
		Sprite: sprite, NameTextIndex: 10, CurrentClassTextID: 5, TargetClassTextID: 21,
	}}, 0, strings, font)
	if err != nil {
		fail(err)
	}
	if _, err := campaign.NativeClassListOpeningFrames(source, frame); err != nil {
		fail(err)
	}
	if _, err := campaign.NativeClassListClosingFrames(source, frame); err != nil {
		fail(err)
	}
	cells, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		fail(err)
	}
	dialogueBase, err := campaign.ComposeNativeChurchDialogueOverlay(source, dialogue, portraits[0])
	if err != nil {
		fail(err)
	}
	confirm, err := campaign.ComposeNativeClassConfirmationFrame(dialogueBase, cells, strings, font, 10, 0, 1)
	if err != nil {
		fail(err)
	}
	if _, err := campaign.NativeClassConfirmationOpeningFrames(confirm, cells); err != nil {
		fail(err)
	}
	if _, err := campaign.NativeClassConfirmationClosingFrames(confirm, cells); err != nil {
		fail(err)
	}
	if _, err := campaign.NativeClassListClosingFrames(source, dialogueBase); err != nil {
		fail(err)
	}
	transfer := append([]byte(nil), source...)
	if err := entries[16].BlitOpaqueAt(transfer, 320, 5, 112, false); err != nil {
		fail(err)
	}
	itemAssets, err := battle.LoadNativeItemPanelDataAssetsArchive(fdotherPath, fdtxtPath)
	if err != nil {
		fail(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(itemRowsPath)
	if err != nil {
		fail(err)
	}
	priceCell, err := fdother.ParseLMI1RawEntry(resource14, 15)
	if err != nil {
		fail(err)
	}
	if err := battle.RenderNativeTransferItemRows(
		itemAssets, priceCell, []int{0, 79, 90}, 0, 1, itemRows, transfer,
	); err != nil {
		fail(err)
	}
	transferFullScene, err := campaign.ComposeNativeChurchScene(
		background, entries[1], dialogue, digits, portraits[0], strings, font, 1000, 510,
	)
	if err != nil {
		fail(err)
	}
	transferFullSource, err := campaign.NativeChurchMenuBase(transferFullScene)
	if err != nil {
		fail(err)
	}
	transferFullBase, err := campaign.ComposeNativeChurchDialogueOverlay(
		transferFullSource, dialogue, portraits[0],
	)
	if err != nil {
		fail(err)
	}
	transferFull, err := campaign.ComposeNativeChurchTextWithNameAt(
		transferFullBase, strings, font, 506, 10, 119*320+12,
	)
	if err != nil {
		fail(err)
	}
	reviveScene, err := campaign.ComposeNativeChurchScene(
		background, entries[1], dialogue, digits, portraits[0], strings, font, 1000, 589,
	)
	if err != nil {
		fail(err)
	}
	reviveSource, err := campaign.NativeChurchMenuBase(reviveScene)
	if err != nil {
		fail(err)
	}
	reviveList := append([]byte(nil), reviveSource...)
	if err := entries[16].BlitOpaqueAt(reviveList, 320, 5, 112, false); err != nil {
		fail(err)
	}
	if err := battle.RenderNativeReviveRows(
		itemAssets, priceCell, []battle.NativeReviveRow{{
			Sprite: sprite, NameTextIndex: 10,
			RaceTextIndex: 141, ClassTextIndex: 155, Fee: 200,
		}}, 0, reviveList,
	); err != nil {
		fail(err)
	}
	reviveDialogue, err := campaign.ComposeNativeChurchDialogueOverlay(
		reviveSource, dialogue, portraits[0],
	)
	if err != nil {
		fail(err)
	}
	reviveQuestion, err := campaign.ComposeNativeReviveConfirmationQuestion(
		reviveDialogue, strings, font, 10, 200,
	)
	if err != nil {
		fail(err)
	}
	reviveConfirm, err := campaign.ComposeNativeConfirmationChoices(
		reviveQuestion, cells, 0, 1,
	)
	if err != nil {
		fail(err)
	}
	reviveEmptyBase, err := campaign.ComposeNativeChurchDialogueOverlay(
		source, dialogue, portraits[0],
	)
	if err != nil {
		fail(err)
	}
	reviveEmpty, err := campaign.ComposeNativeChurchTextAt(
		reviveEmptyBase, strings, font, 588, 119*320+12,
	)
	if err != nil {
		fail(err)
	}
	reviveInsufficient, err := campaign.ComposeNativeChurchTextAt(
		reviveQuestion, strings, font, 504, 157*320+12,
	)
	if err != nil {
		fail(err)
	}
	successScene, err := campaign.ComposeNativeChurchScene(
		background, entries[1], dialogue, digits, portraits[0], strings, font, 800, 589,
	)
	if err != nil {
		fail(err)
	}
	successSource, err := campaign.NativeChurchMenuBase(successScene)
	if err != nil {
		fail(err)
	}
	successDialogue, err := campaign.ComposeNativeChurchDialogueOverlay(
		successSource, dialogue, portraits[0],
	)
	if err != nil {
		fail(err)
	}
	successQuestion, err := campaign.ComposeNativeReviveConfirmationQuestion(
		successDialogue, strings, font, 10, 200,
	)
	if err != nil {
		fail(err)
	}
	successFX := make([]fdother.Frame, 9)
	for i := range successFX {
		successFX[i], err = fdother.ParseLMI1FrameEntry(resource14, 23+i)
		if err != nil {
			fail(err)
		}
	}
	successFrames, _, err := campaign.ComposeNativeChurchReviveSuccessFrames(
		successQuestion, successFX, portraits[0],
	)
	if err != nil {
		fail(err)
	}
	paletteRaw, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		fail(err)
	}
	palette, err := fdother.ParseVGAPalette(paletteRaw)
	if err != nil {
		fail(err)
	}
	palette[0] = color.NRGBA{A: 0xff}
	writePNG(listOutputPath, frame, palette)
	writePNG(confirmOutputPath, confirm, palette)
	writePNG(transferOutputPath, transfer, palette)
	writePNG(transferFullOutputPath, transferFull, palette)
	writePNG(reviveListOutputPath, reviveList, palette)
	writePNG(reviveConfirmOutputPath, reviveConfirm, palette)
	writePNG(reviveEmptyOutputPath, reviveEmpty, palette)
	writePNG(reviveInsufficientOutputPath, reviveInsufficient, palette)
	writePNG(reviveSuccessOutputPath, successFrames[4], palette)
	flashDAC := append([]byte(nil), paletteRaw...)
	if err := fdother.ApplyVGAPaletteDelta(flashDAC, paletteRaw, 0, 255, 62); err != nil {
		fail(err)
	}
	flashPalette, err := fdother.VGAPaletteFromDAC(flashDAC)
	if err != nil {
		fail(err)
	}
	flashPalette[0] = color.NRGBA{A: 0xff}
	writePNG(reviveSuccessFlashOutputPath, successFrames[4], flashPalette)
}

func writePNG(path string, pixels []byte, palette color.Palette) {
	out := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(out.Pix, pixels)
	file, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	if err := png.Encode(file, out); err != nil {
		file.Close()
		fail(err)
	}
	if err := file.Close(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
