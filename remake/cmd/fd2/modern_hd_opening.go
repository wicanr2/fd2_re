package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const modernOpeningRole = "opening_presentation_hd_set"
const modernOpeningContract = "opening_native_schedule_rgba_hd6_v1"

type modernOpeningKey struct {
	archive  string
	resource int
}

type modernOpeningBank struct {
	encoded [][]byte
	hashes  []string
	// 延後到第一次畫那一格才做的檢查：解碼後的尺寸與完全不透明。verified 依 PNG SHA 記錄已通過
	// 的圖；rejected 非 nil 表示這組已退回原版。
	verified map[string]bool
	rejected error
	notices  *modernThemeNotices
}

// Compressed pictures are preflighted before publication. Only the current
// full-resolution picture owns a GPU image; changing a frame releases it.
type modernOpeningHD struct {
	banks    map[modernOpeningKey]*modernOpeningBank
	gpu      *ebiten.Image
	gpuKey   string
	viewport *ebiten.Image
}

func modernOpeningShape(key modernOpeningKey) (width, height, count int) {
	width, height = 1920, 1200
	if key.archive == "ANI.DAT" {
		count = map[int]int{1: 51, 3: 28, 4: 12, 5: 35, 6: 12, 7: 17, 8: 12}[key.resource]
	} else if key.archive == "FDOTHER.DAT" {
		switch key.resource {
		case 7, 74, 75, 100:
			count = 1
		case 69:
			height, count = 4410, 2
		}
	}
	return
}

func modernOpeningSourcePaths(key modernOpeningKey) []string {
	if key.archive == "ANI.DAT" {
		return []string{fmt.Sprintf("animations/ANI_%03d/animation.json", key.resource)}
	}
	if key.resource == 69 {
		var names []string
		for i := 69; i <= 73; i++ {
			names = append(names, fmt.Sprintf("surfaces/FDOTHER_%03d/resource.json", i))
		}
		return append(names, "palette/fdother_101.json", "palette/fdother_102.json")
	}
	palette := map[int]int{7: 8, 74: 76, 75: 76, 100: 99}[key.resource]
	name := fmt.Sprintf("surfaces/FDOTHER_%03d/resource.json", key.resource)
	if key.resource == 7 {
		name = "surfaces/FDOTHER_007/entry_000/resource.json"
	}
	return []string{name, fmt.Sprintf("palette/fdother_%03d.json", palette)}
}

func modernOpeningSourceDigest(key modernOpeningKey) (string, error) {
	var text strings.Builder
	for _, name := range modernOpeningSourcePaths(key) {
		raw, err := os.ReadFile(separatedAssetPath(name))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&text, "%s %x\n", name, sha256.Sum256(raw))
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text.String()))), nil
}

func openingOpaque(picture image.Image) bool {
	switch im := picture.(type) {
	case *image.NRGBA:
		for i := 3; i < len(im.Pix); i += 4 {
			if im.Pix[i] != 255 {
				return false
			}
		}
		return true
	case *image.RGBA:
		return im.Opaque()
	default:
		b := picture.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := picture.At(x, y).RGBA()
				if a != 65535 {
					return false
				}
			}
		}
		return true
	}
}

func loadModernOpeningHDAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	key := modernOpeningKey{asset.SourceArchive, asset.SourceResource}
	w, h, count := modernOpeningShape(key)
	if set == nil || count == 0 || asset.Role != modernOpeningRole ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.ConsumerContract != modernOpeningContract || asset.LayoutPolicy != "full_native_canvas_hd6" ||
		asset.AlphaContract != "opaque" || asset.Width != w || asset.Height != h || asset.FrameCount != count ||
		len(asset.Files) != count || len(asset.FrameSHA256) != count ||
		len(asset.SourceFiles) == 0 || len(asset.SourceFiles) != len(asset.SourceSHA256) {
		return errors.New("modern opening: invalid bank contract")
	}
	if set.openingHD == nil {
		set.openingHD = &modernOpeningHD{banks: map[modernOpeningKey]*modernOpeningBank{}}
	}
	if set.openingHD.banks[key] != nil {
		return errors.New("modern opening: duplicate source")
	}
	digest, err := modernOpeningSourceDigest(key)
	if err != nil {
		return err
	}
	if digest != asset.SourceMetadataSHA256 {
		return errors.New("modern opening: original source metadata differs")
	}
	seen := map[string]bool{}
	master, plans := false, 0
	for i, name := range asset.SourceFiles {
		if !skyKeySafeFile(name) || seen[name] || (filepath.Ext(name) != ".json" && filepath.Ext(name) != ".png") {
			return errors.New("modern opening: unsafe or duplicate provenance")
		}
		seen[name], master = true, master || name == asset.MasterFile
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if skyKeySHA(raw) != asset.SourceSHA256[i] {
			return errors.New("modern opening: provenance hash differs")
		}
		if filepath.Ext(name) == ".json" {
			plans++
			var plan struct {
				Schema    int    `json:"schema_version"`
				Kind      string `json:"kind"`
				Archive   string `json:"source_archive"`
				Resource  int    `json:"source_resource"`
				SourceSHA string `json:"source_metadata_sha256"`
				Frames    []any  `json:"frames"`
			}
			if err := json.Unmarshal(raw, &plan); err != nil {
				return err
			}
			if plan.Schema != 1 || plan.Kind != "opening_hd_art_derivation" || plan.Archive != key.archive || plan.Resource != key.resource || plan.SourceSHA != digest || len(plan.Frames) != count {
				return errors.New("modern opening: derivation plan differs")
			}
		}
	}
	if !master || plans != 1 {
		return errors.New("modern opening: master or unique plan absent")
	}
	bank := &modernOpeningBank{encoded: make([][]byte, count), hashes: append([]string(nil), asset.FrameSHA256...)}
	seen = map[string]bool{}
	shared := map[string][]byte{}
	for i, name := range asset.Files {
		if !skyKeySafeFile(name) || filepath.Ext(name) != ".png" || seen[name] {
			return errors.New("modern opening: unsafe or duplicate frame path")
		}
		seen[name] = true
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if skyKeySHA(raw) != bank.hashes[i] {
			return errors.New("modern opening: frame hash differs")
		}
		if prior := shared[bank.hashes[i]]; prior != nil {
			bank.encoded[i] = prior
			continue
		}
		// 載入時只看 PNG 標頭尺寸；不透明度要解碼像素才看得到，延到第一次畫這一格時檢查（#317 後續）。
		config, err := png.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		if config.Width != w || config.Height != h {
			return errors.New("modern opening: frame geometry or opacity differs")
		}
		bank.encoded[i], shared[bank.hashes[i]] = raw, raw
	}
	bank.notices = set.themeNotices()
	set.openingHD.banks[key] = bank
	set.hasHD = true
	return nil
}

func validateModernOpeningHDGroup(set *modernStoryPortraitSet) error {
	if set.openingHD == nil {
		return nil
	}
	if len(set.openingHD.banks) != 12 || set.skyKeyANI == nil {
		return errors.New("modern opening: scene group or shared ANI0 incomplete")
	}
	return nil
}

func (h *modernOpeningHD) frame(archive string, resource, index int) (*ebiten.Image, error) {
	bank := h.banks[modernOpeningKey{archive, resource}]
	if bank == nil || index < 0 || index >= len(bank.encoded) {
		return nil, errors.New("modern opening: frame outside admitted bank")
	}
	key := bank.hashes[index]
	if bank.rejected != nil {
		return nil, bank.rejected
	}
	if h.gpu != nil && h.gpuKey == key {
		return h.gpu, nil
	}
	picture, err := bank.decode(modernOpeningKey{archive, resource}, index)
	if err != nil {
		return nil, err
	}
	h.replaceGPU(key, picture)
	return h.gpu, nil
}

// decode 解碼一格，第一次用到時順便檢查解碼後的尺寸與完全不透明（載入時延後的檢查）。沒通過時
// 這組退回原版並提示一次，回傳的錯誤符合 errModernHDDeferredRejected。
func (bank *modernOpeningBank) decode(key modernOpeningKey, index int) (image.Image, error) {
	picture, err := png.Decode(bytes.NewReader(bank.encoded[index]))
	hash := bank.hashes[index]
	if err == nil && bank.verified[hash] {
		return picture, nil
	}
	w, h, _ := modernOpeningShape(key)
	if err == nil && (picture.Bounds() != image.Rect(0, 0, w, h) || !openingOpaque(picture)) {
		err = errors.New("frame geometry or opacity differs")
	}
	if err != nil {
		cause := fmt.Errorf("modern opening: %s #%d frame %d: %w", key.archive, key.resource, index, err)
		bank.rejected = fmt.Errorf("%w: %v", errModernHDDeferredRejected, cause)
		bank.notices.reject(cause)
		return nil, bank.rejected
	}
	if bank.verified == nil {
		bank.verified = map[string]bool{}
	}
	bank.verified[hash] = true
	return picture, nil
}

