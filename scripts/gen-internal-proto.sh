#!/usr/bin/env bash
set -euo pipefail

# This helper script regenerates Go protobuf/gRPC source files for the
# internal server protocol. It expects that the generated files live in the same
# directory as the .proto definition so that imports stay relative.

PROTO_DIR="internal/grpc/internalpb"
PROTO_FILE="$PROTO_DIR/internal.proto"

if [ ! -f "$PROTO_FILE" ]; then
    echo "proto file not found: $PROTO_FILE" >&2
    exit 1
fi


protoc \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    "$PROTO_FILE"

echo "generated protobuf sources in $PROTO_DIR"