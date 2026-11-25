#!/usr/bin/env python3
"""Bumble BLE central peer for the llgo ble peripheral emu acceptance run.

esp-emu intercepts the firmware's HCI transport and forwards it to a Bumble
"virtual controller" over TCP (--ble-hci tcp:host:port). The emulated
ESP32-S31 is the peripheral in this probe: it advertises as "llgo-periph" and
serves the GATT table the Go probe registered. This script hosts that virtual
controller plus a second Bumble controller on the same LocalLink whose host is
the central: it scans, connects, discovers the attribute table, reads,
subscribes, writes twice, reads back, unsubscribes and disconnects - every
step logged so the run has evidence from both ends of the link.

Layout:

    [ emulated S31 Bluedroid host ] --HCI/TCP--> emu-controller  ~LocalLink~
                                                                     |
                                              central-controller ----+-- central host
                                              (scanning central: discover, read,
                                               subscribe, write x2, unsubscribe,
                                               disconnect)

The peripheral's attribute table (registered by the Go probe):

    0xFFF0  echo service (16-bit)
      0xFFF1  read | write | write-without-response - the echo value; starts
              as "hello" (WriteCharValue), each write is echoed back as a
              notification on 0xFFF2
      0xFFF2  notify - one notification per echo write, carrying the written
              bytes
    6c6c676f-2222-3333-4444-556677889900  128-bit service
      ...889901  read | write echo characteristic

Run first, then start esp-emu with --ble-hci tcp:127.0.0.1:9545:
    python3 bumble_central.py [--port 9545] [--name llgo-periph]
                              [--timeout 120]
"""
import argparse
import asyncio
import logging
import sys

DEFAULT_PORT = 9545
DEFAULT_NAME = "llgo-periph"
DEFAULT_ADDRESS = "C0:FF:EE:00:00:02"  # the central's own address


def adv_name(data: bytes):
    """The complete (0x09) or shortened (0x08) local name of one AD payload -
    the same walk ble.AdvName does on the firmware side."""
    i = 0
    while i < len(data):
        length = data[i]
        if length == 0 or i + 1 + length > len(data):
            return None
        ad_type = data[i + 1]
        if ad_type in (0x08, 0x09):
            return data[i + 2 : i + 1 + length].decode("utf-8", "replace")
        i += 1 + length
    return None

ECHO_SERVICE_UUID = "FFF0"
ECHO_CHR_UUID = "FFF1"
NOTIFY_CHR_UUID = "FFF2"
WIDE_SERVICE_UUID = "6c6c676f-2222-3333-4444-556677889900"
WIDE_CHR_UUID = "6c6c676f-2222-3333-4444-556677889901"


def patch_controller(Controller, hci):
    """Ack the commands Bumble's virtual controller does not implement.

    The IDF Bluedroid host sends LE-privacy housekeeping and other commands
    the virtual controller has no handler for; an unanswered command parks the
    host until its command timeout. Ack them with success. LE Read RSSI is
    patched for the same reason (unknown command). The same patch the
    peripheral-side peer script applies - the roles are flipped, the gaps are
    not.
    """

    def ack_ok(self, command):
        params = hci.HCI_GenericReturnParameters(data=bytes([0]))
        if isinstance(command, hci.HCI_SyncCommand):
            return params
        self.send_hci_packet(
            hci.HCI_Command_Complete_Event(
                num_hci_command_packets=1,
                command_opcode=command.op_code,
                return_parameters=params,
            )
        )
        return None

    for name in (
        "on_hci_le_set_privacy_mode_command",
        "on_hci_le_clear_resolving_list_command",
        "on_hci_le_add_device_to_resolving_list_command",
        "on_hci_le_remove_device_from_resolving_list_command",
        "on_hci_le_set_address_resolution_enable_command",
        "on_hci_le_read_supported_states_command",
    ):
        if not hasattr(Controller, name):
            setattr(Controller, name, ack_ok)

    def read_rssi(self, command):
        if hasattr(command, "connection_handle"):
            handle = command.connection_handle
        else:
            handle = int.from_bytes(command.parameters[:2], "little")
        data = bytes([0]) + handle.to_bytes(2, "little") + (-60 & 0xFF).to_bytes(1, "little")
        params = hci.HCI_GenericReturnParameters(data=data)
        if isinstance(command, hci.HCI_SyncCommand):
            return params
        self.send_hci_packet(
            hci.HCI_Command_Complete_Event(
                num_hci_command_packets=1,
                command_opcode=command.op_code,
                return_parameters=params,
            )
        )
        return None

    if not hasattr(Controller, "on_hci_read_rssi_command"):
        Controller.on_hci_read_rssi_command = read_rssi


