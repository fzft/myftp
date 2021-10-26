package main

import (
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// FtpPassiveSocket need to implementation io.Reader
type FtpPassiveSocket struct {
	conn net.Conn
	addr string
	sync.Mutex
	port int
}

func NewFtpPassiveSocket(addr string) (*FtpPassiveSocket, error) {
	socket := &FtpPassiveSocket{
		addr: addr,
	}
	retries := 10
	var err error
	for i := 0; i < retries; i++ {
		err = socket.ListenAndServe()
		if err != nil && isErrorAddressAlreadyInUse(err) {
			continue
		}
		break
	}
	return socket, err
}

func (s *FtpPassiveSocket) ListenAndServe() (err error) {
	laddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("", strconv.Itoa(s.port)))
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", laddr)
	parts := strings.Split(ln.Addr().String(), ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return
	}
	s.port = port
	const acceptTimeout = 60 * time.Second
	err = ln.SetDeadline(time.Now().Add(acceptTimeout))
	if err != nil {
		return
	}
	s.Lock()
	go func() {
		defer s.Unlock()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		s.conn = conn
		ln.Close()
	}()
	return nil
}

func (s *FtpPassiveSocket) Port() int {
	return s.port
}

func (s *FtpPassiveSocket) Read(p []byte) (n int, err error) {
	s.Lock()
	defer s.Unlock()
	return s.conn.Read(p)
}

func (s *FtpPassiveSocket) ReadFrom(r io.Reader) (int64, error) {
	s.Lock()
	defer s.Unlock()

	// For normal TCPConn, this will use sendfile syscall; if not,
	// it will just downgrade to normal read/write procedure
	return io.Copy(s.conn, r)
}

func (s *FtpPassiveSocket) Write(p []byte) (n int, err error) {
	s.Lock()
	defer s.Unlock()
	return s.conn.Write(p)
}

func (s *FtpPassiveSocket) Close() error {
	s.Lock()
	defer s.Unlock()
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Detect if an error is "bind: address already in use"
//
// Originally from https://stackoverflow.com/a/52152912/164234
func isErrorAddressAlreadyInUse(err error) bool {
	errOpError, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	errSyscallError, ok := errOpError.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	errErrno, ok := errSyscallError.Err.(syscall.Errno)
	if !ok {
		return false
	}
	if errErrno == syscall.EADDRINUSE {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}
