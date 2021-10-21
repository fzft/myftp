package main

import (
	"net"
	"os"
	"syscall"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

type Server struct {
	ln   net.Listener
	addr string
}

// Conn
type FtpConn struct {
	conn net.Conn
}

// Serve handle read buf
func (c *FtpConn) Serve() {

}

func NewServer(host, port string) *Server {
	return &Server{
		addr: net.JoinHostPort(host, port),
	}
}

func (s *Server) Serve() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}

	for {
		tcpConn, err := s.ln.Accept()
		if err != nil {
			// TODO close
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
		}
		ftpConn := s.NewConn(tcpConn)
		go ftpConn.Serve()
	}
}

func (s *Server) NewConn(conn net.Conn) *FtpConn {
	return &FtpConn{
		conn: conn,
	}
}

//Close use context to close server gracefully
func (s *Server) Close() {

}


func main() {
	host := "127.0.0.1"
	port := "8080"

	term := make(chan os.Signal)
	// stop server gracefully
	go func() {

	}()

	server := NewServer(host, port)
	server.Serve()
}
