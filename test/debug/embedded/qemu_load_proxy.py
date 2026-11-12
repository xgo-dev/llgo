"""Forward RSP to QEMU, translating only OpenOCD's `reset halt` monitor command.

This is a simulator adapter, not an OpenOCD or physical flash implementation.
Memory writes and debugger execution are handled by the real QEMU GDB stub.
"""

import argparse
import json
import os
import queue
import socket
import sys
import threading


def packet(payload):
    return b"$" + payload + b"#" + f"{sum(payload) & 255:02x}".encode()


def translate(payload, log):
    if payload.startswith(b"qRcmd,"):
        command = bytes.fromhex(payload[6:].decode()).decode()
        if command == "reset halt":
            log.write(json.dumps({"event": "reset"}) + "\n")
            log.flush()
            return b"qRcmd," + b"system_reset".hex().encode()
    if payload[:1] in (b"M", b"X"):
        address, size = payload[1:].split(b":", 1)[0].split(b",", 1)
        log.write(json.dumps({"event": "write", "address": int(address, 16),
                              "bytes": int(size, 16)}) + "\n")
        log.flush()
    return payload


def forward(client_input, client_output, upstream, log):
    events = queue.Queue()

    def read(source, origin):
        try:
            while data := source(65536):
                events.put((origin, data))
            events.put((origin, b""))
        except OSError as error:
            events.put((origin, error))

    # Pipe readers work on Windows too, where select only accepts sockets.
    for source, origin in ((lambda size: os.read(client_input.fileno(), size), "client"),
                           (upstream.recv, "qemu")):
        threading.Thread(target=read, args=(source, origin), daemon=True).start()
    pending = b""
    while True:
        try:
            origin, data = events.get(timeout=30)
        except queue.Empty:
            raise TimeoutError("debugger/QEMU session inactive for 30 seconds") from None
        if isinstance(data, OSError):
            raise data
        if not data:
            return
        if origin == "qemu":
            client_output.write(data)
            client_output.flush()
            continue
        pending += data
        # RSP escapes reserved bytes, so a literal '#' terminates the payload.
        while pending:
            if pending[:1] != b"$":
                upstream.sendall(pending[:1])
                pending = pending[1:]
                continue
            end = pending.find(b"#")
            if end < 0 or len(pending) < end + 3:
                break
            payload = pending[1:end]
            rewritten = translate(payload, log)
            upstream.sendall(packet(rewritten) if rewritten != payload else pending[:end + 3])
            pending = pending[end + 3:]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--transport", choices=("stdio",), required=True)
    parser.add_argument("--qemu", type=int, required=True)
    parser.add_argument("--log", required=True)
    args = parser.parse_args()
    with socket.create_connection(("127.0.0.1", args.qemu), timeout=10) as upstream, \
            open(args.log, "w", encoding="utf-8") as log:
        upstream.settimeout(None)
        try:
            forward(sys.stdin.buffer, sys.stdout.buffer, upstream, log)
        finally:
            try:
                upstream.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass


if __name__ == "__main__":
    main()
