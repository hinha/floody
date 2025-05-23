package diode

import (
	diode "code.cloudfoundry.org/go-diodes"
	"context"
	"io"
	"sync"
	"time"
)

var bufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 500)
	},
}

// AlertFunc type is an adapter to allow the use of ordinary functions as
type AlertFunc = diode.AlertFunc

// Writer is a io.Writer wrapper that uses a diode to make Write lock-free,
// non-blocking and thread safe.
type Writer struct {
	w    io.Writer
	d    diodeFetcher
	c    context.CancelFunc
	done chan struct{}
}

type diodeFetcher interface {
	diode.Diode
	Next() diode.GenericDataType
}

func NewWriter(w io.Writer, size int, pollInterval time.Duration, f AlertFunc) Writer {
	ctx, cancel := context.WithCancel(context.Background())
	dw := Writer{
		w:    w,
		c:    cancel,
		done: make(chan struct{}),
	}
	if f == nil {
		f = func(missed int) {}
	}

	d := diode.NewManyToOne(size, f)
	if pollInterval > 0 {
		dw.d = diode.NewPoller(d,
			diode.WithPollingInterval(pollInterval),
			diode.WithPollingContext(ctx))
	} else {
		dw.d = diode.NewWaiter(d, diode.WithWaiterContext(ctx))
	}
	go dw.poll()
	return dw
}

func (dw Writer) Write(p []byte) (n int, err error) {
	// p is pooled in zap, so we can't hold it passed this call, hence the
	// copy.
	p = append(bufPool.Get().([]byte), p...)
	dw.d.Set(diode.GenericDataType(&p))
	return len(p), nil
}

func (dw Writer) Close() error {
	dw.c()
	<-dw.done
	if w, ok := dw.w.(io.Closer); ok {
		return w.Close()
	}
	return nil
}

func (dw Writer) poll() {
	defer close(dw.done)
	for {
		d := dw.d.Next()
		if d == nil {
			return
		}
		p := *(*[]byte)(d)
		_, _ = dw.w.Write(p)

		// Proper usage of a sync.Pool requires each entry to have approximately
		// the same memory cost. To obtain this property when the stored type
		// contains a variably-sized buffer, we add a hard limit on the maximum buffer
		// to place back in the pool.
		//
		// See https://golang.org/issue/23199
		const maxSize = 1 << 16 // 64KiB
		if cap(p) <= maxSize {
			bufPool.Put(p[:0])
		}
	}
}
