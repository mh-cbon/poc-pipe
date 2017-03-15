package t

import (
	"io"
)

type IoOp struct {
	Err  *error
	Data []byte
}

type AsyncReader struct {
	r    io.Reader
	Read chan *IoOp
}

func NewAsyncReader(r io.Reader) *AsyncReader {
	a := &AsyncReader{
		r:    r,
		Read: make(chan *IoOp),
	}
	defer a.start()
	return a
}

func (a *AsyncReader) start() {
	go func() {
		d := make([]byte, 1024)
		res := &IoOp{}
		for {
			n, err := a.r.Read(d)
			res.Err = &err
			res.Data = d[0:n]
			a.Read <- res
		}
	}()
}

func (a *AsyncReader) Close() error {
	if x, ok := a.r.(io.Closer); ok {
		return x.Close()
	}
	return nil
}
