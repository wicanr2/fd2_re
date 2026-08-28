package fdtxt

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	fdtxtSourceSize      = 120502
	fdtxtSourceMD5       = "fe5c487ce4313485f1da9d48d35b05f9"
	fdtxtSourceSHA256    = "a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69"
	fdotherSourceSize    = 3382481
	fdotherSourceMD5     = "22f56e5027edc7c766ad34ca4e5aca93"
	fdotherSourceSHA256  = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
	separatedFontGlyphs  = 1824
	separatedFontColumns = 32
)

type separatedSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedToken struct {
	Kind       string `json:"kind"`
	GlyphIndex *int   `json:"glyph_index,omitempty"`
	Control    string `json:"control,omitempty"`
	Text       string `json:"text,omitempty"`
}

type separatedString struct {
	StringID    string           `json:"string_id"`
	SourceIndex int              `json:"source_index"`
	Text        string           `json:"text"`
	Tokens      []separatedToken `json:"tokens"`
}

type separatedResource struct {
	SchemaVersion int               `json:"schema_version"`
	Kind          string            `json:"kind"`
	AssetID       string            `json:"asset_id"`
	Status        string            `json:"status"`
	Evidence      string            `json:"evidence"`
	ReasonCode    string            `json:"reason_code,omitempty"`
	Source        separatedSource   `json:"source"`
	Strings       []separatedString `json:"strings,omitempty"`
}

type separatedFontDocument struct {
	SchemaVersion int             `json:"schema_version"`
	Kind          string          `json:"kind"`
	AssetID       string          `json:"asset_id"`
	Status        string          `json:"status"`
	Evidence      string          `json:"evidence"`
	GlyphCount    int             `json:"glyph_count"`
	CellWidth     int             `json:"cell_width"`
	CellHeight    int             `json:"cell_height"`
	Columns       int             `json:"columns"`
	Rows          int             `json:"rows"`
	Atlas         string          `json:"atlas"`
	Source        separatedSource `json:"source"`
}

// LoadSeparatedResource loads the lossless editable token projection emitted
// by fd2-asset-import. It never falls back to FDTXT.DAT.
func LoadSeparatedResource(root string, resource int) (*Strings, error) {
	if root == "" || resource < 0 {
		return nil, fmt.Errorf("fdtxt: invalid separated resource request")
	}
	path := filepath.Join(root, fmt.Sprintf("FDTXT_%03d", resource), "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fdtxt: separated resource %d: %w", resource, err)
	}
	var document separatedResource
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdtxt: separated resource %d JSON: %w", resource, err)
	}
	wantAssetID := fmt.Sprintf("text/FDTXT_%03d", resource)
	if document.SchemaVersion != 1 || document.Kind != "fdtxt_word_table" ||
		document.AssetID != wantAssetID || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDTXT.DAT" ||
		document.Source.Resource != resource || document.Source.Size != fdtxtSourceSize ||
		document.Source.MD5 != fdtxtSourceMD5 || document.Source.SHA256 != fdtxtSourceSHA256 ||
		document.Source.RawSize <= 0 || len(document.Strings) == 0 {
		return nil, fmt.Errorf("fdtxt: separated resource %d metadata mismatch", resource)
	}
	words := make([][]uint16, len(document.Strings))
	for index, entry := range document.Strings {
		wantID := fmt.Sprintf("FDTXT_%03d/string_%04d", resource, index)
		if entry.SourceIndex != index || entry.StringID != wantID || entry.Tokens == nil {
			return nil, fmt.Errorf("fdtxt: separated resource %d string %d identity mismatch", resource, index)
		}
		var projection strings.Builder
		for tokenIndex, token := range entry.Tokens {
			switch token.Kind {
			case "glyph":
				if token.GlyphIndex == nil || *token.GlyphIndex < 0 || *token.GlyphIndex >= ControlMin || token.Control != "" || token.Text == "" {
					return nil, fmt.Errorf("fdtxt: resource %d string %d token %d invalid glyph", resource, index, tokenIndex)
				}
				words[index] = append(words[index], uint16(*token.GlyphIndex))
				projection.WriteString(token.Text)
			case "control":
				if token.GlyphIndex != nil || token.Text != "" || len(token.Control) != 4 {
					return nil, fmt.Errorf("fdtxt: resource %d string %d token %d invalid control", resource, index, tokenIndex)
				}
				value, err := strconv.ParseUint(token.Control, 16, 16)
				if err != nil || value < ControlMin || value >= StringEnd {
					return nil, fmt.Errorf("fdtxt: resource %d string %d token %d invalid control", resource, index, tokenIndex)
				}
				words[index] = append(words[index], uint16(value))
			default:
				return nil, fmt.Errorf("fdtxt: resource %d string %d token %d unknown kind", resource, index, tokenIndex)
			}
		}
		if projection.String() != entry.Text {
			return nil, fmt.Errorf("fdtxt: separated resource %d string %d UTF-8 projection mismatch", resource, index)
		}
	}
	return &Strings{words: words}, nil
}

