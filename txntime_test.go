package fauna

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTxnTime(t *testing.T) {
	txnTime := txnTime{}
	require.Equal(t, int64(0), txnTime.get())
	require.Equal(t, "", txnTime.string())

	txnTime.sync(42) // move forward
	require.Equal(t, int64(42), txnTime.get())
	require.Equal(t, "42", txnTime.string())

	txnTime.sync(32) // don't move back
	require.Equal(t, int64(42), txnTime.get())
	require.Equal(t, "42", txnTime.string())
}

func BenchmarkTxnTime(b *testing.B) {
	txnTime := txnTime{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			now := time.Now()
			txnTime.sync(now.UnixMicro())
		}
	})
}
