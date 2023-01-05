package model

import (
	"fmt"
	"net"

	"github.com/hashicorp/yamux"
)

type Session struct {
	yamux *yamux.Session
}

type ForwarderServer struct {
	session *Session
}

func CreateSessionManager() *ForwarderServer {
	server := &ForwarderServer{}
	return server
}

func (sm *ForwarderServer) CheckAccess(user, password string) bool {
	if user == "api" && password == "apipass" {
		return true
	} else if sm.session != nil && password == "dorgproxy420" {
		return true
	}
	return false
}

func (sm *ForwarderServer) HandleConnection(conn net.Conn) error {
	addr := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	if sm.session != nil {
		return fmt.Errorf("someone has already connected")
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	sm.session = &Session{
		session,
	}
	fmt.Printf("%s connected!\n", addr)
	go func() {
		<-session.CloseChan()
		sm.session = nil
		fmt.Printf("%s disconnected\n", addr)
	}()
	return nil
}

func (sm *ForwarderServer) SessionCount() int {
	return 0
}

func (sm *ForwarderServer) Sessions() map[string]*Session {
	return nil
}

func (sm *ForwarderServer) OpenConnection(sourceAddress string, destinationAddress string) (net.Conn, error) {
	session := sm.session
	if session != nil {
		conn, err := session.yamux.Open()
		fmt.Printf("%s -> %s\n", sourceAddress, destinationAddress)
		if err != nil {
			return nil, err
		}

		err = writeConnectionAddress(conn, sourceAddress, destinationAddress)
		if err != nil {
			return nil, err
		}

		err = readError(conn)
		if err != nil {
			return nil, err
		}

		return conn, nil
	} else {
		return nil, fmt.Errorf("client is not connected")
	}
}
