package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/mh-cbon/pipe/t"
)

// func TestOne(tester *testing.T) {
// 	x, _ := os.Open("main.go")
// 	src := t.NewSource(x)
// 	src.Pipe(t.NewSplitLine()).
// 		Pipe(t.NewPrefixer("prefix: ", "\n")).
// 		Sink(t.NewSink(os.Stdout))
//
// 	if err := src.CloseOn(sigTerm).Consume(); err != nil {
// 		panic(err)
// 	}
// }

// func BenchmarkOne(b *testing.B) {
// 	d, _ := ioutil.ReadFile("main.go")
// 	buf := bytes.NewReader(d)
// 	for i := 0; i < b.N; i++ {
// 		src := t.NewSource(buf)
// 		src.Pipe(t.NewSplitLine()).
// 			Pipe(t.NewPrefixer("prefix: ", "\n")).
// 			// Sink(t.NewSink(os.Stdout))
// 			Sink(t.NewSink(ioutil.Discard))
//
// 		if err := src.Consume(); err != nil {
// 			panic(err)
// 		}
// 		buf.Reset(d)
// 	}
// }
//
// func BenchmarkTwo(b *testing.B) {
// 	d, _ := ioutil.ReadFile("main.go")
// 	buf := bytes.NewReader(d)
// 	for i := 0; i < b.N; i++ {
// 		src := t.NewSourceCloser(buf)
// 		src.Pipe(t.NewSplitLine()).
// 			Pipe(t.NewPrefixer("prefix: ", "\n")).
// 			// Sink(t.NewSink(os.Stdout))
// 			Sink(t.NewSink(ioutil.Discard))
//
// 		if err := src.Consume(); err != nil {
// 			panic(err)
// 		}
// 		buf.Reset(d)
// 	}
// }

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
