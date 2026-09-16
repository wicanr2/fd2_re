package main

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// 酒店（0x2FC85，城鎮 hub 變體 0）與商店共用同一套介面框架：0x111BA 載入 FDOTHER
// 資源 13（版面與資源 12 相同：背景、四對服務圖示、310×86 對話框），0x1956B 開對話框並
// 用 [0x52659] 的 DATO 0x81 當店主頭像，0x15F84 在對話框寫文字 0x249「有什麼事嗎？」，
// 0x2D669／0x2D7BD 的四格圖示選單（0 傳聞 0x2FFA5、1 存檔 0x30012、2 讀檔 0x301F4、
// 3 離開）。存檔 0x30012 走 0x30550 的四槽列表（與標題 LOAD 同一支），寫完檔再開一次
// 對話框寫文字 0x294「記錄儲存完畢！」等按鍵。這裡接 1（存檔）與 3（離開）；0／2 的
// 內容仍失敗即關閉回舊版面。
const (
	nativeHotelPortraitID  = 0x81
	nativeHotelPromptText  = 0x249
	nativeHotelSavedText   = 0x294
	nativeHotelResourceID  = 13
	nativeHotelServiceSave = 1
	nativeHotelServiceExit = 3
)

type nativeHotelUIAssets struct {
	assets    *campaign.NativeShopAssets
	portraits []dato.Frame
	slotsBox  fdother.LMI1Entry
}

func loadNativeHotelUIAssets(shared *nativeClassUIAssets) (*nativeHotelUIAssets, error) {
	if shared == nil || shared.strings == nil || shared.font == nil || len(shared.dialogue) <= 17 {
		return nil, errors.New("native hotel UI: shared facility assets unavailable")
	}
	assets, err := campaign.LoadSeparatedNativeShopAssets(separatedAssetPath(""), nativeHotelResourceID)
	if err != nil {
		return nil, err
	}
	portraits, err := loadNativeSeparatedPortrait(nativeHotelPortraitID)
	if err != nil {
		return nil, err
	}
	if len(portraits) == 0 {
		return nil, errors.New("native hotel UI: DATO 0x81 has no frames")
	}
	slotsBox, err := fdother.LoadSeparatedLoadSlotsFrame(separatedAssetPath("ui"))
	if err != nil {
		return nil, err
	}
	return &nativeHotelUIAssets{assets: assets, portraits: portraits, slotsBox: slotsBox}, nil
}

// setupNativeHotel 進酒店節點時切到原生介面；素材不齊就維持舊版面（回 false）。
func (g *Game) setupNativeHotel() bool {
	n := g.camp.Node()
	if n == nil || n.Type != "hotel" || g.nativeHotelUI == nil || g.nativeClassUI == nil {
		return false
	}
	g.nativeHotelMode = "menu"
	g.nativeHotelSlotSel = 0
	g.hotelSel = 0
	g.resetNativeShopUIPulse()
	return true
}

// composeNativeHotelStable 是背景＋對話框＋店主頭像（指定 DATO 幀）＋「有什麼事嗎？」。
func (g *Game) composeNativeHotelStable(portraitFrame int) ([]byte, bool) {
	ui, shared := g.nativeHotelUI, g.nativeClassUI
	if ui == nil || shared == nil || portraitFrame < 0 || portraitFrame >= len(ui.portraits) {
		return nil, false
	}
	frame, err := campaign.ComposeNativeChurchDialogueOverlayAt(
		ui.assets.Background, shared.dialogue, ui.portraits[portraitFrame],
		campaign.NativeFacilityPortraitOffset(nativeHotelPortraitID),
	)
	if err != nil {
		return nil, false
	}
	frame, err = campaign.ComposeNativeChurchTextAt(
		frame, shared.strings, shared.font, nativeHotelPromptText, campaign.NativeShopTextOffset,
	)
	return frame, err == nil
}

