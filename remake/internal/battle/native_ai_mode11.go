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

// NativeAIMode11Stage is one address-level stage in the mode-11 transaction.
// Ordinal is the native call order, not a gameplay action number.  The
// callback owner must supply the complete raw candidate/presentation contract
// for the route before mutating battle state.
type NativeAIMode11Stage struct {
	Ordinal int
	Route   NativeAIMode11Route
}

// Stages returns the exact dispatcher order after 0x1598A and 0x14237 have
// produced their raw scores.  A missing first route is intentionally omitted;
// the second route is still unconditional in the original dispatcher.
func (t NativeAIMode11Transaction) Stages() ([]NativeAIMode11Stage, error) {
	if t.SecondRoute != NativeAIMode11Call14121 && t.SecondRoute != NativeAIMode11Call1548E {
		return nil, fmt.Errorf("native AI mode 11 second route %#x is invalid", t.SecondRoute)
	}
	stages := make([]NativeAIMode11Stage, 0, 2)
	if t.FirstRoute != NativeAIMode11NoFirst {
		if t.FirstRoute != NativeAIMode11Call15311 {
			return nil, fmt.Errorf("native AI mode 11 first route %#x is invalid", t.FirstRoute)
		}
		stages = append(stages, NativeAIMode11Stage{Ordinal: 1, Route: t.FirstRoute})
	}
	stages = append(stages, NativeAIMode11Stage{Ordinal: 2, Route: t.SecondRoute})
	return stages, nil
}

// ExecuteNativeAIMode11Transaction owns only the verified stage ordering. It
// deliberately receives a caller-owned stage callback: command effects,
// movement, indexed presentation and the 0x13FD4 fallback are not silently
// substituted here. A callback failure stops before a later native stage.
func ExecuteNativeAIMode11Transaction(
	t NativeAIMode11Transaction,
	execute func(NativeAIMode11Stage) error,
) error {
	if execute == nil {
		return fmt.Errorf("native AI mode 11 stage executor is unavailable")
	}
	stages, err := t.Stages()
	if err != nil {
		return err
	}
	for _, stage := range stages {
		if err := execute(stage); err != nil {
			return fmt.Errorf("native AI mode 11 stage %d route %#x: %w", stage.Ordinal, stage.Route, err)
		}
	}
	return nil
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
