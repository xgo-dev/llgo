"""Exercise the load adapter over pipes, without reserving TCP ports."""

from contextlib import ExitStack
import importlib.util
import io
import json
import os
from pathlib import Path
import queue
import threading
from types import SimpleNamespace
import unittest


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "qemu_load_proxy", ROOT / "test/debug/embedded/qemu_load_proxy.py")
PROXY = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PROXY)


class LoadProxyTests(unittest.TestCase):
    def read_exact(self, source, size):
        result = queue.Queue()
        threading.Thread(target=lambda: result.put(source.read(size)), daemon=True).start()
        return result.get(timeout=5)

    def test_pipe_transport_preserves_and_rewrites_rsp(self):
        with ExitStack() as stack:
            def pipe():
                reader, writer = os.pipe()
                return (stack.enter_context(os.fdopen(reader, "rb")),
                        stack.enter_context(os.fdopen(writer, "wb")))

            client_input, client_writer = pipe()
            client_reader, client_output = pipe()
            qemu_reader, qemu_input = pipe()
            qemu_output, qemu_writer = pipe()
            def send(data):
                qemu_input.write(data)
                qemu_input.flush()

            upstream = SimpleNamespace(recv=lambda size: os.read(qemu_output.fileno(), size),
                                       sendall=send)
            log, finished = io.StringIO(), queue.Queue()

            def run():
                try:
                    PROXY.forward(client_input, client_output, upstream, log)
                    finished.put(None)
                except Exception as error:
                    finished.put(error)

            worker = threading.Thread(target=run, daemon=True)
            worker.start()
            try:
                reset = PROXY.packet(b"qRcmd," + b"reset halt".hex().encode())
                # Exceed one pipe-read buffer so packet reassembly is required.
                write = PROXY.packet(b"M2000,10000:" + b"ab" * 65536)
                client_writer.write(reset[:5])
                client_writer.flush()
                client_writer.write(reset[5:] + write)
                client_writer.flush()
                expected = PROXY.packet(b"qRcmd," + b"system_reset".hex().encode()) + write
                self.assertEqual(self.read_exact(qemu_reader, len(expected)), expected)

                reply = b"+" + PROXY.packet(b"OK") + b"x" * 100000
                qemu_writer.write(reply)
                qemu_writer.flush()
                self.assertEqual(self.read_exact(client_reader, len(reply)), reply)
                self.assertEqual([json.loads(line) for line in log.getvalue().splitlines()],
                                 [{"event": "reset"},
                                  {"event": "write", "address": 0x2000, "bytes": 65536}])
            finally:
                client_writer.close()
                qemu_writer.close()
                worker.join(timeout=5)
            self.assertFalse(worker.is_alive(), "adapter did not exit on pipe EOF")
            self.assertIsNone(finished.get(timeout=5))


if __name__ == "__main__":
    unittest.main()
