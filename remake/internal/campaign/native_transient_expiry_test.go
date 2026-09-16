package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

func TestNativeTransientExpiryTextContract(t *testing.T) {
	if NativeTransientExpiryTextBase != 481 || NativeTransientExpiryTextCount != 6 ||
		NativeTransientExpiryTextOffset != 0x9f23 {
		t.Fatalf("base=%d count=%d offset=%#x",
			NativeTransientExpiryTextBase,
			NativeTransientExpiryTextCount,
			NativeTransientExpiryTextOffset)
	}
}

func TestNativeTransientExpiryRejectsUnknownCounterBeforeAssets(t *testing.T) {
	if _, err := ComposeNativeTransientExpiryFrame(nil, nil, dato.Frame{}, nil, nil, 6); err == nil {
		t.Fatal("unknown transient counter was accepted")
	}
}

func TestNativeDeathRewardMessageTextContract(t *testing.T) {
	if NativeDeathRewardItemTextIndex != 0x1b0 || NativeDeathRewardGoldTextIndex != 0x1b3 ||
		NativeDeathRewardItemNameBase != 0xb5 || NativeDeathRewardMessageTextOffs != 0x9f23 {
		t.Fatalf("item=%#x gold=%#x name=%#x offset=%#x",
			NativeDeathRewardItemTextIndex, NativeDeathRewardGoldTextIndex,
			NativeDeathRewardItemNameBase, NativeDeathRewardMessageTextOffs)
	}
	if _, err := ComposeNativeDeathRewardMessageFrame(make([]byte, 320*200), nil, nil, 2, 1); err == nil {
		t.Fatal("kind 2 has no 0x1AA1D message and must be rejected")
	}
}

// 0x1B0 的 FFFC 展開成 item+0xB5 的物品名、0x1B3 的 FFFA 展開成金額；兩者都寫在 0x9F23。
func TestComposeNativeDeathRewardMessageExpandsItemNameAndGold(t *testing.T) {
	_, _, dialogueCells, _, portrait, strings, font := loadNativeChurchOriginalSceneAssets(t)
	background := make([]byte, 320*200)
	dialogue, err := ComposeNativeChurchDialogueOverlayAt(background, dialogueCells, portrait, nativeLowerPortraitRightEdge)
	if err != nil {
		t.Fatal(err)
	}
	item, err := ComposeNativeDeathRewardMessageFrame(dialogue, strings, font, 0, 0x00)
	if err != nil {
		t.Fatal(err)
	}
	name, err := strings.Words(0xb5)
	if err != nil {
		t.Fatal(err)
	}
	// 對照：手動展開同一句再用同一個 composer 畫。
	words, _ := strings.Words(0x1b0)
	expanded := []uint16{}
	for _, w := range words {
		if w == 0xfffc {
			expanded = append(expanded, name...)
		} else {
			expanded = append(expanded, w)
		}
	}
	want, err := composeNativeChurchWordsAt(append([]byte(nil), dialogue...), font, expanded, 0x9f23)
	if err != nil {
		t.Fatal(err)
	}
	if string(item) != string(want) {
		t.Fatal("item message frame differs from the FFFC-expanded 0x1B0 composition")
	}
	if string(item) == string(dialogue) {
		t.Fatal("item message wrote nothing")
	}
	gold, err := ComposeNativeDeathRewardMessageFrame(dialogue, strings, font, 1, 150)
	if err != nil {
		t.Fatal(err)
	}
	if string(gold) == string(dialogue) || string(gold) == string(item) {
		t.Fatal("gold message must differ from both the bare dialogue and the item message")
	}
}
