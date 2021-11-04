package main

import (
	"fmt"
	"github.com/fzft/myftp/internal"
	"github.com/sirupsen/logrus"
	"syscall"
)

type Loop struct {
	fdconns      map[int]*FtpConn // fd -> conn
	poll         *internal.Poll
	packet       []byte
	log          *logrus.Logger
	eventHandler Event
}

//Run ...
func (l *Loop) Run() {
	l.poll.Wait(func(fd int) error {
		c := l.fdconns[fd]
		switch {
		case c == nil:
			return l.Accept(fd)
		case len(c.out) > 0:
			return l.Write(c)
		default:
			return l.Read(c)
		}
	})
}

//Accept ...
func (l *Loop) Accept(fd int) error {
	l.log.Infof("accept %d", fd)
	nfd, sa, err := syscall.Accept(fd)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	if err := syscall.SetNonblock(nfd, true); err != nil {
		return err
	}
	c := &FtpConn{fd: nfd, sa: sa, loop: l}
	c.out = nil
	l.fdconns[c.fd] = c
	l.poll.AddReadWrite(c.fd)
	c.out = []byte(fmt.Sprintf("%d %s\n", StatusReadyForNewUser, "welcome to myftp"))
	return nil
}

//Write
func (l *Loop) Write(c *FtpConn) error {
	l.log.Infof("write ")
	n, err := syscall.Write(c.fd, c.out)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return l.CloseConn(c)
	}

	// TODO:
	if n == len(c.out) {
		// release the connection output page if it goes over page size,
		// otherwise keep reusing existing page.
		if cap(c.out) > 4096 {
			c.out = nil
		} else {
			c.out = c.out[:0]
		}
	} else {
		c.out = c.out[n:]
	}
	return nil
}

//Read ...
func (l *Loop) Read(c *FtpConn) error {
	var in []byte
	n, err := syscall.Read(c.fd, l.packet)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return l.CloseConn(c)
	}
	l.log.Infof("read %d bytes", n)
	in = l.packet[:n]

	out, action := l.eventHandler.OnData(c, in)
	c.action = action
	if len(out) > 0 {
		c.out = append(c.out[:0], out...)
	}
	return nil
}

// CloseConn ...
func (l *Loop) CloseConn(c *FtpConn) error {
	return nil
}
