#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Espressif Systems (Shanghai) CO LTD
#
# SPDX-License-Identifier: Apache-2.0

"""Bumble Classic BT peer for esp-emu: serves a virtual controller to the
emulator on tcp-server:_:9544, then a second Bumble device on the same link
connects to the firmware over BR/EDR and drives one profile:

  a2dp FILE.sbc   stream SBC to an A2DP sink        (a2dp_sink_stream)
  spp+a2dp FILE   the llgo demo's whole acceptance: SPP payload round first,
                  then a fresh page for the A2DP stream (one process, because
                  the emulator's HCI bridge is a single TCP connection - a
                  second process cannot take the controller over mid-run)
  spp             RFCOMM SPP client, send data      (bt_spp_acceptor)
  l2cap [PSM]     L2CAP client, send data           (bt_l2cap_server)
  hid             HID host, count input reports     (bt_hid_mouse_device)
  hfp             HF: SLC, then accept the AG's eSCO (hfp_ag; audio data
                  does not cross Bumble's link)

Start it first, then esp-emu with --ble-hci tcp:localhost:9544; it connects
after --wait seconds (default 10). BUMBLE_<PROFILE>_PASS means the Bumble side
completed; whether the firmware received the data is in its own log.
BUMBLE_DEBUG=1 logs the protocols.

Bumble's virtual controller covers far less of BR/EDR than Bluedroid uses; the
Controller patches below fill each gap and say why.

llgo demo copy (this tree's host/): one delta against esp-emulator's
tools/bumble_classic_peer.py: the ACL is dropped before the process exits
(the DELTA in main), and the demo's acceptance runs through the spp+a2dp
combo script. The firmware side can never
observe this process vanishing (the emulator's bridge just drops the TCP and
nothing tells the emulated baseband its peer is gone), so an exit without the
disconnect leaves the ACL "up" in the firmware forever, and Bluedroid's
teardown under an open ACL aborts or hangs. A well-behaved classic client
drops its link; this peer has to be one for the probe's deinit stage to be
testable.

usage: bumble_classic_peer.py [--wait S] PROFILE [ARGS]
"""
import asyncio
import logging
import os
import sys
import types

try:
    from bumble import a2dp as _probe  # noqa: F401
except ImportError:
    sys.exit("ERROR: bumble not installed. Run: pip3 install bumble")
from bumble import a2dp, avdtp, core, hci, hfp, hid, l2cap, rfcomm
from bumble.controller import Controller
from bumble.device import Device
from bumble.host import Host
from bumble.link import LocalLink
from bumble.transport import open_transport

logging.basicConfig(level=logging.WARNING)
if os.environ.get("BUMBLE_DEBUG"):
    for _mod in ("avdtp", "l2cap", "rfcomm", "hfp", "controller"):
        logging.getLogger(f"bumble.{_mod}").setLevel(logging.DEBUG)

PROFILES = ("a2dp", "spp", "spp+a2dp", "l2cap", "hid", "hfp")
EMU_ADDR = "00:11:22:33:44:55"


def sbc_caps():
    info = a2dp.SbcMediaCodecInformation(
        sampling_frequency=a2dp.SbcMediaCodecInformation.SamplingFrequency.SF_44100,
        channel_mode=a2dp.SbcMediaCodecInformation.ChannelMode.JOINT_STEREO,
        block_length=a2dp.SbcMediaCodecInformation.BlockLength.BL_16,
        subbands=a2dp.SbcMediaCodecInformation.Subbands.S_8,
        allocation_method=a2dp.SbcMediaCodecInformation.AllocationMethod.LOUDNESS,
        minimum_bitpool_value=2,
        maximum_bitpool_value=53,
    )
    return avdtp.MediaCodecCapabilities(
        media_type=avdtp.AVDTP_AUDIO_MEDIA_TYPE,
        media_codec_type=a2dp.A2DP_SBC_CODEC_TYPE,
        media_codec_information=info,
    )


