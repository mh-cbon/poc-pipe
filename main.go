package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	// demo6()
	csvdemo()
	// readspeed()
	// gendata()
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
				fmt.Println("PEER LEFT 2")
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

	src := t.NewCsvReader(b)
	src.
		Pipe((&processCsvRecords{}).OnError(func(p t.Flusher, err error) error {
			log.Printf("ERR: %v", err)
			return nil
		})).
		Pipe(&t.CsvWriter{}).
		Pipe(t.NewBytesPrefixer("", "\n")).
		Pipe(&speed{}). // this is the output speed, not the processing speed
		// Sink(t.NewByteSink(os.Stdout))
		Sink(t.NewByteSink(ioutil.Discard))

	if err := src.Consume(); err != nil {
		panic(err)
	}

}

func readspeed() {

	f, err := os.Open("data.csv")
	defer f.Close()
	if err != nil {
		panic(err)
	}
	b := f

	src := t.NewByteReader(b)
	src.
		Pipe(&speed{}).
		// Sink(t.NewByteSink(os.Stdout))
		Sink(t.NewByteSink(ioutil.Discard))

	if err := src.Consume(); err != nil {
		panic(err)
	}

}

type speed struct {
	t.ByteStream
	format string
	c      int
	x      <-chan time.Time
	y      chan int
	s      chan bool
}

func (p *speed) Write(d []byte) error {
	if p.x == nil {
		if p.format == "" {
			p.format = "%vMb/s\n"
		}
		p.x = time.Tick(1 * time.Second)
		p.y = make(chan int)
		p.s = make(chan bool)
		go func() {
			c := 0
			for {
				select {
				case <-p.s:
					return
				case <-p.x:
					fmt.Printf(p.format, c/1024/1024) // not sure 200% :x
					c = 0
				case u := <-p.y:
					c += u
				}
			}
		}()
	}
	go func() {
		p.y <- len(d)
	}()
	return p.ByteStream.Write(d)
}
func (p *speed) Flush() error {
	// close(p.x)
	p.s <- true
	return p.ByteStream.Flush()
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

	if len(rec) < 1 {
		return p.HandleErr(fmt.Errorf("Line %v too short %#v", p.i, rec))
	}

	if !p.d {
		p.d = true
		return p.StringSliceStream.Write(append(rec[:4], "e", "sg", "grade"))
	}

	if len(rec) < 17 {
		return p.HandleErr(fmt.Errorf("Line %v too short %#v", p.i, rec))
	}
	p.i++

	sg := 0.0
	grade := 0.0

	zStr := "0.00000000"

	if rec[17] != zStr {
		sg, _ = strconv.ParseFloat(rec[17], 64)
		if sg < 0 {
			sg = 0
		}
	}
	if rec[13] != zStr {
		grade, _ = strconv.ParseFloat(rec[13], 64)
		if grade < 0 {
			grade = 0
		}
	}

	eStr := zStr
	sgStr := zStr
	gradeStr := zStr
	if grade > 0 {
		grade = 0.1 * grade
		e := calculateE(sg, grade)
		eStr = strconv.FormatFloat(e, 'f', 8, 64)
		sgStr = strconv.FormatFloat(sg, 'f', 8, 64)
		gradeStr = strconv.FormatFloat(grade, 'f', 8, 64)
	}

	// fmt.Println(grade)
	// get economic block value
	s(eStr, sgStr, gradeStr)
	return p.StringSliceStream.Write(
		append(rec[0:4], eStr, sgStr, gradeStr),
	)
	// return p.StringSliceStream.Write(rec)
}
func s(ss ...string) {}

func gendata() {
	l := strings.Split(csvdata, "\n")
	l = l[1:]
	f, err := os.Create("data.csv")
	fmt.Println(err)
	f.Write([]byte(csvdata))
	for i := 0; i < 1000000; i++ {
		f.Write([]byte(strings.Join(l, "\n")))
	}
	f.Close()
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
`

func sigTerm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
