//go:build darwin || netbsd || freebsd || openbsd || dragonfly
// +build darwin netbsd freebsd openbsd dragonfly

package internal

import "syscall"

// Poll ...
type Poll struct {
	fd      int
	changes []syscall.Kevent_t
}

// OpenPoll ...
func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	l.fd = p
	_, err = syscall.Kevent(l.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}

	return l
}

// Wait ...
func (p *Poll) Wait(iter func(fd int) error) error {
	events := make([]syscall.Kevent_t, 128)
	for {
		// listener fd
		n, err := syscall.Kevent(p.fd, p.changes, events, nil)
		if err != nil && err != syscall.EINTR {
			return err
		}
		p.changes = p.changes[:0]
		for i := 0; i < n; i++ {
			// connection fd
			if fd := int(events[i].Ident); fd != 0 {
				if err := iter(fd); err != nil {
					return err
				}
			}
		}
	}
}

// AddRead ...
func (p *Poll) AddRead(fd int) {
	p.changes = append(p.changes,
		syscall.Kevent_t{
			Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_READ,
		},
	)
}

// AddReadWrite ...
func (p *Poll) AddReadWrite(fd int) {
	p.changes = append(p.changes,
		syscall.Kevent_t{
			Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_READ,
		},
		syscall.Kevent_t{
			Ident: uint64(fd), Flags: syscall.EV_ADD, Filter: syscall.EVFILT_WRITE,
		},
	)
}
