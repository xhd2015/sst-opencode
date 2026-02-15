#!/bin/bash

trap 'kill $API_PID $APP_PID 2>/dev/null; exit' INT TERM

pkill -f "bun run.*serve.*5096" 2>/dev/null || true
pkill -f "vite.*4444" 2>/dev/null || true
sleep 1

cd /root/opencode/packages/opencode
OPENCODE_WEB_PROXY_URL=http://localhost:4444 bun run --conditions=browser ./src/index.ts serve --hostname 0.0.0.0 --port 5096 &
API_PID=$!
echo "API server PID: $API_PID"

cd /root/opencode/packages/app
VITE_HOST=0.0.0.0 bun run dev -- --port 4444 --host &
APP_PID=$!
echo "Web app PID: $APP_PID"

wait
