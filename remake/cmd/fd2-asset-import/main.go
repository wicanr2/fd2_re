// fd2-asset-import 將玩家合法持有的原版資源轉成正式 runtime 可消費的分離素材。
// 原版 archive reader 只存在於這個離線命令，不應被遊戲 renderer 呼叫。
package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	fdotherSize   = 3382481
	fdotherMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	fdotherSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	actionCells   = 78
)

type paletteDocument struct {
	SchemaVersion int      `json:"schema_version"`
	AssetID       string   `json:"asset_id"`
	Source        sourceID `json:"source"`
	Components    []int    `json:"dac_6bit_components"`
}

type afmFrameDocument struct {
	FrameID     string `json:"frame_id"`
	Frame       string `json:"frame"`
	Palette     string `json:"palette"`
	SourceFrame int    `json:"source_frame"`
}

type afmAnimationDocument struct {
	SchemaVersion int                `json:"schema_version"`
	Kind          string             `json:"kind"`
	AssetID       string             `json:"asset_id"`
	Status        string             `json:"status"`
	Evidence      string             `json:"evidence"`
	Codec         string             `json:"codec"`
	Title         string             `json:"title"`
	Width         int                `json:"width"`
	Height        int                `json:"height"`
	FrameCount    int                `json:"frame_count"`
	Source        afmSourceDocument  `json:"source"`
	Frames        []afmFrameDocument `json:"frames"`
}

type afmSourceDocument struct {
	File                string `json:"file"`
	Resource            int    `json:"resource"`
	Size                int    `json:"size"`
	MD5                 string `json:"md5"`
	SHA256              string `json:"sha256"`
	RawSize             int    `json:"raw_size"`
	ContainerEntryCount int    `json:"container_entry_count"`
	EmptyTailIndex      int    `json:"empty_tail_index"`
}

type afmPaletteDocument struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Components    []int  `json:"dac_6bit_components"`
}

type sourceID struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Nested   *int   `json:"nested,omitempty"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type surfaceDocument struct {
	SchemaVersion  int      `json:"schema_version"`
	Kind           string   `json:"kind"`
	AssetID        string   `json:"asset_id"`
	Status         string   `json:"status"`
	Codec          string   `json:"codec,omitempty"`
	Width          int      `json:"width,omitempty"`
	Height         int      `json:"height,omitempty"`
	Frame          string   `json:"frame,omitempty"`
	Mask           string   `json:"mask,omitempty"`
	Evidence       string   `json:"evidence"`
	ReasonCode     string   `json:"reason_code,omitempty"`
	ContainerCount int      `json:"container_entry_count,omitempty"`
	EmptyTailIndex *int     `json:"empty_tail_index,omitempty"`
	Source         sourceID `json:"source"`
}

type textToken struct {
	Kind       string `json:"kind"`
	GlyphIndex *int   `json:"glyph_index,omitempty"`
	Control    string `json:"control,omitempty"`
	Text       string `json:"text,omitempty"`
}

type textString struct {
	StringID    string      `json:"string_id"`
	SourceIndex int         `json:"source_index"`
	Text        string      `json:"text"`
	Tokens      []textToken `json:"tokens"`
}

type textDocument struct {
	SchemaVersion int          `json:"schema_version"`
	Kind          string       `json:"kind"`
	AssetID       string       `json:"asset_id"`
	Status        string       `json:"status"`
	Evidence      string       `json:"evidence"`
	ReasonCode    string       `json:"reason_code,omitempty"`
	Source        sourceID     `json:"source"`
	Strings       []textString `json:"strings,omitempty"`
}

type fontDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	AssetID       string   `json:"asset_id"`
	Status        string   `json:"status"`
	Evidence      string   `json:"evidence"`
	GlyphCount    int      `json:"glyph_count"`
	CellWidth     int      `json:"cell_width"`
	CellHeight    int      `json:"cell_height"`
	Columns       int      `json:"columns"`
	Rows          int      `json:"rows"`
	Atlas         string   `json:"atlas"`
	Source        sourceID `json:"source"`
}

type itemPanelEntryDocument struct {
	Index  int    `json:"index"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
	Mask   string `json:"mask,omitempty"`
}

type itemPanelDocument struct {
	SchemaVersion int                      `json:"schema_version"`
	Kind          string                   `json:"kind"`
	AssetID       string                   `json:"asset_id"`
	Status        string                   `json:"status"`
	Evidence      string                   `json:"evidence"`
	Source        sourceID                 `json:"source"`
	Entries       []itemPanelEntryDocument `json:"entries"`
}

type shopEntryDocument struct {
	Index  int    `json:"index"`
	Role   string `json:"role"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
	Mask   string `json:"mask,omitempty"`
}

type shopSuccessDocument struct {
	X               int                 `json:"x"`
	Y               int                 `json:"y"`
	PreTicks        int                 `json:"pre_ticks"`
	TicksPerFrame   int                 `json:"ticks_per_frame"`
	PostTicks       int                 `json:"post_ticks"`
	RestorePortrait bool                `json:"restore_portrait"`
	Frames          []shopEntryDocument `json:"frames"`
}

type shopResourceDocument struct {
	SchemaVersion int                 `json:"schema_version"`
	Kind          string              `json:"kind"`
	AssetID       string              `json:"asset_id"`
	Status        string              `json:"status"`
	Evidence      string              `json:"evidence"`
	ContainerKind string              `json:"container_kind"`
	EntryCount    int                 `json:"entry_count"`
	Source        sourceID            `json:"source"`
	Entries       []shopEntryDocument `json:"entries"`
	Success       shopSuccessDocument `json:"success"`
}

type townLabelDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	AssetID       string   `json:"asset_id"`
	Status        string   `json:"status"`
	Evidence      string   `json:"evidence"`
	Codec         string   `json:"codec"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Frame         string   `json:"frame"`
	Source        sourceID `json:"source"`
}

