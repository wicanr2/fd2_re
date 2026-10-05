package battle

import "testing"

func nativeAI14EF0TestInput() NativeAI14EF0Input {
	return NativeAI14EF0Input{
		HasRawScoreC4F: true, HasRawScoreC23: true, HasRawScoreC33: true,
		HasRawRecord34: true, HasRawActorWord48: true, HasRawTargetWord4A: true,
		HasRawCommandID: true, HasRawCommandWord: true,
		CommandID: 2, CommandWord: 10, ActorWord48: 20, TargetWord4A: 5,
	}
}

func TestSelectNativeAI14EF0TailPreservesRawRoutes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*NativeAI14EF0Input)
		want   NativeAI14EF0Tail
	}{
		{"below threshold", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 5, 5, 5 }, NativeAI14EF0NoTail},
		{"strict c4f", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 7, 7 }, NativeAI14EF0Call1548E},
		{"c4f c23 tie command word", func(in *NativeAI14EF0Input) {
			in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 7
			in.CommandID, in.CommandWord = 2, 15
		}, NativeAI14EF0Call15311},
		{"c4f c23 tie command word too small", func(in *NativeAI14EF0Input) {
			in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 7
			in.CommandID, in.CommandWord = 2, 14
		}, NativeAI14EF0Call1548E},
		{"c4f c23 tie id high bit clear", func(in *NativeAI14EF0Input) {
			in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 7
			in.CommandID, in.Record34 = 11, 0
		}, NativeAI14EF0Call15311},
		{"c4f c23 tie id high bit set", func(in *NativeAI14EF0Input) {
			in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 7
			in.CommandID, in.Record34 = 11, 0x40
		}, NativeAI14EF0Call1548E},
		{"c4f c33 tie bit clear", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 7, 8; in.Record34 = 0 }, NativeAI14EF0Call15055},
		{"c4f c33 tie bit set", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 7, 8; in.Record34 = 0x40 }, NativeAI14EF0Call1548E},
		{"strict c23", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 7, 9, 9 }, NativeAI14EF0Call15311},
		{"strict c33", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 7, 8, 9 }, NativeAI14EF0Call15055},
		{"remaining three-way tie falls through", func(in *NativeAI14EF0Input) { in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 8 }, NativeAI14EF0NoTail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := nativeAI14EF0TestInput()
			tt.mutate(&in)
			got, err := SelectNativeAI14EF0Tail(in)
			if err != nil {
				t.Fatalf("SelectNativeAI14EF0Tail() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SelectNativeAI14EF0Tail() = %d, want %d", got, tt.want)
			}
			decision, err := PlanNativeAI14EF0Dispatch(in)
			wantAccepted := tt.name != "below threshold"
			if err != nil || decision.Tail != tt.want || decision.Accepted != wantAccepted {
				t.Fatalf("dispatch=%+v err=%v want tail=%d accepted=%t", decision, err, tt.want, wantAccepted)
			}
		})
	}
}

func TestNativeAI14EF0NoTailReturnBoundary(t *testing.T) {
	for _, score := range []int32{-1, 0, 5, 6, 8} {
		in := nativeAI14EF0TestInput()
		in.ScoreC4F, in.ScoreC23, in.ScoreC33 = score, score, score
		decision, err := PlanNativeAI14EF0Dispatch(in)
		if err != nil || decision.Tail != NativeAI14EF0NoTail || decision.Accepted != (score >= 6) {
			t.Fatalf("score=%d decision=%+v err=%v", score, decision, err)
		}
	}
	in := nativeAI14EF0TestInput()
	in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 8
	in.HasRawScoreC23 = false
	if decision, err := PlanNativeAI14EF0Dispatch(in); err == nil || decision.Accepted {
		t.Fatalf("missing score admitted: decision=%+v err=%v", decision, err)
	}
}

func TestSelectNativeAI14EF0TailFailsClosedWithoutRawInputs(t *testing.T) {
	in := nativeAI14EF0TestInput()
	in.HasRawTargetWord4A = false
	if _, err := SelectNativeAI14EF0Tail(in); err == nil {
		t.Fatal("expected missing raw target word to fail closed")
	}
	in = nativeAI14EF0TestInput()
	in.ScoreC4F, in.ScoreC23, in.ScoreC33 = 8, 8, 7
	in.HasRawCommandWord = false
	if _, err := SelectNativeAI14EF0Tail(in); err == nil {
		t.Fatal("expected command tie without 0x4e516 word to fail closed")
	}
}
