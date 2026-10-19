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
	Frames []image.Image
	paths  []string
	hashes []string
	sizes  []image.Rectangle
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
	if asset.Role == "battle_animation_hd_set" && asset.SourceArchive == "FIGANI.DAT" && asset.SourceResource >= 0 && asset.SourceResource < 409 {
		return "FIGANI", "native_figani_rgba_hd6_v1", "native_descriptor_key_poses", "contain_native_descriptor_hd6_candidate", true
	}
	if asset.Role == "battle_effect_hd_set" && asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource >= 65 && asset.SourceResource <= 68 {
		return "FDOTHER", "native_fdother_compound_rgba_hd6_v1", "native_compound_frames", "native_descriptor_hd6", true
	}
	return "", "", "", "", false
}

// 所有候選圖片先在區域變數完整驗證；失敗不修改已載入的主題集合。
func loadModernBattleHDAssetWithNative(set *modernStoryPortraitSet, asset modernThemeAsset, root string, native *figani.Animation) error {
	_, consumer, cycle, layout, valid := modernBattleHDContract(asset)
	if !valid || set == nil || native == nil || len(native.Frames) == 0 ||
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
		config, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || config.Width != frame.Width*6 || config.Height != frame.Height*6 {
			return fmt.Errorf("modern HD battle: frame %d has invalid geometry", index)
		}
		picture, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return err
		}
		transparent, opaque := false, false
		for y := picture.Bounds().Min.Y; y < picture.Bounds().Max.Y; y++ {
			for x := picture.Bounds().Min.X; x < picture.Bounds().Max.X; x++ {
				_, _, _, a := picture.At(x, y).RGBA()
				transparent = transparent || a == 0
				opaque = opaque || a == 0xffff
			}
		}
		fullMask := asset.SourceArchive == "FDOTHER.DAT" && len(frame.Mask) == frame.Width*frame.Height
		for _, value := range frame.Mask {
			fullMask = fullMask && value != 0
		}
		if (!transparent && !fullMask) || !opaque {
			return fmt.Errorf("modern HD battle: frame %d lacks transparent background or opaque core", index)
		}
		paths[index] = filepath.Join(root, name)
		sizes[index] = picture.Bounds()
	}
	if entries == nil {
		entries = make(map[int]*modernBattleHDResource)
	}
	entries[asset.SourceResource] = &modernBattleHDResource{Frames: make([]image.Image, asset.FrameCount),
		paths: paths, sizes: sizes, hashes: append([]string(nil), asset.FrameSHA256...)}
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
		frames := make([]image.Image, len(entry.paths))
		for index, path := range entry.paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(raw)
			if hex.EncodeToString(digest[:]) != entry.hashes[index] {
				return nil, fmt.Errorf("modern HD battle: resource %d frame %d changed before preflight", resource, index)
			}
			picture, _, err := image.Decode(bytes.NewReader(raw))
			if err != nil || picture.Bounds() != entry.sizes[index] {
				return nil, fmt.Errorf("modern HD battle: resource %d frame %d cannot be preloaded", resource, index)
			}
			frames[index] = picture
		}
		pending[resource] = frames
	}
	return &modernBattleHDPending{entries: entries, requested: requested, frames: pending}, nil
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
