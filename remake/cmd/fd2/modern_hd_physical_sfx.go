package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const modernPhysicalSFXRole = "battle_physical_sfx_hd"

func modernPhysicalImpactCue(bank, cue int) bool {
	return bank == 49 && (cue == 1 || cue == 3 || cue == 4 || cue == 5) || bank == 53 && cue == 2
}

func modernPhysicalSFXDigest(raw []byte, want string) bool {
	if len(want) != 64 {
		return false
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]) == want
}

func decodeModernPhysicalSFX(raw []byte, rate, channels int) ([]byte, error) {
	identifier := bytes.Index(raw, []byte("\x01vorbis"))
	if len(raw) > 1<<20 || len(raw) < 64 || string(raw[:4]) != "OggS" ||
		identifier < 0 || identifier+30 > len(raw) ||
		binary.LittleEndian.Uint32(raw[identifier+7:identifier+11]) != 0 ||
		int(raw[identifier+11]) != channels ||
		int(binary.LittleEndian.Uint32(raw[identifier+12:identifier+16])) != rate ||
		rate != 48000 || channels != 1 {
		return nil, fmt.Errorf("modern physical sound: invalid Vorbis format")
	}
	stream, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("modern physical sound: %w", err)
	}
	const maximum = 44100 * 4 * 2
	pcm, err := io.ReadAll(io.LimitReader(stream, maximum+4))
	if err != nil || len(pcm) == 0 || len(pcm) > maximum || len(pcm)%4 != 0 {
		return nil, fmt.Errorf("modern physical sound: invalid decoded duration")
	}
	return pcm, nil
}

func loadModernPhysicalSFX(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	if set == nil || asset.Role != modernPhysicalSFXRole ||
		(asset.Status != "runtime_candidate" && asset.Status != "runtime_ready") ||
		asset.ConsumerContract != "physical_impact_ogg_v1" || asset.SourceArchive != "FDOTHER.DAT" ||
		len(asset.CueIDs) == 0 || asset.File == "" || filepath.Base(asset.File) != asset.File ||
		filepath.Ext(asset.File) != ".ogg" {
		return fmt.Errorf("modern physical sound: invalid asset contract")
	}
	seen := make(map[int]bool)
	for _, cue := range asset.CueIDs {
		if !modernPhysicalImpactCue(asset.SourceResource, cue) || seen[cue] ||
			set.physicalSFX[[2]int{asset.SourceResource, cue}] != nil {
			return fmt.Errorf("modern physical sound: duplicate or unsupported cue")
		}
		seen[cue] = true
	}
	metadata, err := os.ReadFile(filepath.Join(separatedAssetPath("sfx"),
		fmt.Sprintf("FDOTHER_%03d", asset.SourceResource), "resource.json"))
	if err != nil || !modernPhysicalSFXDigest(metadata, asset.SourceMetadataSHA256) {
		return fmt.Errorf("modern physical sound: source metadata mismatch")
	}
	raw, err := os.ReadFile(filepath.Join(root, asset.File))
	if err != nil || !modernPhysicalSFXDigest(raw, asset.SHA256) {
		return fmt.Errorf("modern physical sound: asset digest mismatch")
	}
	pcm, err := decodeModernPhysicalSFX(raw, asset.SampleRate, asset.Channels)
	if err != nil {
		return err
	}
	if set.physicalSFX == nil {
		set.physicalSFX = make(map[[2]int][]byte)
	}
	for _, cue := range asset.CueIDs {
		set.physicalSFX[[2]int{asset.SourceResource, cue}] = pcm
	}
	return nil
}

func (g *Game) modernPhysicalSound(bank, cue int, original []byte) []byte {
	if g == nil || g.modernStoryPortraits == nil || len(original) == 0 {
		return original
	}
	if pcm := g.modernStoryPortraits.physicalSFX[[2]int{bank, cue}]; len(pcm) != 0 {
		return pcm
	}
	return original
}

func validateModernPhysicalSFXGroup(set *modernStoryPortraitSet) error {
	if len(set.physicalSFX) == 0 {
		return nil
	}
	if len(set.physicalSFX) != 5 {
		return fmt.Errorf("modern physical sound: incomplete impact group")
	}
	for _, key := range [][2]int{{49, 1}, {49, 3}, {49, 4}, {49, 5}, {53, 2}} {
		if len(set.physicalSFX[key]) == 0 {
			return fmt.Errorf("modern physical sound: incomplete impact group")
		}
	}
	return nil
}
