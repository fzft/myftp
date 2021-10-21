package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

type Server struct {
	ln     net.Listener
	addr   string
	logger *logrus.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

//FtpConn override net.Conn
type FtpConn struct {
	conn net.Conn
}

// Serve handle read buf
func (c *FtpConn) Serve() {

}

func NewServer(host, port string, logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc) *Server {
	return &Server{
		addr:   net.JoinHostPort(host, port),
		logger: logger,
		ctx: ctx,
		cancel: cancel,
	}
}

func (s *Server) Serve() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	s.logger.Infof("ftp server listening on %s", s.addr)
	go func() {
		for {
			tcpConn, err := s.ln.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
				default:
				}
				// TODO close
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					continue
				}
			}
			ftpConn := s.NewConn(tcpConn)
			go ftpConn.Serve()
		}
	}()
	return nil
}

func (s *Server) NewConn(conn net.Conn) *FtpConn {
	return &FtpConn{
		conn: conn,
	}
}

func (s *Server) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.ln != nil {
		return s.ln.Close()
	}
	// server wasnt even started
	return nil
}

//Close use context to close server gracefully
func (s *Server) Close() {

}

func main() {
	logger := logrus.New()
	host := "127.0.0.1"
	port := "8080"

	term := make(chan os.Signal)
	stopCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	// stop server gracefully
	server := NewServer(host, port, logger, ctx, cancel)
	server.Serve()

	go func() {
		signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
		<-term
		logger.Info("start to stop shutting down")
		server.Shutdown()
		stopCh <- struct{}{}
		// TODO teardown
	}()
	<-stopCh
	logger.Info("ftp server has been stopped")
}
