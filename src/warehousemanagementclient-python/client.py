import json
import logging
import sys
import grpc
import os

import demo_pb2 as demo
import warehousemanagement_pb2 as pb
import warehousemanagement_pb2_grpc as pb_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

GRPC_ADDRESS = "ds-exercise-06.netd.cs.tu-dresden.de:30050"
CONFIG_FILE = "config.json"

def load_config(file_path):
    try:
        with open(file_path, "r") as f:
            return json.load(f)
    except FileNotFoundError:
        logging.error(f"Configuration file {file_path} not found.")
        sys.exit(1)
    except json.JSONDecodeError as e:
        logging.error(f"Failed to parse JSON config: {e}")
        sys.exit(1)

def handle_create(stub, data):
    logging.info("--- Calling CreateNewProduct via gRPC ---")

    price_data = data.get("price_usd", {})
    price = demo.Money(
        currency_code=price_data.get("currency_code", "USD"),
        units=price_data.get("units", 0),
        nanos=price_data.get("nanos", 0)
    )

    request = pb.CreateWarehouseProductRequest(
        name=data.get("name", ""),
        description=data.get("description", ""),
        price_usd=price,
        categories=data.get("categories", []),
        initial_stock=data.get("initial_stock", 0)
    )

    try:
        response = stub.CreateNewProductWithXA(request, timeout=5)
        logging.info(f"gRPC: Product Created Successfully!")
        logging.info(f"ID: {response.product.id} | Name: {response.product.name}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not create product: {e.details()} (Code: {e.code()})")

def handle_update(stub, data):
    logging.info("--- Calling UpdateProductStock via gRPC ---")

    product_id = data.get("id")
    if not product_id:
        logging.error("gRPC Update Failed: No 'id' provided in 'update_stock' configuration.")
        return

    request = demo.ChangeInventoryProductStockRequest(
        id=product_id,
        delta=data.get("delta", 0)
    )

    try:
        response = stub.UpdateProductStock(request, timeout=5)
        logging.info(f"gRPC: Stock Updated Successfully!")
        logging.info(f"Product ID: {response.id} | New Stock Level: {response.stock}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not update product stock: {e.details()} (Code: {e.code()})")

def main():
    config = load_config(CONFIG_FILE)
    action = config.get("action", "").lower().strip()

    if action not in ["create", "update"]:
        logging.error(f"Invalid action '{action}' in config. Use 'create' or 'update'.")
        sys.exit(1)

    logging.info(f"Connecting to gRPC server at {GRPC_ADDRESS}...")
    with grpc.insecure_channel(GRPC_ADDRESS) as channel:
        stub = pb_grpc.WarehouseManagementStub(channel)

        if action == "create":
            handle_create(stub, config.get("create_product", {}))
        elif action == "update":
            handle_update(stub, config.get("update_stock", {}))

if __name__ == "__main__":
    main()
