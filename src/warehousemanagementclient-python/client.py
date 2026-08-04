import json
import logging
import sys
import grpc

import proto.common.common_pb2 as money_pb
import proto.inventory.inventory_pb2 as inventory_pb
import proto.warehousemanagement.warehousemanagement_pb2 as whm_pb
import proto.warehousemanagement.warehousemanagement_pb2_grpc as whm_pb_grpc
import proto.auth.auth_pb2 as auth_pb
import proto.auth.auth_pb2_grpc as auth_pb_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

GRPC_ADDRESS = "ds-exercise-01.netd.cs.tu-dresden.de:30050"
CONFIG_FILE = "config.json"

AUTH_GRPC_ADDRESS = "ds-exercise-01.netd.cs.tu-dresden.de:30060"

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

def handle_create(stub, data, jwt):
    logging.info("--- Calling CreateNewProduct via gRPC ---")

    price_data = data.get("price_usd", {})
    price = money_pb.Money(
        currency_code=price_data.get("currency_code", "USD"),
        units=price_data.get("units", 0),
        nanos=price_data.get("nanos", 0)
    )

    request = whm_pb.CreateWarehouseProductRequest(
        name=data.get("name", ""),
        description=data.get("description", ""),
        price_usd=price,
        categories=data.get("categories", []),
        initial_stock=data.get("initial_stock", 0)
    )

    try:
        response = stub.CreateNewProduct(
            request,
            timeout=5,
            metadata=[("authorization", f"Bearer {jwt}")]
        )
        logging.info(f"gRPC: Product Created Successfully!")
        logging.info(f"ID: {response.product.id} | Name: {response.product.name}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not create product: {e.details()} (Code: {e.code()})")

def handle_update(stub, data, jwt):
    logging.info("--- Calling UpdateProductStock via gRPC ---")

    product_id = data.get("id")
    if not product_id:
        logging.error("gRPC Update Failed: No 'id' provided in 'update_stock' configuration.")
        return

    request = inventory_pb.ChangeInventoryProductStockRequest(
        id=product_id,
        delta=data.get("delta", 0)
    )

    try:
        response = stub.UpdateProductStock(
            request,
            timeout=5,
            metadata=[("authorization", f"Bearer {jwt}")]
        )
        logging.info(f"gRPC: Stock Updated Successfully!")
        logging.info(f"Product ID: {response.id} | New Stock Level: {response.stock}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not update product stock: {e.details()} (Code: {e.code()})")

def receiveJWT(stub):
    username = "isabellal"
    password = "isabellal"
    response = stub.Login(auth_pb.LoginRequest(username=username, password=password))
    return response.token

def main():
    config = load_config(CONFIG_FILE)
    dt_function = config.get("dt-function", "").lower().strip()
    if dt_function not in ["naive", "saga", "xa"]:
        logging.error(f"Invalid distributed transaction function '{dt_function}' in config. Use 'naive', 'saga' or 'xa'")
        sys.exit(1)
    else:
        match dt_function:
            case "naive":
                GRPC_ADDRESS = "ds-exercise-01.netd.cs.tu-dresden.de:30051"
            case "saga":
                GRPC_ADDRESS = "ds-exercise-01.netd.cs.tu-dresden.de:30052"
            case "xa":
                GRPC_ADDRESS = "ds-exercise-01.netd.cs.tu-dresden.de:30053"

    action = config.get("action", "").lower().strip()
    if action not in ["create", "update"]:
        logging.error(f"Invalid action '{action}' in config. Use 'create' or 'update'.")
        sys.exit(1)

    channel_credentials = grpc.ssl_channel_credentials()
    options = [('grpc.ssl_target_name_override', 'ds-exercise-01.netd.cs.tu-dresden.de')]

    logging.info(f"Connecting to authservice gRPC server at {AUTH_GRPC_ADDRESS}...")
    jwt = None
    with grpc.secure_channel(AUTH_GRPC_ADDRESS, channel_credentials, options=options) as channel:
        auth_stub = auth_pb_grpc.AuthServiceStub(channel)
        jwt = receiveJWT(auth_stub)
        if not jwt:
            print("Could not acquire JWT token.")
            sys.exit(1)
        else:
            print("Authentication successfully.")
            print(jwt)

    logging.info(f"Connecting to warehousemanagement gRPC server at {GRPC_ADDRESS}...")
    with grpc.secure_channel(GRPC_ADDRESS, channel_credentials, options=options) as channel:
        stub = whm_pb_grpc.WarehouseManagementStub(channel)

        if action == "create":
            handle_create(stub, config.get("create_product", {}), jwt)
        elif action == "update":
            handle_update(stub, config.get("update_stock", {}), jwt)

if __name__ == "__main__":
    main()
