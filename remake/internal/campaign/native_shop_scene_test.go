package campaign

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestDecodeNativeShopAssetsUsesOriginalMixedCodecResources(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}

	// 0x2e341 selects exactly these three backgrounds by hub variant.  The
	// outer containers differ, but cell #1 is always the 0x4e8af opaque
	// 63x15 decoration placed at VGA+0x76c5.
	for _, resourceID := range []int{12, 29, 63} {
		assets, err := DecodeNativeShopAssets(datPath, resourceID)
		if err != nil {
			t.Fatalf("FDOTHER#%d: %v", resourceID, err)
		}
		if assets.ResourceID != resourceID || len(assets.Background) != NativeShopWidth*NativeShopHeight {
			t.Fatalf("FDOTHER#%d assets=%+v background=%d", resourceID, assets, len(assets.Background))
		}
		if len(assets.RawEntries) < 11 {
			t.Fatalf("FDOTHER#%d entries=%d, want at least 11", resourceID, len(assets.RawEntries))
		}
		if cell := assets.Decoration; cell.Width != 63 || cell.Height != 15 || len(cell.Pixels) != 63*15 {
			t.Fatalf("FDOTHER#%d cell#1=%dx%d pixels=%d, want 63x15", resourceID, cell.Width, cell.Height, len(cell.Pixels))
		}
	}
}

