package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 靜態場景圖。原始像素保留供同次 producer 身分核對。載入時只核對路徑、SHA-256 與 PNG 標頭
// 尺寸（#317 後續）；第一次用到時才解碼，同時檢查透明度條件，只有最近用過的幾張留在記憶體。
// picture 為 nil 且 path 非空，表示還沒解碼或已被釋放；path 為空的項目沒有可解碼的來源。
type modernBattleSurfaceHD struct {
	native   figani.Frame
	picture  image.Image
	hash     string
	path     string
	fullMask bool
	cache    *modernBattleSurfaceCache
}

// modernBattleSurfaceCacheSize 是同時保留的已解碼戰鬥背景數。一場戰鬥最多用到基底背景、台座、
// 三張轉場背景與目標側背景共六張；超過時釋放最久沒用的一張，下次用到再解碼。
const modernBattleSurfaceCacheSize = 12

type modernBattleSurfaceCache struct {
	mu      sync.Mutex
	decoded []*modernBattleSurfaceHD // 目前持有解碼圖的項目，最久沒用的在前
	notices *modernThemeNotices
}

func (set *modernStoryPortraitSet) battleSurfaceDecodeCache() *modernBattleSurfaceCache {
	if set.surfaceCache == nil {
		set.surfaceCache = &modernBattleSurfaceCache{notices: set.themeNotices()}
	}
	return set.surfaceCache
}

// decodedPicture 回傳解碼後的圖，必要時讀檔、核對 SHA-256、解碼並檢查透明度條件。檔案在載入後
// 不見或被改是硬性錯誤；解不開或透明度條件不符時回傳符合 errModernHDDeferredRejected 的錯誤，
// 呼叫端把這張背景當成包內沒有登錄。
func (entry *modernBattleSurfaceHD) decodedPicture() (image.Image, error) {
	c := entry.cache
	if c == nil {
		if entry.picture == nil {
			return nil, errors.New("modern HD battle surface: producer source differs")
		}
		return entry.picture, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for n, held := range c.decoded {
		if held == entry {
			c.decoded = append(append(c.decoded[:n:n], c.decoded[n+1:]...), entry)
			break
		}
	}
	if entry.picture != nil {
		return entry.picture, nil
	}
	if entry.path == "" {
		return nil, errors.New("modern HD battle surface: producer source differs")
	}
	raw, err := os.ReadFile(entry.path)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != entry.hash {
		return nil, errors.New("modern HD battle surface: PNG changed before use")
	}
	picture, _, err := image.Decode(bytes.NewReader(raw))
	if err == nil {
		err = modernBattleSurfaceCoverage(picture, entry.fullMask)
	}
	if err != nil {
		cause := fmt.Errorf("modern HD battle surface %s: %w", filepath.Base(entry.path), err)
		c.notices.reject(cause)
		return nil, fmt.Errorf("%w: %v", errModernHDDeferredRejected, cause)
	}
	entry.picture = picture
	c.decoded = append(c.decoded, entry)
	if len(c.decoded) > modernBattleSurfaceCacheSize {
		c.decoded[0].picture = nil
		c.decoded = append(c.decoded[:0:0], c.decoded[1:]...)
	}
	return picture, nil
}

// modernBattleSurfaceCoverage 是載入時延後的透明度條件：原生遮罩全滿的來源必須完全不透明，
// 其餘必須同時有完全不透明與完全透明的像素。與逐點 At().RGBA() 的判斷相同。
func modernBattleSurfaceCoverage(picture image.Image, fullMask bool) error {
	opaque, transparent, partial := false, false, false
	var pix []byte
	stride := 0
	switch p := picture.(type) {
	case *image.NRGBA:
		pix, stride = p.Pix, p.Stride
	case *image.RGBA:
		pix, stride = p.Pix, p.Stride
	}
	rect := picture.Bounds()
	if pix != nil {
		for y := 0; y < rect.Dy(); y++ {
			row := pix[y*stride : y*stride+rect.Dx()*4]
			for x := 3; x < len(row); x += 4 {
				opaque = opaque || row[x] == 0xff
				transparent = transparent || row[x] == 0
				partial = partial || row[x] != 0xff
			}
		}
	} else {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				_, _, _, alpha := picture.At(x, y).RGBA()
				opaque = opaque || alpha == 0xffff
				transparent = transparent || alpha == 0
				partial = partial || alpha != 0xffff
			}
		}
	}
	if fullMask && partial {
		return errors.New("opaque source requires opaque HD")
	}
	if !opaque || (!fullMask && !transparent) {
		return errors.New("missing coverage endpoints")
	}
	return nil
}

