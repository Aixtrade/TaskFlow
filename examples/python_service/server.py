"""
Python Task Executor Service - gRPC Server Example

This module provides a framework for building Python task services that
integrate with TaskFlow's Go worker system via gRPC.
"""

import asyncio
import importlib
import logging
import signal
import time
from concurrent import futures
from typing import Any, AsyncIterator, Callable, Dict, Optional, cast

import grpc
from google.protobuf import struct_pb2

pb: Any = None
pb_grpc: Any = None

# These imports will be generated from the proto file
# Run: python3 -m grpc_tools.protoc -I../../api/proto --python_out=. --grpc_python_out=. ../../api/proto/grpc_task/v1/task.proto
try:
    pb = importlib.import_module("grpc_task.v1.task_pb2")
    pb_grpc = importlib.import_module("grpc_task.v1.task_pb2_grpc")
except ImportError:
    print("Warning: Proto files not generated. Run proto-gen first.")
    pb = None
    pb_grpc = None

if pb is None:
    class _ProtoFallback:
        class ExecuteTaskResponse:
            def __init__(self, **kwargs):
                pass

        class ErrorDetail:
            def __init__(self, **kwargs):
                pass

        class TaskResult:
            def __init__(self, **kwargs):
                pass

        class Progress:
            def __init__(self, **kwargs):
                pass

        class CancelTaskResponse:
            def __init__(self, **kwargs):
                pass

        class HealthCheckResponse:
            def __init__(self, **kwargs):
                pass

        TASK_STATUS_CANCELLED = 0
        TASK_STATUS_COMPLETED = 0
        HEALTH_STATUS_HEALTHY = 0

    pb = _ProtoFallback()

assert pb is not None

pb = cast(Any, pb)
pb_grpc = cast(Any, pb_grpc)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


# Type alias for handler function
HandlerFunc = Callable[[Any, grpc.aio.ServicerContext, dict], AsyncIterator[Any]]


class TaskExecutorServicer:
    """
    gRPC servicer implementation for Python task execution.

    Supports dynamic handler registration and progress reporting.
    """

    def __init__(self):
        self.handlers: Dict[str, HandlerFunc] = {}
        self.active_tasks: Dict[str, dict] = {}
        self._shutdown = False

    def register_handler(self, method: str, handler: HandlerFunc):
        """
        Register a handler function for a specific method.

        Args:
            method: The method name (corresponds to payload.method)
            handler: Async generator function that processes the task
        """
        self.handlers[method] = handler
        logger.info(f"Registered handler for method: {method}")

    async def ExecuteTask(
        self,
        request: Any,
        context: grpc.aio.ServicerContext
    ) -> AsyncIterator[Any]:
        """
        Execute a task and stream progress/results back to the client.
        """
        task_id = request.task_id
        method = request.task_type  # task_type contains the method name

        logger.info(f"Task {task_id}: executing method '{method}'")

        # Initialize task state
        self.active_tasks[task_id] = {
            "cancelled": False,
            "start_time": time.time()
        }

        try:
            # Find handler
            handler = self.handlers.get(method)
            if not handler:
                logger.error(f"Task {task_id}: unknown method '{method}'")
                yield pb.ExecuteTaskResponse(
                    error=pb.ErrorDetail(
                        code="UNKNOWN_METHOD",
                        message=f"Unknown method: {method}",
                        retryable=False
                    )
                )
                return

            # Execute handler
            async for response in handler(request, context, self.active_tasks[task_id]):
                if context.cancelled() or self.active_tasks[task_id].get("cancelled"):
                    logger.warning(f"Task {task_id}: cancelled")
                    yield pb.ExecuteTaskResponse(
                        result=pb.TaskResult(
                            task_id=task_id,
                            status=pb.TASK_STATUS_CANCELLED
                        )
                    )
                    return
                yield response

            # Send completion result
            duration_ms = int((time.time() - self.active_tasks[task_id]["start_time"]) * 1000)
            yield pb.ExecuteTaskResponse(
                result=pb.TaskResult(
                    task_id=task_id,
                    status=pb.TASK_STATUS_COMPLETED,
                    duration_ms=duration_ms
                )
            )
            logger.info(f"Task {task_id}: completed in {duration_ms}ms")

        except asyncio.CancelledError:
            logger.warning(f"Task {task_id}: cancelled")
            yield pb.ExecuteTaskResponse(
                result=pb.TaskResult(
                    task_id=task_id,
                    status=pb.TASK_STATUS_CANCELLED
                )
            )
        except Exception as e:
            logger.exception(f"Task {task_id}: failed with error")
            yield pb.ExecuteTaskResponse(
                error=pb.ErrorDetail(
                    code="EXECUTION_ERROR",
                    message=str(e),
                    retryable=self._is_retryable(e)
                )
            )
        finally:
            if task_id in self.active_tasks:
                del self.active_tasks[task_id]

    async def CancelTask(
        self,
        request: Any,
        context: grpc.aio.ServicerContext
    ) -> Any:
        """Cancel a running task."""
        task_id = request.task_id

        if task_id in self.active_tasks:
            self.active_tasks[task_id]["cancelled"] = True
            logger.info(f"Task {task_id}: cancel requested, reason: {request.reason}")
            return pb.CancelTaskResponse(
                success=True,
                message=f"Task {task_id} cancellation requested"
            )

        return pb.CancelTaskResponse(
            success=False,
            message=f"Task {task_id} not found or already completed"
        )

    async def HealthCheck(
        self,
        request: Any,
        context: grpc.aio.ServicerContext
    ) -> Any:
        """Return health status of the service."""
        return pb.HealthCheckResponse(
            status=pb.HEALTH_STATUS_HEALTHY,
            message="Service is healthy",
            details={
                "active_tasks": str(len(self.active_tasks)),
                "registered_handlers": str(len(self.handlers)),
                "handlers": ",".join(self.handlers.keys())
            }
        )

    def _is_retryable(self, error: Exception) -> bool:
        """Determine if an error should trigger a retry."""
        # Network and timeout errors are retryable
        retryable_types = (
            ConnectionError,
            TimeoutError,
            asyncio.TimeoutError,
        )
        return isinstance(error, retryable_types)


