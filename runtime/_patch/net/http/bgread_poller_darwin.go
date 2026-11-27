//go:build darwin && !baremetal

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import "syscall"

type httpBgPoller struct {
	kq    int
	wakeR int
	wakeW int
	evBuf [64]syscall.Kevent_t
}

func newHTTPBgPoller() (*httpBgPoller, error) {
	kq, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	syscall.CloseOnExec(kq)
	var pipeFds [2]int
	if err := syscall.Pipe(pipeFds[:]); err != nil {
		syscall.Close(kq)
		return nil, err
	}
	syscall.CloseOnExec(pipeFds[0])
	syscall.CloseOnExec(pipeFds[1])
	_ = syscall.SetNonblock(pipeFds[0], true)
	_ = syscall.SetNonblock(pipeFds[1], true)

	ch := [1]syscall.Kevent_t{{
		Ident:  uint64(pipeFds[0]),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
	}}
	if _, err := syscall.Kevent(kq, ch[:], nil, nil); err != nil {
		syscall.Close(pipeFds[0])
		syscall.Close(pipeFds[1])
		syscall.Close(kq)
		return nil, err
	}
	return &httpBgPoller{kq: kq, wakeR: pipeFds[0], wakeW: pipeFds[1]}, nil
}

func (p *httpBgPoller) add(fd int) error {
	ch := [1]syscall.Kevent_t{{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
	}}
	_, err := syscall.Kevent(p.kq, ch[:], nil, nil)
	return err
}

func (p *httpBgPoller) del(fd int) {
	ch := [1]syscall.Kevent_t{{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_DELETE,
	}}
	_, _ = syscall.Kevent(p.kq, ch[:], nil, nil)
}

func (p *httpBgPoller) wake() {
	var b [1]byte
	_, _ = syscall.Write(p.wakeW, b[:])
}

func (p *httpBgPoller) drainWake() {
	var buf [64]byte
	for {
		n, err := syscall.Read(p.wakeR, buf[:])
		if n <= 0 && err != syscall.EINTR {
			return
		}
	}
}

// wait blocks up to timeoutNs nanoseconds (or indefinitely if timeoutNs < 0)
// and appends ready watched fds into out.
func (p *httpBgPoller) wait(timeoutNs int64, out []int) []int {
	var ts *syscall.Timespec
	var tv syscall.Timespec
	if timeoutNs >= 0 {
		tv.Sec = timeoutNs / 1e9
		tv.Nsec = timeoutNs % 1e9
		ts = &tv
	}
	n, err := syscall.Kevent(p.kq, nil, p.evBuf[:], ts)
	if err != nil || n <= 0 {
		return out
	}
	for i := 0; i < n; i++ {
		fd := int(p.evBuf[i].Ident)
		if fd == p.wakeR {
			p.drainWake()
			continue
		}
		out = append(out, fd)
	}
	return out
}
