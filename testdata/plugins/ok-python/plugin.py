#!/usr/bin/env python3
import json
import sys

def emit(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()

emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "ok-python",
    "capabilities": {"ops": ["ping"]},
    "declares": {"side_effects": "none", "idempotent": True},
})

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")

    if op == "ping":
        emit({"type": "response", "request_id": rid, "ok": True, "output": {"pong": True}})
    else:
        emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_UNSUPPORTED", "message": "unsupported op"}})
