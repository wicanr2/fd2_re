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