func loadModernBattleSurfaceHD(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	if set == nil || asset.Role != "battle_surface_hd" ||
		(asset.SourceArchive != "BG.DAT" && asset.SourceArchive != "TAI.DAT") ||
		asset.SourceResource < 0 || asset.SourceResource >= 56 {
		return errors.New("modern HD battle surface: unsupported source")
	}
	prefix := asset.SourceArchive[:len(asset.SourceArchive)-4]
	metadata := filepath.Join(separatedAssetPath("surfaces"), fmt.Sprintf("%s_%03d", prefix, asset.SourceResource), "resource.json")
	raw, err := os.ReadFile(metadata)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SourceMetadataSHA256 {
		return errors.New("modern HD battle surface: metadata sha256 mismatch")
	}
	native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), asset.SourceArchive, asset.SourceResource)
	if err != nil {
		return err
	}
	frame, err := modernHDBattleSurfaceFrame(native)
	if err != nil {
		return err
	}
	return loadModernBattleSurfaceHDWithNative(set, asset, root, frame)
}

func loadModernBattleSurfaceHDWithNative(set *modernStoryPortraitSet, asset modernThemeAsset, root string, frame figani.Frame) error {
	if set == nil || asset.Role != "battle_surface_hd" ||
		(asset.SourceArchive != "BG.DAT" && asset.SourceArchive != "TAI.DAT") || asset.SourceResource < 0 || asset.SourceResource >= 56 ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.ConsumerContract != "native_battle_surface_rgba_hd6_v1" || asset.AlphaContract != "coverage" ||
		asset.LayoutPolicy != "native_surface_hd6" || asset.Width != frame.Width*6 || asset.Height != frame.Height*6 ||
		frame.Width <= 0 || frame.Width > 320 || frame.Height <= 0 || frame.Height > 200 ||
		len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) ||
		len(asset.NativeFrames) != 1 || asset.NativeFrames[0] != (modernBattleHDDescriptor{Width: frame.Width, Height: frame.Height}) ||
		filepath.Base(asset.File) != asset.File || filepath.Ext(asset.File) != ".png" {
		return errors.New("modern HD battle surface: invalid native contract")
	}
	fullMask, visible := true, false
	for _, value := range frame.Mask {
		if value != 0 && value != 255 {
			return errors.New("modern HD battle surface: invalid native mask")
		}
		fullMask = fullMask && value == 255
		visible = visible || value != 0
	}
	if !visible || set.battleSurfaces[asset.SourceArchive][asset.SourceResource] != nil {
		return errors.New("modern HD battle surface: empty or duplicate native source")
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SHA256 {
		return errors.New("modern HD battle surface: PNG sha256 mismatch")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width != asset.Width || config.Height != asset.Height {
		return errors.New("modern HD battle surface: PNG geometry mismatch")
	}
	// 解碼與透明度檢查延到第一次使用（decodedPicture）。
	// 唯一的發布點。失敗不建立 map 或改已載入圖源。
	if set.battleSurfaces == nil {
		set.battleSurfaces = make(map[string]map[int]*modernBattleSurfaceHD)
	}
	if set.battleSurfaces[asset.SourceArchive] == nil {
		set.battleSurfaces[asset.SourceArchive] = make(map[int]*modernBattleSurfaceHD)
	}
	frame.Pixels = append([]byte(nil), frame.Pixels...)
	frame.Mask = append([]byte(nil), frame.Mask...)
	set.battleSurfaces[asset.SourceArchive][asset.SourceResource] = &modernBattleSurfaceHD{native: frame, hash: asset.SHA256,
		path: filepath.Join(root, asset.File), fullMask: fullMask, cache: set.battleSurfaceDecodeCache()}
	set.hasHD = true
	return nil
}

func (set *modernStoryPortraitSet) battleSurfaceHD(source modernHDBattleSource) (image.Image, string, error) {
	if source.Index != 0 || source.Resource < 0 || source.Resource >= 56 {
		return nil, "", errors.New("modern HD battle surface: invalid single-frame source")
	}
	if set == nil {
		return nil, "", nil
	}
	entry := set.battleSurfaces[source.Archive][source.Resource]
	if entry == nil {
		return nil, "", nil
	}
	f := source.Frame
	if entry.hash == "" || f.Width != entry.native.Width || f.Height != entry.native.Height ||
		!bytes.Equal(f.Pixels, entry.native.Pixels) || !bytes.Equal(f.Mask, entry.native.Mask) {
		return nil, "", errors.New("modern HD battle surface: producer source differs")
	}
	picture, err := entry.decodedPicture()
	if errors.Is(err, errModernHDDeferredRejected) {
		// 延後的檢查沒通過：這張背景退回原版（等同包內沒有登錄它），呼叫端的交易照常。
		delete(set.battleSurfaces[source.Archive], source.Resource)
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if picture == nil || picture.Bounds() != image.Rect(0, 0, f.Width*6, f.Height*6) {
		return nil, "", errors.New("modern HD battle surface: producer source differs")
	}
	return picture, entry.hash, nil
}

func (set *modernStoryPortraitSet) validatePhysicalScenery(scene *nativePhysicalScene) error {
	sources, err := modernPhysicalBaseScenery(scene)
	if err != nil {
		return err
	}
	if scene != nil && scene.selection.HasSeparateBackgrounds {
		transitions, err := modernBattleTransitionSources()
		if err != nil {
			return err
		}
		sources = append(sources, transitions...)
		for _, target := range []bool{false, true} {
			frame, index := scene.backgroundFrame, scene.selection.BaseBG
			if target {
				frame, index = scene.targetBackground, modernPhysicalTargetBGIndex(scene)
			}
			more, err := modernPhysicalBackdropScenery(scene, frame, index, true)
			if err != nil {
				return err
			}
			sources = append(sources, more...)
		}
	}
	for _, source := range sources {
		if _, _, err := set.battleSurfaceHD(source); err != nil {
			return err
		}
	}
	return nil
}

func modernBattleTransitionSources() ([]modernHDBattleSource, error) {
	var sources []modernHDBattleSource
	for index := 0; index < 3; index++ {
		native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", index)
		if err != nil {
			return nil, err
		}
		frame, err := modernHDBattleSurfaceFrame(native)
		if err != nil {
			return nil, err
		}
		frame.X, frame.Y = 0, 50
		sources = append(sources, modernHDBattleSource{Archive: "BG.DAT", Resource: index, Frame: frame, Fill: -1})
	}
	return sources, nil
}

func modernPhysicalBaseScenery(scene *nativePhysicalScene) ([]modernHDBattleSource, error) {
	if scene == nil || scene.selection.HasSeparateBackgrounds {
		return nil, nil
	}
	return modernPhysicalScenery(scene, true)
}

func modernPhysicalTargetBGIndex(scene *nativePhysicalScene) byte {
	if scene.rawSide == 0 {
		return scene.selection.SecondaryBG
	}
	return scene.selection.PrimaryBG
}

func modernPhysicalPlatformSource(scene *nativePhysicalScene) (modernHDBattleSource, error) {
	return modernBattlePlatformSource(scene.platformFrame, int(scene.selection.TAI))
}

func modernBattlePlatformSource(platform fdother.Frame, resource int) (modernHDBattleSource, error) {
	frame, err := modernHDBattleSurfaceFrame(platform)
	if err != nil {
		return modernHDBattleSource{}, err
	}
	frame.X, frame.Y = 164, 157
	rect := image.Rect(frame.X, frame.Y, frame.X+frame.Width, frame.Y+frame.Height)
	if !rect.In(image.Rect(0, 0, 320, 200)) || rect.Overlaps(image.Rect(0, 154, 149, 196)) || rect.Overlaps(image.Rect(171, 4, 320, 46)) || rect.Overlaps(image.Rect(0, 50, 320, 150)) {
		return modernHDBattleSource{}, errors.New("modern HD physical platform overlaps a native base component")
	}
	return modernHDBattleSource{Archive: "TAI.DAT", Resource: resource, Frame: frame, Fill: -1}, nil
}

func modernPhysicalScenery(scene *nativePhysicalScene, platform bool) ([]modernHDBattleSource, error) {
	if scene == nil {
		return nil, nil
	}
	return modernPhysicalBackdropScenery(scene, scene.backgroundFrame, scene.selection.BaseBG, platform)
}

func modernBattleBackdropGeometry(frame fdother.Frame) bool {
	return (frame.Width == 320 && (frame.Height == 100 || frame.Height == 101)) ||
		(frame.Width == 318 && frame.Height == 100)
}

func modernPhysicalBackdropScenery(scene *nativePhysicalScene, background fdother.Frame, index byte, platform bool) ([]modernHDBattleSource, error) {
	// 三種原框均落於y50..151，不與面板y4..46及y154..196交會。
	// 固定來源及READY契約見61「其餘可見戰鬥背景與台座HD」。
	if !modernBattleBackdropGeometry(background) {
		return nil, nil
	}
	frame, err := modernHDBattleSurfaceFrame(background)
	if err != nil {
		return nil, err
	}
	frame.X, frame.Y = 0, 50
	sources := []modernHDBattleSource{{Archive: "BG.DAT", Resource: int(index), Frame: frame, Fill: -1}}
	if platform {
		p, err := modernPhysicalPlatformSource(scene)
		if err != nil {
			return nil, err
		}
		sources = append(sources, p)
	}
	return sources, nil
}

// 只還原已證實由memset供應、且不與BG／面板交會的台座底層。
// 不改job的原生base；原生及HD-native fallback分別重組驗證。
func bindModernPhysicalScenery(scene *nativePhysicalScene, sources *modernHDBattleSources) (err error) {
	defer func() {
		if err == nil {
			bindModernPhysicalPanelText(scene, sources)
		}
	}()
	scenery, err := modernPhysicalBaseScenery(scene)
	if err != nil {
		return err
	}
	if sources == nil || len(scenery) == 0 {
		return nil
	}
	return bindModernPhysicalBaseScenery(sources, scenery)
}

func bindModernPhysicalBaseScenery(sources *modernHDBattleSources, scenery []modernHDBattleSource) error {
	if sources == nil || len(scenery) == 0 {
		return nil
	}
	if len(sources.Base) != 64000 {
		return errors.New("modern HD physical clean base unavailable")
	}
	base := append([]byte(nil), sources.Base...)
	for _, source := range scenery {
		if source.Archive != "TAI.DAT" {
			continue
		}
		platform := source.Frame
		for y := platform.Y; y < platform.Y+platform.Height; y++ {
			clear(base[y*320+platform.X : y*320+platform.X+platform.Width])
		}
	}
	for _, source := range sources.BaseScenery {
		if source.PanelText != nil {
			scenery = append(scenery, source)
		}
	}
	sources.Base, sources.BaseScenery = base, scenery
	return nil
}
