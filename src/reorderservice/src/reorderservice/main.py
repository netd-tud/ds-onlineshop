import os
import random
import sys
import time
import threading
import heapq
import itertools
import logging
import grpc
import math

from .generated_proto.notification import notification_pb2, notification_pb2_grpc
from .generated_proto.inventory import inventory_pb2, inventory_pb2_grpc
from .generated_proto.auth import auth_pb2, auth_pb2_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")

alerts_heap = []
heap_lock = threading.Lock()
tie_breaker = itertools.count()

INVENTORY_GRPC_ADDRESS = "inventoryservice:50002"

JWT = None

REORDER_AMOUNT = {
    "lowest": 0,
    "highest": 0,
}

def resolve_alert(alert: "notification_pb2.StockAlert") -> None:
    """The function called when an alert reaches the time threshold."""
    logging.info(f"Alert resolving for product: {alert.product_id}")
    with grpc.insecure_channel(INVENTORY_GRPC_ADDRESS) as channel:
        stub = inventory_pb2_grpc.InventoryServiceStub(channel)
        randReorderAmount = random.randint(REORDER_AMOUNT["lowest"], REORDER_AMOUNT["highest"])
        request = inventory_pb2.ResolveStockAlertRequest(
            product_id=alert.product_id,
            created_at=alert.created_at,
            reorder_amount=randReorderAmount
        )
        logging.info(f"ResolveStockAlert request: {request}")
        response = stub.ResolveStockAlert(
            request,
            metadata=[("authorization", f"Bearer {JWT}")],
        )
        logging.info(f"ResolveStockAlert response: {response}")


def expiration_worker() -> None:
    """Background thread that continuously checks for expired alerts."""
    threshold_minutes = float(os.environ.get("ALERT_EXPIRATION_MINUTES", "5"))
    threshold_seconds = threshold_minutes * 60

    while True:
        now = time.time()

        with heap_lock:
            while alerts_heap:
                oldest_ts, _, oldest_alert = alerts_heap[0]

                if now - oldest_ts >= threshold_seconds:
                    heapq.heappop(alerts_heap)
                    resolve_alert(oldest_alert)
                else:
                    break

        time.sleep(1.0)


def main() -> None:
    NOTIFICATION_GRPC_ADDRESS = os.environ.get("NOTIFICATION_ADDR", "notificationservice:50051")
    AUTH_GRPC_ADDRESS = os.environ.get("AUTH_ADDR", "authservice:50050")

    global INVENTORY_GRPC_ADDRESS
    INVENTORY_GRPC_ADDRESS= os.environ.get("INVENTORY_ADDR", "inventoryservice:50002")

    global REORDER_AMOUNT
    REORDER_AMOUNT = {
        "lowest": int(os.environ.get("LOWEST_REORDER_AMOUNT", "30")),
        "highest": int(os.environ.get("HIGHEST_REORDER_AMOUNT", "100"))
    }

    logging.info(f"Connecting to authservice gRPC server at {AUTH_GRPC_ADDRESS}...")
    global JWT
    with grpc.insecure_channel(AUTH_GRPC_ADDRESS) as channel:
        auth_stub = auth_pb2_grpc.AuthServiceStub(channel)
        response = auth_stub.Login(auth_pb2.LoginRequest(username="admin", password="admin"))
        JWT = response.token
        if not JWT:
            print("Could not acquire JWT token.")
            sys.exit(1)
        else:
            print("Authentication successfully.")
            print(JWT)

    worker = threading.Thread(target=expiration_worker, daemon=True)
    worker.start()

    logging.info(f"Connecting to gRPC server at {NOTIFICATION_GRPC_ADDRESS}...")
    with grpc.insecure_channel(NOTIFICATION_GRPC_ADDRESS) as channel:
        stub = notification_pb2_grpc.NotificationServiceStub(channel)
        logging.info(f"Stub: {stub}")

        request = notification_pb2.StreamStockAlertsRequest(categories=[])
        logging.info(f"Request: {request}")

        try:
            for alert in stub.StreamStockAlerts(request):
                timestamp = alert.created_at.ToDatetime().timestamp()

                with heap_lock:
                    heapq.heappush(alerts_heap, (timestamp, next(tie_breaker), alert))
                    logging.info(f"Alert stored in heap: {alerts_heap}")

                logging.info(f"Received and stored alert for {alert.product_id}")

        except grpc.RpcError as e:
            logging.error(f"Stream disconnected: {e.details()} (Code: {e.code()})")


if __name__ == "__main__":
    main()
