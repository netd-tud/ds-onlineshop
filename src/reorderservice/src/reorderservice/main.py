import os
import random
import sys
import time
import threading
import heapq
import itertools
import logging
import grpc

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

THRESHOLD_SECONDS = {
    "low": 0,
    "critical": 0,
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
    while True:
        now = time.time()

        with heap_lock:
            while alerts_heap:
                oldest_resolve_at, _, oldest_alert = alerts_heap[0]

                if now >= oldest_resolve_at:
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

    global THRESHOLD_SECONDS
    THRESHOLD_SECONDS = {
        "low": int(os.environ.get("LOW_ALERT_EXPIRATION_MINUTES", "3")) * 60,
        "critical": int(os.environ.get("CRITICAL_ALERT_EXPIRATION_MINUTES", "1")) * 60
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
                resolve_at = timestamp + THRESHOLD_SECONDS[alert.severity]

                with heap_lock:
                    existing_index = None
                    for i, item in enumerate(alerts_heap):
                        if item[2].product_id == alert.product_id:
                            existing_index = i
                            break

                    if existing_index is not None:
                        _, _, existing_alert = alerts_heap[existing_index]
                        if alert.severity == "critical" and existing_alert.severity == "low":
                            alerts_heap[existing_index] = (resolve_at, next(tie_breaker), alert)
                            heapq.heapify(alerts_heap)
                            logging.info(f"Upgraded alert to critical for {alert.product_id}")
                    else:
                        heapq.heappush(alerts_heap, (resolve_at, next(tie_breaker), alert))
                        logging.info(f"Received and stored new alert for {alert.product_id}")

                logging.info(f"Current alerts heap: {[item for item in alerts_heap]}")

        except grpc.RpcError as e:
            logging.error(f"Stream disconnected: {e.details()} (Code: {e.code()})")


if __name__ == "__main__":
    main()
