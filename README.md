# poc-pipe

POC in golang to see how to provide faster api to write to deal with io,


#### Read stdin, by line, pipe to stdout, by the way, transform it.

Using only []byte

```go
src := t.NewByteReaderCloser(os.Stdin)
src.CloseOn(sigTerm).
  Pipe(&t.SplitBytesByLine{}).
  Pipe(t.NewBytesPrefixer("prefix: ", "\n")).
  Sink(t.NewByteSink(os.Stdout))

if err := src.Consume(); err != nil {
  panic(err)
}
```


#### Read stdin, pipe to stdout, by the way, transform it.

Reads []byte, decode as t.Line{}, encode back to []byte, writes to stdout

```go
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
```

#### Read stdin, spawn a tcp server, write stdin to peers, prints to stdout peer input, writes peer inputs to other peers

Reads []byte, decode as t.Line{}, encode back to []byte, writes to stdout

In this example, the POC is definitely missing a streamReadWriter to clean a bit.

```go
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
```