def _ack_ok(self, command):
    params = hci.HCI_GenericReturnParameters(data=bytes([0]))
    if isinstance(command, hci.HCI_SyncCommand):
        return params
    self.send_hci_packet(hci.HCI_Command_Complete_Event(
        num_hci_command_packets=1, command_opcode=command.op_code,
        return_parameters=params))
    return None


# Bluedroid writes link-policy (OGF 2), baseband (OGF 3) and ESP vendor
# (OGF 0x3F) settings Bumble has no handler for. Its fallback errors, and for
# an unparsed command sends no reply at all, so the host times out after 8 s.
# They only configure the controller: ack them.
_bumble_on_hci_command = Controller.on_hci_command


def _on_hci_command(self, command):
    if (command.op_code >> 10) in (0x02, 0x03, 0x3F):
        return _ack_ok(self, command)
    return _bumble_on_hci_command(self, command)


Controller.on_hci_command = _on_hci_command


def _on_read_clock_offset(self, command):
    self._send_hci_command_status(hci.HCI_ErrorCode.SUCCESS, command.op_code)
    self.send_hci_packet(hci.HCI_Read_Clock_Offset_Complete_Event(
        status=hci.HCI_ErrorCode.SUCCESS,
        connection_handle=command.connection_handle, clock_offset=0))


def _on_change_packet_type(self, command):
    self._send_hci_command_status(hci.HCI_ErrorCode.SUCCESS, command.op_code)
    self.send_hci_packet(hci.HCI_Connection_Packet_Type_Changed_Event(
        status=hci.HCI_ErrorCode.SUCCESS,
        connection_handle=command.connection_handle,
        packet_type=command.packet_type))


# Bumble's LMP has no BR/EDR authentication, but Bluedroid's A2DP sink
# demands it before AVDTP. Fake a local "just works" SSP exchange on the
# requesting controller: the host sees a normal pairing, the virtual link
# itself carries no security. Events follow the Command Complete (call_soon).
def _later(self, *events):
    loop = asyncio.get_running_loop()
    for e in events:
        loop.call_soon(self.send_hci_packet, e)


def _ok_addr(command):
    return hci.HCI_StatusAndAddressReturnParameters(
        status=hci.HCI_ErrorCode.SUCCESS, bd_addr=command.bd_addr)


def _conn_by_addr(self, addr):
    return self.classic_connections.get(addr)


def _on_auth_requested(self, command):
    conn = self.find_connection_by_handle(command.connection_handle)
    self._send_hci_command_status(hci.HCI_ErrorCode.SUCCESS, command.op_code)
    _later(self, hci.HCI_Link_Key_Request_Event(bd_addr=conn.peer_address))


def _on_link_key_reply(self, command):
    conn = _conn_by_addr(self, command.bd_addr)
    _later(self, hci.HCI_Authentication_Complete_Event(
        status=hci.HCI_ErrorCode.SUCCESS, connection_handle=conn.handle))
    return _ok_addr(command)


def _on_link_key_negative_reply(self, command):
    _later(self, hci.HCI_IO_Capability_Request_Event(bd_addr=command.bd_addr))
    return _ok_addr(command)


def _on_io_cap_reply(self, command):
    _later(self,
           hci.HCI_IO_Capability_Response_Event(
               bd_addr=command.bd_addr, io_capability=3,  # NoInputNoOutput
               oob_data_present=0, authentication_requirements=0),
           hci.HCI_User_Confirmation_Request_Event(bd_addr=command.bd_addr, numeric_value=0))
    return _ok_addr(command)


def _on_user_confirm_reply(self, command):
    conn = _conn_by_addr(self, command.bd_addr)
    _later(self,
           hci.HCI_Simple_Pairing_Complete_Event(
               status=hci.HCI_ErrorCode.SUCCESS, bd_addr=command.bd_addr),
           hci.HCI_Link_Key_Notification_Event(
               bd_addr=command.bd_addr, link_key=bytes(range(16)),
               key_type=7),  # unauthenticated, P-256
           hci.HCI_Authentication_Complete_Event(
               status=hci.HCI_ErrorCode.SUCCESS, connection_handle=conn.handle))
    return _ok_addr(command)


