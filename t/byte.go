package t

import (
	"bytes"
	"fmt"
	"io"
)

type Flusher interface {
	Flush() error
}

type Sinker interface {
	Sink(Flusher)
}

type Piper interface {
	Flusher
	Sinker
	Pipe(Piper) Piper
	Unpipe(Flusher)
}

type ByteWriter interface {
	Flusher
	Write(d []byte) error
}

type ByteStream struct {
	Streams []ByteWriter
}

func (p *ByteStream) Pipe(s Piper) Piper {
	// add lock
	p.Sink(s)
	return s
}
func (p *ByteStream) Sink(s Flusher) {
	x, ok := s.(ByteWriter)
	if !ok {
		panic("nop")
	}
	p.Streams = append(p.Streams, x)
}
func (p *ByteStream) Unpipe(s Flusher) {
	// add lock
	x, ok := s.(ByteWriter)
	if !ok {
		panic("nop")
	}
	i := -1
	for e, pp := range p.Streams {
		if pp == x {
			i = e
			break
		}
	}
	if i > -1 {
		p.Streams = append(p.Streams[:i], p.Streams[i+1:]...)
	}
	fmt.Println(i)
}
func (p *ByteStream) Flush() error {
	for _, pp := range p.Streams {
		if err := pp.Flush(); err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}
func (p *ByteStream) Write(d []byte) error {
	for _, pp := range p.Streams {
		if err := pp.Write(d); err != nil {
			return err
		}
	}
	return nil
}

type ByteReader struct {
	ByteStream
	closed bool
	r      io.Reader
}

func NewByteReader(r io.Reader) *ByteReader {
	return &ByteReader{r: r}
}

func (p *ByteReader) Consume() error {
	var err error
	var n int
	data := make([]byte, 1024)
	for {
		n, err = p.r.Read(data)
		p.closed = err == io.EOF
		data = data[0:n]
		// fmt.Println(len(data))
		// <-time.After(1 * time.Second) // blah.
		if err2 := p.Write(data); err2 != nil {
			p.closed = err2 == io.EOF
			err = err2
		}
		if p.closed {
			err = p.Flush()
			if x, ok := p.r.(io.Closer); ok {
				if err2 := x.Close(); err2 != nil {
					err = err2
				}
			}
			break
		}
	}

	if err == io.EOF {
		err = nil
	}

	return err
}

type ByteReaderCloser struct {
	ByteStream
	closed bool
	a      *AsyncReader
}

func NewByteReaderCloser(r io.Reader) *ByteReaderCloser {
	return &ByteReaderCloser{a: NewAsyncReader(r)}
}

func (p *ByteReaderCloser) Consume() error {
	var err error
	finished := false

	for {
		if finished {
			break
		}
		select {
		case op := <-p.a.Read:
			data := op.Data
			// n := op.N
			err = *op.Err
			p.closed = err == io.EOF
			if err2 := p.Write(data); err2 != nil {
				err = err2
				p.closed = err == io.EOF
			}
		default:
			if p.closed {
				err = p.Flush()
				finished = true
				if err3 := p.a.Close(); err3 != nil {
					return err3
				}
			} else {
				// <-time.After(1 * time.Microsecond) // blah.
			}
		}
	}

	return err
}
func (p *ByteReaderCloser) Close() error {
	if p.closed {
		return ErrAlreadyClosed
	}
	p.closed = true
	return nil
}
func (p *ByteReaderCloser) CloseOn(f func()) Piper {
	go func() {
		f()
		p.Close()
	}()
	return p
}

type ByteSink struct {
	w     io.Writer
	onErr func(Flusher, error) error
}

func NewByteSink(w io.Writer) *ByteSink {
	return &ByteSink{w: w}
}
func (p *ByteSink) OnError(f func(Flusher, error) error) *ByteSink {
	p.onErr = f
	return p
}
func (p *ByteSink) Write(d []byte) error {
	_, err := p.w.Write(d)
	if err != nil && p.onErr != nil {
		err = p.onErr(p, err)
	}
	return err
}
func (p *ByteSink) Flush() error {
	return nil
}

type BytesSplitter struct {
	ByteStream
	buf []byte
	s   []byte
}

func NewBytesSplitter(s ...byte) *BytesSplitter {
	return &BytesSplitter{
		s: s,
	}
}

func (p *BytesSplitter) Write(d []byte) error {
	p.buf = append(p.buf, d...)
	q := false
	for {
		if q {
			break
		}
		for _, s := range p.s {
			if i := bytes.IndexByte(p.buf, s); i >= 0 {
				data := p.buf[:i]
				if len(data) > 0 {
					if err := p.ByteStream.Write(data); err != nil {
						return err
					}
				}
				p.buf = p.buf[i+1:]
			} else {
				q = true
			}
		}
	}
	return nil
}

func (p *BytesSplitter) Flush() error {
	if len(p.buf) > 0 {
		if err := p.ByteStream.Write(p.buf); err != nil {
			return err
		}
	}
	return p.ByteStream.Flush()
}

type SplitBytesByLine struct {
	ByteStream
	buf []byte
}

func (p *SplitBytesByLine) Write(d []byte) error {
	p.buf = append(p.buf, d...)
	for {
		if i := bytes.IndexByte(p.buf, '\n'); i >= 0 {
			data := dropCR(p.buf[:i])
			if len(data) > 0 {
				if err := p.ByteStream.Write(data); err != nil {
					return err
				}
			}
			p.buf = p.buf[i+1:]
		} else {
			break
		}
	}
	return nil
}

func (p *SplitBytesByLine) Flush() error {
	if len(p.buf) > 0 {
		if err := p.ByteStream.Write(dropCR(p.buf[0:])); err != nil {
			return err
		}
	}
	return p.ByteStream.Flush()
}

type BytesPrefixer struct {
	ByteStream
	prefix []byte
	suffix []byte
	b      *bytes.Buffer
}

func NewBytesPrefixer(prefix string, suffix string) *BytesPrefixer {
	return &BytesPrefixer{
		prefix: []byte(prefix),
		suffix: []byte(suffix),
		b:      bytes.NewBuffer(make([]byte, 1024)),
	}
}

func (p *BytesPrefixer) Write(d []byte) error {
	p.b.Truncate(0)
	p.b.Write(p.prefix)
	p.b.Write(d)
	p.b.Write(p.suffix)
	return p.ByteStream.Write(p.b.Bytes())
}

type BytesReverse struct {
	ByteStream
}

func (p *BytesReverse) Write(d []byte) error {
	for i := 0; i < len(d)/2; i++ {
		j := len(d) - i - 1
		d[i], d[j] = d[j], d[i]
	}
	return p.ByteStream.Write(d)
}

type FirstChunkOnly struct {
	ByteStream
	d bool
}

func (p *FirstChunkOnly) Write(d []byte) error {
	if !p.d {
		p.d = true
		return p.ByteStream.Write(d)
	}
	return nil
	// return errors.New("Chunk already received")
}

type LastChunkOnly struct {
	ByteStream
	d     bool
	chunk []byte
}

func (p *LastChunkOnly) Write(d []byte) error {
	p.d = true
	p.chunk = append(p.chunk[:0], d...)
	return nil
}

func (p *LastChunkOnly) Flush() error {
	if p.d {
		if err := p.ByteStream.Write(p.chunk); err != nil {
			return err
		}
	}
	return p.ByteStream.Flush()
}
