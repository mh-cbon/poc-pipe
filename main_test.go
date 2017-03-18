package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/mh-cbon/pipe/t"
)

var csvbuf bytes.Buffer
var csvdatafile []byte

var buf *bytes.Reader
var data []byte

func init() {
	data, _ = ioutil.ReadFile("main.go")
	buf = bytes.NewReader(data)

	csvbuf.Write([]byte(csvdata))
	csvdatafile = []byte(csvdata)
}
func BenchmarkOne(b *testing.B) {
	for i := 0; i < b.N; i++ {
		src := t.NewByteReader(buf)
		src.
			Pipe(&t.SplitBytesByLine{}).
			Pipe(t.NewBytesPrefixer("prefix: ", "\n")).
			Sink(t.NewByteSink(ioutil.Discard))

		if err := src.Consume(); err != nil {
			panic(err)
		}
		buf.Reset(data)
	}
}

func BenchmarkTwo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		src := t.NewByteReaderCloser(buf)
		src.
			Pipe(&t.SplitBytesByLine{}).
			Pipe(t.NewBytesPrefixer("prefix: ", "\n")).
			Sink(t.NewByteSink(ioutil.Discard))

		if err := src.Consume(); err != nil {
			panic(err)
		}
		buf.Reset(data)
	}
}
func BenchmarkThree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		src := t.NewCsvReader(&csvbuf)
		src.
			Pipe((&processCsvRecords{}).OnError(func(p t.Flusher, err error) error {
				// log.Printf("ERR: %v", err)
				return nil
			})).
			Pipe(&t.CsvWriter{}).
			Pipe(t.NewBytesPrefixer("", "\n")).
			Sink(t.NewByteSink(ioutil.Discard))
			// Sink(t.NewByteSink(os.Stdout))

		if err := src.Consume(); err != nil {
			panic(err)
		}

		// csvbuf.Reset()
		csvbuf.Write(csvdatafile)
	}
}
