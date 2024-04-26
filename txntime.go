package fauna

import (
	"strconv"
	"sync/atomic"
)

type txnTime struct {
	value atomic.Int64
}

func (t *txnTime) get() int64 {
	return t.value.Load()
}

func (t *txnTime) sync(newTxnTime int64) {
	for {
		oldTxnTime := t.value.Load()
		if oldTxnTime >= newTxnTime ||
			t.value.CompareAndSwap(oldTxnTime, newTxnTime) {
			break
		}
	}
}

func (t *txnTime) string() (str string) {
	if lastSeen := t.value.Load(); lastSeen != 0 {
		str = strconv.FormatInt(lastSeen, 10)
	}
	return
}
