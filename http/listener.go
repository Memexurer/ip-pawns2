package http

import (
	"errors"
	"net"
)

type Listener struct {
	ch   chan net.Conn
	addr net.Addr
}

// newListener creates a channel listener. The addr argument
// is the listener's network address.
func newListener() *Listener {
	return &Listener{
		ch:   make(chan net.Conn),
		addr: &net.IPAddr{},
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	c, ok := <-l.ch
	if !ok {
		return nil, errors.New("closed")
	}
	return c, nil
}

func (l *Listener) Close() error { return nil }

func (l *Listener) Addr() net.Addr { return l.addr }

func (l *Listener) Handle(c net.Conn) error {
	l.ch <- c
	return nil
}
