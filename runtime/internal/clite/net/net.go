/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package net

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = true
)

const (
	AF_UNSPEC          = 0       // unspecified
	AF_UNIX            = 1       // local to host (pipes)
	AF_LOCAL           = AF_UNIX // backward compatibility
	AF_INET            = 2       // internetwork: UDP, TCP, etc.
	AF_IMPLINK         = 3       // arpanet imp addresses
	AF_PUP             = 4       // pup protocols: e.g. BSP
	AF_CHAOS           = 5       // mit CHAOS protocols
	AF_NS              = 6       // XEROX NS protocols
	AF_ISO             = 7       // ISO protocols
	AF_OSI             = AF_ISO
	AF_ECMA            = 8       // European computer manufacturers
	AF_DATAKIT         = 9       // datakit protocols
	AF_CCITT           = 10      // CCITT protocols, X.25 etc
	AF_SNA             = 11      // IBM SNA
	AF_DECnet          = 12      // DECnet
	AF_DLI             = 13      // DEC Direct data link interface
	AF_LAT             = 14      // LAT
	AF_HYLINK          = 15      // NSC Hyperchannel
	AF_APPLETALK       = 16      // Apple Talk
	AF_ROUTE           = 17      // Internal Routing Protocol
	AF_LINK            = 18      // Link layer interface
	pseudo_AF_XTP      = 19      // eXpress Transfer Protocol (no AF)
	AF_COIP            = 20      // connection-oriented IP, aka ST II
	AF_CNT             = 21      // Computer Network Technology
	pseudo_AF_RTIP     = 22      // Help Identify RTIP packets
	AF_IPX             = 23      // Novell Internet Protocol
	AF_SIP             = 24      // Simple Internet Protocol
	pseudo_AF_PIP      = 25      // Help Identify PIP packets
	AF_NDRV            = 27      // Network Driver 'raw' access
	AF_ISDN            = 28      // Integrated Services Digital Network
	AF_E164            = AF_ISDN // CCITT E.164 recommendation
	pseudo_AF_KEY      = 29      // Internal key-management function
	AF_INET6           = 30      // IPv6
	AF_NATM            = 31      // native ATM access
	AF_SYSTEM          = 32      // Kernel event messages
	AF_NETBIOS         = 33      // NetBIOS
	AF_PPP             = 34      // PPP communication protocol
	pseudo_AF_HDRCMPLT = 35      // Used by BPF to not rewrite headers in interface output routine
	AF_RESERVED_36     = 36      // Reserved for internal usage
	AF_IEEE80211       = 37      // IEEE 802.11 protocol
	AF_UTUN            = 38
	AF_VSOCK           = 40 // VM Sockets
	AF_MAX             = 41
)

const (
	SOCK_STREAM    = 1 // stream socket
	SOCK_DGRAM     = 2 // datagram socket
	SOCK_RAW       = 3 // raw-protocol interface
	SOCK_RDM       = 4 // reliably-delivered message
	SOCK_SEQPACKET = 5 // sequenced packet stream
)

const (
	EAI_ADDRFAMILY = iota + 1 /* address family for hostname not supported */
	EAI_AGAIN                 /* temporary failure in name resolution */
	EAI_BADFLAGS              /* invalid value for ai_flags */
	EAI_FAIL                  /* non-recoverable failure in name resolution */
	EAI_FAMILY                /* ai_family not supported */
	EAI_MEMORY                /* memory allocation failure */
	EAI_NODATA                /* no address associated with hostname */
	EAI_NONAME                /* hostname nor servname provided, or not known */
	EAI_SERVICE               /* servname not supported for ai_socktype */
	EAI_SOCKTYPE              /* ai_socktype not supported */
	EAI_SYSTEM                /* system error returned in errno */
	EAI_BADHINTS              /* invalid value for hints */
	EAI_PROTOCOL              /* resolved protocol is unknown */
	EAI_OVERFLOW              /* argument buffer overflow */
)

