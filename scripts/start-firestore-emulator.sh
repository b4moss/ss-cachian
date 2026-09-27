#!/usr/bin/env bash
# Start Cloud Firestore emulator on 127.0.0.1:8080 (Java JAR).
set -euo pipefail
PORT="${FIRESTORE_EMULATOR_PORT:-8080}"
HOST="${FIRESTORE_EMULATOR_LISTEN:-127.0.0.1}"
JAR="${FIRESTORE_EMULATOR_JAR:-/tmp/cloud-firestore-emulator.jar}"
URL="${FIRESTORE_EMULATOR_JAR_URL:-https://storage.googleapis.com/firebase-preview-drop/emulator/cloud-firestore-emulator-v1.19.8.jar}"

if [[ ! -f "$JAR" ]]; then
  echo "Downloading Firestore emulator JAR..."
  curl -fsSL "$URL" -o "$JAR"
fi

if nc -z "$HOST" "$PORT" 2>/dev/null; then
  echo "Port $HOST:$PORT already open; assuming emulator is running."
  exit 0
fi

echo "Starting Firestore emulator on $HOST:$PORT"
java -jar "$JAR" --host="$HOST" --port="$PORT" > /tmp/firestore-emu.log 2>&1 &
echo $! > /tmp/firestore-emu.pid
for i in $(seq 1 60); do
  if grep -q 'Dev App Server is now running' /tmp/firestore-emu.log 2>/dev/null; then
    echo "Firestore emulator ready (FIRESTORE_EMULATOR_HOST=$HOST:$PORT)"
    exit 0
  fi
  sleep 0.5
done
echo "Emulator failed to start; log:" >&2
tail -50 /tmp/firestore-emu.log >&2
exit 1
