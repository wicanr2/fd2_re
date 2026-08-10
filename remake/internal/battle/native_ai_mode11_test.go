package battle

import "testing"

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
