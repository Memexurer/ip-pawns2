package http

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
	"wasiproxy/model"
)

type HttpServer struct {
	listener  *Listener
	startTime time.Time
	sessions  *model.ForwarderServer
}

func CreateServer(sessions *model.ForwarderServer) *HttpServer {
	return &HttpServer{
		newListener(),
		time.Now(),
		sessions,
	}
}

func (s *HttpServer) Handle(conn net.Conn) {
	s.listener.Handle(conn)
}

func (s *HttpServer) Serve() {
	http.HandleFunc("/", s.handleInfoEndpoint)
	http.HandleFunc("/users", s.handleUsersEndpoint)
	http.Serve(s.listener, nil)
}

func (s *HttpServer) handleUsersEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s.sessions.Sessions())
}

func (s *HttpServer) handleInfoEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Motd   string `json:"motd"`
		Uptime string `json:"uptime"`
		Users  int    `json:"users"`
	}{"Running WasiProxy v1.0", time.Since(s.startTime).String(), s.sessions.SessionCount()})
}