def make_legacy_controller():
    """A controller that speaks legacy advertising only.

    The emulated device advertises with the legacy commands (the glue's
    advertising path is the legacy API set). Dropping the
    LE_EXTENDED_ADVERTISING feature bit keeps both sides consistent: the
    scanner reports through the legacy LE Advertising Report event and never
    sends extended scan requests, and a firmware built without extended
    advertising sees a controller it fully understands (the same patch the
    peripheral-side peer script applies for the flipped roles).
    """
    from bumble import hci
    from bumble.controller import Controller

    class LegacyOnly(Controller):
        le_features = Controller.le_features & ~hci.LeFeatureMask.LE_EXTENDED_ADVERTISING

    return LegacyOnly, hci, Controller


async def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--name", default=DEFAULT_NAME)
    parser.add_argument("--address", default=DEFAULT_ADDRESS)
    parser.add_argument("--timeout", type=float, default=120.0,
                        help="exit after this many seconds (keeps CI runs bounded)")
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    logging.basicConfig(level=logging.DEBUG if args.verbose else logging.WARNING)

    from bumble import hci
    from bumble.device import Device, Peer
    from bumble.host import Host
    from bumble.link import LocalLink
    from bumble.transport import open_transport

    Controller_, hci_, _Controller = make_legacy_controller()
    patch_controller(Controller_, hci_)

    link = LocalLink()

    # Controller the emulator attaches to over TCP; its address is what the
    # firmware sees as its own adapter address.
    emu_transport = await open_transport(f"tcp-server:_:{args.port}")
    emu_controller = Controller_(
        "emu-controller",
        host_source=emu_transport.source,
        host_sink=emu_transport.sink,
        link=link,
        public_address="00:11:22:33:44:55",
    )
    # LocalLink routes ACL data by the sender controller's random address.
    emu_controller.random_address = hci.Address(
        "00:11:22:33:44:55", hci.Address.PUBLIC_DEVICE_ADDRESS
    )
    print(f"[peer] virtual controller listening on tcp:{args.port}", flush=True)

    # The central: its own controller on the same link, its own host.
    central_controller = Controller_(
        "central-controller", link=link, public_address=args.address
    )
    central_host = Host()
    central_host.controller = central_controller
    central_controller.host = central_host
    central = Device(name="llgo-central", address=hci.Address(args.address), host=central_host)
    await central.power_on()
    print(f"[peer] central ready at {args.address}, scanning for '{args.name}'", flush=True)

    # Scan until the peripheral's advertisement shows up (legacy, active: the
    # scan response may carry the name when the advertising packet is full).
    found = asyncio.Event()
    target_address = None
    target_addr_type = None

    def on_advertisement(advertisement):
        nonlocal target_address, target_addr_type
        try:
            if found.is_set():
                return
            name = adv_name(advertisement.data_bytes)
            if name != args.name:
                return
            target_address = advertisement.address
            target_addr_type = advertisement.address.address_type
            print(f"[peer] scan found name={name} addr={target_address} "
                  f"addrtype={target_addr_type} rssi={advertisement.rssi}", flush=True)
            found.set()
        except Exception as exc:
            print(f"[peer] adv listener error: {exc!r}", flush=True)

    central.on("advertisement", on_advertisement)
    await central.start_scanning(legacy=True, active=True)
    try:
        await asyncio.wait_for(found.wait(), timeout=30)
    except asyncio.TimeoutError:
        print(f"[peer] FAIL: no advertisement named {args.name} within 30s", flush=True)
        await central.stop_scanning()
        return 1
    await central.stop_scanning()

    connection = await central.connect_le(target_address, timeout=15.0)
    print(f"[peer] connected handle={connection.handle} peer={connection.peer_address} "
          f"mtu={connection.att_mtu}", flush=True)

    def on_mtu():
        print(f"[peer] mtu update handle={connection.handle} mtu={connection.att_mtu}",
              flush=True)

    connection.on(connection.EVENT_CONNECTION_ATT_MTU_UPDATE, on_mtu)

    def on_disconnection(reason):
        print(f"[peer] disconnected reason=0x{reason:02x}", flush=True)

    connection.on(connection.EVENT_DISCONNECTION, on_disconnection)

    peer = Peer(connection)
    print("[peer] discovering services...", flush=True)
    await peer.discover_services()
    await peer.discover_characteristics()

    for svc in peer.services:
        chars = " ".join(str(c.uuid) for c in svc.characteristics)
        print(f"[peer] service uuid={svc.uuid} chars=[{chars}]", flush=True)

    echo_svcs = peer.get_services_by_uuid(ECHO_SERVICE_UUID)
    wide_svcs = peer.get_services_by_uuid(WIDE_SERVICE_UUID)
    echo_chars = peer.get_characteristics_by_uuid(ECHO_CHR_UUID)
    notify_chars = peer.get_characteristics_by_uuid(NOTIFY_CHR_UUID)
    wide_chars = peer.get_characteristics_by_uuid(WIDE_CHR_UUID)
    if not echo_svcs or not wide_svcs or not echo_chars or not notify_chars or not wide_chars:
        print(f"[peer] FAIL: table incomplete svc={bool(echo_svcs)}/{bool(wide_svcs)} "
              f"chr={bool(echo_chars)}/{bool(notify_chars)}/{bool(wide_chars)}", flush=True)
        return 1
    echo_chr, notify_chr, wide_chr = echo_chars[0], notify_chars[0], wide_chars[0]
    print(f"[peer] discovered echo uuid={ECHO_CHR_UUID} props={echo_chr.properties!r}",
          flush=True)
    print(f"[peer] discovered notify uuid={NOTIFY_CHR_UUID} props={notify_chr.properties!r}",
          flush=True)
    print(f"[peer] discovered wide uuid={WIDE_CHR_UUID} props={wide_chr.properties!r}",
          flush=True)

    # The initial value: what WriteCharValue put in the cache before the
    # advertisement started.
    initial = await peer.read_value(echo_chr)
    print(f"[peer] read echo len={len(initial)} value={initial!r}", flush=True)
    if bytes(initial) != b"hello":
        print(f"[peer] FAIL: initial echo value {initial!r} != b'hello'", flush=True)
        return 1

    # Notifications arrive as the peripheral echoes the writes.
    received = asyncio.Queue()
    await notify_chr.subscribe(lambda value: received.put_nowait(bytes(value)),
                               prefer_notify=True)
    print("[peer] subscribed notify", flush=True)

    # Phase 1: acknowledged write, echoed back.
    await peer.write_value(echo_chr, b"ping-00", with_response=True)
    print("[peer] echo write uuid=FFF1 len=7 value=b'ping-00'", flush=True)
    got = await asyncio.wait_for(received.get(), timeout=10)
    print(f"[peer] notify n=1 len={len(got)} value={got!r}", flush=True)
    if got != b"ping-00":
        print(f"[peer] FAIL: notify mismatch {got!r}", flush=True)
        return 1

    # Phase 2: unacknowledged write command, echoed back the same way.
    await peer.write_value(echo_chr, b"ping-01", with_response=False)
    print("[peer] echo write uuid=FFF1 len=7 value=b'ping-01'", flush=True)
    got = await asyncio.wait_for(received.get(), timeout=10)
    print(f"[peer] notify n=2 len={len(got)} value={got!r}", flush=True)
    if got != b"ping-01":
        print(f"[peer] FAIL: notify mismatch {got!r}", flush=True)
        return 1

    # The write also landed in the peripheral's value cache: the read-back
    # returns the last written value.
    back = await peer.read_value(echo_chr)
    print(f"[peer] read back len={len(back)} value={back!r}", flush=True)
    if bytes(back) != b"ping-01":
        print(f"[peer] FAIL: read-back mismatch {back!r}", flush=True)
        return 1

    # Phase 3: the 128-bit path, write and read back.
    await peer.write_value(wide_chr, b"u128", with_response=True)
    print("[peer] wide write len=4 value=b'u128'", flush=True)
    back = await peer.read_value(wide_chr)
    print(f"[peer] wide read back len={len(back)} value={back!r}", flush=True)
    if bytes(back) != b"u128":
        print(f"[peer] FAIL: wide read-back mismatch {back!r}", flush=True)
        return 1

    # Phase 4: unsubscribe and disconnect; the peripheral logs the CCCD write
    # and the disconnection on its side.
    await notify_chr.unsubscribe()
    print("[peer] unsubscribed notify", flush=True)
    await connection.disconnect()

    print("[peer] PASS: scan, connect, discover, read, subscribe, write, echo, "
          "unsubscribe, disconnect", flush=True)
    print("[peer] done", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
