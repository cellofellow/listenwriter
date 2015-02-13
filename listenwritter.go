/* listenWriter provides an io.Writer that is backed by a socket that
* broadcasts anything written to the Writer to all connected clients. */

package listenWriter

import (
	"container/list"
	"errors"
	"net"
	"strings"
	"time"
)

// ListenWriter is an object that implements io.Writer
type ListenWriter struct {
	listener net.Listener
	conns    *list.List
}

func NewListenWriter(n, laddr string) (*ListenWriter, error) {
	listener, err := net.Listen(n, laddr)
	if err != nil {
		return nil, err
	}
	lw := &ListenWriter{listener: listener, conns: list.New()}
	lw.accepter()
	return lw, nil
}

func (lw *ListenWriter) accepter() {
	go func() {
		for {
			conn, err := lw.listener.Accept()
			if err != nil {
				continue
			}
			lw.conns.PushFront(conn)
		}
	}()
}

func (lw *ListenWriter) Write(p []byte) (int, error) {
	errs := []string{}
	count := 0
	el := lw.conns.Front()
	for el != nil {
		conn := el.Value.(net.Conn)
		conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
		i, err := conn.Write(p)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			// Write timedout. Assuming connection closed.
			nextel := el.Next()
			conn.Close()
			lw.conns.Remove(el)
			el = nextel
		} else {
			if err != nil {
				errs = append(errs, err.Error())
			}
			count += i
			el = el.Next()
		}
	}

	var err error = nil
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}
	return count, err
}

func (lw *ListenWriter) Close() error {
	err := lw.listener.Close()
	el := lw.conns.Front()
	for el != nil {
		conn := el.Value.(net.Conn)
		conn.Close()
		el = el.Next()
	}
	return err
}
