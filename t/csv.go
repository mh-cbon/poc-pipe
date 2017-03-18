package t

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type StringSlicePipeWriter interface {
	Flusher
	Write([]string) error
}

type StringSliceStream struct {
	Streams []StringSlicePipeWriter
}

func (p *StringSliceStream) Pipe(s Piper) Piper {
	// add lock
	p.Sink(s)
	return s
}
func (p *StringSliceStream) Sink(s Flusher) {
	// add lock
	x, ok := s.(StringSlicePipeWriter)
	if !ok {
		panic("nop")
	}
	p.Streams = append(p.Streams, x)
}
func (p *StringSliceStream) Unpipe(s Flusher) {
	// add lock
}
func (p *StringSliceStream) Flush() error {
	for _, pp := range p.Streams {
		if err := pp.Flush(); err != nil {
			return err
		}
	}
	return nil
}
func (p *StringSliceStream) Write(d []string) error {
	for _, pp := range p.Streams {
		if err := pp.Write(d); err != nil {
			return err
		}
	}
	return nil
}

type CsvReader struct {
	StringSliceStream
	closed bool
	r      *csv.Reader
}

func NewCsvReader(r io.Reader) *CsvReader {
	return &CsvReader{
		r: csv.NewReader(r),
	}
}
func (p *CsvReader) Consume() error {
	var err error
	var record []string
	for {
		record, err = p.r.Read()
		p.closed = err == io.EOF
		// fmt.Println(len(data))
		// <-time.After(1 * time.Second) // blah.
		if err2 := p.Write(record); err2 != nil {
			p.closed = p.closed || err2 == io.EOF
			err = err2
		}
		if p.closed {
			err = p.Flush()
			break
		}
	}

	if err == io.EOF {
		err = nil
	}

	return err
}

type CsvWriter struct {
	ByteStream
}

func (p *CsvWriter) Write(d []string) error {
	data := strings.Join(d, ",")            // Quick and dirty
	return p.ByteStream.Write([]byte(data)) // bad, so much allocations.
}
func (p *CsvWriter) Flush() error {
	return nil
}

type StringSliceToByte struct {
	ByteStream
}

func (p *StringSliceToByte) Write(a []string) error {
	return p.ByteStream.Write([]byte(fmt.Sprintf("%v", a)))
}

type AnythingToByte struct {
	ByteStream
}

func (p *AnythingToByte) Write(a interface{}) error {
	return p.ByteStream.Write([]byte(fmt.Sprintf("%v", a)))
}
