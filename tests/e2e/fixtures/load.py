"""
Generates e2e fixture log data with current timestamps and pushes to Loki.
Timestamps are always relative to now so Loki never rejects them as too old.
seed=42, ~2-hour window ending at now.
"""
import json
import urllib.request
import time
from datetime import datetime, timezone, timedelta

LOKI_URL = "http://loki:3100/loki/api/v1/push"

now = datetime.now(timezone.utc)
base = now - timedelta(hours=2)

levels = ["info", "warn", "error"]
apps = ["frontend", "backend"]

streams = []
for app in apps:
    for level in levels:
        values = []
        for i in range(12):
            ts = base + timedelta(minutes=i * 10)
            ns = str(int(ts.timestamp() * 1e9))
            if level == "error":
                msg = f"[ERROR] {app}: request failed code=500 latency={i * 50}ms"
            elif level == "warn":
                msg = f"[WARN] {app}: high latency latency={i * 100}ms threshold=500ms"
            else:
                msg = f"[INFO] {app}: processed request code=200 latency={i * 5}ms"
            values.append([ns, msg])
        streams.append({"stream": {"app": app, "level": level, "job": "e2e-test"}, "values": values})

payload = json.dumps({"streams": streams}).encode()

for attempt in range(5):
    try:
        req = urllib.request.Request(
            LOKI_URL,
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req) as resp:
            print(f"Pushed {sum(len(s['values']) for s in streams)} log entries to Loki")
            break
    except Exception as e:
        print(f"Attempt {attempt + 1} failed: {e}")
        if attempt < 4:
            time.sleep(3)
        else:
            raise
