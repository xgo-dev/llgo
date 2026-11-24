#!/usr/bin/env python3
"""Bumble BLE peripheral peer for the llgo ble emu acceptance run.

esp-emu intercepts the firmware's HCI transport and forwards it to a Bumble
"virtual controller" over TCP (--ble-hci tcp:host:port). This script hosts
that controller plus a second Bumble controller on the same LocalLink whose
host advertises as a connectable peripheral, so the emulated ESP32-S31 (a BLE
central in these probes) can find it by scanning and connect to it.

Layout:

    [ emulated S31 NimBLE host ] --HCI/TCP--> emu-controller  ~LocalLink~
                                                                 |
                                              peer-controller ---+-- peer host
                                              (advertising peripheral)

Run first, then start esp-emu with --ble-hci tcp:127.0.0.1:9544:
    python3 bumble_peer.py [--port 9544] [--address C0:FF:EE:00:00:01]
                           [--name llgo-peer] [--timeout 90]
"""
import argparse
import asyncio
import logging
import sys

DEFAULT_PORT = 9544
DEFAULT_ADDRESS = "C0:FF:EE:00:00:01"
DEFAULT_NAME = "llgo-peer"


def patch_controller(Controller, hci):
    """Ack the commands Bumble's virtual controller does not implement.

    The IDF NimBLE host sends LE-privacy housekeeping right after host sync
    (LE Clear/Add/Remove Resolving List, LE Set Privacy Mode). Bumble's
    controller does not implement them and answers nothing for unparsed
    commands, so NimBLE waits out its 20 s command timeout and declares the
    controller unresponsive. Ack them with success. LE Read RSSI is patched
    for the same reason (unknown command)
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


async def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--address", default=DEFAULT_ADDRESS)
    parser.add_argument("--name", default=DEFAULT_NAME)
    parser.add_argument("--timeout", type=float, default=90.0,
                        help="exit after this many seconds (keeps CI runs bounded)")
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    logging.basicConfig(level=logging.DEBUG if args.verbose else logging.WARNING)

    from bumble import hci
    from bumble.controller import Controller
    from bumble.device import Device
    from bumble.host import Host
    from bumble.link import LocalLink
    from bumble.transport import open_transport

    patch_controller(Controller, hci)

    class LegacyOnlyController(Controller):
        """A controller that speaks legacy advertising only.

        Two Bumble branches key off the LE_EXTENDED_ADVERTISING feature bit,
        and both would defeat a firmware that scans with NimBLE's legacy
        discovery procedure (ble_gap_disc, which parses only the legacy LE
        Advertising Report event):

          * Device.start_advertising sends the extended advertising commands
            and an AdvExtInd PDU (Device.supports_le_extended_advertising);
          * Controller.on_advertising_pdu reports every received advertising
            PDU as HCI_LE_EXTENDED_ADVERTISING_REPORT when the feature is
            set - including on the scanning controller.

        Dropping the bit makes the peer advertise with the legacy commands
        (ADV_IND) and the scanning controller report through the legacy event,
        which is exactly what an IDF NimBLE host built without
        CONFIG_BT_NIMBLE_EXT_ADV expects.
        """

        le_features = Controller.le_features & ~hci.LeFeatureMask.LE_EXTENDED_ADVERTISING

    link = LocalLink()

    # Controller the emulator attaches to over TCP; its address is what the
    # firmware sees as its own adapter address.
    emu_transport = await open_transport(f"tcp-server:_:{args.port}")
    emu_controller = LegacyOnlyController(
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
    print(f"[peer] virtual controller listening on tcp:{args.port}, "
          f"peer address {args.address} name '{args.name}'", flush=True)

    # The peripheral: its own controller on the same link, its own host.
    peer_controller = LegacyOnlyController(
        "peer-controller", link=link, public_address=args.address
    )
    peer_host = Host()
    peer_host.controller = peer_controller
    peer_controller.host = peer_host
    peer = Device(name=args.name, address=hci.Address(args.address), host=peer_host)
    await peer.power_on()

    connected = asyncio.Event()

    def on_connection(connection):
        print(f"[peer] central connected handle={connection.handle} "
              f"peer={connection.peer_address}", flush=True)

    def on_disconnection(reason):
        print(f"[peer] central disconnected reason=0x{reason:02x}", flush=True)

    peer.on(peer.EVENT_CONNECTION, on_connection)
    peer.on(peer.EVENT_CONNECTION_FAILURE, lambda e: print(f"[peer] connection failed: {e}", flush=True))

    from bumble.core import AdvertisingData
    from bumble import data_types

    await peer.start_advertising(
        advertising_data=bytes(AdvertisingData([data_types.CompleteLocalName(args.name)])),
        auto_restart=True,
    )
    print(f"[peer] advertising as {peer.random_address} (static random), "
          f"connectable, name '{args.name}'", flush=True)

    # Per-connection disconnection hooks are registered when a connection
    # arrives; keep the run bounded either way.
    async def watch():
        while True:
            for connection in peer.connections.values():
                if not getattr(connection, "_llgo_hooked", False):
                    connection._llgo_hooked = True
                    connection.on(connection.EVENT_DISCONNECTION, on_disconnection)
            if peer.connections:
                connected.set()
            await asyncio.sleep(0.2)

    watcher = asyncio.create_task(watch())
    try:
        await asyncio.wait_for(connected.wait(), timeout=args.timeout)
        print("[peer] PASS: a central connected", flush=True)
        # Stay up so the central side can disconnect on its own and the
        # emulator's HCI stream never sees the transport vanish mid-run.
        await asyncio.sleep(max(args.timeout - 10, 5))
    except asyncio.TimeoutError:
        print(f"[peer] FAIL: no connection within {args.timeout}s", flush=True)
        return 1
    finally:
        watcher.cancel()
        try:
            await peer.stop_advertising()
        except Exception:
            pass
    print("[peer] done", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