func TestComposeNativeShopSceneUsesOriginalStableResources(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	assets, err := DecodeNativeShopAssets(fdotherPath, 12)
	if err != nil {
		t.Fatal(err)
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	dialogue := make([]fdother.RawCell, 18)
	for index := 1; index <= 17; index++ {
		dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			t.Fatalf("dialogue cell %d: %v", index, err)
		}
	}
	digits := make([]fdother.Frame, 10)
	for digit := range digits {
		digits[digit], err = fdother.ParseLMI1FrameEntry(resource5, 31+digit)
		if err != nil {
			t.Fatalf("digit %d: %v", digit, err)
		}
	}
	portraits, err := dato.DecodeResource(filepath.Join(base, "DATO.DAT"), 0x81)
	if err != nil {
		t.Fatal(err)
	}
	textRaw, err := fdother.ReadResource(filepath.Join(base, "FDTXT.DAT"), 0)
	if err != nil {
		t.Fatal(err)
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		t.Fatal(err)
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		t.Fatal(err)
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := ComposeNativeShopScene(
		assets, dialogue, digits, portraits[0], 0x81,
		strings, font, 12345678, 0x1b8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != NativeShopWidth*NativeShopHeight {
		t.Fatalf("frame bytes=%d", len(frame))
	}
	if string(frame) == string(assets.Background) {
		t.Fatal("stable shop overlays did not change the original background")
	}
	stableBefore := append([]byte(nil), frame...)
	for step := 0; step < 4; step++ {
		opening, err := ComposeNativeShopServiceOpeningFrame(frame, assets, step)
		if err != nil {
			t.Fatalf("service opening step %d: %v", step, err)
		}
		if len(opening) != NativeShopWidth*NativeShopHeight ||
			string(opening) == string(frame) {
			t.Fatalf("service opening step %d did not render", step)
		}
		closing, err := ComposeNativeShopServiceClosingFrame(frame, assets, step)
		if err != nil {
			t.Fatalf("service closing step %d: %v", step, err)
		}
		if len(closing) != NativeShopWidth*NativeShopHeight ||
			string(closing) == string(frame) {
			t.Fatalf("service closing step %d did not render", step)
		}
	}
	normal, err := ComposeNativeShopServiceSteadyFrame(frame, assets, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := ComposeNativeShopServiceSteadyFrame(frame, assets, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if string(normal) == string(selected) {
		t.Fatal("selected service pulse variant did not change the frame")
	}
	if string(frame) != string(stableBefore) {
		t.Fatal("service compositors mutated the caller-owned stable frame")
	}
	itemAssets, err := battle.LoadNativeItemPanelDataAssetsArchive(
		fdotherPath, filepath.Join(base, "FDTXT.DAT"),
	)
	if err != nil {
		t.Fatal(err)
	}
	effectRows, err := battle.LoadNativeItemEffectRowPrefix(
		"../../assets/data/native_item_effect_rows.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	purchase, err := ComposeNativeShopItemListFrame(
		frame, assets, itemAssets, []int{0, 1}, 0, 1, effectRows,
		battle.NativeFacilityFullPrice,
	)
	if err != nil {
		t.Fatal(err)
	}
	sale, err := ComposeNativeShopItemListFrame(
		frame, assets, itemAssets, []int{0, 1}, 0, 1, effectRows,
		battle.NativeFacilityThreeQuarterPrice,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(purchase) == string(sale) {
		t.Fatal("purchase and sale price modes rendered the same child panel")
	}
	if string(frame) != string(stableBefore) {
		t.Fatal("child-panel compositor mutated the stable shop frame")
	}
	purchasePortraits, err := dato.DecodeResource(
		filepath.Join(base, "DATO.DAT"), 0x80,
	)
	if err != nil {
		t.Fatal(err)
	}
	purchaseSource, err := ComposeNativeShopScene(
		assets, dialogue, digits, purchasePortraits[0], 0x80,
		strings, font, 12345678, 0x1f5,
	)
	if err != nil {
		t.Fatal(err)
	}
	choices, err := fdother.DecodeRawCellResource(fdotherPath, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range []NativeShopPurchaseMessage{
		NativeShopPurchaseQuestion,
		NativeShopPurchaseNoEligibleRecipient,
		NativeShopPurchaseEquipQuestion,
	} {
		messageFrame, err := ComposeNativeShopPurchaseMessage(
			purchaseSource, dialogue, purchasePortraits[0], 0x80,
			strings, font, message, 1, 0, 50,
		)
		if err != nil {
			t.Fatalf("purchase message %d: %v", message, err)
		}
		if len(messageFrame) != NativeShopWidth*NativeShopHeight {
			t.Fatalf("purchase message %d bytes=%d", message, len(messageFrame))
		}
	}
	confirmation, err := ComposeNativeShopPurchaseConfirmation(
		purchaseSource, dialogue, purchasePortraits[0], 0x80,
		strings, font, choices, NativeShopPurchaseQuestion,
		1, 0, 50, 0, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(confirmation) == string(purchaseSource) {
		t.Fatal("purchase confirmation did not change the source frame")
	}
	question, err := ComposeNativeShopPurchaseMessage(
		purchaseSource, dialogue, purchasePortraits[0], 0x80,
		strings, font, NativeShopPurchaseQuestion, 1, 0, 50,
	)
	if err != nil {
		t.Fatal(err)
	}
	closing, err := NativeClassConfirmationClosingFrames(question, choices)
	if err != nil || len(closing) != 4 {
		t.Fatalf("purchase confirmation closing: frames=%d err=%v", len(closing), err)
	}
	postChoiceClose := closing[len(closing)-1]
	postChoiceCloseBefore := append([]byte(nil), postChoiceClose...)
	insufficient, err := ComposeNativeShopPurchaseInsufficientGold(
		postChoiceClose, strings, font, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(insufficient) == string(postChoiceCloseBefore) {
		t.Fatal("insufficient-gold feedback did not append after choice closing")
	}
	if string(insufficient[:157*NativeShopWidth+12]) !=
		string(postChoiceCloseBefore[:157*NativeShopWidth+12]) {
		t.Fatal("insufficient-gold feedback changed pixels before literal VGA target")
	}
	if _, err := ComposeNativeShopPurchaseMessage(
		purchaseSource, dialogue, purchasePortraits[0], 0x80,
		strings, font, NativeShopPurchaseInsufficientGold, 1, 0, 50,
	); err == nil {
		t.Fatal("insufficient-gold feedback accepted a fresh dialogue source")
	}
	fdicons, err := fdicon.DecodeFile(filepath.Join(base, "FDICON.B24"))
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]NativeRosterRow, 6)
	for identity := range rows {
		sprite, err := fdicons.SpriteFor(identity, 0, 0)
		if err != nil {
			t.Fatalf("recipient sprite %d: %v", identity, err)
		}
		rows[identity] = NativeRosterRow{
			Sprite: sprite, NameTextIndex: identity + 1,
		}
	}
	recipient, err := ComposeNativeShopConsumableRecipientFrame(
		purchaseSource, assets, rows, 0, 0x20, strings, font,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(recipient) == string(purchaseSource) {
		t.Fatal("consumable recipient list did not change the source frame")
	}
	if _, err := ComposeNativeShopConsumableRecipientFrame(
		purchaseSource, assets, rows, 0, 0x1f, strings, font,
	); err == nil {
		t.Fatal("equipment recipient accepted the consumable roster renderer")
	}
	full, err := ComposeNativeShopPurchaseRecipientFull(
		purchaseSource, dialogue, purchasePortraits[0], 0x80,
		strings, font, 1, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(full) == string(purchaseSource) {
		t.Fatal("recipient-full feedback did not change the source frame")
	}
	equipmentRows := make([]NativeShopEquipmentRecipientRow, 3)
	for identity := range equipmentRows {
		record := make([]byte, 0x50)
		record[7], record[8] = byte(identity), byte(identity)
		binary.LittleEndian.PutUint16(record[0x37:], uint16(40+identity))
		binary.LittleEndian.PutUint16(record[0x39:], uint16(30+identity))
		binary.LittleEndian.PutUint16(record[0x3e:], uint16(20+identity))
		for stat, offset := range []int{0x48, 0x4a, 0x4c, 0x4e} {
			binary.LittleEndian.PutUint16(record[offset:], uint16(50+stat+identity))
		}
		candidate, err := NativeShopEquipmentCandidateStats(
			record, 0, effectRows,
		)
		if err != nil {
			t.Fatal(err)
		}
		current, err := NativeShopEquipmentCurrentStats(record)
		if err != nil {
			t.Fatal(err)
		}
		equipmentRows[identity] = NativeShopEquipmentRecipientRow{
			Sprite: rows[identity].Sprite, NameTextIndex: identity + 1,
			Current: current, Candidate: candidate,
		}
	}
	equipment, err := ComposeNativeShopEquipmentRecipientFrame(
		purchaseSource, assets, itemAssets, equipmentRows, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(equipment) == string(purchaseSource) {
		t.Fatal("equipment recipient comparison did not change the source frame")
	}
	opening, err := NativeClassListOpeningFrames(purchaseSource, equipment)
	if err != nil || len(opening) != 6 {
		t.Fatalf("equipment recipient opening=%d,%v", len(opening), err)
	}
	listClosing, err := NativeClassListClosingFrames(purchaseSource, equipment)
	if err != nil || len(listClosing) != 5 {
		t.Fatalf("equipment recipient closing=%d,%v", len(listClosing), err)
	}
}

func TestNativeShopEquipmentCandidateStatsReplacesSameCategoryOnly(t *testing.T) {
	rows := make([]byte, 3*battle.NativeItemEffectRowSize)
	// New weapon: AP/HIT/DP/EV = 10/20/30/40.
	rows[0] = 0x10
	for offset, value := range map[int]int16{1: 10, 3: 20, 5: 30, 7: 40} {
		binary.LittleEndian.PutUint16(rows[offset:], uint16(value))
	}
	// Equipped weapon must be replaced and excluded.
	weapon := rows[battle.NativeItemEffectRowSize:]
	weapon[0] = 0x11
	for _, offset := range []int{1, 3, 5, 7} {
		binary.LittleEndian.PutUint16(weapon[offset:], 100)
	}
	// Equipped armor is the opposite category and remains included.
	armor := rows[2*battle.NativeItemEffectRowSize:]
	armor[0] = 0x15
	for _, offset := range []int{1, 3, 5, 7} {
		binary.LittleEndian.PutUint16(armor[offset:], 5)
	}
	record := make([]byte, 0x50)
	binary.LittleEndian.PutUint16(record[0x37:], 1)
	binary.LittleEndian.PutUint16(record[0x39:], 2)
	binary.LittleEndian.PutUint16(record[0x3e:], 3)
	record[0x0a], record[0x0b] = 0x40, 1
	record[0x0c], record[0x0d] = 0x40, 2
	got, err := NativeShopEquipmentCandidateStats(record, 0, rows)
	if err != nil {
		t.Fatal(err)
	}
	want := [4]int{16, 37, 28, 48}
	if got != want {
		t.Fatalf("candidate=%v want %v", got, want)
	}
}

func TestPlanNativeShopPurchaseSuccessPreservesThreeShopCases(t *testing.T) {
	tests := []NativeShopPurchaseSuccessPlan{
		{
			HubVariant: 1, ResourceID: 12, FrameCount: 5,
			EffectOffset:           45*320 + 169,
			PerFrameDelayBIOSTicks: 2, RestorePortraitMode0: true,
			OptionalEquipBefore: true, DebitAfterPresentation: true,
			ReturnToProductLoop: true,
		},
		{
			HubVariant: 3, ResourceID: 29, FrameCount: 1,
			EffectOffset:      39*320 + 148,
			PreDelayBIOSTicks: 1, PostDelayBIOSTicks: 8,
			RestorePortraitMode0: true,
			OptionalEquipBefore:  true, DebitAfterPresentation: true,
			ReturnToProductLoop: true,
		},
		{
			HubVariant: 5, ResourceID: 63, FrameCount: 7,
			EffectOffset:           28*320 + 131,
			PerFrameDelayBIOSTicks: 2,
			OptionalEquipBefore:    true, DebitAfterPresentation: true,
			ReturnToProductLoop: true,
		},
	}
	for _, want := range tests {
		got, err := PlanNativeShopPurchaseSuccess(want.HubVariant)
		if err != nil || got != want {
			t.Fatalf("variant%d plan=%+v,%v want %+v", want.HubVariant, got, err, want)
		}
	}
	if _, err := PlanNativeShopPurchaseSuccess(4); err == nil {
		t.Fatal("church success variant accepted by shop plan")
	}
}

func TestNativeShopPurchaseSuccessUsesOriginalVariantResources(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	datoPath := filepath.Join(base, "DATO.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	resource5, err := fdother.ReadResource(fdotherPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	digits := make([]fdother.Frame, 10)
	for digit := range digits {
		digits[digit], err = fdother.ParseLMI1FrameEntry(
			resource5, 31+digit,
		)
		if err != nil {
			t.Fatalf("digit%d: %v", digit, err)
		}
	}
	tests := []struct {
		variant, resourceID, portraitID int
	}{
		{1, 12, 0x80},
		{3, 29, 0x82},
		{5, 63, 0x84},
	}
	for _, test := range tests {
		assets, err := DecodeNativeShopAssets(fdotherPath, test.resourceID)
		if err != nil {
			t.Fatalf("variant%d assets: %v", test.variant, err)
		}
		portraits, err := dato.DecodeResource(datoPath, test.portraitID)
		if err != nil {
			t.Fatalf("variant%d portrait: %v", test.variant, err)
		}
		bare, err := ComposeNativeShopBareScene(assets, digits, 1000)
		if err != nil {
			t.Fatalf("variant%d bare scene: %v", test.variant, err)
		}
		animation, final, err := ComposeNativeShopPurchaseSuccessFrames(
			bare, assets, portraits[0],
			test.portraitID, test.variant,
		)
		if err != nil {
			t.Fatalf("variant%d success: %v", test.variant, err)
		}
		changed := false
		for _, frame := range animation {
			if string(frame) != string(assets.Background) {
				changed = true
				break
			}
		}
		if len(animation) != len(assets.SuccessFrames) ||
			len(final) != NativeShopWidth*NativeShopHeight ||
			!changed {
			t.Fatalf(
				"variant%d frames=%d final=%d changed=%v assets=%dx%d",
				test.variant, len(animation), len(final),
				changed, assets.SuccessFrames[0].Width, assets.SuccessFrames[0].Height,
			)
		}
		if test.variant == 1 {
			const first = 90*NativeShopWidth + 175
			if bare[first] != 190 || bare[first+1] != 190 {
				t.Fatalf(
					"variant1 caller backing pixels=(%d,%d), want (190,190)",
					bare[first], bare[first+1],
				)
			}
			if animation[0][first] != 190 || animation[0][first+1] != 190 {
				t.Fatalf(
					"variant1 success started after portrait restore: (%d,%d)",
					animation[0][first], animation[0][first+1],
				)
			}
			if final[first] != 96 || final[first+1] != 191 {
				t.Fatalf(
					"variant1 final portrait pixels=(%d,%d), want (96,191)",
					final[first], final[first+1],
				)
			}
		}
	}
}

func TestNativeShopEquipmentRecordRequiresAndPreservesBaseStats(t *testing.T) {
	unit := battle.Unit{
		BattleFig: 2, NativeIdentity: 2, HasNativeIdentity: true,
		NativeRecordByte6: 3, HasNativeRecordByte6: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 4, HasNativeRecordClass: true,
		Lv: 8, MV: 5, Exp: 20, DX: 17,
		HP: 30, MaxHP: 35, MP: 5, MaxMP: 9,
		AP: 41, DP: 32, HIT: 70, EV: 22,
		InventorySlots:       []int{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		BaseAP:               29, BaseDP: 25,
	}
	if _, err := NativeShopEquipmentRecordForUnit(&unit); err == nil {
		t.Fatal("equipment preview accepted unproven base AP/DP")
	}
	unit.EquipmentBaseSet = true
	record, err := NativeShopEquipmentRecordForUnit(&unit)
	if err != nil {
		t.Fatal(err)
	}
	if got := int(int16(binary.LittleEndian.Uint16(record[0x37:]))); got != 29 {
		t.Fatalf("base AP=%d, want 29", got)
	}
	if got := int(int16(binary.LittleEndian.Uint16(record[0x39:]))); got != 25 {
		t.Fatalf("base DP=%d, want 25", got)
	}
	if got := int(int16(binary.LittleEndian.Uint16(record[0x3e:]))); got != 17 {
		t.Fatalf("base DX=%d, want 17", got)
	}
	if got := int(int16(binary.LittleEndian.Uint16(record[0x48:]))); got != 41 {
		t.Fatalf("current AP=%d, want 41", got)
	}
}

func TestNativeShopEquipmentCurrentStatsPreservesSignedWords(t *testing.T) {
	record := make([]byte, 0x50)
	for index, value := range [...]int16{-1, -32768, 32767, -23} {
		offset := [...]int{0x48, 0x4a, 0x4c, 0x4e}[index]
		binary.LittleEndian.PutUint16(record[offset:], uint16(value))
	}
	got, err := NativeShopEquipmentCurrentStats(record)
	if err != nil {
		t.Fatal(err)
	}
	if got != [4]int{-1, -32768, 32767, -23} {
		t.Fatalf("signed current stats=%v", got)
	}
}

func TestNativeShopPurchaseTextTablesPreserveSixVariants(t *testing.T) {
	want := [4][6]int{
		{1, 502, 1, 439, 1, 439},
		{1, 504, 1, 438, 1, 438},
		{1, 505, 1, 437, 1, 437},
		{1, 507, 1, 507, 1, 507},
	}
	for message := NativeShopPurchaseQuestion; message <= NativeShopPurchaseEquipQuestion; message++ {
		for variant := 0; variant < 6; variant++ {
			got, ok := NativeShopPurchaseTextIndex(message, variant)
			if !ok || got != want[message][variant] {
				t.Fatalf("message=%d variant=%d: %d,%v", message, variant, got, ok)
			}
		}
	}
	if _, ok := NativeShopPurchaseTextIndex(NativeShopPurchaseMessage(4), 0); ok {
		t.Fatal("unknown purchase message was accepted")
	}
}

func TestComposeNativeShopServiceMenuRejectsInvalidState(t *testing.T) {
	stable := make([]byte, NativeShopWidth*NativeShopHeight)
	assets := &NativeShopAssets{}
	for _, call := range []func() error{
		func() error {
			_, err := ComposeNativeShopServiceOpeningFrame(stable, assets, -1)
			return err
		},
		func() error {
			_, err := ComposeNativeShopServiceSteadyFrame(stable, assets, 4, 0)
			return err
		},
		func() error {
			_, err := ComposeNativeShopServiceSteadyFrame(stable, assets, 0, 4)
			return err
		},
	} {
		if err := call(); err == nil {
			t.Fatal("invalid native service-menu state was accepted")
		}
	}
}

func TestDecodeNativeShopAssetsRejectsUnselectedResource(t *testing.T) {
	if _, err := DecodeNativeShopAssets("irrelevant", 13); err == nil {
		t.Fatal("unselected shop resource was accepted")
	}
}
