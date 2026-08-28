// Command fd2-shop-scene-oracle renders the recovered stable target of
// 0x2E341+0x1956B from player-provided original indexed resources. The
// deterministic fixture uses hub variant zero, DATO#129, gold=12345678,
// FDTXT_000 string #440, and service option zero at native pulse phase two.
package main

import (
	"encoding/binary"
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
	if len(os.Args) < 5 || len(os.Args) == 9 || len(os.Args) > 15 {
		fmt.Fprintln(os.Stderr, "usage: fd2-shop-scene-oracle FDOTHER.DAT FDTXT.DAT DATO.DAT menu.png [purchase-list.png [purchase-confirm.png [purchase-insufficient.png [FDICON.B24 recipient.png [recipient-full.png [equipment-recipient.png [purchase-success.png [sell-list.png [gold-debit.png]]]]]]]]]")
		os.Exit(2)
	}
	fdotherPath, fdtxtPath, datoPath, outputPath := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	assets, err := campaign.DecodeNativeShopAssets(fdotherPath, 12)
	check(err)
	resource5 := mustResource(fdotherPath, 5)
	dialogue := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		check(err)
	}
	digits := make([]fdother.Frame, 10)
	for digit := range digits {
		digits[digit], err = fdother.ParseLMI1FrameEntry(resource5, 31+digit)
		check(err)
	}
	portraits, err := dato.DecodeResource(datoPath, 0x81)
	check(err)
	strings, err := fdtxt.Parse(mustResource(fdtxtPath, 0))
	check(err)
	font, err := fdtxt.ParseFont(mustResource(fdotherPath, 4))
	check(err)
	stable, err := campaign.ComposeNativeShopScene(
		assets, dialogue, digits, portraits[0], 0x81,
		strings, font, 12345678, 0x1b8,
	)
	check(err)
	frame, err := campaign.ComposeNativeShopServiceSteadyFrame(stable, assets, 0, 2)
	check(err)
	palette, err := fdother.ParseVGAPalette(mustResource(fdotherPath, 0))
	check(err)
	palette[0] = color.NRGBA{A: 0xff}
	writePNG(outputPath, frame, palette)
	if len(os.Args) >= 6 {
		itemAssets, err := battle.LoadNativeItemPanelDataAssetsArchive(fdotherPath, fdtxtPath)
		check(err)
		effectRows, err := battle.LoadNativeItemEffectRowPrefix(
			"assets/data/native_item_effect_rows.json",
		)
		check(err)
		purchase, err := campaign.ComposeNativeShopItemListFrame(
			stable, assets, itemAssets, []int{0, 1, 2, 3, 4, 5},
			0, 0, effectRows, battle.NativeFacilityFullPrice,
		)
		check(err)
		writePNG(os.Args[5], purchase, palette)
		if len(os.Args) >= 14 {
			sale, err := campaign.ComposeNativeShopItemListFrame(
				stable, assets, itemAssets, []int{0, 1, 2, 3, 4, 5},
				0, 0, effectRows, battle.NativeFacilityThreeQuarterPrice,
			)
			check(err)
			writePNG(os.Args[13], sale, palette)
		}
	}
	if len(os.Args) >= 7 {
		purchasePortraits, err := dato.DecodeResource(datoPath, 0x80)
		check(err)
		purchaseSource, err := campaign.ComposeNativeShopScene(
			assets, dialogue, digits, purchasePortraits[0], 0x80,
			strings, font, 12345678, 0x1f5,
		)
		check(err)
		choices, err := fdother.DecodeRawCellResource(fdotherPath, 2)
		check(err)
		confirmation, err := campaign.ComposeNativeShopPurchaseConfirmation(
			purchaseSource, dialogue, purchasePortraits[0], 0x80,
			strings, font, choices, campaign.NativeShopPurchaseQuestion,
			1, 0, 50, 0, 1,
		)
		check(err)
		writePNG(os.Args[6], confirmation, palette)
		if len(os.Args) >= 8 {
			question, err := campaign.ComposeNativeShopPurchaseMessage(
				purchaseSource, dialogue, purchasePortraits[0], 0x80,
				strings, font, campaign.NativeShopPurchaseQuestion,
				1, 0, 50,
			)
			check(err)
			closing, err := campaign.NativeClassConfirmationClosingFrames(
				question, choices,
			)
			check(err)
			insufficient, err := campaign.ComposeNativeShopPurchaseInsufficientGold(
				closing[len(closing)-1], strings, font, 1,
			)
			check(err)
			writePNG(os.Args[7], insufficient, palette)
		}
		if len(os.Args) >= 10 {
			units, err := fdicon.DecodeFile(os.Args[8])
			check(err)
			rows := make([]campaign.NativeRosterRow, 6)
			for identity := range rows {
				sprite, err := units.SpriteFor(identity, 0, 0)
				check(err)
				rows[identity] = campaign.NativeRosterRow{
					Sprite: sprite, NameTextIndex: identity + 1,
				}
			}
			recipient, err := campaign.ComposeNativeShopConsumableRecipientFrame(
				purchaseSource, assets, rows, 0, 0x20, strings, font,
			)
			check(err)
			writePNG(os.Args[9], recipient, palette)
			if len(os.Args) >= 11 {
				full, err := campaign.ComposeNativeShopPurchaseRecipientFull(
					purchaseSource, dialogue, purchasePortraits[0], 0x80,
					strings, font, 1, 1,
				)
				check(err)
				writePNG(os.Args[10], full, palette)
			}
			if len(os.Args) >= 12 {
				itemAssets, err := battle.LoadNativeItemPanelDataAssetsArchive(
					fdotherPath, fdtxtPath,
				)
				check(err)
				effectRows, err := battle.LoadNativeItemEffectRowPrefix(
					"assets/data/native_item_effect_rows.json",
				)
				check(err)
				equipmentRows := make([]campaign.NativeShopEquipmentRecipientRow, 3)
				for identity := range equipmentRows {
					record := make([]byte, 0x50)
					record[7], record[8] = byte(identity), byte(identity)
					binary.LittleEndian.PutUint16(record[0x37:], uint16(40+identity))
					binary.LittleEndian.PutUint16(record[0x39:], uint16(30+identity))
					binary.LittleEndian.PutUint16(record[0x3e:], uint16(20+identity))
					for stat, offset := range []int{0x48, 0x4a, 0x4c, 0x4e} {
						binary.LittleEndian.PutUint16(
							record[offset:], uint16(50+stat+identity),
						)
					}
					candidate, err := campaign.NativeShopEquipmentCandidateStats(
						record, 0, effectRows,
					)
					check(err)
					current, err := campaign.NativeShopEquipmentCurrentStats(record)
					check(err)
					equipmentRows[identity] = campaign.NativeShopEquipmentRecipientRow{
						Sprite: rows[identity].Sprite, NameTextIndex: identity + 1,
						Current: current, Candidate: candidate,
					}
				}
				equipment, err := campaign.ComposeNativeShopEquipmentRecipientFrame(
					purchaseSource, assets, itemAssets, equipmentRows, 0,
				)
				check(err)
				writePNG(os.Args[11], equipment, palette)
			}
			if len(os.Args) >= 13 {
				bare, err := campaign.ComposeNativeShopBareScene(
					assets, digits, 1000,
				)
				check(err)
				animation, final, err := campaign.ComposeNativeShopPurchaseSuccessFrames(
					bare, assets, purchasePortraits[0], 0x80, 1,
				)
				check(err)
				writePNGFrames(
					os.Args[12], append(animation, final), palette,
				)
				if len(os.Args) >= 15 {
					debit, next, err := campaign.ComposeNativeGoldDebitFrames(
						final, assets.GoldRollStrip, 1000, 50,
					)
					check(err)
					if next != 950 || len(debit) != 45 {
						check(fmt.Errorf(
							"gold debit next=%d frames=%d", next, len(debit),
						))
					}
					writePNGFrames(os.Args[14], debit, palette)
				}
			}
		}
	}
}

func mustResource(path string, index int) []byte {
	resource, err := fdother.ReadResource(path, index)
	check(err)
	return resource
}

func writePNG(path string, pixels []byte, palette color.Palette) {
	out := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(out.Pix, pixels)
	file, err := os.Create(path)
	check(err)
	check(png.Encode(file, out))
	check(file.Close())
}

func writePNGFrames(path string, frames [][]byte, palette color.Palette) {
	out := image.NewPaletted(image.Rect(0, 0, 320*len(frames), 200), palette)
	for frameIndex, pixels := range frames {
		if len(pixels) != 320*200 {
			check(fmt.Errorf("frame %d has %d pixels", frameIndex, len(pixels)))
		}
		for y := 0; y < 200; y++ {
			copy(
				out.Pix[y*out.Stride+frameIndex*320:],
				pixels[y*320:(y+1)*320],
			)
		}
	}
	file, err := os.Create(path)
	check(err)
	check(png.Encode(file, out))
	check(file.Close())
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
