// Command udpsrv is the host-side acceptance peer for the esp-emu UDP
// probe (_demo/esp-idf-wifi): a UDP echo server on 127.0.0.1:9002, the
// datagram analog of echosrv. The emulator's user-mode NAT (slirp-style)
// aliases its gateway address (192.168.4.1 in the default 192.168.4.0/24
// subnet) to the host, and forwards each guest datagram to the same port
// on the host's loopback, which is where this server listens. The emu
// firmware sends to 192.168.4.1:9002 and this process echoes each datagram
// back to its source address.
//
// Usage:
//
//	go run ./idfproj/host/udpsrv [listen-addr]   (default 127.0.0.1:9002)
//
// Every datagram and sender is logged so the firmware-side probe output
// can be cross-checked. There is no connection lifetime to manage: each
// datagram is answered as it arrives, forever.
package main

import (
	"log"
	"net"
	"os"
)

func main() {
	addr := "127.0.0.1:9002"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	srv, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatalf("udpsrv: listen %s: %v", addr, err)
	}
	log.Printf("udpsrv: listening on %s", srv.LocalAddr())
	buf := make([]byte, 2048)
	for {
		n, raddr, err := srv.ReadFrom(buf)
		if err != nil {
			log.Fatalf("udpsrv: read: %v", err)
		}
		log.Printf("udpsrv: datagram %d bytes from %s", n, raddr)
		if _, err := srv.WriteTo(buf[:n], raddr); err != nil {
			log.Printf("udpsrv: echo to %s: %v", raddr, err)
		}
	}
}
