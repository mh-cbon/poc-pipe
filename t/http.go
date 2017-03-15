package t

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type HttpReader struct {
	ByteStream
	closed bool
	h      *http.Server
	sync   chan []byte
}

func NewHttpReader(h *http.Server) *HttpReader {
	return &HttpReader{h: h, sync: make(chan []byte)}
}
func (p *HttpReader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err == nil && len(data) > 0 {
		// synced ? Not sure, just a demo
		p.sync <- data
	}
}
func (p *HttpReader) Consume() error {
	p.h.Handler = p
	go func() {
		finished := false
		for {
			if finished {
				break
			}
			select {
			case data := <-p.sync:
				/*err := ?*/ p.Write(data)
			default:
				if p.closed {
					finished = true
				}
			}
		}
	}()
	return p.h.ListenAndServe()
}
func (p *HttpReader) Close() error {
	if p.closed {
		return ErrAlreadyClosed
	}
	p.closed = true
	return p.h.Close()
}
func (p *HttpReader) CloseOn(f func()) Piper {
	go func() {
		f()
		p.Close()
	}()
	return p
}

type HttpSink struct {
	url string
}

func NewHttpSink(url string) *HttpSink {
	return &HttpSink{url: url}
}
func (p *HttpSink) Write(d []byte) error {
	req, err := http.NewRequest("POST", p.url, bytes.NewBuffer(d))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
func (p *HttpSink) Flush() error {
	return nil
}
