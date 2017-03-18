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

#### Read a string as csv data, process data, change columns, encode back to csv, write stdout

It also demosntrate bandwidth meter

```sh
....
READ      39Mb/s
WRITE     13Mb/s

real	0m44.591s // processed 1.9G, is it fast ?
user	1m35.527s
sys	0m6.060s

```

```go

// demo csv read/write.
func csvdemo() {

	f, err := os.Open("data.csv")
	defer f.Close()
	if err != nil {
		panic(err)
	}
	b := f
	// var k bytes.Buffer
	// k.Write([]byte(csvdata))
	// b := &k

	/* cool stuff here,
	if you construct a simpler csv parser using a
	- ByteReader to read the source data
	-> ByteSplitter to generate chunks of Line
	-> StringSliceFromByte to split lines into col=>val
	instead of a regular CsvReader source

	the processing is much faster.
	No magic here,
	it only got ride of utf-8 support (in my understanding),
	which is totally suitable in that case.
	*/
	// src := t.NewCsvReader(b)

	src := t.NewByteReader(b)
	src.
		Pipe(&speed{format: "READ      %v%v\n"}).
		Pipe(t.NewBytesSplitter('\n')).
		Pipe(t.NewStringSliceFromByte(",")).
		Pipe((&processCsvRecords{}).OnError(func(p t.Flusher, err error) error {
			log.Printf("ERR: %v", err)
			return nil
		})).
		Pipe(&t.CsvWriter{}).
		Pipe(t.NewBytesPrefixer("", "\n")).
		Pipe(&speed{format: "WRITE     %v%v\n"}). // this is the output speed, not the processing speed
		// Sink(t.NewByteSink(os.Stdout))
		Sink(t.NewByteSink(ioutil.Discard))

	if err := src.Consume(); err != nil {
		panic(err)
	}
}

// Calculate economic block value given a 40x20x12 block of density sg and grade
func calculateE(sg float64, grade float64) float64 {
	return 40 * 20 * 12 * sg * grade * 80 / 10 * 0.66
}

type processCsvRecords struct {
	t.StringSliceStream
	i int
	d bool
}

func (p *processCsvRecords) OnError(f func(t.Flusher, error) error) *processCsvRecords {
	p.StringSliceStream.OnError(f)
	return p
}
func (p *processCsvRecords) Write(rec []string) error {
	// skip headers
	if !p.d {
		p.d = true
		return p.StringSliceStream.Write(append(rec[:4], "e", "sg", "grade"))
	}

	if len(rec) < 17 {
		return p.HandleErr(fmt.Errorf("Line %v too short %#v", p.i, rec))
	}
	p.i++

	sg, _ := strconv.ParseFloat(rec[17], 64)
	grade, _ := strconv.ParseFloat(rec[13], 64)

	if grade < 0 {
		grade = 0
	} else {
		grade = 0.1 * grade
	}
	if sg < 0 {
		sg = 0
	}
	// fmt.Println(grade)
	// get economic block value
	e := calculateE(sg, grade)
	eStr := strconv.FormatFloat(e, 'f', 8, 64)
	sgStr := strconv.FormatFloat(sg, 'f', 8, 64)
	gradeStr := strconv.FormatFloat(grade, 'f', 8, 64)
	return p.StringSliceStream.Write(
		append(rec[0:4], eStr, sgStr, gradeStr),
	)
}

// 500mb csv with head of the data looks like:
const csvdata = `xcentre,ycentre,zcentre,xlength,ylength,zlength,fe_percent,fe_recovery,oxide,rescat,sg,mat_type_8,fillpc,mass_recovery,mass_recovery_percent,air,al2o3,cao,k2o,loi,mgo,mno,phos,sio2,tio2
556960.000,6319980.000,-1100.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,91,1.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
557000.000,6319980.000,-1100.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,91,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
556960.000,6320000.000,-1100.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,91,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
557000.000,6320000.000,-1100.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,91,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
556960.000,6319980.000,-1088.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,100,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
557000.000,6319980.000,-1088.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,100,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
556960.000,6320000.000,-1088.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,100,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
557000.000,6320000.000,-1088.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,100,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
557040.000,6319980.000,-1100.000,40.000,20.000,12.000,-99.00000000,66.00000000,-99,4,2.84999990,2,91,0.00000000,0.00000000,0,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000,0.00000000
line,too,short
`
```