def _on_set_encryption(self, command):
    self._send_hci_command_status(hci.HCI_ErrorCode.SUCCESS, command.op_code)
    _later(self, hci.HCI_Encryption_Change_Event(
        status=hci.HCI_ErrorCode.SUCCESS, connection_handle=command.connection_handle,
        encryption_enabled=command.encryption_enable))


def _on_read_key_size(self, command):
    return hci.HCI_Read_Encryption_Key_Size_ReturnParameters(
        status=hci.HCI_ErrorCode.SUCCESS, connection_handle=command.connection_handle,
        key_size=16)


for _name, _fn in (
    ("on_hci_authentication_requested_command", _on_auth_requested),
    ("on_hci_link_key_request_reply_command", _on_link_key_reply),
    ("on_hci_link_key_request_negative_reply_command", _on_link_key_negative_reply),
    ("on_hci_io_capability_request_reply_command", _on_io_cap_reply),
    ("on_hci_user_confirmation_request_reply_command", _on_user_confirm_reply),
    ("on_hci_set_connection_encryption_command", _on_set_encryption),
    ("on_hci_read_encryption_key_size_command", _on_read_key_size),
):
    if not hasattr(Controller, _name):
        setattr(Controller, _name, _fn)


# Bluedroid sets up eSCO with the v1 Setup_Synchronous_Connection, which
# Bumble leaves unparsed; its enhanced-variant handler reads nothing but the
# connection handle and the opcode it echoes in Command Status.
def _on_setup_sync_conn(self, command):
    params = getattr(command, "parameters", b"")
    cmd = types.SimpleNamespace(op_code=command.op_code,
                                connection_handle=int.from_bytes(params[:2], "little"))
    return self.on_hci_enhanced_setup_synchronous_connection_command(cmd)


if not hasattr(Controller, "on_hci_setup_synchronous_connection_command"):
    Controller.on_hci_setup_synchronous_connection_command = _on_setup_sync_conn
if not hasattr(Controller, "on_hci_change_connection_packet_type_command"):
    Controller.on_hci_change_connection_packet_type_command = _on_change_packet_type
if not hasattr(Controller, "on_hci_read_clock_offset_command"):
    Controller.on_hci_read_clock_offset_command = _on_read_clock_offset


async def run_a2dp(dev, conn, sbc_path):
    version = await avdtp.find_avdtp_service_with_connection(conn)
    protocol = await avdtp.Protocol.connect(conn, version)
    print(f"[+] AVDTP {version} signalling open", flush=True)
    with open(sbc_path, "rb") as f:
        async def read(n):
            return f.read(n)
        pump = avdtp.MediaPacketPump(
            a2dp.SbcPacketSource(read, protocol.l2cap_channel.peer_mtu).packets)
        source = protocol.add_source(sbc_caps(), pump)
        await protocol.discover_remote_endpoints()
        sink = protocol.find_remote_sink_by_codec(
            avdtp.AVDTP_AUDIO_MEDIA_TYPE, a2dp.A2DP_SBC_CODEC_TYPE)
        if sink is None:
            raise RuntimeError("no SBC sink endpoint")
        stream = await protocol.create_stream(source, sink)
        await stream.start()
        print("[+] streaming", flush=True)
        await pump.wait_for_completion()
        await stream.stop()
        print("[+] stream stopped", flush=True)


