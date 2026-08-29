package localization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePack(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.json")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validPack = `{"schema_version":1,"pack_id":"test-en","locale":"en","kind":"official","layout_profile":"dos-8x8-latin","font":{"profile":"fd2-latin-8x8","path":"fonts/atlas.png"},"entries":{"battle.attack.hit":{"text":"%s deals %d damage","variables":["%s","%d"],"source_string_id":"native-1"}}}`

func TestLoadAndFormat(t *testing.T) {
	catalog, err := Load(writePack(t, validPack))
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.Format("battle.attack.hit", "A", 3)
	if err != nil || got != "A deals 3 damage" {
		t.Fatalf("format = %q, %v", got, err)
	}
}

func TestFormatFailsClosed(t *testing.T) {
	catalog, err := Load(writePack(t, validPack))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Format("missing"); err == nil {
		t.Fatal("unknown key accepted")
	}
	if _, err := catalog.Format("battle.attack.hit", "A"); err == nil {
		t.Fatal("wrong argument count accepted")
	}
	if _, err := catalog.Format("battle.attack.hit", "A", "not-a-number"); err == nil {
		t.Fatal("incompatible argument type accepted")
	}
}

func TestLoadRejectsUnknownAndUnsafeFields(t *testing.T) {
	for name, body := range map[string]string{
		"unknown top-level": strings.Replace(validPack, `"entries":`, `"future":true,"entries":`, 1),
		"unknown entry":     strings.Replace(validPack, `"source_string_id":"native-1"`, `"source_string_id":"native-1","future":true`, 1),
		"unsafe font":       strings.Replace(validPack, `fonts/atlas.png`, `../atlas.png`, 1),
		"legacy key":        strings.Replace(validPack, `battle.attack.hit`, `legacy.go.main.l1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writePack(t, body)); err == nil {
				t.Fatal("invalid pack accepted")
			}
		})
	}
}

func TestCommunityBaseLocaleRules(t *testing.T) {
	base := strings.Replace(validPack, `"kind":"official"`, `"kind":"community","base_locale":"zh-Hant"`, 1)
	base = strings.Replace(base, `"locale":"en"`, `"locale":"zh-Hans"`, 1)
	if _, err := Load(writePack(t, base)); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(base, `"base_locale":"zh-Hant"`, `"base_locale":"zh-Hant-x-test"`, 1)
	if _, err := Load(writePack(t, bad)); err == nil {
		t.Fatal("same base locale accepted")
	}
}

func TestLoadOfficialUsesLocalePathAndIdentity(t *testing.T) {
	dir := t.TempDir()
	localeDir := filepath.Join(dir, "en")
	if err := os.Mkdir(localeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "pack.json"), []byte(validPack), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOfficial(dir, "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOfficial(dir, "ja"); err == nil {
		t.Fatal("missing official pack accepted")
	}
}

func TestOfficialLocaleAssetsLoad(t *testing.T) {
	for _, locale := range []string{"zh-Hant", "zh-Hans", "ja", "en"} {
		if _, err := LoadOfficial(filepath.Join("..", "..", "assets", "locales"), locale); err != nil {
			t.Fatalf("official %s pack: %v", locale, err)
		}
	}
}