type fdiconSpriteDocument struct {
	Index     int    `json:"index"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Frame     string `json:"frame"`
	Mask      string `json:"mask"`
	RemapMask string `json:"remap_mask"`
}

type fdiconBankDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	Kind          string                 `json:"kind"`
	AssetID       string                 `json:"asset_id"`
	Status        string                 `json:"status"`
	Evidence      string                 `json:"evidence"`
	Source        sourceID               `json:"source"`
	Sprites       []fdiconSpriteDocument `json:"sprites"`
}

type fdshapBankDocument struct {
	SchemaVersion   int                    `json:"schema_version"`
	Kind            string                 `json:"kind"`
	AssetID         string                 `json:"asset_id"`
	Status          string                 `json:"status"`
	Evidence        string                 `json:"evidence"`
	Source          sourceID               `json:"source"`
	MapIndex        int                    `json:"map_index"`
	ImageResource   int                    `json:"image_resource"`
	ControlResource int                    `json:"control_resource"`
	Controls        []int                  `json:"controls"`
	Sprites         []fdiconSpriteDocument `json:"sprites"`
}

type fdfieldPositionDocument struct {
	XWord  int `json:"x_word"`
	YWord  int `json:"y_word"`
	RawKey int `json:"raw_key"`
}

type fdfieldSelectorDocument struct {
	SchemaVersion     int                       `json:"schema_version"`
	Kind              string                    `json:"kind"`
	DocumentID        string                    `json:"document_id"`
	Status            string                    `json:"status"`
	Evidence          string                    `json:"evidence"`
	Source            sourceID                  `json:"source"`
	MapResource       int                       `json:"map_resource"`
	ControlResource   int                       `json:"control_resource"`
	PositionsResource int                       `json:"positions_resource"`
	MapSHA256         string                    `json:"map_sha256"`
	ControlSHA256     string                    `json:"control_sha256"`
	PositionsSHA256   string                    `json:"positions_sha256"`
	Width             int                       `json:"width"`
	Height            int                       `json:"height"`
	Tiles             []int                     `json:"tiles"`
	EventBytes        []int                     `json:"event_bytes"`
	BlitModes         []int                     `json:"blit_modes"`
	ControlBytes      []int                     `json:"control_bytes"`
	Positions         []fdfieldPositionDocument `json:"positions"`
}

type archiveIdentity struct {
	file, prefix string
	size         int
	md5, sha256  string
}

var surfaceArchives = []archiveIdentity{
	{file: "BG.DAT", prefix: "BG", size: 624564, md5: "4b5414c92b40ef25ba0ee10c80f9e149", sha256: "b9fc21d019d6256a4bb7e6da1cefcb0bfe331d8ff74a52a8201570afc98b56de"},
	{file: "TAI.DAT", prefix: "TAI", size: 94917, md5: "7cfe4b9ad2cbff44b2ebd7ab2f94e4aa", sha256: "d56fea9c43f8bb59aad89ad76698885d7e07f380d12a4547888a0b60ea5e0410"},
}

var fdtxtArchive = archiveIdentity{
	file: "FDTXT.DAT", prefix: "FDTXT", size: 120502,
	md5:    "fe5c487ce4313485f1da9d48d35b05f9",
	sha256: "a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69",
}

var fdiconArchive = archiveIdentity{
	file: "FDICON.B24", prefix: "FDICON", size: 624010,
	md5:    "46f793540209a063ea73a5373ca14bf4",
	sha256: "7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44",
}

var fdshapArchive = archiveIdentity{
	file: "FDSHAP.DAT", prefix: "FDSHAP", size: 3557794,
	md5:    "9b0d356074f57cc27aebf3bb89aae247",
	sha256: "901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c",
}

var fdotherArchive = archiveIdentity{
	file: "FDOTHER.DAT", prefix: "FDOTHER", size: fdotherSize,
	md5: fdotherMD5, sha256: fdotherSHA256,
}

var fdfieldArchive = archiveIdentity{
	file: "FDFIELD.DAT", prefix: "FDFIELD", size: 243169,
	md5:    "ecdb0436d26adfe5d107f2713fa7e9a2",
	sha256: "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513",
}

var aniArchive = archiveIdentity{
	file: "ANI.DAT", prefix: "ANI", size: afm.ANIFileSize,
	md5: afm.ANIMD5, sha256: afm.ANISHA256,
}

func verifyFDOTHER(path string) error {
	return verifyArchive(path, fdotherArchive)
}

func verifyArchive(path string, identity archiveIdentity) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	md5sum, sha := md5.Sum(raw), sha256.Sum256(raw)
	if len(raw) != identity.size || hex.EncodeToString(md5sum[:]) != identity.md5 || hex.EncodeToString(sha[:]) != identity.sha256 {
		return fmt.Errorf("%s 版本不符：size=%d md5=%x sha256=%x", identity.file, len(raw), md5sum, sha)
	}
	return nil
}

func exportSingleFrameArchive(path, outputRoot string, identity archiveIdentity) (int, int, error) {
	if err := verifyArchive(path, identity); err != nil {
		return 0, 0, err
	}
	count, err := fdother.ArchiveResourceCount(path)
	if err != nil {
		return 0, 0, err
	}
	decoded, blocked := 0, 0
	for resource := 0; resource < count; resource++ {
		raw, readErr := fdother.ReadResource(path, resource)
		directory := filepath.Join(outputRoot, "surfaces", fmt.Sprintf("%s_%03d", identity.prefix, resource))
		doc := surfaceDocument{
			SchemaVersion: 1, Kind: "indexed_surface", AssetID: fmt.Sprintf("surface/%s_%03d", identity.prefix, resource),
			Status: "blocked", Evidence: "confirmed", ReasonCode: "decode_failed",
			Source: sourceID{File: identity.file, Resource: resource, Size: identity.size, MD5: identity.md5, SHA256: identity.sha256, RawSize: len(raw)},
		}
		if readErr == nil {
			doc.Source.Size = identity.size
			frame, decodeErr := fdother.ParseSingleFrame(raw)
			if decodeErr == nil {
				indexed, mask, layerErr := frame.IndexedLayers()
				if layerErr == nil {
					doc.Status, doc.Codec, doc.Width, doc.Height = "decoded", "fd2_4e63d_single_frame", frame.Width, frame.Height
					doc.Frame, doc.Mask, doc.ReasonCode = "frame.png", "mask.png", ""
					if err := writeSurfacePNGs(directory, frame.Width, frame.Height, indexed, mask); err != nil {
						return decoded, blocked, err
					}
					decoded++
				} else {
					blocked++
				}
			} else {
				blocked++
			}
		} else {
			blocked++
		}
		if err := writeJSON(filepath.Join(directory, "resource.json"), doc); err != nil {
			return decoded, blocked, err
		}
	}
	return decoded, blocked, nil
}

func exportSelectedSingleFrame(path, outputRoot string, identity archiveIdentity, resource int) error {
	raw, err := fdother.ReadResource(path, resource)
	if err != nil {
		return err
	}
	frame, err := fdother.ParseSingleFrame(raw)
	if err != nil {
		return err
	}
	indexed, mask, err := frame.IndexedLayers()
	if err != nil {
		return err
	}
	directory := filepath.Join(outputRoot, "surfaces", fmt.Sprintf("%s_%03d", identity.prefix, resource))
	document := surfaceDocument{
		SchemaVersion: 1, Kind: "indexed_surface", AssetID: fmt.Sprintf("surface/%s_%03d", identity.prefix, resource),
		Status: "decoded", Codec: "fd2_4e63d_single_frame", Width: frame.Width, Height: frame.Height,
		Frame: "frame.png", Mask: "mask.png", Evidence: "confirmed",
		Source: sourceID{File: identity.file, Resource: resource, Size: identity.size, MD5: identity.md5, SHA256: identity.sha256, RawSize: len(raw)},
	}
	if err := writeSurfacePNGs(directory, frame.Width, frame.Height, indexed, mask); err != nil {
		return err
	}
	return writeJSON(filepath.Join(directory, "resource.json"), document)
}

func writeSurfacePNGs(directory string, width, height int, indexed, mask []byte) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 0xff}
	}
	frame := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	copy(frame.Pix, indexed)
	maskImage := image.NewGray(image.Rect(0, 0, width, height))
	copy(maskImage.Pix, mask)
	for name, value := range map[string]image.Image{"frame.png": frame, "mask.png": maskImage} {
		file, err := os.Create(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, value)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func exportANI(path, outputRoot string) (int, error) {
	if err := verifyArchive(path, aniArchive); err != nil {
		return 0, err
	}
	count, err := afm.ArchiveResourceCount(path)
	if err != nil || count != 10 {
		return 0, fmt.Errorf("ANI.DAT resource count=%d err=%v, want 10", count, err)
	}
	tail, err := afm.ReadArchiveResource(path, 9)
	if err != nil || len(tail) != 0 {
		return 0, fmt.Errorf("ANI.DAT empty tail bytes=%d err=%v, want 0", len(tail), err)
	}
	totalFrames := 0
	for resource := 0; resource < 9; resource++ {
		raw, err := afm.ReadArchiveResource(path, resource)
		if err != nil {
			return totalFrames, err
		}
		clip, err := afm.DecodeResource(path, resource)
		if err != nil {
			return totalFrames, fmt.Errorf("ANI #%d: %w", resource, err)
		}
		directory := filepath.Join(outputRoot, "animations", fmt.Sprintf("ANI_%03d", resource))
		document := afmAnimationDocument{
			SchemaVersion: 1, Kind: "afm_indexed_animation", AssetID: fmt.Sprintf("animation/ANI_%03d", resource),
			Status: "decoded", Evidence: "confirmed", Codec: "fd2_afm_vm_v1",
			Title: strings.TrimRight(clip.Title, "\x00 "), Width: afm.Width, Height: afm.Height,
			FrameCount: clip.HeaderFrames,
			Source: afmSourceDocument{File: aniArchive.file, Resource: resource, Size: aniArchive.size,
				MD5: aniArchive.md5, SHA256: aniArchive.sha256, RawSize: len(raw), ContainerEntryCount: 10, EmptyTailIndex: 9},
			Frames: make([]afmFrameDocument, clip.HeaderFrames),
		}
		for frame := 0; frame < clip.HeaderFrames; frame++ {
			frameName := fmt.Sprintf("frame_%03d.png", frame)
			paletteName := fmt.Sprintf("palette_%03d.json", frame)
			if err := writeAFMIndexedPNG(filepath.Join(directory, frameName), clip.IndexedFrames[frame], clip.Palettes[frame]); err != nil {
				return totalFrames, fmt.Errorf("ANI #%d frame %d: %w", resource, frame, err)
			}
			if err := writeJSON(filepath.Join(directory, paletteName), afmPaletteDocument{
				SchemaVersion: 1, Kind: "vga_dac_6bit_snapshot", Components: bytesToInts(clip.Palettes[frame]),
			}); err != nil {
				return totalFrames, err
			}
			document.Frames[frame] = afmFrameDocument{FrameID: fmt.Sprintf("frame/%03d", frame),
				Frame: frameName, Palette: paletteName, SourceFrame: frame}
		}
		if err := writeJSON(filepath.Join(directory, "animation.json"), document); err != nil {
			return totalFrames, err
		}
		totalFrames += clip.HeaderFrames
	}
	return totalFrames, nil
}

func writeAFMIndexedPNG(path string, indexed, dac []byte) error {
	if len(indexed) != afm.Width*afm.Height || len(dac) != 768 {
		return errors.New("AFM frame or palette length mismatch")
	}
	palette := make(color.Palette, 256)
	for index := range palette {
		r, g, b := dac[index*3], dac[index*3+1], dac[index*3+2]
		palette[index] = color.RGBA{R: (r << 2) | (r >> 4), G: (g << 2) | (g >> 4), B: (b << 2) | (b >> 4), A: 0xff}
	}
	output := image.NewPaletted(image.Rect(0, 0, afm.Width, afm.Height), palette)
	copy(output.Pix, indexed)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(file, output)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func loadGlyphMap(path string) (map[int]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var encoded map[string]string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, err
	}
	result := make(map[int]string, len(encoded))
	for key, value := range encoded {
		if key == "_comment" {
			continue
		}
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= fdtxt.ControlMin || value == "" {
			return nil, fmt.Errorf("glyph map key/value invalid: %q", key)
		}
		result[index] = value
	}
	return result, nil
}

func exportFDTXT(path, glyphMapPath, outputRoot string) (int, int, error) {
	if err := verifyArchive(path, fdtxtArchive); err != nil {
		return 0, 0, err
	}
	glyphs, err := loadGlyphMap(glyphMapPath)
	if err != nil {
		return 0, 0, fmt.Errorf("glyph map: %w", err)
	}
	count, err := fdother.ArchiveResourceCount(path)
	if err != nil {
		return 0, 0, err
	}
	decoded, blocked := 0, 0
	for resource := 0; resource < count; resource++ {
		raw, readErr := fdother.ReadResource(path, resource)
		document := textDocument{
			SchemaVersion: 1, Kind: "fdtxt_word_table",
			AssetID: fmt.Sprintf("text/FDTXT_%03d", resource), Status: "blocked",
			Evidence: "confirmed", ReasonCode: "decode_failed",
			Source: sourceID{File: fdtxtArchive.file, Resource: resource, Size: fdtxtArchive.size,
				MD5: fdtxtArchive.md5, SHA256: fdtxtArchive.sha256, RawSize: len(raw)},
		}
		if readErr == nil && len(raw) == 0 {
			document.ReasonCode = "empty_resource"
			blocked++
		} else if readErr != nil {
			blocked++
		} else if stringsTable, parseErr := fdtxt.Parse(raw); parseErr != nil {
			blocked++
		} else {
			document.Status, document.ReasonCode = "decoded", ""
			document.Strings = make([]textString, stringsTable.Count())
			for index := range document.Strings {
				words, _ := stringsTable.Words(index)
				entry := textString{StringID: fmt.Sprintf("FDTXT_%03d/string_%04d", resource, index), SourceIndex: index, Tokens: make([]textToken, 0, len(words))}
				for _, word := range words {
					if word >= fdtxt.ControlMin {
						entry.Tokens = append(entry.Tokens, textToken{Kind: "control", Control: fmt.Sprintf("%04X", word)})
						continue
					}
					glyphIndex := int(word)
					text := glyphs[glyphIndex]
					if text == "" {
						text = fmt.Sprintf("〈glyph:%d〉", glyphIndex)
					}
					entry.Text += text
					entry.Tokens = append(entry.Tokens, textToken{Kind: "glyph", GlyphIndex: &glyphIndex, Text: text})
				}
				document.Strings[index] = entry
			}
			decoded++
		}
		directory := filepath.Join(outputRoot, "text", fmt.Sprintf("FDTXT_%03d", resource))
		if err := writeJSON(filepath.Join(directory, "resource.json"), document); err != nil {
			return decoded, blocked, err
		}
	}
	return decoded, blocked, nil
}

func exportCommandGrid(fdotherPath, outputRoot string) error {
	if err := verifyFDOTHER(fdotherPath); err != nil {
		return err
	}
	dac, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		return err
	}
	if _, err := fdother.ParseVGAPalette(dac); err != nil {
		return err
	}
	components := make([]int, len(dac))
	for i, component := range dac {
		components[i] = int(component)
	}
	document := paletteDocument{
		SchemaVersion: 1,
		AssetID:       "palette/fdother_000",
		Source: sourceID{
			File: "FDOTHER.DAT", Resource: 0, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(dac),
		},
		Components: components,
	}
	if err := writeJSON(filepath.Join(outputRoot, "palette", "fdother_000.json"), document); err != nil {
		return err
	}

	palette, _ := fdother.ParseVGAPalette(dac)
	cells, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		return err
	}
	if len(cells) != actionCells {
		return fmt.Errorf("FDOTHER #2 cells=%d, want %d", len(cells), actionCells)
	}
	directory := filepath.Join(outputRoot, "ui", "action_cells")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	actionDocument := itemPanelDocument{
		SchemaVersion: 1,
		Kind:          "fdother_raw_cell_bank",
		AssetID:       "ui/FDOTHER_002/action_cells",
		Status:        "decoded",
		Evidence:      "confirmed",
		Source: sourceID{
			File: "FDOTHER.DAT", Resource: 2, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: 37680,
		},
	}
	for index, cell := range cells {
		image, err := cell.Paletted(palette)
		if err != nil {
			return fmt.Errorf("cell %d: %w", index, err)
		}
		path := filepath.Join(directory, fmt.Sprintf("cell_%03d.png", index))
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, image)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
		actionDocument.Entries = append(actionDocument.Entries, itemPanelEntryDocument{
			Index: index, Codec: "raw_indexed_transparent", Width: cell.Width,
			Height: cell.Height, Frame: fmt.Sprintf("cell_%03d.png", index),
		})
	}
	if err := writeJSON(filepath.Join(directory, "resource.json"), actionDocument); err != nil {
		return err
	}
	if err := exportFont(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportItemPanel(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportLoadSlots(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportChurchUI(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportNativeShops(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportNativeTown(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportTitlePublisher(fdotherPath, outputRoot); err != nil {
		return err
	}
	if err := exportTitleFDOTHER(fdotherPath, outputRoot); err != nil {
		return err
	}
	return exportEndingFDOTHER(fdotherPath, outputRoot)
}

func exportTitlePublisher(fdotherPath, outputRoot string) error {
	if err := exportSelectedSingleFrame(fdotherPath, outputRoot, fdotherArchive, 74); err != nil {
		return fmt.Errorf("FDOTHER #74 title publisher: %w", err)
	}
	dac, err := fdother.ReadResource(fdotherPath, 76)
	if err != nil {
		return err
	}
	if _, err := fdother.ParseVGAPalette(dac); err != nil {
		return fmt.Errorf("FDOTHER #76 publisher palette: %w", err)
	}
	return writeJSON(filepath.Join(outputRoot, "palette", "fdother_076.json"), paletteDocument{
		SchemaVersion: 1, AssetID: "palette/fdother_076",
		Source:     sourceID{File: fdotherArchive.file, Resource: 76, Size: fdotherArchive.size, MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(dac)},
		Components: bytesToInts(dac),
	})
}

func exportTitleFDOTHER(fdotherPath, outputRoot string) error {
	for _, resource := range []int{69, 70, 71, 72, 73, 75, 100} {
		if err := exportSelectedSingleFrame(fdotherPath, outputRoot, fdotherArchive, resource); err != nil {
			return fmt.Errorf("FDOTHER #%d title surface: %w", resource, err)
		}
	}
	for _, resource := range []int{8, 99, 101} {
		dac, err := fdother.ReadResource(fdotherPath, resource)
		if err != nil {
			return err
		}
		if _, err := fdother.ParseVGAPalette(dac); err != nil {
			return fmt.Errorf("FDOTHER #%d title palette: %w", resource, err)
		}
		if err := writeJSON(filepath.Join(outputRoot, "palette", fmt.Sprintf("fdother_%03d.json", resource)), paletteDocument{
			SchemaVersion: 1,
			AssetID:       fmt.Sprintf("palette/fdother_%03d", resource),
			Source:        sourceID{File: fdotherArchive.file, Resource: resource, Size: fdotherArchive.size, MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(dac)},
			Components:    bytesToInts(dac),
		}); err != nil {
			return err
		}
	}
	return exportTitleNestedSurfaces(fdotherPath, outputRoot)
}

func exportTitleNestedSurfaces(fdotherPath, outputRoot string) error {
	outer, err := fdother.ReadResource(fdotherPath, 7)
	if err != nil {
		return err
	}
	if count, err := fdother.ArchiveDataResourceCount(outer); err != nil || count != 8 {
		return fmt.Errorf("FDOTHER #7 nested count=%d err=%v, want 8", count, err)
	}
	tail, err := fdother.ArchiveEntry(outer, 7)
	if err != nil || len(tail) != 0 {
		return fmt.Errorf("FDOTHER #7 nested tail bytes=%d err=%v, want 0", len(tail), err)
	}
	for nested := 0; nested < 7; nested++ {
		raw, err := fdother.ArchiveEntry(outer, nested)
		if err != nil {
			return err
		}
		frame, err := fdother.ParseSingleFrame(raw)
		if err != nil {
			return fmt.Errorf("FDOTHER #7/%d title surface: %w", nested, err)
		}
		indexed, mask, err := frame.IndexedLayers()
		if err != nil {
			return err
		}
		directory := filepath.Join(outputRoot, "surfaces", "FDOTHER_007", fmt.Sprintf("entry_%03d", nested))
		entryIndex, emptyTail := nested, 7
		document := surfaceDocument{
			SchemaVersion: 1, Kind: "indexed_surface", AssetID: fmt.Sprintf("surface/FDOTHER_007/entry_%03d", nested),
			Status: "decoded", Codec: "fd2_4e63d_single_frame", Width: frame.Width, Height: frame.Height,
			Frame: "frame.png", Mask: "mask.png", Evidence: "confirmed", ContainerCount: 8, EmptyTailIndex: &emptyTail,
			Source: sourceID{File: fdotherArchive.file, Resource: 7, Nested: &entryIndex, Size: fdotherArchive.size, MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(raw)},
		}
		if err := writeSurfacePNGs(directory, frame.Width, frame.Height, indexed, mask); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(directory, "resource.json"), document); err != nil {
			return err
		}
	}
	return nil
}

func exportNativeTown(fdotherPath, outputRoot string) error {
	for _, resource := range []int{11, 61, 62} {
		if err := exportSelectedSingleFrame(fdotherPath, outputRoot, fdotherArchive, resource); err != nil {
			return fmt.Errorf("FDOTHER #%d town background: %w", resource, err)
		}
	}
	raw, err := fdother.ReadResource(fdotherPath, 10)
	if err != nil {
		return err
	}
	label, err := fdother.ParseOpaqueRunCell(raw)
	if err != nil || label.Width != 62 || label.Height != 26 {
		return errors.New("FDOTHER #10 town label is not 62x26 opaque cell")
	}
	directory := filepath.Join(outputRoot, "ui", "fdother_010_town_label")
	if err := writeIndexedPNG(filepath.Join(directory, "frame.png"), label.Width, label.Height, label.Pixels); err != nil {
		return err
	}
	return writeJSON(filepath.Join(directory, "resource.json"), townLabelDocument{
		SchemaVersion: 1, Kind: "native_town_label", AssetID: "ui/FDOTHER_010/town_label",
		Status: "decoded", Evidence: "confirmed", Codec: "opaque_high_run",
		Width: label.Width, Height: label.Height, Frame: "frame.png",
		Source: sourceID{File: fdotherArchive.file, Resource: 10, Size: fdotherArchive.size,
			MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(raw)},
	})
}

func exportNativeShops(fdotherPath, outputRoot string) error {
	for _, resource := range []int{12, 29, 63} {
		assets, err := campaign.DecodeNativeShopAssets(fdotherPath, resource)
		if err != nil {
			return fmt.Errorf("FDOTHER #%d shop: %w", resource, err)
		}
		raw, err := fdother.ReadResource(fdotherPath, resource)
		if err != nil {
			return err
		}
		directory := filepath.Join(outputRoot, "shop", fmt.Sprintf("FDOTHER_%03d", resource))
		doc := shopResourceDocument{SchemaVersion: 1, Kind: "native_shop_indexed_assets",
			AssetID: fmt.Sprintf("shop/FDOTHER_%03d", resource), Status: "decoded", Evidence: "confirmed",
			ContainerKind: map[bool]string{true: "scene_lmi1", false: "llllll"}[resource == 29],
			EntryCount:    len(assets.RawEntries), Source: sourceID{File: fdotherArchive.file, Resource: resource,
				Size: fdotherArchive.size, MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(raw)}}
		add := func(index int, role, codec string, width, height int, pixels []byte, transparent bool) error {
			name := fmt.Sprintf("entry_%03d", index)
			entry := shopEntryDocument{Index: index, Role: role, Codec: codec, Width: width, Height: height,
				Frame: filepath.ToSlash(filepath.Join(name, "frame.png"))}
			if transparent {
				entry.Mask = filepath.ToSlash(filepath.Join(name, "mask.png"))
				mask := make([]byte, len(pixels))
				for i, value := range pixels {
					if value != 0 {
						mask[i] = 255
					}
				}
				if err := writeSurfacePNGs(filepath.Join(directory, name), width, height, pixels, mask); err != nil {
					return err
				}
			} else if err := writeIndexedPNG(filepath.Join(directory, entry.Frame), width, height, pixels); err != nil {
				return err
			}
			doc.Entries = append(doc.Entries, entry)
			return nil
		}
		if err := add(0, "background", "fd2_4e63d_single_frame", campaign.NativeShopWidth, campaign.NativeShopHeight, assets.Background, false); err != nil {
			return err
		}
		if err := add(1, "decoration", "opaque_high_run", assets.Decoration.Width, assets.Decoration.Height, assets.Decoration.Pixels, false); err != nil {
			return err
		}
		if err := add(2, "gold_roll_strip", "raw_indexed_transparent", assets.GoldRollStrip.Width, assets.GoldRollStrip.Height, assets.GoldRollStrip.Pixels, true); err != nil {
			return err
		}
		for option := range assets.ServiceCells {
			for variant := range assets.ServiceCells[option] {
				cell := assets.ServiceCells[option][variant]
				index := 3 + option*2 + variant
				if err := add(index, "service_cell", "raw_indexed_transparent", cell.Width, cell.Height, cell.Pixels, true); err != nil {
					return err
				}
			}
		}
		if err := add(15, "price_cell", "raw_indexed_transparent", assets.PriceCell.Width, assets.PriceCell.Height, assets.PriceCell.Pixels, true); err != nil {
			return err
		}
		if err := add(16, "panel", "opaque_high_run", assets.Panel.Width, assets.Panel.Height, assets.Panel.Pixels, false); err != nil {
			return err
		}
		for i, cell := range assets.CompareCells {
			if err := add(18+i, "compare_cell", "opaque_high_run", cell.Width, cell.Height, cell.Pixels, false); err != nil {
				return err
			}
		}
		plan := map[int]shopSuccessDocument{12: {X: 169, Y: 45, TicksPerFrame: 2, RestorePortrait: true}, 29: {X: 148, Y: 39, PreTicks: 1, PostTicks: 8, RestorePortrait: true}, 63: {X: 131, Y: 28, TicksPerFrame: 2}}[resource]
		for i, frame := range assets.SuccessFrames {
			indexed, mask, err := frame.IndexedLayers()
			if err != nil {
				return err
			}
			index := 23 + i
			name := fmt.Sprintf("success_%03d", i)
			if err := writeSurfacePNGs(filepath.Join(directory, name), frame.Width, frame.Height, indexed, mask); err != nil {
				return err
			}
			entry := shopEntryDocument{Index: index, Role: "success_frame", Codec: "fd2_4e63d_single_frame", Width: frame.Width, Height: frame.Height, Frame: name + "/frame.png", Mask: name + "/mask.png"}
			plan.Frames = append(plan.Frames, entry)
			doc.Entries = append(doc.Entries, entry)
		}
		doc.Success = plan
		if err := writeJSON(filepath.Join(directory, "resource.json"), doc); err != nil {
			return err
		}
	}
	return nil
}

func exportEndingFDOTHER(fdotherPath, outputRoot string) error {
	for _, resource := range []int{56, 59, 60} {
		if err := exportSelectedSingleFrame(fdotherPath, outputRoot, fdotherArchive, resource); err != nil {
			return fmt.Errorf("FDOTHER #%d ending surface: %w", resource, err)
		}
	}
	dac, err := fdother.ReadResource(fdotherPath, 57)
	if err != nil {
		return err
	}
	if _, err := fdother.ParseVGAPalette(dac); err != nil {
		return fmt.Errorf("FDOTHER #57 ending palette: %w", err)
	}
	return writeJSON(filepath.Join(outputRoot, "palette", "fdother_057.json"), paletteDocument{
		SchemaVersion: 1,
		AssetID:       "palette/fdother_057",
		Source: sourceID{File: fdotherArchive.file, Resource: 57, Size: fdotherArchive.size,
			MD5: fdotherArchive.md5, SHA256: fdotherArchive.sha256, RawSize: len(dac)},
		Components: bytesToInts(dac),
	})
}

func exportFont(fdotherPath, outputRoot string) error {
	raw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		return err
	}
	font, err := fdtxt.ParseFont(raw)
	if err != nil {
		return err
	}
	const columns = 32
	rows := (font.GlyphCount() + columns - 1) / columns
	atlas := image.NewGray(image.Rect(0, 0, columns*fdtxt.GlyphWidth, rows*fdtxt.GlyphHeight))
	for glyph := 0; glyph < font.GlyphCount(); glyph++ {
		baseX := (glyph % columns) * fdtxt.GlyphWidth
		baseY := (glyph / columns) * fdtxt.GlyphHeight
		for y := 0; y < fdtxt.GlyphHeight; y++ {
			for x := 0; x < fdtxt.GlyphWidth; x++ {
				set, err := font.GlyphBit(glyph, x, y)
				if err != nil {
					return err
				}
				if set {
					atlas.SetGray(baseX+x, baseY+y, color.Gray{Y: 0xff})
				}
			}
		}
	}
	directory := filepath.Join(outputRoot, "fonts", "fdother_004")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(directory, "atlas.png"))
	if err != nil {
		return err
	}
	encodeErr := png.Encode(file, atlas)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	document := fontDocument{
		SchemaVersion: 1, Kind: "bitmap_font", AssetID: "font/FDOTHER_004",
		Status: "decoded", Evidence: "confirmed", GlyphCount: font.GlyphCount(),
		CellWidth: fdtxt.GlyphWidth, CellHeight: fdtxt.GlyphHeight,
		Columns: columns, Rows: rows, Atlas: "atlas.png",
		Source: sourceID{File: "FDOTHER.DAT", Resource: 4, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(raw)},
	}
	return writeJSON(filepath.Join(directory, "font.json"), document)
}

func exportItemPanel(fdotherPath, outputRoot string) error {
	raw, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		return err
	}
	if len(raw) != 44181 {
		return fmt.Errorf("FDOTHER #5 raw size=%d, want 44181", len(raw))
	}
	directory := filepath.Join(outputRoot, "ui", "fdother_005_item_panel")
	document := itemPanelDocument{
		SchemaVersion: 1, Kind: "fdother_lmi1_item_panel", AssetID: "ui/FDOTHER_005/item_panel",
		Status: "decoded", Evidence: "confirmed",
		Source: sourceID{File: "FDOTHER.DAT", Resource: 5, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(raw)},
	}
	for _, index := range []int{22} {
		entry, err := fdother.ParseLMI1OpaqueEntry(raw, index)
		if err != nil {
			return err
		}
		metadata := itemPanelEntryDocument{Index: index, Codec: "opaque_high_run", Width: entry.Width, Height: entry.Height, Frame: fmt.Sprintf("entry_%03d/frame.png", index)}
		if err := writeIndexedPNG(filepath.Join(directory, metadata.Frame), entry.Width, entry.Height, entry.Pixels); err != nil {
			return err
		}
		document.Entries = append(document.Entries, metadata)
	}
	rawIndexes := make([]int, 0, 43)
	for index := 0; index <= 19; index++ {
		rawIndexes = append(rawIndexes, index)
	}
	rawIndexes = append(rawIndexes, 23, 24, 25, 26, 27, 28, 29, 30, 53, 54, 55, 56, 57, 59, 60, 61, 62, 63, 64, 65, 66, 67, 92)
	for _, index := range rawIndexes {
		entry, err := fdother.ParseLMI1RawEntry(raw, index)
		if err != nil {
			return err
		}
		metadata := itemPanelEntryDocument{Index: index, Codec: "raw_indexed_opaque", Width: entry.Width, Height: entry.Height, Frame: fmt.Sprintf("entry_%03d/frame.png", index)}
		if err := writeIndexedPNG(filepath.Join(directory, metadata.Frame), entry.Width, entry.Height, entry.Pixels); err != nil {
			return err
		}
		document.Entries = append(document.Entries, metadata)
	}
	frameIndexes := make([]int, 0, 34)
	for index := 31; index <= 52; index++ {
		frameIndexes = append(frameIndexes, index)
	}
	frameIndexes = append(frameIndexes, 93)
	for index := 119; index <= 129; index++ {
		frameIndexes = append(frameIndexes, index)
	}
	for _, index := range frameIndexes {
		entry, err := fdother.ParseLMI1FrameEntry(raw, index)
		if err != nil {
			return err
		}
		indexed, mask, err := entry.IndexedLayers()
		if err != nil {
			return err
		}
		entryDirectory := filepath.Join(directory, fmt.Sprintf("entry_%03d", index))
		if err := writeSurfacePNGs(entryDirectory, entry.Width, entry.Height, indexed, mask); err != nil {
			return err
		}
		document.Entries = append(document.Entries, itemPanelEntryDocument{
			Index: index, Codec: "four_mode_frame", Width: entry.Width, Height: entry.Height,
			Frame: fmt.Sprintf("entry_%03d/frame.png", index), Mask: fmt.Sprintf("entry_%03d/mask.png", index),
		})
	}
	return writeJSON(filepath.Join(directory, "resource.json"), document)
}

func exportLoadSlots(fdotherPath, outputRoot string) error {
	raw, err := fdother.ReadResource(fdotherPath, 13)
	if err != nil {
		return err
	}
	if len(raw) != 53210 {
		return fmt.Errorf("FDOTHER #13 raw size=%d, want 53210", len(raw))
	}
	entry, err := fdother.ParseLMI1OpaqueEntry(raw, 16)
	if err != nil {
		return err
	}
	if entry.Width != 310 || entry.Height != 86 {
		return fmt.Errorf("FDOTHER #13 entry16=%dx%d, want 310x86", entry.Width, entry.Height)
	}
	directory := filepath.Join(outputRoot, "ui", "fdother_013_load_slots")
	document := itemPanelDocument{
		SchemaVersion: 1,
		Kind:          "fdother_lmi1_load_slots",
		AssetID:       "ui/FDOTHER_013/load_slots",
		Status:        "decoded",
		Evidence:      "confirmed",
		Source: sourceID{
			File: "FDOTHER.DAT", Resource: 13, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(raw),
		},
		Entries: []itemPanelEntryDocument{{
			Index: 16, Codec: "opaque_high_run", Width: entry.Width,
			Height: entry.Height, Frame: "entry_016/frame.png",
		}},
	}
	if err := writeIndexedPNG(
		filepath.Join(directory, document.Entries[0].Frame),
		entry.Width, entry.Height, entry.Pixels,
	); err != nil {
		return err
	}
	return writeJSON(filepath.Join(directory, "resource.json"), document)
}

func exportChurchUI(fdotherPath, outputRoot string) error {
	raw, err := fdother.ReadResource(fdotherPath, 14)
	if err != nil {
		return err
	}
	if len(raw) != 51157 {
		return fmt.Errorf("FDOTHER #14 raw size=%d, want 51157", len(raw))
	}
	directory := filepath.Join(outputRoot, "ui", "fdother_014_church")
	document := itemPanelDocument{
		SchemaVersion: 1,
		Kind:          "fdother_lmi1_church_ui",
		AssetID:       "ui/FDOTHER_014/church",
		Status:        "decoded",
		Evidence:      "confirmed",
		Source: sourceID{
			File: "FDOTHER.DAT", Resource: 14, Size: fdotherSize,
			MD5: fdotherMD5, SHA256: fdotherSHA256, RawSize: len(raw),
		},
	}
	for _, index := range []int{1, 3, 4, 5, 6, 7, 8, 9, 10, 16} {
		entry, err := fdother.ParseLMI1OpaqueEntry(raw, index)
		if err != nil {
			return err
		}
		metadata := itemPanelEntryDocument{
			Index: index, Codec: "opaque_high_run", Width: entry.Width,
			Height: entry.Height, Frame: fmt.Sprintf("entry_%03d/frame.png", index),
		}
		if err := writeIndexedPNG(
			filepath.Join(directory, metadata.Frame), entry.Width, entry.Height, entry.Pixels,
		); err != nil {
			return err
		}
		document.Entries = append(document.Entries, metadata)
	}
	priceCell, err := fdother.ParseLMI1RawEntry(raw, 15)
	if err != nil {
		return err
	}
	priceMetadata := itemPanelEntryDocument{
		Index: 15, Codec: "raw_indexed_opaque", Width: priceCell.Width,
		Height: priceCell.Height, Frame: "entry_015/frame.png",
	}
	if err := writeIndexedPNG(
		filepath.Join(directory, priceMetadata.Frame),
		priceCell.Width, priceCell.Height, priceCell.Pixels,
	); err != nil {
		return err
	}
	document.Entries = append(document.Entries, priceMetadata)
	for _, index := range []int{0, 23, 24, 25, 26, 27, 28, 29, 30, 31} {
		entry, err := fdother.ParseLMI1FrameEntry(raw, index)
		if err != nil {
			return err
		}
		indexed, mask, err := entry.IndexedLayers()
		if err != nil {
			return err
		}
		entryDirectory := filepath.Join(directory, fmt.Sprintf("entry_%03d", index))
		if err := writeSurfacePNGs(entryDirectory, entry.Width, entry.Height, indexed, mask); err != nil {
			return err
		}
		document.Entries = append(document.Entries, itemPanelEntryDocument{
			Index: index, Codec: "four_mode_frame", Width: entry.Width, Height: entry.Height,
			Frame: fmt.Sprintf("entry_%03d/frame.png", index),
			Mask:  fmt.Sprintf("entry_%03d/mask.png", index),
		})
	}
	return writeJSON(filepath.Join(directory, "resource.json"), document)
}

func writeIndexedPNG(path string, width, height int, pixels []byte) error {
	if width <= 0 || height <= 0 || len(pixels) != width*height {
		return fmt.Errorf("indexed PNG geometry mismatch")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	palette := make(color.Palette, 256)
	for index := range palette {
		palette[index] = color.RGBA{uint8(index), uint8(index), uint8(index), 0xff}
	}
	output := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	copy(output.Pix, pixels)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(file, output)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func exportFDICON(path, outputRoot string) error {
	if err := verifyArchive(path, fdiconArchive); err != nil {
		return err
	}
	bank, err := fdicon.DecodeFile(path)
	if err != nil {
		return err
	}
	if len(bank.Sprites) != fdicon.SeparatedSpriteCount {
		return fmt.Errorf("FDICON.B24 sprite count=%d, want %d", len(bank.Sprites), fdicon.SeparatedSpriteCount)
	}
	directory := filepath.Join(outputRoot, "sprites", "fdicon")
	document := fdiconBankDocument{
		SchemaVersion: 1, Kind: "fdicon_sprite_bank", AssetID: "sprites/fdicon",
		Status: "decoded", Evidence: "confirmed",
		Source:  sourceID{File: fdiconArchive.file, Size: fdiconArchive.size, MD5: fdiconArchive.md5, SHA256: fdiconArchive.sha256, RawSize: fdiconArchive.size},
		Sprites: make([]fdiconSpriteDocument, 0, len(bank.Sprites)),
	}
	for index, sprite := range bank.Sprites {
		if len(sprite.Pixels) != fdicon.NativeSize*fdicon.NativeSize ||
			len(sprite.Mask) != len(sprite.Pixels) || len(sprite.RemapMask) != len(sprite.Pixels) {
			return fmt.Errorf("FDICON.B24 sprite %d has incomplete decoded layers", index)
		}
		prefix := fmt.Sprintf("sprite_%04d", index)
		if err := writeIndexedPNG(filepath.Join(directory, prefix, "frame.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.Pixels); err != nil {
			return err
		}
		if err := writeBinaryMaskPNG(filepath.Join(directory, prefix, "mask.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.Mask); err != nil {
			return err
		}
		if err := writeBinaryMaskPNG(filepath.Join(directory, prefix, "remap_mask.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.RemapMask); err != nil {
			return err
		}
		document.Sprites = append(document.Sprites, fdiconSpriteDocument{
			Index: index, Width: fdicon.NativeSize, Height: fdicon.NativeSize,
			Frame: prefix + "/frame.png", Mask: prefix + "/mask.png", RemapMask: prefix + "/remap_mask.png",
		})
	}
	return writeJSON(filepath.Join(directory, "bank.json"), document)
}

func exportFDSHAP(path, outputRoot string) error {
	if err := verifyArchive(path, fdshapArchive); err != nil {
		return err
	}
	for mapIndex := 0; mapIndex < fdicon.SeparatedFDSHAPMapCount; mapIndex++ {
		bank, controls, err := fdother.DecodeMapTerrainResources(path, mapIndex)
		if err != nil {
			return fmt.Errorf("FDSHAP map %d: %w", mapIndex, err)
		}
		directory := filepath.Join(outputRoot, "tilesets", "fdshap", fmt.Sprintf("map_%02d", mapIndex))
		controlValues := make([]int, len(controls))
		for index, value := range controls {
			controlValues[index] = int(value)
		}
		document := fdshapBankDocument{
			SchemaVersion: 1, Kind: "fdshap_terrain_bank",
			AssetID: fmt.Sprintf("tilesets/fdshap/map_%02d", mapIndex),
			Status:  "decoded", Evidence: "confirmed",
			Source:   sourceID{File: fdshapArchive.file, Size: fdshapArchive.size, MD5: fdshapArchive.md5, SHA256: fdshapArchive.sha256, RawSize: fdshapArchive.size},
			MapIndex: mapIndex, ImageResource: mapIndex * 2, ControlResource: mapIndex*2 + 1,
			Controls: controlValues, Sprites: make([]fdiconSpriteDocument, 0, len(bank.Sprites)),
		}
		for index, sprite := range bank.Sprites {
			if len(sprite.Pixels) != fdicon.NativeSize*fdicon.NativeSize || len(sprite.Mask) != len(sprite.Pixels) || len(sprite.RemapMask) != len(sprite.Pixels) {
				return fmt.Errorf("FDSHAP map %d tile %d has incomplete layers", mapIndex, index)
			}
			prefix := fmt.Sprintf("tile_%04d", index)
			if err := writeIndexedPNG(filepath.Join(directory, prefix, "frame.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.Pixels); err != nil {
				return err
			}
			if err := writeBinaryMaskPNG(filepath.Join(directory, prefix, "mask.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.Mask); err != nil {
				return err
			}
			if err := writeBinaryMaskPNG(filepath.Join(directory, prefix, "remap_mask.png"), fdicon.NativeSize, fdicon.NativeSize, sprite.RemapMask); err != nil {
				return err
			}
			document.Sprites = append(document.Sprites, fdiconSpriteDocument{Index: index, Width: fdicon.NativeSize, Height: fdicon.NativeSize, Frame: prefix + "/frame.png", Mask: prefix + "/mask.png", RemapMask: prefix + "/remap_mask.png"})
		}
		if err := writeJSON(filepath.Join(directory, "bank.json"), document); err != nil {
			return err
		}
	}
	return nil
}

func bytesToInts(raw []byte) []int {
	values := make([]int, len(raw))
	for index, value := range raw {
		values[index] = int(value)
	}
	return values
}

func exportFDFIELDSelector30(path, outputRoot string) error {
	if err := verifyArchive(path, fdfieldArchive); err != nil {
		return err
	}
	resources := make([][]byte, 3)
	for offset := range resources {
		resource, err := fdother.ReadResource(path, 90+offset)
		if err != nil {
			return fmt.Errorf("FDFIELD #%d: %w", 90+offset, err)
		}
		resources[offset] = resource
	}
	field, control, positions := resources[0], resources[1], resources[2]
	if len(field) < 4 {
		return errors.New("FDFIELD #90 is too short")
	}
	width, height := int(binary.LittleEndian.Uint16(field)), int(binary.LittleEndian.Uint16(field[2:]))
	if width != 35 || height != 45 || len(field) != 4+width*height*4 {
		return fmt.Errorf("FDFIELD #90 geometry=%dx%d size=%d", width, height, len(field))
	}
	doc := fdfieldSelectorDocument{
		SchemaVersion: 1, Kind: "fdfield_selector", DocumentID: "field/fdfield/selector_30",
		Status: "decoded", Evidence: "confirmed",
		Source:      sourceID{File: fdfieldArchive.file, Resource: 90, Size: fdfieldArchive.size, MD5: fdfieldArchive.md5, SHA256: fdfieldArchive.sha256, RawSize: len(field)},
		MapResource: 90, ControlResource: 91, PositionsResource: 92,
		Width: width, Height: height, Tiles: make([]int, width*height), EventBytes: make([]int, width*height), BlitModes: make([]int, width*height),
		ControlBytes: bytesToInts(control),
	}
	for index := range doc.Tiles {
		offset := 4 + index*4
		doc.Tiles[index] = int(binary.LittleEndian.Uint16(field[offset:]))
		doc.EventBytes[index] = int(field[offset+2])
		doc.BlitModes[index] = int(field[offset+3])
	}
	if len(positions) < 2 || len(positions) != 2+int(binary.LittleEndian.Uint16(positions))*6 {
		return fmt.Errorf("FDFIELD #92 position size=%d", len(positions))
	}
	for offset := 2; offset < len(positions); offset += 6 {
		doc.Positions = append(doc.Positions, fdfieldPositionDocument{
			XWord:  int(binary.LittleEndian.Uint16(positions[offset:])),
			YWord:  int(binary.LittleEndian.Uint16(positions[offset+2:])),
			RawKey: int(binary.LittleEndian.Uint16(positions[offset+4:])),
		})
	}
	mapHash, controlHash, positionsHash := sha256.Sum256(field), sha256.Sum256(control), sha256.Sum256(positions)
	doc.MapSHA256, doc.ControlSHA256, doc.PositionsSHA256 = hex.EncodeToString(mapHash[:]), hex.EncodeToString(controlHash[:]), hex.EncodeToString(positionsHash[:])
	return writeJSON(filepath.Join(outputRoot, "fields", "fdfield", "selector_30", "field.json"), doc)
}

func writeBinaryMaskPNG(path string, width, height int, mask []byte) error {
	if width <= 0 || height <= 0 || len(mask) != width*height {
		return errors.New("binary mask PNG geometry mismatch")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	output := image.NewGray(image.Rect(0, 0, width, height))
	for index, value := range mask {
		if value != 0 && value != 1 {
			return errors.New("binary mask contains a non-binary value")
		}
		if value != 0 {
			output.Pix[index] = 0xff
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(file, output)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func main() {
	fdotherPath := flag.String("fdother", "", "固定版本 FDOTHER.DAT 路徑")
	bgPath := flag.String("bg", "", "固定版本 BG.DAT 路徑")
	taiPath := flag.String("tai", "", "固定版本 TAI.DAT 路徑")
	fdtxtPath := flag.String("fdtxt", "", "固定版本 FDTXT.DAT 路徑")
	fdiconPath := flag.String("fdicon", "", "固定版本 FDICON.B24 路徑")
	fdshapPath := flag.String("fdshap", "", "固定版本 FDSHAP.DAT 路徑")
	fdfieldPath := flag.String("fdfield", "", "固定版本 FDFIELD.DAT 路徑")
	aniPath := flag.String("ani", "", "固定版本 ANI.DAT 路徑")
	glyphMapPath := flag.String("glyph-map", "", "受版控 glyph_map.json 路徑")
	outputRoot := flag.String("out", "", "分離素材包根目錄")
	flag.Parse()
	if *outputRoot == "" || (*fdotherPath == "" && *bgPath == "" && *taiPath == "" && *fdtxtPath == "" && *fdiconPath == "" && *fdshapPath == "" && *fdfieldPath == "" && *aniPath == "") || (*fdtxtPath != "" && *glyphMapPath == "") {
		fmt.Fprintln(os.Stderr, "用法：fd2-asset-import [-fdother FDOTHER.DAT] [-bg BG.DAT] [-tai TAI.DAT] [-fdtxt FDTXT.DAT -glyph-map glyph_map.json] [-fdicon FDICON.B24] [-fdshap FDSHAP.DAT] [-fdfield FDFIELD.DAT] [-ani ANI.DAT] -out ASSET_PACK")
		os.Exit(2)
	}
	if *fdotherPath != "" {
		if err := exportCommandGrid(*fdotherPath, *outputRoot); err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Println("已匯出戰場指令格：256色調色盤＋78張 PNG")
	}
	for _, request := range []struct {
		path     string
		identity archiveIdentity
	}{{*bgPath, surfaceArchives[0]}, {*taiPath, surfaceArchives[1]}} {
		if request.path == "" {
			continue
		}
		decoded, blocked, err := exportSingleFrameArchive(request.path, *outputRoot, request.identity)
		if err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 %s：decoded=%d blocked=%d\n", request.identity.file, decoded, blocked)
	}
	if *fdtxtPath != "" {
		decoded, blocked, err := exportFDTXT(*fdtxtPath, *glyphMapPath, *outputRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 FDTXT.DAT：decoded=%d blocked=%d\n", decoded, blocked)
	}
	if *fdiconPath != "" {
		if err := exportFDICON(*fdiconPath, *outputRoot); err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 FDICON.B24：%d 張 indexed frame＋mask＋remap mask\n", fdicon.SeparatedSpriteCount)
	}
	if *fdshapPath != "" {
		if err := exportFDSHAP(*fdshapPath, *outputRoot); err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 FDSHAP.DAT：%d 張地圖的 indexed tile＋mask＋remap mask＋control\n", fdicon.SeparatedFDSHAPMapCount)
	}
	if *fdfieldPath != "" {
		if err := exportFDFIELDSelector30(*fdfieldPath, *outputRoot); err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Println("已匯出 FDFIELD.DAT selector 30：地圖、控制列與 32 筆位置列")
	}
	if *aniPath != "" {
		frames, err := exportANI(*aniPath, *outputRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, "匯入失敗：", err)
			os.Exit(1)
		}
		fmt.Printf("已匯出 ANI.DAT：9 組動畫、%d 幀 indexed PNG＋六位元 DAC\n", frames)
	}
}