// composeNativeHotelFrame 依目前模式組出整幀；portraitFrame 是店主頭像的 DATO 幀
// （0x16559 等待期間會眨眼，重播對每一幀各出一張）。
func (g *Game) composeNativeHotelFrame(portraitFrame int) ([]byte, bool) {
	ui, shared := g.nativeHotelUI, g.nativeClassUI
	if ui == nil || shared == nil {
		return nil, false
	}
	stable, ok := g.composeNativeHotelStable(portraitFrame)
	if !ok {
		return nil, false
	}
	switch g.nativeHotelMode {
	case "menu":
		frame, err := campaign.ComposeNativeShopServiceSteadyFrame(
			stable, ui.assets, g.hotelSel, g.nativeShopUIPulse,
		)
		return frame, err == nil
	case "slots":
		slots, ok := nativeLoadSlotMetadata()
		if !ok {
			return nil, false
		}
		frame, err := campaign.ComposeNativeLoadSlotsFrame(
			stable, ui.slotsBox, shared.strings, shared.font, slots, g.nativeHotelSlotSel,
		)
		return frame, err == nil
	case "saved":
		frame, err := campaign.ComposeNativeChurchDialogueOverlayAt(
			ui.assets.Background, shared.dialogue, ui.portraits[portraitFrame],
			campaign.NativeFacilityPortraitOffset(nativeHotelPortraitID),
		)
		if err != nil {
			return nil, false
		}
		frame, err = campaign.ComposeNativeChurchTextAt(
			frame, shared.strings, shared.font, nativeHotelSavedText, campaign.NativeShopTextOffset,
		)
		return frame, err == nil
	}
	return nil, false
}

func (g *Game) drawNativeHotel(screen *ebiten.Image) bool {
	if g.nativeHotelMode == "" {
		return false
	}
	frame, ok := g.composeNativeHotelFrame(0)
	if !ok {
		return false
	}
	g.presentNativeClassFrame(screen, frame)
	return true
}

// nativeHotelInput 是酒店原生介面的輸入；與 0x2D7BD 相同：左右換圖示、enter 選、
// esc 離開（0x2FEA0）。存檔列表上下選槽、enter 寫檔、esc 回選單；「記錄儲存完畢」
// 任一鍵回選單。
type nativeHotelInput struct {
	delta                int
	up, down, enter, esc bool
}

func currentNativeHotelInput(enter bool) nativeHotelInput {
	in := nativeHotelInput{enter: enter, esc: inpututil.IsKeyJustPressed(ebiten.KeyEscape)}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		in.delta = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		in.delta = 1
	}
	in.up = inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
	in.down = inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)
	return in
}

func (g *Game) handleNativeHotelInput(in nativeHotelInput) bool {
	switch g.nativeHotelMode {
	case "menu":
		if in.delta == -1 {
			g.hotelSel = (g.hotelSel + 3) % 4
			g.resetNativeShopUIPulse()
		}
		if in.delta == 1 {
			g.hotelSel = (g.hotelSel + 1) % 4
			g.resetNativeShopUIPulse()
		}
		if in.esc {
			g.leaveHotel()
			return true
		}
		if in.enter {
			switch g.hotelSel {
			case nativeHotelServiceSave:
				g.nativeHotelMode = "slots"
				g.nativeHotelSlotSel = 0
			case nativeHotelServiceExit:
				g.leaveHotel()
			case 0:
				// 0x2FFA5 打聽消息：進本章的傳聞 story 節點（沒有就留在選單）。
				if n := g.camp.Node(); n != nil && n.Rumor != "" {
					g.nativeHotelMode = ""
					g.camp.Advance("rumor")
					g.enterNode()
				}
			default:
				g.applyHotelServiceSelection(byte(g.hotelSel))
			}
		}
		return true
	case "slots":
		if in.up && g.nativeHotelSlotSel > 0 {
			g.nativeHotelSlotSel--
		}
		if in.down && g.nativeHotelSlotSel < 3 {
			g.nativeHotelSlotSel++
		}
		if in.esc {
			g.nativeHotelMode = "menu"
			return true
		}
		if in.enter {
			g.saveGameToSlot(g.nativeHotelSlotSel)
			g.nativeHotelMode = "saved"
		}
		return true
	case "saved":
		if in.enter || in.esc {
			g.nativeHotelMode = "menu"
		}
		return true
	}
	return false
}
