package fdother

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeSeparatedSoundFixture(t *testing.T, resource int) string {
	t.Helper()
	count := separatedSoundCounts[resource]
	root := t.TempDir()
	bankRoot := filepath.Join(root, "FDOTHER_082")
	if err := os.MkdirAll(bankRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := separatedSoundDocument{
		SchemaVersion: 1,
		Kind:          "fd2_pcm_sound_bank",
		AssetID:       "sfx/FDOTHER_082",
		Status:        "converted",
		Source: separatedSoundSource{
			Name: "FDOTHER.DAT", Size: separatedFDOTHERSize,
			MD5: separatedFDOTHERMD5, SHA256: separatedFDOTHERSHA256,
		},
		Resource: resource, ContainerCount: count + 1, ZeroLengthTailIndex: count,
		SampleRate: 11025, Channels: 1, SampleFormat: "unsigned_u8",
		TimingEvidence: "hardware-spec_approximation",
	}
	for index := 0; index < count; index++ {
		path := "sample_00" + string(rune('0'+index)) + ".ogg"
		doc.Samples = append(doc.Samples, separatedSoundSample{
			Subresource: index, SourceByteCount: 4, SourcePCMSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			Path: path, CueEvidence: "typed_schedule",
		})
		if err := os.WriteFile(filepath.Join(bankRoot, path), []byte("OggSfixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bankRoot, "resource.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadSeparatedSoundBank(t *testing.T) {
	root := writeSeparatedSoundFixture(t, 82)
	bank, err := LoadSeparatedSoundBank(root, 82)
	if err != nil {
		t.Fatal(err)
	}
	if bank.Resource != 82 || len(bank.Encoded) != 2 || string(bank.Encoded[1][:4]) != "OggS" {
		t.Fatalf("bank=%+v", bank)
	}
}

func TestLoadSeparatedSoundBankFailsClosed(t *testing.T) {
	root := writeSeparatedSoundFixture(t, 82)
	if err := os.Remove(filepath.Join(root, "FDOTHER_082", "sample_001.ogg")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSeparatedSoundBank(root, 82); err == nil {
		t.Fatal("missing OGG was accepted")
	}
	if _, err := LoadSeparatedSoundBank(root, 81); err == nil {
		t.Fatal("unsupported resource was accepted")
	}
}
