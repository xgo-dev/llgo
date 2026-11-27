//go:build linux && !baremetal && go1.27 && !go1.28

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

const (
	bgEpollOneShot uint32 = 1 << 30
	bgEpollET      uint32 = 1 << 31
)

type httpBgPoller struct {
	epfd  int
	wakeR int
	wakeW int
	evBuf [64]syscall.EpollEvent
}

func newHTTPBgPoller() (*httpBgPoller, error) {
	epfd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	var pipeFds [2]int
	if err := syscall.Pipe2(pipeFds[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC); err != nil {
		syscall.Close(epfd)
		return nil, err
	}
	ev := syscall.EpollEvent{
		Events: uint32(syscall.EPOLLIN) | bgEpollET,
		Fd:     int32(pipeFds[0]),
	}
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, pipeFds[0], &ev); err != nil {
		syscall.Close(pipeFds[0])
		syscall.Close(pipeFds[1])
		syscall.Close(epfd)
		return nil, err
	}
	return &httpBgPoller{epfd: epfd, wakeR: pipeFds[0], wakeW: pipeFds[1]}, nil
}

func (p *httpBgPoller) add(fd int) error {
	ev := syscall.EpollEvent{
		Events: uint32(syscall.EPOLLIN|syscall.EPOLLRDHUP) | bgEpollOneShot,
		Fd:     int32(fd),
	}
	err := syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_ADD, fd, &ev)
	if err == syscall.EEXIST {
		err = syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_MOD, fd, &ev)
	}
	return err
}

func (p *httpBgPoller) del(fd int) {
	_ = syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
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

func (p *httpBgPoller) wait(timeoutNs int64, out []int) []int {
	ms := -1
	if timeoutNs >= 0 {
		ms = int((timeoutNs + 1e6 - 1) / 1e6)
		if ms < 1 {
			ms = 1
		}
	}
	n, err := syscall.EpollWait(p.epfd, p.evBuf[:], ms)
	if err != nil || n <= 0 {
		return out
	}
	for i := 0; i < n; i++ {
		fd := int(p.evBuf[i].Fd)
		if fd == p.wakeR {
			p.drainWake()
			continue
		}
		out = append(out, fd)
	}
	return out
}