async def run_spp(dev, conn):
    channel = await rfcomm.find_rfcomm_channel_with_uuid(
        conn, core.BT_SERIAL_PORT_SERVICE)
    if channel is None:
        raise RuntimeError("no SPP record in SDP")
    print(f"[+] SDP: SPP on RFCOMM channel {channel}", flush=True)
    # Bluedroid offers ERTM and Bumble aborts on a mode mismatch instead of
    # falling back to basic, so open the RFCOMM L2CAP channel in ERTM.
    chan = await conn.create_l2cap_channel(spec=l2cap.ClassicChannelSpec(
        psm=rfcomm.RFCOMM_PSM, mtu=rfcomm.RFCOMM_DEFAULT_L2CAP_MTU,
        mode=l2cap.TransmissionMode.ENHANCED_RETRANSMISSION, fcs_enabled=True))
    mux = rfcomm.Multiplexer(chan, rfcomm.Multiplexer.Role.INITIATOR)
    await mux.connect()
    dlc = await mux.open_dlc(channel)
    print("[+] RFCOMM DLC open", flush=True)
    # bt_spp_acceptor only reports (as a throughput line) after >= 3 s of data.
    for i in range(25):
        dlc.write(bytes([0x30 + i % 10]) * 512)
        await asyncio.sleep(0.2)
    await dlc.disconnect()
    print("[+] RFCOMM DLC closed", flush=True)


async def run_l2cap(dev, conn, psm):
    chan = await conn.create_l2cap_channel(spec=l2cap.ClassicChannelSpec(
        psm=psm, mode=l2cap.TransmissionMode.ENHANCED_RETRANSMISSION, fcs_enabled=True))
    print(f"[+] L2CAP channel open on PSM 0x{psm:04X}", flush=True)
    for i in range(5):
        chan.write(f"hello from bumble {i}".encode())
        await asyncio.sleep(0.5)
    await chan.disconnect()
    print("[+] L2CAP channel closed", flush=True)


async def run_hid(hid_host):
    reports = []
    hid_host.on(hid_host.EVENT_INTERRUPT_DATA, reports.append)
    await hid_host.connect_control_channel()
    await hid_host.connect_interrupt_channel()
    print("[+] HID control + interrupt channels open", flush=True)
    await asyncio.sleep(5)
    print(f"[+] {len(reports)} input reports, first: "
          f"{reports[0].hex() if reports else None}", flush=True)
    await hid_host.disconnect_interrupt_channel()
    await hid_host.disconnect_control_channel()
    if not reports:
        raise RuntimeError("no HID input reports")


HF_CONFIG = hfp.HfConfiguration(
    supported_hf_features=[hfp.HfFeature.CODEC_NEGOTIATION,
                           hfp.HfFeature.ESCO_S4_SETTINGS_SUPPORTED],
    supported_hf_indicators=[],
    supported_audio_codecs=[hfp.AudioCodec.CVSD])


async def run_hfp(dev, conn):
    found = await hfp.find_ag_sdp_record(conn)
    if found is None:
        raise RuntimeError("no HFP AG record in SDP")
    channel, version, _ = found
    print(f"[+] SDP: HFP AG {version} on RFCOMM channel {channel}", flush=True)
    sco = asyncio.get_running_loop().create_future()

    def on_sco_request(connection, link_type):
        params = hfp.ESCO_PARAMETERS[hfp.DefaultCodecParameters.ESCO_CVSD_S1]
        asyncio.ensure_future(dev.send_command(
            hci.HCI_Enhanced_Accept_Synchronous_Connection_Request_Command(
                bd_addr=connection.peer_address, **params.asdict())))

    dev.on(dev.EVENT_SCO_REQUEST, on_sco_request)
    dev.on(dev.EVENT_SCO_CONNECTION, lambda link: sco.done() or sco.set_result(link))
    # Unlike SPP, Bluedroid's HFP port leaves ERTM off and answers an ERTM
    # request with "use basic"; Bumble then renegotiates but keeps ERTM
    # framing, so ask for basic mode up front.
    mux = await rfcomm.Client(conn).start()
    dlc = await mux.open_dlc(channel)
    hf = hfp.HfProtocol(dlc, HF_CONFIG)
    # run() brings up the SLC, then serves unsolicited results such as the
    # AG's +BCS codec proposal, which must be answered before eSCO setup.
    slc = asyncio.ensure_future(hf.run())
    while not hf.supported_ag_features and not slc.done():
        await asyncio.sleep(0.1)
    print("[+] HFP service level connection up", flush=True)
    link = await asyncio.wait_for(sco, 30)
    print(f"[+] eSCO link up: {link}", flush=True)