const (
	ALIGNSIZE = unsafe.Sizeof(c.LongLong(0))
	MAXSIZE   = 128
	PAD1_SIZE = ALIGNSIZE - unsafe.Sizeof(byte(0)) - unsafe.Sizeof(byte(0))
	PAD2_SIZE = MAXSIZE - unsafe.Sizeof(byte(0)) - unsafe.Sizeof(byte(0)) - PAD1_SIZE - ALIGNSIZE
)

// (TODO) merge to inet
const INET_ADDRSTRLEN = 16

type SockaddrIn struct {
	Len    uint8
	Family uint8
	Port   uint16
	Addr   InAddr
	Zero   [8]c.Char
}

type SockaddrIn6 struct {
	Len      uint8
	Family   uint8
	Port     uint16
	Flowinfo c.Uint
	Addr     In6Addr
	ScopeId  c.Uint
}

type SockaddrStorage struct {
	Len    uint8
	Family uint8
	pad1   [PAD1_SIZE]c.Char
	align  c.LongLong
	pad2   [PAD2_SIZE]c.Char
}

type InAddr struct {
	Addr c.Uint
}

type In6Addr struct {
	U6Addr [16]uint8
}

type SockAddr struct {
	Len    uint8
	Family uint8
	Data   [14]c.Char
}

type Hostent struct {
	Name     *c.Char  // official name of host
	Aliases  **c.Char // null-terminated array of alternate names for the host
	AddrType c.Int    // host address type
	Length   c.Int    // length of address
	AddrList **c.Char // null-terminated array of addresses for the host
}

//go:linkname Socket C.socket
func Socket(domain c.Int, typ c.Int, protocol c.Int) c.Int

//go:linkname Bind C.bind
func Bind(sockfd c.Int, addr *SockaddrIn, addrlen c.Uint) c.Int

//go:linkname Connect C.connect
func Connect(sockfd c.Int, addr *SockAddr, addrlen c.Uint) c.Int

//go:linkname Listen C.listen
func Listen(sockfd c.Int, backlog c.Int) c.Int

//go:linkname Accept C.accept
func Accept(sockfd c.Int, addr *SockaddrIn, addrlen *c.Uint) c.Int

//go:linkname GetHostByName C.gethostbyname
func GetHostByName(name *c.Char) *Hostent

// (TODO) merge to inet
//
//go:linkname InetNtop C.inet_ntop
func InetNtop(af c.Int, src c.Pointer, dst *c.Char, size c.Uint) *c.Char

//go:linkname InetAddr C.inet_addr
func InetAddr(s *c.Char) c.Uint

//go:linkname Send C.send
func Send(c.Int, c.Pointer, uintptr, c.Int) c.Long

//go:linkname Recv C.recv
func Recv(c.Int, c.Pointer, uintptr, c.Int) c.Long

//go:linkname SetSockOpt C.setsockopt
func SetSockOpt(socket c.Int, level c.Int, optionName c.Int, optionValue c.Pointer, sockLen c.Uint) c.Int

//go:linkname Ntohs C.ntohs
func Ntohs(x uint16) uint16

//go:linkname Htons C.htons
func Htons(x uint16) uint16

//go:linkname Ntohl C.ntohl
func Ntohl(x c.Uint) c.Uint

//go:linkname Htonl C.htonl
func Htonl(x c.Uint) c.Uint

// -----------------------------------------------------------------------------

type AddrInfo struct {
	Flags     c.Int
	Family    c.Int
	SockType  c.Int
	Protocol  c.Int
	AddrLen   c.Uint
	CanOnName *c.Char
	Addr      *SockAddr
	Next      *AddrInfo
}

//go:linkname Getaddrinfo C.getaddrinfo
func Getaddrinfo(host *c.Char, port *c.Char, addrInfo *AddrInfo, result **AddrInfo) c.Int

//go:linkname Freeaddrinfo C.freeaddrinfo
func Freeaddrinfo(addrInfo *AddrInfo) c.Int

// -----------------------------------------------------------------------------