// modernOpeningSource 回傳目前開場畫面要用的 HD 圖組（與 drawModernHDOpening 的選擇相同）。
func (g *Game) modernOpeningSource() (archive string, resource int) {
	archive, resource = "FDOTHER.DAT", 7
	if g.titlePhase != "cutscene" || g.cutIdx < 0 || g.cutIdx >= len(cutScript) {
		return
	}
	switch step := cutScript[g.cutIdx]; step.kind {
	case "publisher":
		resource = 74
	case "static":
		resource = []int{100, 75}[step.res]
	case "scroll", "hold", "redfade":
		resource = 69
	case "afm":
		archive, resource = "ANI.DAT", step.res
	}
	return
}

// modernOpeningRejected 判斷目前這一幕的 HD 圖組是否已因延後檢查退回原版。
func (g *Game) modernOpeningRejected() bool {
	set := g.modernStoryPortraits
	archive, resource := g.modernOpeningSource()
	if archive == "ANI.DAT" && resource == 0 {
		return set.skyKeyANI.rejectedErr() != nil
	}
	bank := set.openingHD.banks[modernOpeningKey{archive, resource}]
	return bank != nil && bank.rejected != nil
}

func (h *modernOpeningHD) replaceGPU(key string, picture image.Image) {
	if h.gpu != nil {
		h.gpu.Dispose()
	}
	h.gpu, h.gpuKey = ebiten.NewImageFromImage(picture), key
}

func (g *Game) modernOpeningEnabled() bool {
	return g.modernStoryPortraits != nil && g.modernStoryPortraits.openingHD != nil &&
		g.titleAssets != nil && !g.helpVisible && !g.startupBlocked &&
		(g.titlePhase == "cutscene" || g.titlePhase == "menu") && !g.modernOpeningRejected()
}

func (p *modernHDPresentation) drawModernHDOpening(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	g := p.Game
	if !g.modernOpeningEnabled() {
		return
	}
	h := g.modernStoryPortraits.openingHD
	archive, resource, frame := "FDOTHER.DAT", 7, 0
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	if g.titlePhase == "cutscene" {
		if g.cutIdx < 0 || g.cutIdx >= len(cutScript) {
			return
		}
		step := cutScript[g.cutIdx]
		switch step.kind {
		case "publisher":
			resource = 74
			b := titlePublisherBrightness(g.cutTick)
			op.ColorScale.Scale(b, b, b, 1)
		case "static":
			resource = []int{100, 75}[step.res]
		case "scroll", "hold":
			resource = 69
			if step.kind == "scroll" && titleScrollUsesPulsePalette(g.scrollY) {
				frame = 1
			}
			op.GeoM.Translate(0, -g.scrollY*6)
		case "redfade":
			resource = 69
			factor := float64(40-titlePalettePhase(g.cutTick)) / 40
			titlePaletteBlend(op, factor, color.RGBA{R: 255, A: 255})
		case "redhold":
			// The ordinary title renderer owns this original full red flash.
			return
		case "titlefade":
			factor := float64(titlePalettePhase(g.cutTick)) / 40
			titlePaletteBlend(op, factor, color.RGBA{R: 0xe3, G: 0xf3, B: 255, A: 255})
		case "afm":
			archive, resource, frame = "ANI.DAT", step.res, g.cutFrame
		default:
			return
		}
	}
	var picture *ebiten.Image
	var err error
	if archive == "ANI.DAT" && resource == 0 {
		key := fmt.Sprintf("shared-ANI0-%d", frame)
		if h.gpu == nil || h.gpuKey != key {
			var decoded *image.RGBA
			decoded, err = g.modernStoryPortraits.skyKeyANI.picture(frame)
			if err == nil {
				h.replaceGPU(key, decoded)
			}
		}
		picture = h.gpu
	} else {
		picture, err = h.frame(archive, resource, frame)
	}
	if errors.Is(err, errModernHDDeferredRejected) {
		return // 這組已退回原版；下一幀起 modernOpeningEnabled 為假，改畫原版
	}
	if err != nil {
		g.startupBlocked, g.loadErr = true, err.Error()
		return
	}
	// A bounded source plane clips the long scroll before scaling to the
	// final screen. Posters and the floating toolbar retain their own areas.
	if h.viewport == nil {
		h.viewport = ebiten.NewImage(modernHDWidth, modernHDHeight)
	}
	h.viewport.Clear()
	h.viewport.DrawImage(picture, op)
	final := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	final.GeoM.Scale(1.0/3, 1.0/3)
	final.GeoM.Concat(geometry)
	screen.DrawImage(h.viewport, final)
}
