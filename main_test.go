package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/mh-cbon/pipe/t"
)

var buf *bytes.Reader
var data []byte

func init() {
	data, _ = ioutil.ReadFile("main.go")
	buf = bytes.NewReader(data)
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
