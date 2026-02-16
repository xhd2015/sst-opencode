#!/bin/bash

set -e

kool kill-port 4731 2>/dev/null || true
kool kill-port 4444 2>/dev/null || true
kool kill-port 6096 2>/dev/null || true
sleep 1

cd /root/opencode/script/fe-proxy
go run main.go &
PROXY_PID=$!
echo "FE proxy PID: $PROXY_PID"
sleep 1

cd /root/opencode/packages/app
VITE_HOST=0.0.0.0 bun run dev -- --port 4444 --host &
VITE_PID=$!
echo "Vite PID: $VITE_PID"
sleep 2

cd /root/opencode/packages/opencode
unset OPENCODE_SERVER_PASSWORD
OPENCODE_WEB_PROXY_URL=http://localhost:4731 bun run --conditions=browser ./src/index.ts serve --hostname 0.0.0.0 --port 6096 &
API_PID=$!
echo "API server PID: $API_PID"
echo "Server running at http://localhost:6096"
echo "Vite at http://localhost:4444"
echo "FE proxy at http://localhost:4731"

cleanup() {
    echo "Cleaning up..."
    kill $PROXY_PID $VITE_PID $API_PID 2>/dev/null || true
    kool kill-port 4731 2>/dev/null || true
    kool kill-port 4444 2>/dev/null || true
    kool kill-port 6096 2>/dev/null || true
}

trap cleanup EXIT INT TERM

wait
