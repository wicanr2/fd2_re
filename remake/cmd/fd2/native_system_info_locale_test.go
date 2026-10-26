package main

import (
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSystemInfoLocaleAllSourcesAndBothFrames(t *testing.T) {
	g, in := systemInfoHDTestSource(t)
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		catalog, err := loadOfficialLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeID, g.localeCatalog = locale, catalog
		for chapter := 0; chapter < 30; chapter++ {
			indices := []int{597 + chapter*2}
			if chapter == 16 {
				indices = append(indices, 627)
			}
			for _, index := range indices {
				in.RawChapter = byte(chapter)
				for i := range in.Lines {
					in.Lines[i], err = g.nativeSystemInfoAssets.Numbers.Strings.Words(index + i)
					if err != nil {
						t.Fatal(err)
					}
				}
				p, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, in, g.nativeBattleGlyphs)
				if err != nil {
					t.Fatal(err)
				}
				before := p.input
				if err := g.localizeSystemInfo(p, index, catalog); err != nil {
					t.Fatal(locale, chapter, err)
				}
				for _, native := range []bool{false, true} {
					p.nativeChrome, p.picture = native, nil
					if err := p.compose(g.font, systemInfoTestPalette()); err != nil {
						t.Fatal(locale, chapter, native, err)
					}
					if !reflect.DeepEqual(before, p.input) {
						t.Fatal("localization changed original values/words")
					}
					if out := os.Getenv("FD2_INFO_LOCALE_UNIT_OUT"); out != "" && chapter == 22 {
						visual := *p
						visual.input.RawRound, visual.input.Currency, visual.input.RawCampCounts = 1, 2009, [3]int{24, 0, 9}
						visual.picture = nil
						if err := visual.compose(g.font, systemInfoTestPalette()); err != nil {
							t.Fatal(err)
						}
						if err := os.MkdirAll(out, 0755); err != nil {
							t.Fatal(err)
						}
						f, err := os.Create(filepath.Join(out, fmt.Sprintf("%s-native-%v.png", locale, native)))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, visual.picture)
						closeErr := f.Close()
						if err != nil || closeErr != nil {
							t.Fatal(err, closeErr)
						}
					}
				}
			}
		}
	}
}

func TestSystemInfoLocaleRejectsAndInvalidatesCache(t *testing.T) {
	g, in := systemInfoHDTestSource(t)
	p, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, in, g.nativeBattleGlyphs)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := loadOfficialLocale("en")
	if err != nil {
		t.Fatal(err)
	}
	if err = g.localizeSystemInfo(p, 597, catalog); err != nil {
		t.Fatal(err)
	}
	if err = p.compose(g.font, systemInfoTestPalette()); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(p.picture.Pix)
	for _, kind := range []string{"missing", "wrong-source", "empty", "three-rows", "empty-row"} {
		c, err := loadOfficialLocale("en")
		if err != nil {
			t.Fatal(err)
		}
		key := "battle.info.condition.0597"
		e := c.Entries[key]
		switch kind {
		case "missing":
			delete(c.Entries, key)
		case "wrong-source":
			e.SourceStringID = "FDTXT_000/string_0599"
			c.Entries[key] = e
		case "empty":
			e.Text = ""
			c.Entries[key] = e
		case "three-rows":
			e.Text = "one\ntwo\nthree"
			c.Entries[key] = e
		case "empty-row":
			e.Text = "one\n"
			c.Entries[key] = e
		}
		if err := g.localizeSystemInfo(p, 597, c); err == nil {
			t.Fatal("accepted", kind)
		}
		if sha256.Sum256(p.picture.Pix) != before {
			t.Fatal("failure mutated published art")
		}
	}
	if err := g.localizeSystemInfo(p, 601, catalog); err != nil {
		t.Fatal("same original words at distinct ID should be supported", err)
	}
	in.Lines[0][0]++
	p.input.Lines[0] = in.Lines[0]
	if err := g.localizeSystemInfo(p, 597, catalog); err == nil {
		t.Fatal("wrong words admitted")
	}
	p.input.Lines[0], _ = g.nativeSystemInfoAssets.Numbers.Strings.Words(597)
	ja, err := loadOfficialLocale("ja")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.localizeSystemInfo(p, 597, ja); err != nil {
		t.Fatal(err)
	}
	if p.picture != nil {
		t.Fatal("locale switch retained old cached image")
	}
	if err := p.compose(g.font, systemInfoTestPalette()); err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(p.picture.Pix) == before {
		t.Fatal("language picture unchanged")
	}
}

func TestSystemInfoLocaleNativeProducerChapterBranch(t *testing.T) {
	real, _ := systemInfoHDTestSource(t)
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		catalog, err := loadOfficialLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		for _, present := range []bool{false, true} {
			g := nativeSystemInfoRuntimeTestGame(t)
			g.nativeSystemInfoAssets, g.nativeBattleGlyphs, g.font = real.nativeSystemInfoAssets, real.nativeBattleGlyphs, real.font
			g.localeID, g.localeCatalog = locale, catalog
			g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true}
			g.handlerChapter, g.partyMembers[18] = 16, present
			s, err := g.prepareNativeSystemInfoUI()
			if err != nil {
				t.Fatal(locale, present, err)
			}
			index := 627
			if present {
				index = 629
			}
			if s.textIndex != index || s.hd == nil {
				t.Fatal("native branch lost")
			}
			text, err := catalog.Format(fmt.Sprintf("battle.info.condition.%04d", index+1))
			if err != nil || !reflect.DeepEqual(s.hd.lines[1], strings.Split(text, "\n")) {
				t.Fatal("wrong branch translation", err)
			}
		}
	}
}
