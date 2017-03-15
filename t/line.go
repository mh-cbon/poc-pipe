package t

import "fmt"

// Line is really just a type for testing,
// it really does not mean anything special.
type Line struct {
	content string
	n       int
}

type LinePipeWriter interface {
	Flusher
	Write(Line) error
}

type LineStream struct {
	Streams []LinePipeWriter
}

func (p *LineStream) Pipe(s Piper) Piper {
	// add lock
	p.Sink(s)
	return s
}
func (p *LineStream) Sink(s Flusher) {
	// add lock
	x, ok := s.(LinePipeWriter)
	if !ok {
		panic("nop")
	}
	p.Streams = append(p.Streams, x)
}
func (p *LineStream) Unpipe(s Flusher) {
	// add lock
}
func (p *LineStream) Flush() error {
	for _, pp := range p.Streams {
		if err := pp.Flush(); err != nil {
			return err
		}
	}
	return nil
}
func (p *LineStream) Write(d Line) error {
	for _, pp := range p.Streams {
		if err := pp.Write(d); err != nil {
			return err
		}
	}
	return nil
}

type LineFromByte struct {
	LineStream
	n int
}

func (p *LineFromByte) Write(d []byte) error {
	o := Line{content: string(d), n: p.n}
	p.n++
	return p.LineStream.Write(o)
}

type LineTransform struct {
	LineStream
}

func (p *LineTransform) Write(d Line) error {
	d.n += 100
	return p.LineStream.Write(d)
}

type LineToByte struct {
	ByteStream
}

func (p *LineToByte) Write(d Line) error {
	c := fmt.Sprintf("%v: %v", d.n, d.content)
	return p.ByteStream.Write([]byte(c))
}
