package main

import (
	"fmt"
	"net"
    "crypto/tls"
    "io"
    "log"
    "strings"
    "net/http"
    "encoding/base64"
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
 
    fmt.Printf("%s is connecting to %s\n", user, r.Host)
    dest_conn, err := sessions.OpenConnection(user, r.Host)
    if err != nil {
        fmt.Printf("Open connection: %v\n", err)
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        panic(err)
        return
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
func handleHTTP(w http.ResponseWriter, req *http.Request) {
    resp, err := http.DefaultTransport.RoundTrip(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    defer resp.Body.Close()
    copyHeader(w.Header(), resp.Header)
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
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
                handleHTTP(w, r)
            }
        }),
        // Disable HTTP/2.
        TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
    }
    fmt.Println("serving!")
    log.Fatal(server.ListenAndServe())
//2139 port
}