async def main():
    args = sys.argv[1:]
    wait_s = 10.0
    if args[:1] == ["--wait"]:
        wait_s = float(args[1]); args = args[2:]
    if not args or args[0] not in PROFILES:
        sys.exit(__doc__)
    profile, rest = args[0], args[1:]

    link = LocalLink()
    transport = await open_transport("tcp-server:_:9544")
    emu = Controller("emu", host_source=transport.source, host_sink=transport.sink,
                     link=link, public_address=EMU_ADDR)
    # Bluedroid keeps feature pages 0..2 only and asserts on a max page of 3,
    # Bumble's default; real BTDM controllers report 2.
    emu.lmp_features_max_page_number = 2
    # Without eSCO in the features Bluedroid falls back to the legacy
    # Add_SCO_Connection, which Bumble's controller does not implement.
    emu.lmp_features |= (hci.LmpFeatureMask.SCO_LINK | hci.LmpFeatureMask.HV3_PACKETS
                         | hci.LmpFeatureMask.EXTENDED_SCO_LINK_EV3_PACKETS)
    print("[*] virtual controller on :9544, waiting for esp-emu", flush=True)

    peer_ctrl = Controller("peer", link=link, public_address="AA:BB:CC:DD:EE:01")
    host = Host()
    host.controller = peer_ctrl
    peer_ctrl.host = host
    dev = Device(name="Bumble Peer", host=host)
    dev.classic_enabled = True
    if profile == "a2dp":
        dev.sdp_service_records = {
            0x00010001: a2dp.make_audio_source_service_sdp_records(0x00010001)}
    elif profile == "hfp":
        dev.sdp_service_records = {
            0x00010001: hfp.make_hf_sdp_records(0x00010001, 1, HF_CONFIG)}
    # hid.Host must exist before the ACL comes up: it binds on connection.
    hid_host = hid.Host(dev) if profile == "hid" else None
    await dev.power_on()

    await asyncio.sleep(wait_s)
    print(f"[*] connecting to {EMU_ADDR} over BR/EDR", flush=True)
    # Pairing, when the firmware asks for it, is the faked SSP above.
    conn = await dev.connect(EMU_ADDR, transport=hci.PhysicalTransport.BR_EDR)
    print("[+] ACL connected", flush=True)
    if profile == "a2dp":
        await run_a2dp(dev, conn, rest[0])
    elif profile == "spp":
        await run_spp(dev, conn)
    elif profile == "spp+a2dp":
        await run_spp(dev, conn)
        await conn.disconnect()
        print("[+] ACL disconnected (spp round)", flush=True)
        await asyncio.sleep(2)
        conn = await dev.connect(EMU_ADDR, transport=hci.PhysicalTransport.BR_EDR)
        print("[+] ACL connected (a2dp round)", flush=True)
        await run_a2dp(dev, conn, rest[0])
    elif profile == "l2cap":
        await run_l2cap(dev, conn, int(rest[0], 0) if rest else 0x1001)
    elif profile == "hfp":
        await run_hfp(dev, conn)
    elif profile == "hid":
        await run_hid(hid_host)
    # DELTA (llgo demo copy): a well-behaved classic client drops the ACL
    # before leaving: the firmware side never sees this process vanish, and
    # Bluedroid cannot tear down under an open ACL (its disable path aborts
    # or hangs on the controller's answer to a link that outlived the host).
    await conn.disconnect()
    print("[+] ACL disconnected", flush=True)
    print(f"BUMBLE_{profile.upper()}_PASS", flush=True)
    await asyncio.sleep(2)


asyncio.run(main())
