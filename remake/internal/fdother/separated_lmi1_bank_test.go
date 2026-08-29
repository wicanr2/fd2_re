package fdother

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedCompleteLMI1BanksMatchFixedResources(t *testing.T) {
	for _, sample := range []struct {
		resource int
		root     string
	}{
		{5, "ui/fdother_005_lmi1_opaque"},
		{6, "effects/fdother_006_lmi1_opaque"},
		{9, "animations/fdother_009_spawn_intro"},
	} {
		rawPath := filepath.Join("..", "..", "..", "extracted", "raw", "FDOTHER", "FDOTHER_"+formatResource(sample.resource)+".bin")
		raw, err := os.ReadFile(rawPath)
		if err != nil {
			t.Skipf("player-provided FDOTHER #%d is absent", sample.resource)
		}
		want, err := ParseLMI1(raw)
		if err != nil {
			t.Fatal(err)
		}
		root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22", filepath.FromSlash(sample.root))
		got, err := LoadSeparatedLMI1Bank(root, sample.resource)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("FDOTHER #%d entries=%d, want %d", sample.resource, len(got), len(want))
		}
		for index := range want {
			if got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
				!bytes.Equal(got[index].Pixels, want[index].Pixels) {
				t.Fatalf("FDOTHER #%d entry %d differs", sample.resource, index)
			}
		}
	}
}

func TestSeparatedCompleteLMI1BanksFailClosed(t *testing.T) {
	if _, err := LoadSeparatedLMI1Bank(t.TempDir(), 5); err == nil {
		t.Fatal("missing FDOTHER #5 bank was accepted")
	}
	if _, err := LoadSeparatedLMI1Bank(t.TempDir(), 4); err == nil {
		t.Fatal("unsupported FDOTHER bank was accepted")
	}
}

func formatResource(resource int) string {
	return string([]byte{'0' + byte(resource/100), '0' + byte(resource/10%10), '0' + byte(resource%10)})
}