// LoadSeparatedFont reconstructs FDOTHER#4's exact 1bpp glyph bank from the
// standard atlas. The PNG is a transport format; only binary 0/255 pixels are
// admitted, so an editor cannot silently introduce antialiasing into the
// original renderer contract.
func LoadSeparatedFont(root string) (*Font, error) {
	if root == "" {
		return nil, fmt.Errorf("fdtxt: separated font root is empty")
	}
	directory := filepath.Join(root, "fdother_004")
	raw, err := os.ReadFile(filepath.Join(directory, "font.json"))
	if err != nil {
		return nil, fmt.Errorf("fdtxt: separated font metadata: %w", err)
	}
	var document separatedFontDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdtxt: separated font JSON: %w", err)
	}
	wantRows := (separatedFontGlyphs + separatedFontColumns - 1) / separatedFontColumns
	if document.SchemaVersion != 1 || document.Kind != "bitmap_font" ||
		document.AssetID != "font/FDOTHER_004" || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.GlyphCount != separatedFontGlyphs ||
		document.CellWidth != GlyphWidth || document.CellHeight != GlyphHeight ||
		document.Columns != separatedFontColumns || document.Rows != wantRows ||
		document.Atlas != "atlas.png" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != 4 || document.Source.Size != fdotherSourceSize ||
		document.Source.MD5 != fdotherSourceMD5 || document.Source.SHA256 != fdotherSourceSHA256 ||
		document.Source.RawSize != separatedFontGlyphs*GlyphBytes {
		return nil, fmt.Errorf("fdtxt: separated font metadata mismatch")
	}
	file, err := os.Open(filepath.Join(directory, document.Atlas))
	if err != nil {
		return nil, fmt.Errorf("fdtxt: separated font atlas: %w", err)
	}
	atlas, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, fmt.Errorf("fdtxt: separated font atlas decode: %w", decodeErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if atlas.Bounds().Dx() != document.Columns*GlyphWidth || atlas.Bounds().Dy() != document.Rows*GlyphHeight {
		return nil, fmt.Errorf("fdtxt: separated font atlas geometry mismatch")
	}
	packed := make([]byte, document.GlyphCount*GlyphBytes)
	for glyph := 0; glyph < document.GlyphCount; glyph++ {
		baseX := (glyph % document.Columns) * GlyphWidth
		baseY := (glyph / document.Columns) * GlyphHeight
		for y := 0; y < GlyphHeight; y++ {
			var row uint16
			for x := 0; x < GlyphWidth; x++ {
				r, g, b, _ := atlas.At(baseX+x, baseY+y).RGBA()
				if (r != 0 || g != 0 || b != 0) && (r != 0xffff || g != 0xffff || b != 0xffff) {
					return nil, fmt.Errorf("fdtxt: separated font atlas contains a non-binary pixel")
				}
				if r == 0xffff {
					row |= uint16(0x8000) >> x
				}
			}
			packed[glyph*GlyphBytes+y*2] = byte(row >> 8)
			packed[glyph*GlyphBytes+y*2+1] = byte(row)
		}
	}
	return ParseFont(packed)
}
