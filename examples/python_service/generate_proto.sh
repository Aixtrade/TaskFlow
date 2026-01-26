#!/bin/bash

# Generate Python protobuf files from proto definitions
# Run this from the examples/python_service directory

set -e

PROTO_DIR="../../api/proto"
OUT_DIR="."

echo "Generating Python protobuf code..."

python3 -m grpc_tools.protoc \
    -I${PROTO_DIR} \
    --python_out=${OUT_DIR} \
    --grpc_python_out=${OUT_DIR} \
    ${PROTO_DIR}/grpc_task/v1/task.proto

# Fix imports in generated files (Python 3 style)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' 's/^import grpc_task/from . import grpc_task/' grpc_task/v1/task_pb2_grpc.py
    sed -i '' 's/from grpc_task\.v1 import task_pb2/from . import task_pb2/' grpc_task/v1/task_pb2_grpc.py
else
    # Linux
    sed -i 's/^import grpc_task/from . import grpc_task/' grpc_task/v1/task_pb2_grpc.py
    sed -i 's/from grpc_task\.v1 import task_pb2/from . import task_pb2/' grpc_task/v1/task_pb2_grpc.py
fi

echo "Proto generation complete!"
echo "Generated files:"
ls -la grpc_task/v1/*.py
