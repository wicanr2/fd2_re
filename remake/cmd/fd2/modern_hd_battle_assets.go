package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type modernBattleHDDescriptor struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	Delay    int  `json:"delay_native"`
	RawByte4 byte `json:"raw_byte_4"`
	RawByte5 byte `json:"raw_byte_5"`
	RawByte7 byte `json:"raw_byte_7"`
}

type modernBattleHDResource struct {
	Frames           []image.Image
	paths            []string
	hashes           []string
	sizes            []image.Rectangle
	hdRects          []image.Rectangle
	visualScale      modernHDVisualScale
	flameEmberSprite image.Image
	// coverage 是載入時由原生 descriptor 決定、延到第一次預備時才對解碼像素檢查的透明度條件；
	// nil 表示這筆資源不是由 catalog 載入器建立，沒有延後的條件。verified 記錄已通過的畫格
	// （條件只取決於已核對 SHA-256 的 PNG 位元組，所以通過一次即可）。
	coverage []modernBattleHDCoverageRule
	verified []bool
	notices  *modernThemeNotices
}

// modernBattleHDCoverageRule 是一個畫格的透明度契約：原生全空白的畫格不得有墨跡；其餘畫格
// 必須有不透明核心，且除非原生遮罩全滿，也必須有透明背景。
type modernBattleHDCoverageRule struct {
	nativeEmpty bool
	fullMask    bool
}

func (rule modernBattleHDCoverageRule) check(coverage modernHDAlphaCoverage) error {
	if rule.nativeEmpty && coverage.visible {
		return errors.New("adds ink to empty native frame")
	}
	if !rule.nativeEmpty && ((!coverage.transparent && !rule.fullMask) || !coverage.opaque) {
		return errors.New("lacks transparent background or opaque core")
	}
	return nil
}

// HD coordinates are independent of the immutable native FIGANI descriptor.
type modernBattleHDRectangle struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (r *modernBattleHDRectangle) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if len(fields) != 4 {
		return errors.New("modern HD battle: HD rectangle requires four integer fields")
	}
	var value modernBattleHDRectangle
	for key, target := range map[string]*int{"x": &value.X, "y": &value.Y, "width": &value.Width, "height": &value.Height} {
		data, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return errors.New("modern HD battle: incomplete HD rectangle")
		}
		if err := json.Unmarshal(data, target); err != nil {
			return err
		}
	}
	*r = value
	return nil
}

func (r modernBattleHDRectangle) valid() bool {
	return r.Width > 0 && r.Width <= modernHDWidth && r.Height > 0 && r.Height <= modernHDHeight &&
		r.X >= 0 && r.Y >= 0 && r.X <= modernHDWidth-r.Width && r.Y <= modernHDHeight-r.Height
}

func loadModernBattleHDAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	prefix, _, _, _, ok := modernBattleHDContract(asset)
	if !ok {
		return errors.New("modern HD battle: invalid source archive or resource")
	}
	metadataPath := filepath.Join(separatedAssetPath("animations"), fmt.Sprintf("%s_%03d", prefix, asset.SourceResource), "animation.json")
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("modern HD battle source metadata: %w", err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != asset.SourceMetadataSHA256 {
		return errors.New("modern HD battle: source metadata sha256 mismatch")
	}
	native, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), asset.SourceArchive, asset.SourceResource)
	if err != nil {
		return err
	}
	return loadModernBattleHDAssetWithNative(set, asset, root, native)
}

func modernBattleHDContract(asset modernThemeAsset) (prefix, consumer, cycle, layout string, ok bool) {
	if asset.Role == "battle_effect_hd_set" && asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource == 44 {
		return "FDOTHER", "native_fdother_command9_ai_rgba_hd6_v1", "native_command9_ai_frames", "native_descriptor_hd6", true
	}
	if asset.Role == "battle_animation_hd_set" && asset.SourceArchive == "FIGANI.DAT" && asset.SourceResource >= 0 && asset.SourceResource < 409 {
		if asset.ConsumerContract == "native_figani_rgba_hd6_v2" {
			return "FIGANI", "native_figani_rgba_hd6_v2", "native_descriptor_key_poses", "explicit_hd_frame_rects_v2", true
		}
		return "FIGANI", "native_figani_rgba_hd6_v1", "native_descriptor_key_poses", "contain_native_descriptor_hd6_candidate", true
	}
	if asset.Role == "battle_effect_hd_set" && asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource >= 65 && asset.SourceResource <= 68 {
		return "FDOTHER", "native_fdother_compound_rgba_hd6_v1", "native_compound_frames", "native_descriptor_hd6", true
	}
	if asset.Role == "battle_effect_hd_set" && asset.SourceArchive == "FDOTHER.DAT" && modernHDOrdinaryEffectResource(asset.SourceResource) {
		return "FDOTHER", "native_fdother_ordinary_rgba_hd6_v1", "native_ordinary_frames", "native_descriptor_hd6", true
	}
	return "", "", "", "", false
}

