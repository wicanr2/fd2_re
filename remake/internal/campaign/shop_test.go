package campaign

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestBuyGoodUsesSelectedInventoryAndIsAtomicOnFailure(t *testing.T) {
	good := Good{ID: 0xc0, Name: "藥草", Price: 10}
	receiver := &battle.Unit{Inventory: []int{1, 2}}
	gold, err := BuyGood(50, receiver, good)
	if err != nil || gold != 40 || !reflect.DeepEqual(receiver.Inventory, []int{1, 2, 0xc0}) {
		t.Fatalf("purchase gold=%d err=%v inventory=%#v", gold, err, receiver.Inventory)
	}

	full := &battle.Unit{Inventory: make([]int, 8)}
	if got, err := BuyGood(50, full, good); err == nil || got != 50 || len(full.Inventory) != 8 {
		t.Fatalf("full inventory changed gold=%d err=%v inventory=%#v", got, err, full.Inventory)
	}
	if got, err := BuyGood(9, receiver, good); err == nil || got != 9 || len(receiver.Inventory) != 3 {
		t.Fatalf("insufficient gold changed state gold=%d err=%v inventory=%#v", got, err, receiver.Inventory)
	}
}
