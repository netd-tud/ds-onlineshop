import os
import random
import sys
import time
import threading
import heapq
import itertools
import logging
import grpc
from dataclasses import dataclass
from typing import List, Tuple

from .generated_proto.notification import notification_pb2, notification_pb2_grpc
from .generated_proto.inventory import inventory_pb2, inventory_pb2_grpc
from .generated_proto.auth import auth_pb2, auth_pb2_grpc
StockAlert = notification_pb2.StockAlert

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")


@dataclass
class Config:
    """Config holds microservice connection addresses, reorder boundaries, and alert thresholds.

    Attributes:
        notification_addr: Network address for the notification grpc service.
        auth_addr: Network address for the authentication grpc service.
        inventory_addr: Network address for the inventory grpc service.
        reorder_bounds: Dictionary defining lowest and highest inventory reorder quantities.
        thresholds: Dictionary defining low and critical alert expiration intervals in seconds.
    """
    notification_addr: str
    auth_addr: str
    inventory_addr: str
    reorder_bounds: dict
    thresholds: dict

    @classmethod
    def from_env(cls):
        """Creates and populates a Config instance using environment variables with sensible defaults."""
        return cls(
            notification_addr=os.environ.get("NOTIFICATION_ADDR", "notificationservice:50051"),
            auth_addr=os.environ.get("AUTH_ADDR", "authservice:50050"),
            inventory_addr=os.environ.get("INVENTORY_ADDR", "inventoryservice:50002"),
            reorder_bounds={
                "lowest": int(os.environ.get("LOWEST_REORDER_AMOUNT", "30")),
                "highest": int(os.environ.get("HIGHEST_REORDER_AMOUNT", "100"))
            },
            thresholds={
                "low": int(os.environ.get("LOW_ALERT_EXPIRATION_MINUTES", "3")) * 60,
                "critical": int(os.environ.get("CRITICAL_ALERT_EXPIRATION_MINUTES", "1")) * 60
            }
        )


class AlertManager:
    """Manages stock alerts, storing them in a thread-safe priority queue based on expiration timestamps,
    handling severity upgrades, and triggering automated inventory reorder resolutions.

    Attributes:
        inventory_stub: gRPC client stub for interacting with the inventory service.
        jwt: Authentication token used for secure gRPC requests.
        thresholds: Dictionary mapping alert severities to expiration time intervals in seconds.
        reorder_bounds: Dictionary defining the lower and upper bounds for automated reorder amounts.
        alerts_heap: Thread-safe priority queue storing active alerts sorted by resolution time.
        heap_lock: Thread lock protecting concurrent access to the alerts heap.
        tie_breaker: Counter used to maintain stable heap ordering for items with identical timestamps.
    """

    def __init__(self, inventory_stub, jwt, thresholds, reorder_bounds):
        """Initializes the AlertManager with service clients, authentication credentials, and operational parameters."""
        self.inventory_stub = inventory_stub
        self.jwt = jwt
        self.thresholds = thresholds
        self.reorder_bounds = reorder_bounds
        self.alerts_heap: List[Tuple[float, int, StockAlert]] = []
        self.heap_lock = threading.Lock()
        self.tie_breaker = itertools.count()

    def add_alert(self, alert):
        """Adds a new stock alert to the heap or upgrades an existing alert's severity if appropriate.

        Calculates the expiration timestamp based on alert severity thresholds and manages
        thread-safe updates to the underlying priority queue.
        """
        timestamp = alert.created_at.ToDatetime().timestamp()
        resolve_at = timestamp + self.thresholds.get(alert.severity, 0)

        with self.heap_lock:
            existing_index = None
            for i, item in enumerate(self.alerts_heap):
                if item[2].product_id == alert.product_id:
                    existing_index = i
                    break

            if existing_index is not None:
                _, _, existing_alert = self.alerts_heap[existing_index]
                if alert.severity == "critical" and existing_alert.severity == "low":
                    self.alerts_heap[existing_index] = (resolve_at, next(self.tie_breaker), alert)
                    heapq.heapify(self.alerts_heap)
                    logging.info(f"Upgraded alert to critical for {alert.product_id}")
            else:
                heapq.heappush(self.alerts_heap, (resolve_at, next(self.tie_breaker), alert))
                logging.info(f"Received and stored new alert for {alert.product_id}")

    def resolve_alert(self, alert):
        """Triggers the resolution of an expired stock alert by sending a gRPC request to the inventory service.

        Computes a randomized reorder amount within configured bounds and authenticates the request using the stored JWT.
        """
        logging.info(f"Alert resolving for product: {alert.product_id}")

        rand_amount = random.randint(self.reorder_bounds["lowest"], self.reorder_bounds["highest"])
        request = inventory_pb2.ResolveStockAlertRequest(
            product_id=alert.product_id,
            created_at=alert.created_at,
            reorder_amount=rand_amount
        )
        logging.info(f"ResolveStockAlert request: {request}")
        try:
            response = self.inventory_stub.ResolveStockAlert(
                request,
                metadata=[("authorization", f"Bearer {self.jwt}")]
            )
            logging.info(f"ResolveStockAlert response: {response}")
        except grpc.RpcError as e:
            logging.error(f"Failed to resolve alert for {alert.product_id}: {e.details()} (Code: {e.code()})")

    def expiration_worker(self):
        """Runs a continuous background loop that monitors the alert heap and processes expired stock alerts."""
        while True:
            now = time.time()
            with self.heap_lock:
                while self.alerts_heap:
                    oldest_resolve_at, _, oldest_alert = self.alerts_heap[0]
                    if now >= oldest_resolve_at:
                        heapq.heappop(self.alerts_heap)
                        self.resolve_alert(oldest_alert)
                    else:
                        break
            time.sleep(1.0)


