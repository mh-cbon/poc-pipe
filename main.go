package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mh-cbon/pipe/t"
)

func main() {
	// demo2()
	// demo1()
	// demo()
	// demo3()
	// demo4()
	// demo5()
	demo6()
}

func demo() {
	src := t.NewByteReader(os.Stdin)
	src.
		Pipe(&t.SplitBytesByLine{}).
		Pipe(&t.BytesReverse{}).
		Pipe(&t.BytesReverse{}).
		Pipe(&t.LineFromByte{}).
		Pipe(&t.LineTransform{}).
		Pipe(&t.LineToByte{}).
		Pipe(t.NewBytesPrefixer("prefix ", "\n")).
		Pipe(&t.LastChunkOnly{}).
		// Pipe(&t.FirstChunkOnly{}).
		Pipe(&t.BytesReverse{}).
		Sink(t.NewByteSink(os.Stdout))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

func demo1() {
	src := t.NewByteReader(os.Stdin)
	src.
		Pipe(&t.SplitBytesByLine{}).
		Pipe(t.NewBytesPrefixer("prefix: ", "\n")).
		Sink(t.NewByteSink(os.Stdout))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

func demo2() {
	src := t.NewByteReaderCloser(os.Stdin)
	src.CloseOn(sigTerm).
		Pipe(&t.SplitBytesByLine{}).
		Pipe(t.NewBytesPrefixer("prefix: ", "\n")).
		Sink(t.NewByteSink(os.Stdout))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

func demo3() {
	src := t.NewCsvReader(os.Stdin)
	src.
		Pipe(&t.StringSliceToByte{}).
		Pipe(t.NewBytesPrefixer("", "\n")).
		Sink(t.NewByteSink(os.Stdout))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

func demo4() {
	src := t.NewByteReader(os.Stdin)
	src.
		Pipe(t.NewBytesSplitter(' ', '\n')).
		Pipe(&t.VersionFromByte{}).
		Pipe(&t.VersionSorter{Asc: true}).
		Pipe(&t.VersionToByte{}).
		Pipe(&t.FirstChunkOnly{}).
		Pipe(t.NewBytesPrefixer("- ", "\n")).
		Sink(t.NewByteSink(os.Stdout))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

func demo5() {
	go func() {
		<-time.After(1 * time.Second)
		send := t.NewByteReaderCloser(os.Stdin)
		send.CloseOn(sigTerm).Sink(t.NewHttpSink("http://localhost:8080/"))
		if err := send.Consume(); err != nil {
			panic(err)
		}
	}()
	rcv := t.NewHttpReader(&http.Server{Addr: ":8080"})
	rcv.CloseOn(sigTerm).
		Pipe(t.NewBytesPrefixer("============\n", "\n")).
		Sink(t.NewByteSink(os.Stdout))
	if err := rcv.Consume(); err != nil {
		panic(err)
	}
}

func demo6() {
	// minus all the bug related to unpiping,
	// multiplexing made in < 100 loc
	send := t.NewByteReaderCloser(os.Stdin)
	send.CloseOn(sigTerm)
	send.Pipe(t.NewBytesPrefixer("SEND:", "\n")).Sink(t.NewByteSink(os.Stdout))

	go func() {
		if err := send.Consume(); err != nil {
			panic(err)
		}
	}()
	// socket should be a streamReadWriter
	// instead of those two slices.
	socketsReader := []t.Piper{}
	socketsWriter := []*t.ByteSink{}

	close, err := t.Serve("tcp", "localhost", "8080", func(socket io.ReadWriter) {
		fmt.Println("GOT A PEER")
		go func() {
			rcv := t.NewByteReader(socket)
			rcv.Pipe(&t.SplitBytesByLine{}).
				Pipe(&t.BytesReverse{}).
				Pipe(t.NewBytesPrefixer("RCV:", "\n")).
				Sink(t.NewByteSink(os.Stdout))

			for _, s := range socketsWriter {
				if s != nil {
					rcv.Sink(s)
				}
			}
			w := t.NewByteSink(socket)
			for _, s := range socketsReader {
				if s != nil {
					s.Sink(w)
				}
			}
			socketsReader = append(socketsReader, rcv)
			socketsWriter = append(socketsWriter, w)

			if err := rcv.Consume(); err != nil {
				panic(err)
			}
		}()
		send.Sink(t.NewByteSink(socket).OnError(func(p t.Flusher, err error) error {
			send.Unpipe(p)
			fmt.Println("PEER LEFT")
			socket = nil
			return nil
		}))
	})
	if err != nil {
		panic(err)
	}
	sigTerm()
	close <- true
}

func sigTerm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
