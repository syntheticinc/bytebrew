#!/usr/bin/env python3
"""
Diagnostic WS transport test through Bridge.

Simulates a CLI (server) and a Device (mobile) connecting through the bridge,
then exchanges ping-pong messages every 5 seconds for 3 minutes to measure
round-trip time and detect connection drops.

Usage:
    python3 test_ws_transport.py                    # on the bridge server
    python3 test_ws_transport.py --bridge-url wss://bridge.bytebrew.ai  # custom URL
    python3 test_ws_transport.py --duration 60      # run for 60 seconds
    python3 test_ws_transport.py --interval 3       # ping every 3 seconds
"""

import argparse
import asyncio
import json
import os
import sys
import time
from dataclasses import dataclass, field

try:
    import websockets
except ImportError:
    print("ERROR: websockets not installed. Run: pip3 install websockets")
    sys.exit(1)


@dataclass
class Stats:
    sent: int = 0
    received: int = 0
    rtts: list = field(default_factory=list)
    bridge_pongs: int = 0
    drops: int = 0
    errors: list = field(default_factory=list)


def now_ms() -> int:
    return int(time.time() * 1000)


def elapsed_str(start: float) -> str:
    secs = int(time.time() - start)
    m, s = divmod(secs, 60)
    return f"{m:02d}:{s:02d}"


def log(start: float, msg: str):
    print(f"[{elapsed_str(start)}] {msg}", flush=True)


def read_auth_token() -> str:
    """Read BRIDGE_AUTH_TOKEN from /etc/bytebrew/bridge.env if available."""
    env_path = "/etc/bytebrew/bridge.env"
    if not os.path.exists(env_path):
        return ""
    try:
        with open(env_path) as f:
            for line in f:
                line = line.strip()
                if line.startswith("BRIDGE_AUTH_TOKEN="):
                    val = line.split("=", 1)[1].strip()
                    # Remove surrounding quotes if present
                    if len(val) >= 2 and val[0] == val[-1] and val[0] in ('"', "'"):
                        val = val[1:-1]
                    return val
    except OSError:
        pass
    return ""


async def run_cli(bridge_url: str, server_id: str, device_id: str,
                  auth_token: str, start: float, stats: Stats,
                  stop_event: asyncio.Event):
    """CLI side: register, then echo pings back as pongs."""
    url = f"{bridge_url}/register"
    log(start, f"CLI connecting to {url}")

    try:
        async with websockets.connect(url, ping_interval=None, ping_timeout=None) as ws:
            # Send register message
            register_msg = {
                "type": "register",
                "server_id": server_id,
                "server_name": "transport-test",
                "auth_token": auth_token,
            }
            await ws.send(json.dumps(register_msg))

            # Wait for registered confirmation
            raw = await asyncio.wait_for(ws.recv(), timeout=10)
            msg = json.loads(raw)
            if msg.get("type") != "registered":
                log(start, f"CLI ERROR: expected 'registered', got: {msg}")
                return
            log(start, "CLI registered successfully")

            # Read loop: echo pings back as pongs, log other messages
            while not stop_event.is_set():
                try:
                    raw = await asyncio.wait_for(ws.recv(), timeout=1.0)
                except asyncio.TimeoutError:
                    continue
                except websockets.ConnectionClosed as e:
                    log(start, f"CLI connection closed: {e}")
                    stats.drops += 1
                    stats.errors.append(f"CLI closed: {e}")
                    return

                msg = json.loads(raw)

                if msg.get("type") == "device_connected":
                    log(start, f"CLI got device_connected: {msg.get('device_id')}")
                    continue

                if msg.get("type") == "device_disconnected":
                    log(start, f"CLI got device_disconnected: {msg.get('device_id')}")
                    stats.drops += 1
                    continue

                if msg.get("type") == "data":
                    payload = msg.get("payload", {})
                    if isinstance(payload, str):
                        payload = json.loads(payload)

                    if payload.get("type") == "ping":
                        # Echo back as pong
                        pong_payload = {
                            "type": "pong",
                            "seq": payload.get("seq"),
                            "ts": payload.get("ts"),
                            "server_ts": now_ms(),
                        }
                        reply = {
                            "type": "data",
                            "device_id": msg.get("device_id", device_id),
                            "payload": pong_payload,
                        }
                        await ws.send(json.dumps(reply))
                    else:
                        log(start, f"CLI got unexpected payload: {payload}")
                else:
                    log(start, f"CLI got unexpected message: {msg.get('type')}")

    except Exception as e:
        log(start, f"CLI ERROR: {e}")
        stats.errors.append(f"CLI: {e}")
        stats.drops += 1


