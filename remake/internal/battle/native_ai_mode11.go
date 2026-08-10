package battle

import "fmt"

// NativeAIMode11Transaction is the raw two-stage branch after 0x13A9F has
// selected mode 11.  The fields are address-level routes only; they are not
// command, attack, spell or movement names.
// NativeAIMode11Route keeps the mode-11 branch separate from the 0x14ef0
// tail enum: 0x14121 is a pre-tail movement/recovery route, not a 0x14ef0
// consumer.
type NativeAIMode11Route uint8

const (
	NativeAIMode11NoFirst NativeAIMode11Route = iota
	NativeAIMode11Call15311
	NativeAIMode11Call14121
	NativeAIMode11Call1548E
)

type NativeAIMode11Transaction struct {
	FirstRoute  NativeAIMode11Route // 0x15311 when raw [0x53C23] >= 6, otherwise none
	SecondRoute NativeAIMode11Route // 0x1548E when raw [0x53C4F] >= 6, otherwise 0x14121
}

// SelectNativeAIMode11Transaction preserves the direct mode-11 control flow:
// 0x1598A has already produced [0x53C23], 0x15311 is called at signed >=6,
// 0x14237 then runs unconditionally, and [0x53C4F] selects 0x1548E or
// 0x14121.  It is deliberately a pure E0 adapter; it does not execute either
// stage or claim that the two-stage presentation/commit owner is recovered.
func SelectNativeAIMode11Transaction(scoreC23, scoreC4F int32, hasRawScoreC23, hasRawScoreC4F bool) (NativeAIMode11Transaction, error) {
	if !hasRawScoreC23 || !hasRawScoreC4F {
		return NativeAIMode11Transaction{}, fmt.Errorf("native AI mode 11 raw scores are incomplete")
	}
	transaction := NativeAIMode11Transaction{SecondRoute: NativeAIMode11Call14121}
	if scoreC23 >= 6 {
		transaction.FirstRoute = NativeAIMode11Call15311
	}
	if scoreC4F >= 6 {
		transaction.SecondRoute = NativeAIMode11Call1548E
	}
	return transaction, nil
}
