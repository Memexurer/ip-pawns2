package main

import (
	"io"
	"log"
	"net"
        "syscall"
        "bytes"

        "golang.org/x/sys/unix"
	"github.com/hashicorp/yamux"
)

var session *yamux.Session

func main() {
	// Get a TCP connection
	conn, err := net.Dial("tcp", "ssh.grupahakerskapiotr.us:2140")
	if err != nil {
		panic(err)
	}

	session, err = yamux.Server(conn, nil)
	if err != nil {
		panic(err)
	}

	for {
		stream, err := session.Accept()
		println("Acceping stream")
		if err != nil {
			panic(err)
		}

                go handleStream(stream)

	}
}

func handleStream(stream net.Conn) {

		buf := make([]byte, 48) //16 srcip + 32 domain
		_, err := stream.Read(buf)
		if err != nil {
			log.Fatalln(err)
			return
		}

                srcIp := net.IP(buf[0:16])
                destString := string(bytes.Trim(buf[16:], "\x00"))

		log.Printf("dialing %s -> %s", srcIp.String(), destString)
                 
		dialer := &net.Dialer{
			Control: func(network, address string, conn syscall.RawConn) error {
				var operr error
				if err := conn.Control(func(fd uintptr) {
					operr = syscall.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_FREEBIND, 1)
				}); err != nil {
					return err
				}
				return operr
			},
			LocalAddr: &net.TCPAddr { IP: srcIp },
		}

		prox, err := dialer.Dial("tcp", destString)
		buf = nil
		if err != nil {
			stream.Write([]byte{0x00})
			errBuff := make([]byte, 256)
			copy(errBuff, []byte(err.Error()))
			stream.Write(errBuff)
			return
		}
		stream.Write([]byte{0x01})
		// Start proxying

		go proxy(prox, stream)
		go proxy(stream, prox)
}

type closeWriter interface {
	CloseWrite() error
}

// proxy is used to suffle data from src to destination, and sends errors
// down a dedicated channel
func proxy(dst io.Writer, src io.Reader) {
	io.Copy(dst, src)
	if tcpConn, ok := dst.(closeWriter); ok {
		tcpConn.CloseWrite()
	}
}