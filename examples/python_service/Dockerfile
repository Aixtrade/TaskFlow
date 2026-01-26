FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY examples/python_service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy proto files and generate Python code
COPY api/proto /proto
RUN python -m grpc_tools.protoc \
    -I/proto \
    --python_out=. \
    --grpc_python_out=. \
    /proto/grpc_task/v1/task.proto

# Fix Python imports in generated files
RUN sed -i 's/^import grpc_task/from . import grpc_task/' grpc_task/v1/task_pb2_grpc.py && \
    sed -i 's/from grpc_task\.v1 import task_pb2/from . import task_pb2/' grpc_task/v1/task_pb2_grpc.py

# Copy application code
COPY examples/python_service/server.py .

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["python", "server.py", "--port", "50051"]