func modernHDOrdinaryEffectResource(resource int) bool {
	switch resource {
	case 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 30, 32, 33, 37, 38, 39, 43:
		return true
	}
	return false
}

// 所有候選圖片先在區域變數完整驗證；失敗不修改已載入的主題集合。
func loadModernBattleHDAssetWithNative(set *modernStoryPortraitSet, asset modernThemeAsset, root string, native *figani.Animation) error {
	_, consumer, cycle, layout, valid := modernBattleHDContract(asset)
	if !valid || !asset.VisualScalePercent.valid() || (asset.VisualScalePercent != 0 && asset.Role != "battle_animation_hd_set") || set == nil || native == nil || len(native.Frames) == 0 ||
		asset.ConsumerContract != consumer || asset.AlphaContract != "coverage" ||
		asset.Width != modernHDWidth || asset.Height != modernHDHeight ||
		asset.CyclePolicy != cycle || asset.LayoutPolicy != layout ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.FrameCount != len(native.Frames) || len(asset.Files) != asset.FrameCount ||
		len(asset.FrameSHA256) != asset.FrameCount || len(asset.NativeFrames) != asset.FrameCount ||
		len(asset.NativeHeader) != 3 || asset.NativeHeader[0] != int(native.HeaderByte1) ||
		asset.NativeHeader[1] != int(native.HeaderByte2) || asset.NativeHeader[2] != int(native.HeaderByte4) {
		return errors.New("modern HD battle: incomplete or mismatched animation")
	}
	if asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource == 44 &&
		(len(native.Frames) != 31 || native.HeaderByte1 != 0 || native.HeaderByte2 != 31 || native.HeaderByte4 != 0) {
		return errors.New("modern HD battle: command9 AI native shape differs")
	}
	var entries map[int]*modernBattleHDResource
	if asset.SourceArchive == "FDOTHER.DAT" {
		entries = set.battleEffects
	} else {
		entries = set.battleAnimations
	}
	if entries[asset.SourceResource] != nil {
		return errors.New("modern HD battle: duplicate resource")
	}
	paths := make([]string, asset.FrameCount)
	sizes := make([]image.Rectangle, asset.FrameCount)
	coverage := make([]modernBattleHDCoverageRule, asset.FrameCount)
	var hdRects []image.Rectangle
	if consumer == "native_figani_rgba_hd6_v2" {
		if len(asset.HDFrames) != asset.FrameCount {
			return errors.New("modern HD battle: incomplete HD rectangles")
		}
		hdRects = make([]image.Rectangle, asset.FrameCount)
	} else if len(asset.HDFrames) != 0 {
		return errors.New("modern HD battle: unexpected HD rectangles for v1")
	}
	for index, frame := range native.Frames {
		want := modernBattleHDDescriptor{X: frame.X, Y: frame.Y, Width: frame.Width, Height: frame.Height, Delay: frame.Delay,
			RawByte4: frame.RawByte4, RawByte5: frame.RawByte5, RawByte7: frame.RawByte7}
		if want != asset.NativeFrames[index] || frame.Width <= 0 || frame.Height <= 0 || frame.Width > 1024 || frame.Height > 1024 {
			return fmt.Errorf("modern HD battle: native frame %d differs", index)
		}
		name := asset.Files[index]
		if filepath.Base(name) != name || filepath.Ext(name) != ".png" {
			return errors.New("modern HD battle: unsafe PNG path")
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != asset.FrameSHA256[index] {
			return fmt.Errorf("modern HD battle: frame %d sha256 mismatch", index)
		}
		width, height := frame.Width*6, frame.Height*6
		if hdRects != nil {
			r := asset.HDFrames[index]
			if !r.valid() {
				return fmt.Errorf("modern HD battle: frame %d has invalid HD rectangle", index)
			}
			width, height = r.Width, r.Height
			hdRects[index] = image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || config.Width != width || config.Height != height {
			return fmt.Errorf("modern HD battle: frame %d has invalid geometry", index)
		}
		// 完整解碼與透明度檢查延到第一次預備這筆資源時（stageModernBattleHDResources 本來就
		// 要解碼）。這裡只決定條件：載入時解碼全部戰鬥畫格佔主題載入約三分之二的時間，解出的
		// 圖片又立刻丟掉（#317）。
		fullMask := asset.SourceArchive == "FDOTHER.DAT" && len(frame.Mask) == frame.Width*frame.Height
		for _, value := range frame.Mask {
			fullMask = fullMask && value != 0
		}
		emptyAllowed := asset.SourceArchive == "FIGANI.DAT" ||
			(asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource == 44 && len(native.Frames) == 31 && index == 30)
		nativeEmpty := emptyAllowed && len(frame.Mask) == frame.Width*frame.Height && bytes.Count(frame.Mask, []byte{0}) == len(frame.Mask)
		coverage[index] = modernBattleHDCoverageRule{nativeEmpty: nativeEmpty, fullMask: fullMask}
		paths[index] = filepath.Join(root, name)
		sizes[index] = image.Rect(0, 0, config.Width, config.Height)
	}
	if entries == nil {
		entries = make(map[int]*modernBattleHDResource)
	}
	entries[asset.SourceResource] = &modernBattleHDResource{Frames: make([]image.Image, asset.FrameCount),
		paths: paths, sizes: sizes, hdRects: hdRects, visualScale: asset.VisualScalePercent, hashes: append([]string(nil), asset.FrameSHA256...),
		coverage: coverage, verified: make([]bool, asset.FrameCount), notices: set.themeNotices()}
	if asset.SourceArchive == "FDOTHER.DAT" {
		set.battleEffects = entries
	} else {
		set.battleAnimations = entries
	}
	return nil
}

func (set *modernStoryPortraitSet) battleHDFrame(resource, frame int) (image.Image, error) {
	if set == nil {
		return nil, nil
	}
	return modernBattleHDFrame(set.battleAnimations, resource, frame)
}

func modernBattleHDFrame(entries map[int]*modernBattleHDResource, resource, frame int) (image.Image, error) {
	entry := entries[resource]
	if entry == nil {
		return nil, nil
	}
	if frame < 0 || frame >= len(entry.Frames) || entry.Frames[frame] == nil {
		return nil, errors.New("modern HD battle: frame was not preflighted")
	}
	return entry.Frames[frame], nil
}

// 僅在戰鬥結算前呼叫；所有請求成功後一起發布，再釋放其他資源。
func (set *modernStoryPortraitSet) prepareBattleHDResources(resources ...int) error {
	if set == nil {
		return nil
	}
	return prepareModernBattleHDResources(set.battleAnimations, resources...)
}

func (set *modernStoryPortraitSet) prepareBattleHDEffects(resources ...int) error {
	if set == nil {
		return nil
	}
	return prepareModernBattleHDResources(set.battleEffects, resources...)
}

func prepareModernBattleHDResources(entries map[int]*modernBattleHDResource, resources ...int) error {
	pending, err := stageModernBattleHDResources(entries, resources...)
	if err != nil {
		return err
	}
	pending.publish()
	return nil
}

type modernBattleHDPending struct {
	entries   map[int]*modernBattleHDResource
	requested map[int]bool
	frames    map[int][]image.Image
}

// prepareBattleHDBanks commits both archive families only after every requested
// registered source has decoded. A failure retains both prior frame banks.
func (set *modernStoryPortraitSet) prepareBattleHDBanks(figures, effects []int) error {
	publish, err := set.stageBattleHDBanks(figures, effects)
	if err != nil {
		return err
	}
	publish()
	return nil
}

// Validate/decode before constructing expensive presentation images, but keep
// both banks unpublished until all native frame construction has succeeded.
func (set *modernStoryPortraitSet) stageBattleHDBanks(figures, effects []int) (func(), error) {
	if set == nil {
		return func() {}, nil
	}
	a, err := stageModernBattleHDResources(set.battleAnimations, figures...)
	if err != nil {
		return nil, err
	}
	b, err := stageModernBattleHDResources(set.battleEffects, effects...)
	if err != nil {
		return nil, err
	}
	return func() { a.publish(); b.publish() }, nil
}

func stageModernBattleHDResources(entries map[int]*modernBattleHDResource, resources ...int) (*modernBattleHDPending, error) {
	requested := make(map[int]bool)
	pending := make(map[int][]image.Image)
	// Images are immutable. Share only newly decoded, verified sources within
	// this transaction, retaining independent path/hash/geometry validation.
	decoded := make(map[string]image.Image)
	coverage := make(map[string]modernHDAlphaCoverage)
	for _, resource := range resources {
		if requested[resource] {
			continue
		}
		requested[resource] = true
		entry := entries[resource]
		if entry == nil {
			continue
		}
		if len(entry.Frames) == 0 {
			return nil, errors.New("modern HD battle: empty registered resource")
		}
		if entry.Frames[0] != nil {
			continue
		}
		frames, rejected, err := stageModernBattleHDResource(entry, resource, decoded, coverage)
		if err != nil {
			return nil, err
		}
		if rejected != nil {
			// 延後的檢查失敗：這筆資源整組退回原版畫格（等同包內沒有登錄它），其他資源照常；
			// 不發布任何一格，也不讓呼叫端的交易因此失敗（#317）。
			delete(entries, resource)
			entry.notices.reject(rejected)
			continue
		}
		pending[resource] = frames
	}
	return &modernBattleHDPending{entries: entries, requested: requested, frames: pending}, nil
}

// stageModernBattleHDResource 讀取、核對並解碼一筆資源的全部畫格。err 是原有的硬性失敗（檔案
// 不見或在載入後被改），呼叫端整個交易拒收；rejected 是載入時延後的檢查沒有通過。
func stageModernBattleHDResource(entry *modernBattleHDResource, resource int, decoded map[string]image.Image, coverage map[string]modernHDAlphaCoverage) (frames []image.Image, rejected, err error) {
	frames = make([]image.Image, len(entry.paths))
	for index, path := range entry.paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != entry.hashes[index] {
			return nil, nil, fmt.Errorf("modern HD battle: resource %d frame %d changed before preflight", resource, index)
		}
		picture := decoded[entry.hashes[index]]
		if picture == nil {
			picture, _, err = image.Decode(bytes.NewReader(raw))
			if err != nil {
				if entry.coverage != nil {
					// 位元組已通過 SHA-256，解不開是清冊登錄了壞圖，不是檔案在載入後被改。
					return nil, fmt.Errorf("modern HD battle: resource %d frame %d cannot be decoded: %w", resource, index, err), nil
				}
				return nil, nil, fmt.Errorf("modern HD battle: resource %d frame %d cannot be preloaded", resource, index)
			}
		}
		if picture.Bounds() != entry.sizes[index] {
			return nil, nil, fmt.Errorf("modern HD battle: resource %d frame %d cannot be preloaded", resource, index)
		}
		if err := entry.verifyCoverage(index, picture, coverage); err != nil {
			return nil, fmt.Errorf("modern HD battle: resource %d frame %d %w", resource, index, err), nil
		}
		frames[index] = picture
		decoded[entry.hashes[index]] = picture
	}
	return frames, nil, nil
}

// verifyBattleHDResource 立即做一筆資源延後的解碼與透明度檢查，不發布任何畫格；未通過時
// 和第一次預備相同，退回原版畫格並回傳原因。供清冊核對與測試使用。
func (set *modernStoryPortraitSet) verifyBattleHDResource(archive string, resource int) error {
	if set == nil {
		return nil
	}
	entries := set.battleAnimations
	if archive == "FDOTHER.DAT" {
		entries = set.battleEffects
	}
	entry := entries[resource]
	if entry == nil {
		return nil
	}
	_, rejected, err := stageModernBattleHDResource(entry, resource, map[string]image.Image{}, map[string]modernHDAlphaCoverage{})
	if err != nil {
		return err
	}
	if rejected != nil {
		delete(entries, resource)
		entry.notices.reject(rejected)
		return rejected
	}
	return nil
}

func (pending *modernBattleHDPending) publish() {
	for resource, frames := range pending.frames {
		pending.entries[resource].Frames = frames
	}
	for resource, entry := range pending.entries {
		if !pending.requested[resource] {
			entry.Frames = make([]image.Image, len(entry.paths))
		}
	}
}

// verifyCoverage 檢查載入時延後的透明度條件；同一次預備內相同 SHA-256 的圖只掃一次。
func (entry *modernBattleHDResource) verifyCoverage(index int, picture image.Image, cache map[string]modernHDAlphaCoverage) error {
	if entry.coverage == nil || (index < len(entry.verified) && entry.verified[index]) {
		return nil
	}
	if index >= len(entry.coverage) || index >= len(entry.verified) {
		return errors.New("has no coverage rule")
	}
	summary, ok := cache[entry.hashes[index]]
	if !ok {
		summary = modernHDAlphaCoverageOf(picture)
		cache[entry.hashes[index]] = summary
	}
	if err := entry.coverage[index].check(summary); err != nil {
		return err
	}
	entry.verified[index] = true
	return nil
}
