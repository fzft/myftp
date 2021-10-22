package main

import (
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type FtpPassiveSocket struct {
	conn net.Conn
	addr string
	sync.Mutex
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
	laddr, err := net.ResolveTCPAddr("tcp", s.addr)
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", laddr)
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

		}
		s.conn = conn
		ln.Close()
	}()
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
