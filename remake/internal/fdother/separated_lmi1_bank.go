package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type separatedLMI1BankSource struct {
	File      string `json:"file"`
	Resource  int    `json:"resource"`
	Size      int    `json:"size"`
	MD5       string `json:"md5"`
	SHA256    string `json:"sha256"`
	RawSize   int    `json:"raw_size"`
	RawMD5    string `json:"raw_md5"`
	RawSHA256 string `json:"raw_sha256"`
}

type separatedLMI1BankEntry struct {
	Index  int    `json:"index"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Frame  string `json:"frame"`
}

type separatedLMI1BankDocument struct {
	SchemaVersion int                      `json:"schema_version"`
	Kind          string                   `json:"kind"`
	AssetID       string                   `json:"asset_id"`
	Status        string                   `json:"status"`
	Evidence      string                   `json:"evidence"`
	Source        separatedLMI1BankSource  `json:"source"`
	Entries       []separatedLMI1BankEntry `json:"entries"`
}

type separatedLMI1BankContract struct {
	assetID, label                string
	resource, rawSize, entryCount int
	rawMD5, rawSHA256             string
}

var separatedLMI1BankContracts = map[int]separatedLMI1BankContract{
	5: {assetID: "ui/FDOTHER_005/lmi1_opaque", label: "FDOTHER #5 LMI1", resource: 5,
		rawSize: 44181, entryCount: 138, rawMD5: "646bfc938f3f459ed54a8af34909ee47",
		rawSHA256: "561eb8ca579b13a6f2bc1f36436ad3af7d8d0bf77eab25a4ab6411142a7eb118"},
	6: {assetID: "effects/FDOTHER_006/lmi1_opaque", label: "FDOTHER #6 LMI1", resource: 6,
		rawSize: 33415, entryCount: 230, rawMD5: "19e0e00500eb0ce41739f17615238cc3",
		rawSHA256: "47cc4f7136b553879f960e9902d223c9f7e4462d726fb7ea3124bf51c912f71c"},
	9: {assetID: "animation/FDOTHER_009/spawn_intro", label: "FDOTHER #9 spawn intro", resource: 9,
		rawSize: 3999, entryCount: 12, rawMD5: "a20e79010a333abaab060f9199eb6d7c",
		rawSHA256: "0eeaeb88320287e0c60c530c9d3a2592fea54378ebe5d5ac6a321e4fb1de7f0b"},
}

// LoadSeparatedLMI1Bank loads one of the exact complete LMI1 banks consumed
// by the native map and battle presenters. It never opens FDOTHER.DAT.
func LoadSeparatedLMI1Bank(root string, resource int) ([]LMI1Entry, error) {
	contract, ok := separatedLMI1BankContracts[resource]
	if !ok {
		return nil, errors.New("fdother: unsupported separated LMI1 bank")
	}
	raw, err := os.ReadFile(filepath.Join(root, "bank.json"))
	if err != nil {
		return nil, fmt.Errorf("fdother: separated %s metadata: %w", contract.label, err)
	}
	var document separatedLMI1BankDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("fdother: separated %s JSON: %w", contract.label, err)
	}
	if document.SchemaVersion != 1 || document.Kind != "fdother_lmi1_bank" ||
		document.AssetID != contract.assetID || document.Status != "decoded" ||
		document.Evidence != "confirmed" || document.Source.File != "FDOTHER.DAT" ||
		document.Source.Resource != contract.resource || document.Source.Size != 3382481 ||
		document.Source.MD5 != "22f56e5027edc7c766ad34ca4e5aca93" ||
		document.Source.SHA256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" ||
		document.Source.RawSize != contract.rawSize || document.Source.RawMD5 != contract.rawMD5 ||
		document.Source.RawSHA256 != contract.rawSHA256 || len(document.Entries) != contract.entryCount {
		return nil, fmt.Errorf("fdother: separated %s contract mismatch", contract.label)
	}
	entries := make([]LMI1Entry, contract.entryCount)
	seen := make([]bool, contract.entryCount)
	for _, entry := range document.Entries {
		if entry.Index < 0 || entry.Index >= contract.entryCount || seen[entry.Index] ||
			entry.Codec != "opaque_high_run" || entry.Width <= 0 || entry.Height <= 0 ||
			entry.Frame != fmt.Sprintf("entry_%03d/frame.png", entry.Index) {
			return nil, fmt.Errorf("fdother: invalid separated %s entry %d", contract.label, entry.Index)
		}
		pixels, err := loadItemPanelIndexedPNG(filepath.Join(root, filepath.FromSlash(entry.Frame)), entry.Width, entry.Height)
		if err != nil {
			return nil, fmt.Errorf("fdother: separated %s entry %d: %w", contract.label, entry.Index, err)
		}
		entries[entry.Index] = LMI1Entry{Width: entry.Width, Height: entry.Height, Pixels: pixels}
		seen[entry.Index] = true
	}
	for index, present := range seen {
		if !present {
			return nil, fmt.Errorf("fdother: separated %s entry %d is missing", contract.label, index)
		}
	}
	if resource == NativeSpawnIntroFrameResource {
		if err := validateNativeSpawnIntroFrames(entries); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func validateNativeSpawnIntroFrames(entries []LMI1Entry) error {
	want := [NativeSpawnIntroPassCount][2]int{
		{42, 24}, {44, 24}, {45, 24}, {46, 24}, {46, 24}, {66, 24},
		{69, 24}, {70, 24}, {69, 24}, {63, 24}, {65, 24}, {66, 24},
	}
	if len(entries) != len(want) {
		return fmt.Errorf("fdother: spawn-intro resource has %d entries, want %d", len(entries), len(want))
	}
	for index, geometry := range want {
		entry := entries[index]
		if entry.Width != geometry[0] || entry.Height != geometry[1] || len(entry.Pixels) != entry.Width*entry.Height {
			return fmt.Errorf("fdother: spawn-intro entry %d geometry differs", index)
		}
	}
	return nil
}
