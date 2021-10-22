package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

type Auth struct {
	need     bool
	username string
	password string
}

type Server struct {
	ln     net.Listener
	addr   string
	logger *logrus.Logger
	ctx    context.Context
	cancel context.CancelFunc
	auth   Auth
}

//FtpConn override net.Conn
type FtpConn struct {
	conn          net.Conn
	controlReader *bufio.Reader
	controlWriter *bufio.Writer
	logger        *logrus.Logger
	closed        bool
	auth          Auth
	isLogin       bool
	dataConn *FtpPassiveSocket
}

// Serve handle read buf
func (c *FtpConn) Serve() {
	c.logger.Infof("from %s connection established", c.conn.RemoteAddr().String())
	c.WriteMessage(StatusReadyForNewUser, "welcome to myftp")
	for {
		line, err := c.controlReader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
			}
			break
		}
		c.receiveLine(line)
		if c.closed == true {
			break
		}
	}
	c.Close()
	c.logger.Infof("from %s connection closed", c.conn.RemoteAddr().String())
}

func (c *FtpConn) passiveListenIP() string {
	var listenIP string
	listenIP = c.conn.LocalAddr().(*net.TCPAddr).IP.String()

	lastIdx := strings.LastIndex(listenIP, ":")
	if lastIdx <= 0 {
		return listenIP
	}
	return listenIP[:lastIdx]
}

func (c *FtpConn) WriteMessage(code int, message string) (int, error) {
	line := fmt.Sprintf("%d %s\r\n", code, message)
	wrote, err := c.controlWriter.WriteString(line)
	c.controlWriter.Flush()
	return wrote, err
}

func (c *FtpConn) Close() {
	defer func() {
		c.closed = true
	}()
	c.conn.Close()
}

// receiveLine handle concrete read
func (c *FtpConn) receiveLine(line string) {
	command, params := c.parseLine(line)
	c.logger.Infof("command %s, params %s\n", command, params)
	commandObj := commands[command]
	if commandObj == nil {
		c.WriteMessage(StatusUnrecognizedCommand, "command unrecognized")
		return
	}
	// validate auth required
	if commandObj.AuthRequired() && c.isLogin == false {
		c.WriteMessage(StatusNotLogin, "Not logged in.")
		return
	}
	commandObj.Execute(c, params)
}

func (c *FtpConn) parseLine(line string) (string, string) {
	params := strings.SplitN(strings.Trim(line, "\r\n"), " ", 2)
	if len(params) == 1 {
		return params[0], ""
	}
	return params[0], params[1]
}

func NewServer(host, port string, logger *logrus.Logger, ctx context.Context, cancel context.CancelFunc) *Server {
	return &Server{
		addr:   net.JoinHostPort(host, port),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		auth: Auth{
			need:     true,
			username: "mos",
			password: "mos",
		},
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
					return
				default:
				}
				// TODO close
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					continue
				}
			}
			ftpConn := s.NewConn(tcpConn, s.logger, s.auth)
			go ftpConn.Serve()
		}
	}()
	return nil
}

func (s *Server) NewConn(conn net.Conn, logger *logrus.Logger, auth Auth) *FtpConn {
	return &FtpConn{
		conn:          conn,
		logger:        logger,
		controlReader: bufio.NewReader(conn),
		controlWriter: bufio.NewWriter(conn),
		auth:          auth,
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
