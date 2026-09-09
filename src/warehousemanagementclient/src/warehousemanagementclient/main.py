import json
import logging
from pathlib import Path

import sys
import grpc
import paho.mqtt.client as mqtt

from warehousemanagementclient.genproto.common import common_pb2 as money_pb
from warehousemanagementclient.genproto.inventory import inventory_pb2 as inventory_pb
from warehousemanagementclient.genproto.warehousemanagement import warehousemanagement_pb2 as whm_pb
from warehousemanagementclient.genproto.warehousemanagement import warehousemanagement_pb2_grpc as whm_pb_grpc
from warehousemanagementclient.genproto.auth import auth_pb2 as auth_pb
from warehousemanagementclient.genproto.auth import auth_pb2_grpc as auth_pb_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

BASE_HOST = "ds-exercise-01.netd.cs.tu-dresden.de"
AUTH_GRPC_ADDRESS = f"{BASE_HOST}:30060"
MQTT_BROKER_PORT = 31883
CONFIG_FILE = Path(__file__).parent / "config.json"

DT_PORTS = {
    "naive": 30051,
    "saga": 30052,
    "xa": 30053
}

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

def create_secure_channel(target_address):
    credentials = grpc.ssl_channel_credentials()
    options = [('grpc.ssl_target_name_override', BASE_HOST)]
    return grpc.secure_channel(target_address, credentials, options=options)


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

def receiveJWT(stub, config):

    username = config.get("username")
    password = config.get("password")
    response = stub.Login(auth_pb.LoginRequest(username=username, password=password))
    return response.token


def mqtt_execution(config=None):
    if config.get("action", "").lower().strip() != "create":
        logging.error("MQTT: Only 'create' action is supported for MQTT execution.")
        return

    logging.info("--- MQTT PUBLISHING ---")
    try:
        mqtt_client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2, client_id="warehousemanagement_client")
    except AttributeError:
        mqtt_client = mqtt.Client(client_id="warehousemanagement_client")

    try:
        mqtt_client.connect(BASE_HOST, MQTT_BROKER_PORT, keepalive=60)
    except Exception as e:
        logging.error(f"MQTT: Connection failed: {e}")
        return

    mqtt_client.loop_start()
    logging.info("MQTT: Connected successfully to broker")

    try:
        create_topic = "inventory/create-item"
        product_data = config.get("create_product", {}) if config else {}
        price_data = product_data.get("price_usd", {})

        new_product_payload = {
            "name": product_data.get("name", "Lighter"),
            "description": product_data.get("description", "Simple, light Lighter."),
            "price_usd": {
                "currency_code": price_data.get("currency_code", "USD"),
                "units": price_data.get("units", 1),
                "nanos": price_data.get("nanos", 500000000),
            },
            "categories": product_data.get("categories", ["utility"]),
            "initial_stock": product_data.get("initial_stock", 50),
        }

        create_bytes = json.dumps(new_product_payload)
        logging.info(
            f"MQTT: Publishing creation request for '{new_product_payload['name']}' to topic '{create_topic}'...")
        token = mqtt_client.publish(create_topic, create_bytes, qos=1)
        token.wait_for_publish(timeout=5)

        if token.is_published():
            logging.info("MQTT: Creation request published successfully")
        else:
            logging.error("MQTT: Publishing creation failed")
    finally:
        mqtt_client.loop_stop()
        mqtt_client.disconnect()


def main():
    config = load_config(CONFIG_FILE)

    dt_function = config.get("dt-function", "").lower().strip()
    if dt_function not in DT_PORTS:
        logging.error(f"Invalid distributed transaction function '{dt_function}'. Use 'naive', 'saga' or 'xa'")
        sys.exit(1)

    grpc_address = f"{BASE_HOST}:{DT_PORTS[dt_function]}"

    action = config.get("action", "").lower().strip()
    if action not in ["create", "update"]:
        logging.error(f"Invalid action '{action}' in config. Use 'create' or 'update'.")
        sys.exit(1)

    connection_type = config.get("connection-type", "").lower().strip()
    if connection_type not in ["grpc", "mqtt"]:
        logging.error(f"Invalid connection type '{connection_type}' in config. Use 'grpc' or 'mqtt'.")
        sys.exit(1)

    if connection_type == "mqtt":
        mqtt_execution(config)
        return

    logging.info(f"Connecting to authservice gRPC server at {AUTH_GRPC_ADDRESS}...")
    with create_secure_channel(AUTH_GRPC_ADDRESS) as channel:
        auth_stub = auth_pb_grpc.AuthServiceStub(channel)
        jwt = receiveJWT(auth_stub, config)
        if not jwt:
            logging.error("Could not acquire JWT token. Exiting.")
            sys.exit(1)

        logging.info(f"Authenticated successfully. Token: {jwt[:5]}...{jwt[-5:]}")

    logging.info(f"Connecting to warehousemanagement gRPC server at {grpc_address}...")
    with create_secure_channel(grpc_address) as channel:
        stub = whm_pb_grpc.WarehouseManagementStub(channel)

        if action == "create":
            handle_create(stub, config.get("create_product", {}), jwt)
        elif action == "update":
            handle_update(stub, config.get("update_stock", {}), jwt)


if __name__ == "__main__":
    main()