def create_progress(
    task_id: str,
    percentage: int,
    stage: str,
    message: str = "",
    metadata: Optional[Dict[str, str]] = None
) -> Any:
    """
    Helper function to create a progress response.

    Args:
        task_id: The task ID
        percentage: Completion percentage (0-100)
        stage: Current processing stage
        message: Optional progress message
        metadata: Optional additional metadata

    Returns:
        ExecuteTaskResponse containing progress
    """
    return pb.ExecuteTaskResponse(
        progress=pb.Progress(
            task_id=task_id,
            percentage=percentage,
            stage=stage,
            message=message,
            timestamp_ms=int(time.time() * 1000),
            metadata=metadata or {}
        )
    )


def payload_to_dict(payload: struct_pb2.Struct) -> Dict[str, Any]:
    """Convert protobuf Struct to Python dict."""
    from google.protobuf.json_format import MessageToDict
    return MessageToDict(payload)


# =============================================================================
# Example Handlers
# =============================================================================

async def demo_handler(
    request: Any,
    context: grpc.aio.ServicerContext,
    task_state: dict
) -> AsyncIterator[Any]:
    """
    Demo handler that simulates work with progress updates.
    """
    task_id = request.task_id
    payload = payload_to_dict(request.payload)

    message = payload.get("message", "Hello")
    count = int(payload.get("count", 5))

    logger.info(f"Demo task {task_id}: message={message}, count={count}")

    for i in range(count):
        if task_state.get("cancelled"):
            return

        await asyncio.sleep(0.5)  # Simulate work

        yield create_progress(
            task_id=task_id,
            percentage=int((i + 1) / count * 100),
            stage="processing",
            message=f"Step {i + 1}/{count}: {message}"
        )

    # Handler doesn't need to yield final result - servicer handles it


async def chat_handler(
    request: Any,
    context: grpc.aio.ServicerContext,
    task_state: dict
) -> AsyncIterator[Any]:
    """
    Example LLM chat handler that simulates streaming response.
    """
    task_id = request.task_id
    payload = payload_to_dict(request.payload)

    prompt = payload.get("prompt", "")
    max_tokens = int(payload.get("max_tokens", 100))

    logger.info(f"Chat task {task_id}: prompt='{prompt[:50]}...', max_tokens={max_tokens}")

    # Simulate streaming LLM response
    response_tokens = ["Hello", " there", "!", " I'm", " a", " simulated",
                      " LLM", " response", " for", " testing", "."]

    for i, token in enumerate(response_tokens):
        if task_state.get("cancelled"):
            return

        await asyncio.sleep(0.1)  # Simulate token generation

        yield create_progress(
            task_id=task_id,
            percentage=int((i + 1) / len(response_tokens) * 100),
            stage="generating",
            message=token,  # Stream token as message
            metadata={"token_index": str(i)}
        )


async def backtest_handler(
    request: Any,
    context: grpc.aio.ServicerContext,
    task_state: dict
) -> AsyncIterator[Any]:
    """
    Example trading strategy backtest handler.
    """
    task_id = request.task_id
    payload = payload_to_dict(request.payload)

    strategy_id = payload.get("strategy_id", "unknown")
    start_date = payload.get("start_date", "2024-01-01")

    logger.info(f"Backtest task {task_id}: strategy={strategy_id}, start={start_date}")

    stages = [
        ("loading_data", "Loading historical data..."),
        ("preprocessing", "Preprocessing data..."),
        ("running_backtest", "Running backtest simulation..."),
        ("calculating_metrics", "Calculating performance metrics..."),
        ("generating_report", "Generating report...")
    ]

    for i, (stage, message) in enumerate(stages):
        if task_state.get("cancelled"):
            return

        await asyncio.sleep(1.0)  # Simulate work

        yield create_progress(
            task_id=task_id,
            percentage=int((i + 1) / len(stages) * 100),
            stage=stage,
            message=message
        )


# =============================================================================
# Server Entry Point
# =============================================================================

async def serve(port: int = 50051):
    """
    Start the gRPC server.

    Args:
        port: Port number to listen on
    """
    if pb is None or pb_grpc is None:
        raise RuntimeError("Proto files not generated. Run generate_proto.sh first.")

    servicer = TaskExecutorServicer()

    # Register handlers
    servicer.register_handler("demo", demo_handler)
    servicer.register_handler("chat", chat_handler)
    servicer.register_handler("backtest", backtest_handler)

    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    pb_grpc.add_TaskExecutorServiceServicer_to_server(servicer, server)

    listen_addr = f'[::]:{port}'
    server.add_insecure_port(listen_addr)

    # Setup graceful shutdown
    loop = asyncio.get_event_loop()

    async def shutdown():
        logger.info("Shutting down server...")
        await server.stop(5)  # 5 second grace period

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, lambda: asyncio.create_task(shutdown()))

    logger.info(f"Starting Python Task Executor Service on {listen_addr}")
    logger.info(f"Registered handlers: {list(servicer.handlers.keys())}")

    await server.start()
    await server.wait_for_termination()


if __name__ == '__main__':
    import argparse

    parser = argparse.ArgumentParser(description='Python Task Executor Service')
    parser.add_argument('--port', type=int, default=50051, help='Port to listen on')
    args = parser.parse_args()

    asyncio.run(serve(args.port))