async def run_device(bridge_url: str, server_id: str, device_id: str,
                     start: float, stats: Stats, stop_event: asyncio.Event,
                     interval: float, cli_ready: asyncio.Event):
    """Device side: connect, send pings, measure RTT from pongs."""
    # Wait a moment for CLI to register
    await asyncio.sleep(2)

    url = f"{bridge_url}/connect?server_id={server_id}&device_id={device_id}"
    log(start, f"DEVICE connecting to {url}")

    try:
        async with websockets.connect(url, ping_interval=None, ping_timeout=None) as ws:
            log(start, "DEVICE connected")
            cli_ready.set()

            seq = 0
            pending: dict[int, int] = {}  # seq -> ts

            async def sender():
                nonlocal seq
                while not stop_event.is_set():
                    seq += 1
                    ts = now_ms()
                    pending[seq] = ts
                    stats.sent += 1

                    ping_payload = {"type": "ping", "seq": seq, "ts": ts}
                    msg = {"type": "data", "payload": ping_payload}
                    try:
                        await ws.send(json.dumps(msg))
                    except websockets.ConnectionClosed as e:
                        log(start, f"DEVICE send failed: {e}")
                        stats.drops += 1
                        stats.errors.append(f"Device send: {e}")
                        return

                    await asyncio.sleep(interval)

            async def receiver():
                while not stop_event.is_set():
                    try:
                        raw = await asyncio.wait_for(ws.recv(), timeout=1.0)
                    except asyncio.TimeoutError:
                        continue
                    except websockets.ConnectionClosed as e:
                        log(start, f"DEVICE recv closed: {e}")
                        stats.drops += 1
                        stats.errors.append(f"Device recv: {e}")
                        return

                    msg = json.loads(raw)

                    # Bridge keepalive pong (application-level)
                    if msg.get("type") == "pong":
                        stats.bridge_pongs += 1
                        log(start, "BRIDGE_PONG received")
                        continue

                    if msg.get("type") == "data":
                        payload = msg.get("payload", {})
                        if isinstance(payload, str):
                            payload = json.loads(payload)

                        if payload.get("type") == "pong":
                            s = payload.get("seq")
                            original_ts = payload.get("ts")
                            if original_ts is not None:
                                rtt = now_ms() - original_ts
                                stats.received += 1
                                stats.rtts.append(rtt)
                                pending.pop(s, None)
                                log(start, f"PING seq={s} -> RTT={rtt}ms")
                            else:
                                log(start, f"DEVICE got pong without ts: {payload}")
                        else:
                            log(start, f"DEVICE got unexpected payload: {payload}")
                    else:
                        log(start, f"DEVICE got unexpected: {msg.get('type')}")

            # Check for timed-out pings periodically
            async def timeout_checker():
                while not stop_event.is_set():
                    await asyncio.sleep(interval * 3)
                    now = now_ms()
                    timed_out = [
                        s for s, ts in pending.items()
                        if now - ts > interval * 3 * 1000
                    ]
                    for s in timed_out:
                        ts = pending.pop(s)
                        log(start, f"TIMEOUT seq={s} (no pong after {now - ts}ms)")

            await asyncio.gather(
                sender(),
                receiver(),
                timeout_checker(),
            )

    except Exception as e:
        log(start, f"DEVICE ERROR: {e}")
        stats.errors.append(f"Device: {e}")
        stats.drops += 1


async def main():
    parser = argparse.ArgumentParser(description="WS transport diagnostic test")
    parser.add_argument("--bridge-url", default="wss://bridge.bytebrew.ai",
                        help="Bridge WebSocket URL (default: wss://bridge.bytebrew.ai)")
    parser.add_argument("--duration", type=int, default=180,
                        help="Test duration in seconds (default: 180)")
    parser.add_argument("--interval", type=float, default=5.0,
                        help="Ping interval in seconds (default: 5)")
    parser.add_argument("--auth-token", default=None,
                        help="Bridge auth token (default: read from /etc/bytebrew/bridge.env)")
    args = parser.parse_args()

    auth_token = args.auth_token if args.auth_token else read_auth_token()
    server_id = f"test-transport-{int(time.time())}"
    device_id = "test-device-1"

    print(f"=== WS Transport Diagnostic ===")
    print(f"Bridge:   {args.bridge_url}")
    print(f"Server:   {server_id}")
    print(f"Device:   {device_id}")
    print(f"Duration: {args.duration}s")
    print(f"Interval: {args.interval}s")
    print(f"Auth:     {'yes' if auth_token else 'no'}")
    print(f"===============================")
    print()

    stats = Stats()
    stop_event = asyncio.Event()
    cli_ready = asyncio.Event()
    start = time.time()

    # Stop after duration
    async def timer():
        await asyncio.sleep(args.duration)
        log(start, "Duration reached, stopping...")
        stop_event.set()

    try:
        await asyncio.gather(
            timer(),
            run_cli(args.bridge_url, server_id, device_id,
                    auth_token, start, stats, stop_event),
            run_device(args.bridge_url, server_id, device_id,
                       start, stats, stop_event, args.interval, cli_ready),
        )
    except KeyboardInterrupt:
        log(start, "Interrupted by user")
        stop_event.set()

    # Summary
    print()
    print(f"=== SUMMARY ===")
    print(f"Sent: {stats.sent}, Received: {stats.received}, "
          f"Lost: {stats.sent - stats.received}")

    if stats.rtts:
        avg_rtt = sum(stats.rtts) / len(stats.rtts)
        max_rtt = max(stats.rtts)
        min_rtt = min(stats.rtts)
        print(f"Avg RTT: {avg_rtt:.0f}ms, Min RTT: {min_rtt}ms, Max RTT: {max_rtt}ms")
    else:
        print("No RTT data (no pongs received)")

    print(f"Bridge pongs received: {stats.bridge_pongs}")
    print(f"Connection drops: {stats.drops}")

    if stats.errors:
        print(f"Errors:")
        for e in stats.errors:
            print(f"  - {e}")


if __name__ == "__main__":
    asyncio.run(main())
