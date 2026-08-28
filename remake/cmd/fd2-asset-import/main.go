// fd2-asset-import 將玩家合法持有的原版資源轉成正式 runtime 可消費的分離素材。
// 原版 archive reader 只存在於這個離線命令，不應被遊戲 renderer 呼叫。
package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

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

type sourceID struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type surfaceDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	AssetID       string   `json:"asset_id"`
	Status        string   `json:"status"`
	Codec         string   `json:"codec,omitempty"`
	Width         int      `json:"width,omitempty"`
	Height        int      `json:"height,omitempty"`
	Frame         string   `json:"frame,omitempty"`
	Mask          string   `json:"mask,omitempty"`
	Evidence      string   `json:"evidence"`
	ReasonCode    string   `json:"reason_code,omitempty"`
	Source        sourceID `json:"source"`
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

func verifyFDOTHER(path string) error {
	return verifyArchive(path, archiveIdentity{file: "FDOTHER.DAT", size: fdotherSize, md5: fdotherMD5, sha256: fdotherSHA256})
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
	}
	if err := exportFont(fdotherPath, outputRoot); err != nil {
		return err
	}
	return exportItemPanel(fdotherPath, outputRoot)
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
	for _, index := range []int{23, 24, 25, 26, 27, 28, 29, 30, 53, 54, 55, 56, 57, 59, 60, 61, 62, 63, 64, 65, 66, 67, 92} {
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

func main() {
	fdotherPath := flag.String("fdother", "", "固定版本 FDOTHER.DAT 路徑")
	bgPath := flag.String("bg", "", "固定版本 BG.DAT 路徑")
	taiPath := flag.String("tai", "", "固定版本 TAI.DAT 路徑")
	fdtxtPath := flag.String("fdtxt", "", "固定版本 FDTXT.DAT 路徑")
	glyphMapPath := flag.String("glyph-map", "", "受版控 glyph_map.json 路徑")
	outputRoot := flag.String("out", "", "分離素材包根目錄")
	flag.Parse()
	if *outputRoot == "" || (*fdotherPath == "" && *bgPath == "" && *taiPath == "" && *fdtxtPath == "") || (*fdtxtPath != "" && *glyphMapPath == "") {
		fmt.Fprintln(os.Stderr, "用法：fd2-asset-import [-fdother FDOTHER.DAT] [-bg BG.DAT] [-tai TAI.DAT] [-fdtxt FDTXT.DAT -glyph-map glyph_map.json] -out ASSET_PACK")
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
}
