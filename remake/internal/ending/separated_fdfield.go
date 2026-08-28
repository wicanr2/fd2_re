package ending

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	separatedFDFIELDSize      = 243169
	separatedFDFIELDMD5       = "ecdb0436d26adfe5d107f2713fa7e9a2"
	separatedFDFIELDSHA256    = "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513"
	selector30MapSHA256       = "fd44ebb3f269a45e16f84e3ea1ec3458aa672d7c85caa5ffc4dd9c628db168d3"
	selector30ControlSHA256   = "c17581097e0beb63ff892ea6acf0b50a9ab5e62aa94ceab560c46a6f2ede625a"
	selector30PositionsSHA256 = "371a43900435ff22887a94dc6c96341585e73d933beaf23fd2b6e77b6d3dc4f9"
)

type separatedFDFIELDSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type separatedFDFIELDPosition struct {
	XWord  int `json:"x_word"`
	YWord  int `json:"y_word"`
	RawKey int `json:"raw_key"`
}

type separatedFDFIELDDocument struct {
	SchemaVersion     int                        `json:"schema_version"`
	Kind              string                     `json:"kind"`
	DocumentID        string                     `json:"document_id"`
	Status            string                     `json:"status"`
	Evidence          string                     `json:"evidence"`
	Source            separatedFDFIELDSource     `json:"source"`
	MapResource       int                        `json:"map_resource"`
	ControlResource   int                        `json:"control_resource"`
	PositionsResource int                        `json:"positions_resource"`
	MapSHA256         string                     `json:"map_sha256"`
	ControlSHA256     string                     `json:"control_sha256"`
	PositionsSHA256   string                     `json:"positions_sha256"`
	Width             int                        `json:"width"`
	Height            int                        `json:"height"`
	Tiles             []int                      `json:"tiles"`
	EventBytes        []int                      `json:"event_bytes"`
	BlitModes         []int                      `json:"blit_modes"`
	ControlBytes      []int                      `json:"control_bytes"`
	Positions         []separatedFDFIELDPosition `json:"positions"`
}

func hashBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func loadSeparatedFDFIELDSelector30(root string) ([]byte, []byte, []byte, error) {
	if root == "" {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD root is unavailable")
	}
	raw, err := os.ReadFile(filepath.Join(root, "selector_30", "field.json"))
	if err != nil {
		return nil, nil, nil, err
	}
	var doc separatedFDFIELDDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, nil, err
	}
	if doc.SchemaVersion != 1 || doc.Kind != "fdfield_selector" || doc.DocumentID != "field/fdfield/selector_30" || doc.Status != "decoded" || doc.Evidence != "confirmed" {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 identity is invalid")
	}
	if doc.Source.File != "FDFIELD.DAT" || doc.Source.Resource != 90 || doc.Source.Size != separatedFDFIELDSize || doc.Source.MD5 != separatedFDFIELDMD5 || doc.Source.SHA256 != separatedFDFIELDSHA256 || doc.Source.RawSize != 6304 ||
		doc.MapResource != 90 || doc.ControlResource != 91 || doc.PositionsResource != 92 ||
		doc.MapSHA256 != selector30MapSHA256 || doc.ControlSHA256 != selector30ControlSHA256 || doc.PositionsSHA256 != selector30PositionsSHA256 {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 provenance is invalid")
	}
	if doc.Width != 35 || doc.Height != 45 {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 geometry=%dx%d", doc.Width, doc.Height)
	}
	cells := doc.Width * doc.Height
	if len(doc.Tiles) != cells || len(doc.EventBytes) != cells || len(doc.BlitModes) != cells {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 cell arrays are incomplete")
	}
	fieldMap := make([]byte, 4+cells*4)
	binary.LittleEndian.PutUint16(fieldMap, uint16(doc.Width))
	binary.LittleEndian.PutUint16(fieldMap[2:], uint16(doc.Height))
	for index := 0; index < cells; index++ {
		if doc.Tiles[index] < 0 || doc.Tiles[index] > 0xffff || doc.EventBytes[index] < 0 || doc.EventBytes[index] > 0xff || doc.BlitModes[index] < 0 || doc.BlitModes[index] > 0xff {
			return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 cell %d is outside raw range", index)
		}
		offset := 4 + index*4
		binary.LittleEndian.PutUint16(fieldMap[offset:], uint16(doc.Tiles[index]))
		fieldMap[offset+2] = byte(doc.EventBytes[index])
		fieldMap[offset+3] = byte(doc.BlitModes[index])
	}
	control := make([]byte, len(doc.ControlBytes))
	for index, value := range doc.ControlBytes {
		if value < 0 || value > 0xff {
			return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 control byte %d is outside raw range", index)
		}
		control[index] = byte(value)
	}
	positions := make([]byte, 2+len(doc.Positions)*6)
	binary.LittleEndian.PutUint16(positions, uint16(len(doc.Positions)))
	for index, row := range doc.Positions {
		if row.XWord < 0 || row.XWord > 0xffff || row.YWord < 0 || row.YWord > 0xffff || row.RawKey < 0 || row.RawKey > 0xffff {
			return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 position %d is outside raw range", index)
		}
		offset := 2 + index*6
		binary.LittleEndian.PutUint16(positions[offset:], uint16(row.XWord))
		binary.LittleEndian.PutUint16(positions[offset+2:], uint16(row.YWord))
		binary.LittleEndian.PutUint16(positions[offset+4:], uint16(row.RawKey))
	}
	if hashBytes(fieldMap) != doc.MapSHA256 || hashBytes(control) != doc.ControlSHA256 || hashBytes(positions) != doc.PositionsSHA256 {
		return nil, nil, nil, fmt.Errorf("ending: separated FDFIELD selector 30 content hash mismatch")
	}
	return fieldMap, control, positions, nil
}
