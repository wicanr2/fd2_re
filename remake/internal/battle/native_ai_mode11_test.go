package battle

import (
	"fmt"
	"testing"
)

func TestSelectNativeAIMode11TransactionPreservesIndependentGates(t *testing.T) {
	tests := []struct {
		name               string
		scoreC23, scoreC4F int32
		first, second      NativeAIMode11Route
	}{
		{name: "both below", scoreC23: 5, scoreC4F: 5, first: NativeAIMode11NoFirst, second: NativeAIMode11Call14121},
		{name: "command only", scoreC23: 6, scoreC4F: 5, first: NativeAIMode11Call15311, second: NativeAIMode11Call14121},
		{name: "physical only", scoreC23: 5, scoreC4F: 6, first: NativeAIMode11NoFirst, second: NativeAIMode11Call1548E},
		{name: "both", scoreC23: 6, scoreC4F: 6, first: NativeAIMode11Call15311, second: NativeAIMode11Call1548E},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := SelectNativeAIMode11Transaction(test.scoreC23, test.scoreC4F, true, true)
			if err != nil {
				t.Fatal(err)
			}
			if got.FirstRoute != test.first || got.SecondRoute != test.second {
				t.Fatalf("transaction=%+v want first=%v second=%v", got, test.first, test.second)
			}
		})
	}
}

func TestSelectNativeAIMode11TransactionFailsClosedWithoutRawScore(t *testing.T) {
	if _, err := SelectNativeAIMode11Transaction(6, 6, false, true); err == nil {
		t.Fatal("missing raw command score unexpectedly accepted")
	}
	if _, err := SelectNativeAIMode11Transaction(6, 6, true, false); err == nil {
		t.Fatal("missing raw physical score unexpectedly accepted")
	}
}

func TestNativeAIMode11StagesPreserveNativeOrder(t *testing.T) {
	tx, err := SelectNativeAIMode11Transaction(6, 5, true, true)
	if err != nil {
		t.Fatal(err)
	}
	var got []NativeAIMode11Stage
	if err := ExecuteNativeAIMode11Transaction(tx, func(stage NativeAIMode11Stage) error {
		got = append(got, stage)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := []NativeAIMode11Stage{
		{Ordinal: 1, Route: NativeAIMode11Call15311},
		{Ordinal: 2, Route: NativeAIMode11Call14121},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("stages=%#v, want %#v", got, want)
	}
}

func TestNativeAIMode11StageFailureStopsLaterRoute(t *testing.T) {
	tx, err := SelectNativeAIMode11Transaction(6, 6, true, true)
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	if err := ExecuteNativeAIMode11Transaction(tx, func(stage NativeAIMode11Stage) error {
		called++
		if stage.Ordinal == 1 {
			return fmt.Errorf("presentation owner unavailable")
		}
		return nil
	}); err == nil {
		t.Fatal("failed first stage unexpectedly accepted")
	}
	if called != 1 {
		t.Fatalf("later stage executed after failure: called=%d", called)
	}
}

func TestNativeAIMode11StagesRejectInvalidRoute(t *testing.T) {
	if _, err := (NativeAIMode11Transaction{
		FirstRoute:  NativeAIMode11Call14121,
		SecondRoute: NativeAIMode11Call14121,
	}).Stages(); err == nil {
		t.Fatal("invalid first route unexpectedly accepted")
	}
	if err := ExecuteNativeAIMode11Transaction(NativeAIMode11Transaction{
		SecondRoute: NativeAIMode11NoFirst,
	}, func(NativeAIMode11Stage) error { return nil }); err == nil {
		t.Fatal("invalid second route unexpectedly accepted")
	}
}
