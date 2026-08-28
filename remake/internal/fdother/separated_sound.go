package fdother

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	separatedFDOTHERSize   = 3382481
	separatedFDOTHERMD5    = "22f56e5027edc7c766ad34ca4e5aca93"
	separatedFDOTHERSHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
)

var separatedSoundCounts = map[int]int{
	31: 13,
	82: 2,
	83: 4,
	84: 3,
	85: 2,
	86: 2,
	87: 4,
	88: 2,
	90: 3,
}

type separatedSoundSource struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

type separatedSoundSample struct {
	Subresource     int    `json:"subresource"`
	SourceByteCount int    `json:"source_byte_count"`
	SourcePCMSHA256 string `json:"source_pcm_sha256"`
	Path            string `json:"path"`
	CueEvidence     string `json:"cue_evidence"`
}

type separatedSoundDocument struct {
	SchemaVersion       int                    `json:"schema_version"`
	Kind                string                 `json:"kind"`
	AssetID             string                 `json:"asset_id"`
	Status              string                 `json:"status"`
	Source              separatedSoundSource   `json:"source"`
	Resource            int                    `json:"resource"`
	ContainerCount      int                    `json:"container_count"`
	ZeroLengthTailIndex int                    `json:"zero_length_tail_index"`
	SampleRate          int                    `json:"sample_rate"`
	Channels            int                    `json:"channels"`
	SampleFormat        string                 `json:"sample_format"`
	TimingEvidence      string                 `json:"timing_evidence"`
	Samples             []separatedSoundSample `json:"samples"`
}

// SeparatedSoundBank 保存固定 FDOTHER 巢狀 PCM 音效庫的 OGG 編碼內容。
// 正式執行期只讀分離素材，不會回退原始 archive。
type SeparatedSoundBank struct {
	Resource int
	Encoded  map[int][]byte
}

// LoadSeparatedSoundBank 嚴格驗證一個已納入契約的FDOTHER音效庫及其所有OGG檔。
func LoadSeparatedSoundBank(sfxRoot string, resource int) (SeparatedSoundBank, error) {
	wantCount, supported := separatedSoundCounts[resource]
	if sfxRoot == "" || !supported {
		return SeparatedSoundBank{}, fmt.Errorf("fdother: unsupported separated sound request %d", resource)
	}
	bankRoot := filepath.Join(sfxRoot, fmt.Sprintf("FDOTHER_%03d", resource))
	raw, err := os.ReadFile(filepath.Join(bankRoot, "resource.json"))
	if err != nil {
		return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound metadata %d: %w", resource, err)
	}
	var doc separatedSoundDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound metadata %d: %w", resource, err)
	}
	if doc.SchemaVersion != 1 || doc.Kind != "fd2_pcm_sound_bank" ||
		doc.AssetID != fmt.Sprintf("sfx/FDOTHER_%03d", resource) || doc.Status != "converted" ||
		doc.Source.Name != "FDOTHER.DAT" || doc.Source.Size != separatedFDOTHERSize ||
		doc.Source.MD5 != separatedFDOTHERMD5 || doc.Source.SHA256 != separatedFDOTHERSHA256 ||
		doc.Resource != resource || doc.ContainerCount != wantCount+1 ||
		doc.ZeroLengthTailIndex != wantCount || doc.SampleRate != 11025 || doc.Channels != 1 ||
		doc.SampleFormat != "unsigned_u8" || doc.TimingEvidence != "hardware-spec_approximation" ||
		len(doc.Samples) != wantCount {
		return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d contract mismatch", resource)
	}
	bank := SeparatedSoundBank{Resource: resource, Encoded: make(map[int][]byte, wantCount)}
	for _, sample := range doc.Samples {
		digest, digestErr := hex.DecodeString(sample.SourcePCMSHA256)
		if sample.Subresource < 0 || sample.Subresource >= wantCount || sample.SourceByteCount <= 0 ||
			digestErr != nil || len(digest) != 32 || sample.CueEvidence == "" {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d invalid sample metadata", resource)
		}
		if _, duplicate := bank.Encoded[sample.Subresource]; duplicate {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d duplicate subresource %d", resource, sample.Subresource)
		}
		wantPath := fmt.Sprintf("sample_%03d.ogg", sample.Subresource)
		if sample.Path != wantPath || filepath.Base(sample.Path) != sample.Path || strings.Contains(sample.Path, "..") {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d unsafe sample path", resource)
		}
		encoded, readErr := os.ReadFile(filepath.Join(bankRoot, sample.Path))
		if readErr != nil {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d subresource %d: %w", resource, sample.Subresource, readErr)
		}
		if len(encoded) < 4 || string(encoded[:4]) != "OggS" {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d subresource %d is not OGG", resource, sample.Subresource)
		}
		bank.Encoded[sample.Subresource] = encoded
	}
	for index := 0; index < wantCount; index++ {
		if len(bank.Encoded[index]) == 0 {
			return SeparatedSoundBank{}, fmt.Errorf("fdother: separated sound %d missing subresource %d", resource, index)
		}
	}
	return bank, nil
}
