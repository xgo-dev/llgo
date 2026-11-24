// Host side of the soft-AP acceptance run: the station that dials the
// firmware's AP.
//
// esp-emu joins the firmware's soft AP as a station, leases an address from the
// firmware's own DHCP server, and forwards host connections from
// 127.0.0.1:<host port> into the firmware through that link. This client is
// therefore an ordinary host TCP client: if it gets a reply, the whole chain
// worked - emulator station associations, the firmware's DHCP lease, the
// firmware's lwIP listener on the AP netif, and the return path.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "hostfwd address esp-emu listens on")
	request := flag.String("request", "hello llgo-ap", "request line sent to the firmware")
	want := flag.String("want", "llgo-ap-ok", "reply the firmware must send")
	timeout := flag.Duration("timeout", 60*time.Second, "overall budget for connecting and reading the reply")
	flag.Parse()

	deadline := time.Now().Add(*timeout)
	var conn net.Conn
	for {
		c, err := net.DialTimeout("tcp", *addr, 2*time.Second)
		if err == nil {
			conn = c
			break
		}
		if time.Now().After(deadline) {
			fmt.Printf("[host] ap client FAIL dial %s: %v\n", *addr, err)
			os.Exit(1)
		}
		time.Sleep(500 * time.Millisecond)
	}
	defer conn.Close()
	fmt.Printf("[host] ap client connected to %s\n", *addr)

	if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		fmt.Printf("[host] ap client FAIL deadline: %v\n", err)
		os.Exit(1)
	}
	if _, err := conn.Write([]byte(*request + "\n")); err != nil {
		fmt.Printf("[host] ap client FAIL write: %v\n", err)
		os.Exit(1)
	}
	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("[host] ap client FAIL read: %v\n", err)
		os.Exit(1)
	}
	reply := string(buf[:n])
	fmt.Printf("[host] ap client request = %s\n", *request)
	fmt.Printf("[host] ap client reply = %s", reply)
	if len(reply) < len(*want) || reply[:len(*want)] != *want {
		fmt.Printf("[host] ap client FAIL unexpected reply\n")
		os.Exit(1)
	}
	fmt.Printf("[host] ap client PASS\n")
}
