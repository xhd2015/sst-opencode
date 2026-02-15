#!/bin/bash

trap 'kill $PROXY_PID $API_PID 2>/dev/null; exit' INT TERM

pkill -f "go run.*fe-proxy" 2>/dev/null || true
pkill -f "serve.*6096" 2>/dev/null || true
sleep 1

cd /root/opencode/script/fe-proxy
go run main.go &
PROXY_PID=$!
echo "FE proxy PID: $PROXY_PID"
sleep 1

cd /root/opencode/packages/opencode
unset OPENCODE_SERVER_PASSWORD
OPENCODE_WEB_PROXY_URL=http://localhost:4731 bun run --conditions=browser ./src/index.ts serve --hostname 0.0.0.0 --port 6096 &
API_PID=$!
echo "API server PID: $API_PID"
echo "Server running at http://localhost:6096"
echo "FE proxy at http://localhost:4731"

wait
