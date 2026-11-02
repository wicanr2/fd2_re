package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var modernServiceMetadata = map[int]string{
	12: "9a80b9a9d23b0d89d242c785900fbcf1a7048f540fbc609ae125d37a05a6a775",
	13: "4cb14409dbd9d2eb25fdf4d6a50f5f8a19e1313094e3d16ac3f71fce694491c5",
	14: "fabec30f3791617f5449239ac235bbeb69cf0b8ba51ad1ff9cc2e357c2728a12",
	29: "732b27ddb48a416d4efa0ff61355f446bacc10083726792b813370e31f6bb096",
	63: "c461d881cdfef918c09bcbdd7940d556b74e554a543f18509981b2b18831c838",
}

type modernHDServiceIcon struct {
	native  modernLMIHDDescriptor
	picture *image.RGBA
	digest  [32]byte
	gpu     *ebiten.Image
}
type modernHDServiceIconPlacement struct {
	icon   *modernHDServiceIcon
	origin image.Point
	clip   image.Rectangle
}

func loadModernHDServiceIcon(set *modernStoryPortraitSet, a modernThemeAsset, root string) error {
	d := a.NativeRawCell
	metadata, known := modernServiceMetadata[a.SourceResource]
	if set == nil || !known || a.Role != "native_ui_icon_hd" || a.SourceArchive != "FDOTHER.DAT" ||
		a.SourceMetadataSHA256 != metadata || a.ConsumerContract != "native_raw_ui_icon_hd6_v1" ||
		a.LayoutPolicy != "native_raw_cell_rect_hd6" || a.AlphaContract != "opaque" ||
		a.Status != "runtime_candidate" && a.Status != "runtime_ready" || d.Index < 3 || d.Index > 10 ||
		d.Width != 24 || d.Height != 20 || a.Width != 144 || a.Height != 120 ||
		len(a.SourceFiles) != 1 || len(a.SourceSHA256) != 1 || a.SourceFiles[0] != a.MasterFile {
		return errors.New("modern service icon: unproved source contract")
	}
	key := [2]int{a.SourceResource, d.Index}
	if set.uiServiceIcons[key] != nil {
		return errors.New("modern service icon: duplicate source")
	}
	if filepath.Base(a.File) != a.File || filepath.Ext(a.File) != ".png" ||
		filepath.Base(a.MasterFile) != a.MasterFile || filepath.Ext(a.MasterFile) != ".png" {
		return errors.New("modern service icon: unsafe path")
	}
	if h, err := hex.DecodeString(a.SourceSHA256[0]); err != nil || len(h) != 32 {
		return errors.New("modern service icon: invalid master SHA")
	}
	folder := fmt.Sprintf("shop/FDOTHER_%03d", a.SourceResource)
	if a.SourceResource == 14 {
		folder = "ui/fdother_014_church"
	}
	directory := separatedAssetPath(folder)
	data, err := os.ReadFile(filepath.Join(directory, "resource.json"))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != metadata {
		return errors.New("modern service icon: original metadata differs")
	}
	data, err = os.ReadFile(filepath.Join(directory, fmt.Sprintf("entry_%03d/frame.png", d.Index)))
	if err != nil {
		return err
	}
	original, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	indexed, ok := original.(*image.Paletted)
	if !ok || indexed.Bounds() != image.Rect(0, 0, 24, 20) {
		return errors.New("modern service icon: original indexed shape differs")
	}
	pixels := make([]byte, 480)
	for y := 0; y < 20; y++ {
		copy(pixels[y*24:(y+1)*24], indexed.Pix[y*indexed.Stride:y*indexed.Stride+24])
	}
	if d != modernUIRawDescription(d.Index, fdother.RawCell{Width: 24, Height: 20, Pixels: pixels}) {
		return errors.New("modern service icon: original raw pixels differ")
	}
	data, err = os.ReadFile(filepath.Join(root, a.File))
	if err != nil {
		return err
	}
	sum = sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != a.SHA256 {
		return errors.New("modern service icon: PNG SHA differs")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != 144 || config.Height != 120 {
		return errors.New("modern service icon: PNG geometry differs")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	picture := image.NewRGBA(image.Rect(0, 0, 144, 120))
	draw.Draw(picture, picture.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	for at := 3; at < len(picture.Pix); at += 4 {
		if picture.Pix[at] != 255 {
			return errors.New("modern service icon: PNG is not opaque")
		}
	}
	if set.uiServiceIcons == nil {
		set.uiServiceIcons = make(map[[2]int]*modernHDServiceIcon)
	}
	set.uiServiceIcons[key] = &modernHDServiceIcon{native: d, picture: picture, digest: sha256.Sum256(picture.Pix)}
	return nil
}

func validateModernHDServiceIconGroup(set *modernStoryPortraitSet) error {
	if len(set.uiServiceIcons) == 0 {
		return nil
	}
	if len(set.uiServiceIcons) != 40 {
		return errors.New("modern service icon: incomplete40-source group")
	}
	for resource := range modernServiceMetadata {
		for index := 3; index <= 10; index++ {
			if set.uiServiceIcons[[2]int{resource, index}] == nil {
				return errors.New("modern service icon: missing required source")
			}
		}
	}
	return nil
}

func (g *Game) modernServiceMenuSource() (resource, selected, phase int, cells [4][2]fdother.RawCell, ok bool) {
	if g.camp == nil || g.camp.Node() == nil || g.nativeClassUI == nil ||
		g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil {
		return
	}
	switch g.camp.Node().Type {
	case "shop":
		if g.nativeShopMode != "menu" {
			return
		}
		assets, _, _, valid := g.nativeShopState()
		if !valid || assets == nil {
			return
		}
		resource, selected, phase, cells = assets.ResourceID, g.nativeShopServiceSel, g.nativeShopUIPulse, assets.ServiceCells
	case "hotel":
		if g.nativeHotelMode != "menu" || g.nativeHotelUI == nil || g.nativeHotelUI.assets == nil {
			return
		}
		assets := g.nativeHotelUI.assets
		resource, selected, phase, cells = assets.ResourceID, g.hotelSel, g.nativeShopUIPulse, assets.ServiceCells
		if resource != 13 {
			return
		}
	case "church":
		if g.churchMode != "menu" || len(g.nativeClassUI.entries) <= 10 {
			return
		}
		resource, selected, phase = 14, g.churchSel, g.nativeChurchUIPulse
		for option := 0; option < 4; option++ {
			for pulse := 0; pulse < 2; pulse++ {
				c := g.nativeClassUI.entries[3+2*option+pulse]
				cells[option][pulse] = fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}
			}
		}
	default:
		return
	}
	if _, known := modernServiceMetadata[resource]; !known || selected < 0 || selected > 3 || phase < 0 || phase > 3 {
		return
	}
	ok = true
	return
}

func (g *Game) appendModernHDServiceIcons(frame []byte) {
	if g.modernHDText == nil {
		return
	}
	g.modernHDText.serviceIcons = nil
	set := g.interfaceAssetTheme()
	resource, selected, phase, cells, ok := g.modernServiceMenuSource()
	if !ok || set == nil || len(set.uiServiceIcons) != 40 || len(frame) != 64000 || len(g.nativeClassUI.palette) != 256 ||
		g.nativeClassUI.palette[74] == nil || color.RGBAModel.Convert(g.nativeClassUI.palette[74]).(color.RGBA) != (color.RGBA{40, 69, 138, 255}) {
		return
	}
	var placements []modernHDServiceIconPlacement
	for option := 0; option < 4; option++ {
		pulse := 0
		if option == selected {
			pulse = phase / 2
		}
		index := 3 + 2*option + pulse
		cell := cells[option][pulse]
		icon := set.uiServiceIcons[[2]int{resource, index}]
		if cell.Width != 24 || cell.Height != 20 || len(cell.Pixels) != 480 || icon == nil || icon.picture == nil ||
			icon.native != modernUIRawDescription(index, cell) || icon.picture.Bounds() != image.Rect(0, 0, 144, 120) ||
			sha256.Sum256(icon.picture.Pix) != icon.digest {
			return
		}
		x0, y0 := 201+26*option, 169
		for y := 0; y < 20; y++ {
			for x := 0; x < 24; x++ {
				value := cell.Pixels[y*24+x]
				if value == 0 {
					value = 74
				}
				if frame[(y0+y)*320+x0+x] != value {
					return
				}
			}
		}
		placements = append(placements, modernHDServiceIconPlacement{icon: icon, origin: image.Pt(x0, y0)})
	}
	g.modernHDText.serviceIcons = placements
}

func (p *modernHDPresentation) drawModernHDServiceIcons(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	frame := p.Game.modernHDText
	if frame == nil {
		return
	}
	for _, placement := range frame.serviceIcons {
		icon := placement.icon
		if icon.gpu == nil {
			icon.gpu = ebiten.NewImageFromImage(icon.picture)
		}
		options := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		picture := icon.gpu
		at := placement.origin.Mul(6)
		if !placement.clip.Empty() {
			rect := image.Rectangle{Min: at, Max: at.Add(icon.picture.Bounds().Size())}
			visible := rect.Intersect(image.Rectangle{Min: placement.clip.Min.Mul(6), Max: placement.clip.Max.Mul(6)})
			if visible.Empty() {
				continue
			}
			picture = picture.SubImage(visible.Sub(rect.Min)).(*ebiten.Image)
			at = visible.Min
		}
		options.GeoM.Scale(1.0/3, 1.0/3)
		options.GeoM.Translate(float64(at.X)/3, float64(at.Y)/3)
		options.GeoM.Concat(geometry)
		value := float32(math.Max(0, math.Min(1, brightness)))
		options.ColorScale.Scale(value, value, value, 1)
		screen.DrawImage(picture, options)
	}
}
