import os
import time
import threading
import heapq
import itertools
import logging
import grpc
from .generated_proto.notification import notification_pb2, notification_pb2_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")

alerts_heap = []
heap_lock = threading.Lock()
tie_breaker = itertools.count()


def resolve_alert(alert: "notification_pb2.StockAlert") -> None:
    """The function called when an alert reaches the time threshold."""
    logging.info(f"Alert resolved for product: {alert.product_id}")


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
    addr = os.environ.get("NOTIFICATION_ADDR", "notificationservice:50051")

    worker = threading.Thread(target=expiration_worker, daemon=True)
    worker.start()

    logging.info(f"Connecting to gRPC server at {addr}...")
    credentials = grpc.ssl_channel_credentials()
    with grpc.secure_channel(addr, credentials) as channel:
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
