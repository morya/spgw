#!/usr/bin/env bash
set -e

./spclient \
  --host 127.0.0.1:7891 \
  --protocol cmpp3.0 \
  --spid 200261 \
  --password 888888 \
  --src-id 10659 \
  --rate 5 \
  --total 100 \
  --result-file ./result.txt \
  --log-level info \
  --drain-timeout 30s &

PID=$!

sleep 3
kill -TERM "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true

echo "===== result.txt ====="
cat ./result.txt
