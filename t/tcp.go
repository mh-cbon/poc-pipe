package t

import (
	"fmt"
	"io"
	"net"
)

func Serve(p, host, port string, handleSocket func(socket io.ReadWriter)) (chan bool, error) {
	l, err := net.Listen(p, host+":"+port)
	if err != nil {
		return nil, err
	}
	close := make(chan bool)
	closed := false
	conns := []io.ReadCloser{}
	go func() {
		// Close the listener when the application closes.
		<-close
		// close all sockets.
		for _, c := range conns {
			c.Close()
		}
		closed = true
	}()
	go func() {
		// Close the listener when the application closes.
		defer l.Close()
		for {
			// Listen for an incoming connection.
			conn, err2 := l.Accept()
			if closed {
				conn.Close()
				return
			}
			conns = append(conns, conn)
			if err2 != nil {
				fmt.Println("Error accepting: ", err2.Error())
			} else {
				// Handle connections in a new goroutine.
				go handleSocket(conn)
			}
		}
	}()
	return close, err
}
