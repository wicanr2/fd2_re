package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"reflect"
	"sort"
	"testing"
)

// TestHDThemeLoadMeasureResourceDigest 以真實 HD 包證明按需解碼不改變消費端拿到的像素（#317 後續）。
// 天空之鑰、開場、地形與戰鬥背景四類逐格經正式取用路徑解碼（同時跑完延後的檢查），其餘欄位
// 以 hdLoadSetDigest 的同一套走訪逐欄雜湊。在修改前後的樹各跑一次，比較輸出的 JSON。
// 只在 FD2_HD_LOAD_MEASURE=1 時執行。取用路徑在兩棵樹不同，由 hdDigest* 包裝。
func TestHDThemeLoadMeasureResourceDigest(t *testing.T) {
	out := hdLoadMeasureSetup(t)
	set, err := loadPlayerModernTheme()
	if err != nil {
		t.Fatal(err)
	}
	imageDigest := func(picture image.Image) string {
		h := sha256.New()
		fmt.Fprint(h, reflect.TypeOf(picture).String(), picture.Bounds())
		switch p := picture.(type) {
		case *image.RGBA:
			fmt.Fprint(h, p.Stride)
			h.Write(p.Pix)
		case *image.NRGBA:
			fmt.Fprint(h, p.Stride)
			h.Write(p.Pix)
		default:
			t.Fatalf("unexpected decoded image type %T", picture)
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	combine := func(parts []string) string {
		h := sha256.New()
		for _, p := range parts {
			fmt.Fprintln(h, p)
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	groups := map[string]any{}
	frames := 0
	for name, bank := range map[string]*modernSkyKeyHDBank{"sky-fdother34": set.skyKeyFD34, "sky-ani0": set.skyKeyANI} {
		var parts []string
		for i := range bank.asset.Files {
			picture, err := bank.picture(i + bank.offset)
			if err != nil {
				t.Fatal(name, i, err)
			}
			parts = append(parts, imageDigest(picture))
		}
		frames += len(parts)
		groups[name] = map[string]any{"frames": len(parts), "sha256": combine(parts)}
	}
	var openingKeys []modernOpeningKey
	for key := range set.openingHD.banks {
		openingKeys = append(openingKeys, key)
	}
	sort.Slice(openingKeys, func(i, j int) bool {
		if openingKeys[i].archive != openingKeys[j].archive {
			return openingKeys[i].archive < openingKeys[j].archive
		}
		return openingKeys[i].resource < openingKeys[j].resource
	})
	var openingParts []string
	for _, key := range openingKeys {
		bank := set.openingHD.banks[key]
		for i := range bank.encoded {
			picture, err := hdDigestOpeningFrame(bank, key, i)
			if err != nil {
				t.Fatal(key, i, err)
			}
			openingParts = append(openingParts, fmt.Sprintf("%s#%d/%d %s", key.archive, key.resource, i, imageDigest(picture)))
		}
	}
	frames += len(openingParts)
	groups["opening"] = map[string]any{"frames": len(openingParts), "sha256": combine(openingParts)}
	var mapIDs []int
	for id := range set.terrainHD {
		mapIDs = append(mapIDs, id)
	}
	sort.Ints(mapIDs)
	var terrainParts []string
	for _, id := range mapIDs {
		assets, err := hdDigestTerrainMap(set, id)
		if err != nil || len(assets) != len(set.terrainHD[id]) {
			t.Fatal("terrain map", id, err)
		}
		var ids []int
		for frame := range assets {
			ids = append(ids, frame)
		}
		sort.Ints(ids)
		for _, frame := range ids {
			terrainParts = append(terrainParts, fmt.Sprintf("%d/%d %s %s", id, frame, assets[frame].Role, imageDigest(assets[frame].Image)))
		}
	}
	frames += len(terrainParts)
	groups["terrain"] = map[string]any{"maps": len(mapIDs), "frames": len(terrainParts), "sha256": combine(terrainParts)}
	var surfaceParts []string
	for _, archive := range []string{"BG.DAT", "TAI.DAT"} {
		var resources []int
		for resource := range set.battleSurfaces[archive] {
			resources = append(resources, resource)
		}
		sort.Ints(resources)
		for _, resource := range resources {
			entry := set.battleSurfaces[archive][resource]
			picture, err := hdDigestBattleSurface(entry)
			if err != nil {
				t.Fatal(archive, resource, err)
			}
			native := sha256.Sum256(append(append([]byte(nil), entry.native.Pixels...), entry.native.Mask...))
			surfaceParts = append(surfaceParts, fmt.Sprintf("%s#%d %s %x %s", archive, resource, entry.hash, native, imageDigest(picture)))
		}
	}
	frames += len(surfaceParts)
	groups["battle-surface"] = map[string]any{"surfaces": len(surfaceParts), "sha256": combine(surfaceParts)}
	// 其餘頂層欄位逐欄雜湊；上面四類與只屬於按需解碼狀態的欄位略過。
	skip := map[string]bool{"skyKeyFD34": true, "skyKeyANI": true, "openingHD": true, "terrainHD": true, "terrainLazy": true,
		"battleSurfaces": true, "surfaceCache": true, "notices": true}
	fields := map[string]string{}
	root := reflect.ValueOf(set).Elem()
	for i := 0; i < root.NumField(); i++ {
		name := root.Type().Field(i).Name
		if skip[name] {
			continue
		}
		fields[name] = hdLoadValueDigest(root.Field(i))
	}
	if notices := set.notices.take(); len(notices) != 0 {
		t.Fatal("deferred checks rejected registered resources", notices)
	}
	hdLoadWriteReceipt(t, out, "resource-digest", map[string]any{
		"decoded_on_use": groups,
		"frames":         frames,
		"other_fields":   fields,
	})
}

// 以下是目前樹的正式取用路徑。修改前的樹（全部在載入時解碼）改成直接讀欄位。
func hdDigestOpeningFrame(bank *modernOpeningBank, key modernOpeningKey, index int) (image.Image, error) {
	return bank.decode(key, index)
}

func hdDigestTerrainMap(set *modernStoryPortraitSet, mapID int) (map[int]*modernTerrainHDAsset, error) {
	return set.terrainHDMap(mapID)
}

func hdDigestBattleSurface(entry *modernBattleSurfaceHD) (image.Image, error) {
	return entry.decodedPicture()
}
