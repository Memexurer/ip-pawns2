package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"wasiproxy/model"
)

var sessions *model.ForwarderServer

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	val, ok := r.Header["Proxy-Authorization"]
	if !ok {
		w.Header().Set("Proxy-Authenticate", "Basic realm=\"dorgproxy\"")
		http.Error(w, "Authorization required", http.StatusProxyAuthRequired)
		return
	}

	split := strings.Split(val[0], " ")
	if split[0] != "Basic" {
		http.Error(w, "Authorization required", http.StatusProxyAuthRequired)
		return
	}

	decodedRaw, err := base64.StdEncoding.DecodeString(split[1])
	if err != nil {
		http.Error(w, "Authorization required", http.StatusProxyAuthRequired)
		return
	}
	decoded := string(decodedRaw)

	user := decoded[:strings.LastIndex(decoded, ":")]
	pass := decoded[strings.LastIndex(decoded, ":")+1:]
	if pass != "dorgproxy420" {
		http.Error(w, "Authorization required", http.StatusProxyAuthRequired)
		return
	}

	var dest_conn net.Conn

	attempts := 3
	for {
		if attempts == 0 {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			panic(err)
		}

		fmt.Printf("%s is connecting to %s (attempt %d)\n", user, r.Host, attempts)
		conn, err := sessions.OpenConnection(user, r.Host)
		if err != nil {
			fmt.Printf("Open connection (attempt %d): %v\n", attempts, err)
			attempts--
		} else {
			dest_conn = conn
			break
		}
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		fmt.Println("epic hijack fail!!")
		http.Error(w, "Hijacking not supported", http.StatusServiceUnavailable)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	sessions = model.CreateSessionManager()

	ln, err := net.Listen("tcp", "0.0.0.0:2140")
	if err != nil {
		return
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				println("Errors accepting!")
			}

			err = sessions.HandleConnection(conn)
			if err != nil {
				fmt.Printf("Error accepting: %s\n", err)
			}
		}
	}()

	server := &http.Server{
		Addr: "0.0.0.0:2139",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				http.Error(w, "We don't support http.", http.StatusServiceUnavailable)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	fmt.Println("serving!")
	log.Fatal(server.ListenAndServe())
	//2139 port
}