def main() -> None:
    """Entry point for the stock alert monitoring microservice.

    Loads runtime configuration, authenticates against the auth service to acquire a JWT token,
    initializes the AlertManager, launches a background thread for alert expiration processing,
    and continuously streams incoming stock alerts from the notification service with automatic retry logic.
    """
    config = Config.from_env()

    logging.info(f"Connecting to authservice gRPC server at {config.auth_addr}...")
    jwt_token = None
    with grpc.insecure_channel(config.auth_addr) as auth_channel:
        auth_stub = auth_pb2_grpc.AuthServiceStub(auth_channel)
        try:
            response = auth_stub.Login(auth_pb2.LoginRequest(username="admin", password="admin"))
            jwt_token = response.token
            logging.info("Authentication successfully.")
        except grpc.RpcError as e:
            logging.error(f"Could not acquire JWT token: {e.details()}")
            sys.exit(1)

    logging.info(f"Connecting to inventoryservice gRPC server at {config.inventory_addr}...")
    inventory_channel = grpc.insecure_channel(config.inventory_addr)
    inventory_stub = inventory_pb2_grpc.InventoryServiceStub(inventory_channel)

    manager = AlertManager(inventory_stub, jwt_token, config.thresholds, config.reorder_bounds)

    worker = threading.Thread(target=manager.expiration_worker, daemon=True)
    worker.start()

    logging.info(f"Connecting to gRPC server at {config.notification_addr}...")
    with grpc.insecure_channel(config.notification_addr) as notification_channel:
        notification_stub = notification_pb2_grpc.NotificationServiceStub(notification_channel)
        request = notification_pb2.StreamStockAlertsRequest(categories=[])

        while True:
            try:
                for alert in notification_stub.StreamStockAlerts(request):
                    manager.add_alert(alert)
            except grpc.RpcError as e:
                logging.error(f"Stream disconnected: {e.details()} (Code: {e.code()}). Retrying in 5 seconds...")
                time.sleep(5.0)


if __name__ == "__main__":
    main()
