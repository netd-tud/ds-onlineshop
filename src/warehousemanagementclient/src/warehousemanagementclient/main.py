import json
import logging
from pathlib import Path

import ssl
import sys
import grpc
import paho.mqtt.client as mqtt
from typing import Dict, Any, Optional

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


def load_config(file_path: Path) -> Dict[str, Any]:
    """
    Loads and parses the JSON configuration file from the specified path.

    Exits the process with status code 1 if the file is missing or contains invalid JSON.
    """
    try:
        with open(file_path, "r") as f:
            return json.load(f)
    except FileNotFoundError:
        logging.error(f"Configuration file {file_path} not found.")
        sys.exit(1)
    except json.JSONDecodeError as e:
        logging.error(f"Failed to parse JSON config: {e}")
        sys.exit(1)


def create_secure_channel(target_address: str) -> grpc.Channel:
    """
    Creates an SSL/TLS-encrypted gRPC channel target_address with SNI hostname override.
    """
    credentials = grpc.ssl_channel_credentials()
    options = [('grpc.ssl_target_name_override', BASE_HOST)]
    return grpc.secure_channel(target_address, credentials, options=options)


def receive_jwt(stub: auth_pb_grpc.AuthServiceStub, config: Dict[str, Any]) -> str:
    """
    Authenticates against the AuthService via gRPC using configured credentials
    and returns the acquired JSON Web Token (JWT).
    """
    username = config.get("username")
    password = config.get("password")

    if not username or not password:
        logging.error("Missing 'username' or 'password' in configuration.")
        return None

    try:
        response = stub.Login(auth_pb.LoginRequest(username=username, password=password))
        return response.token
    except grpc.RpcError as e:
        logging.error(f"Auth failed: {e.details()}")
        return None


def handle_create_grpc(stub: whm_pb_grpc.WarehouseManagementStub, data: Dict[str, Any], jwt: str, caller_id: str):
    """
    Executes the CreateNewProduct gRPC request using product parameters from the configuration
    and attaches the JWT bearer token for authorization.
    """
    logging.info("--- Calling CreateNewProduct via gRPC ---")

    price_data = data.get("price_usd", {})
    price = money_pb.Money(
        currency_code=price_data.get("currency_code", ""),
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
            metadata=[
                ("authorization", f"Bearer {jwt}"),
                ("x-caller-id-secret", f"{caller_id}")
            ]
        )
        logging.info(f"gRPC: Product Created Successfully!")
        logging.info(f"ID: {response.product.id} | Name: {response.product.name}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not create product: {e.details()} (Code: {e.code()})")


def handle_update_grpc(stub: whm_pb_grpc.WarehouseManagementStub, data: Dict[str, Any], jwt: str, caller_id: str):
    """
    Executes the UpdateProductStock gRPC request to change inventory stock levels,
    validating the target product ID and attaching the JWT bearer token.
    """
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
            metadata=[
                ("authorization", f"Bearer {jwt}"),
                ("x-caller-id-secret", f"{caller_id}")
            ]
        )
        logging.info(f"gRPC: Stock Updated Successfully!")
        logging.info(f"Product ID: {response.id} | New Stock Level: {response.stock}")
    except grpc.RpcError as e:
        logging.error(f"gRPC: Could not update product stock: {e.details()} (Code: {e.code()})")


def grpc_execution(config: Dict[str, Any], action: str, grpc_address: str, jwt: str, caller_id: str):
    """
    Establishes a secure gRPC channel and routes the specified action ('create' or 'update')
    to its corresponding handler with the provided JWT authentication data.
    """
    logging.info(f"Connecting to warehousemanagement gRPC server at {grpc_address}...")
    try:
        with create_secure_channel(grpc_address) as channel:
            stub = whm_pb_grpc.WarehouseManagementStub(channel)
            if action == "create":
                handle_create_grpc(stub, config.get("create_product", {}), jwt, caller_id)
            elif action == "update":
                handle_update_grpc(stub, config.get("update_stock", {}), jwt, caller_id)
    except Exception as e:
        logging.error(f"gRPC execution failed: {e}")


def handle_create_mqtt(client: mqtt.Client, config: Dict[str, Any], jwt: str):
    """
    Serializes product creation metadata and authorization token into JSON format and publishes
    it to the 'inventory/create-item' MQTT topic with QoS 1.
    """
    create_topic = "inventory/create-item"
    product_data = config.get("create_product", {})
    price_data = product_data.get("price_usd", {})

    new_product_payload = {
        "name": product_data.get("name", ""),
        "description": product_data.get("description", ""),
        "price_usd": {
            "currency_code": price_data.get("currency_code", ""),
            "units": price_data.get("units", 0),
            "nanos": price_data.get("nanos", 0),
        },
        "categories": product_data.get("categories", []),
        "initial_stock": product_data.get("initial_stock", 0),
        "token": jwt
    }

    create_bytes = json.dumps(new_product_payload)
    logging.info(
        f"MQTT: Publishing creation request for '{new_product_payload['name']}' to topic '{create_topic}'...")
    token = client.publish(create_topic, create_bytes, qos=1)

    try:
        token.wait_for_publish(timeout=5)
        if token.is_published():
            logging.info("MQTT: Creation request published successfully")
        else:
            logging.error("MQTT: Publishing creation timed out or failed")
    except RuntimeError as e:
        logging.error(f"MQTT: Failed during publish wait: {e}")


def handle_update_mqtt(client: mqtt.Client, config: Dict[str, Any], jwt: str):
    """
    Serializes stock modification parameters and authorization token into JSON format and publishes
    it to the 'inventory/update-product-stock' MQTT topic with QoS 1.
    """
    update_topic = "inventory/update-product-stock"
    update_data = config.get("update_stock", {})

    update_payload = {
        "id": update_data.get("id", ""),
        "delta": update_data.get("delta", 0),
        "token": jwt
    }

    update_bytes = json.dumps(update_payload)
    logging.info(
        f"MQTT: Publishing update request for product '{update_payload['id']}' to topic '{update_topic}'...")
    token = client.publish(update_topic, update_bytes, qos=1)

    try:
        token.wait_for_publish(timeout=5)
        if token.is_published():
            logging.info("MQTT: Update request published successfully")
        else:
            logging.error("MQTT: Publishing update timed out or failed")
    except RuntimeError as e:
        logging.error(f"MQTT: Failed during publish wait: {e}")


def mqtt_execution(config: Dict[str, Any], jwt: str):
    """
    Initializes a TLS-enabled MQTT client, connects to the broker, loop-starts the background thread,
    and dispatches requested operations ('create' or 'update') before disconnecting.
    """
    action = config.get("action", "").lower().strip()
    logging.info("--- MQTT PUBLISHING ---")

    try:
        mqtt_client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2, client_id="warehousemanagement_client")
    except AttributeError:
        mqtt_client = mqtt.Client(client_id="warehousemanagement_client")

    mqtt_client.tls_set()

    try:
        mqtt_client.connect(BASE_HOST, MQTT_BROKER_PORT, keepalive=60)
        mqtt_client.loop_start()
        logging.info("MQTT: Connected successfully to broker")

        if action == "create":
            handle_create_mqtt(mqtt_client, config, jwt)
        elif action == "update":
            handle_update_mqtt(mqtt_client, config, jwt)
    except Exception as e:
        logging.error(f"MQTT: Connection failed: {e}")
    finally:
        mqtt_client.loop_stop()
        mqtt_client.disconnect()
        logging.info("MQTT: Disconnected from broker")


def main():
    """
    Entry point for the client utility. Validates configuration settings, acquires an auth token,
    and executes product operations over gRPC or MQTT based on selected transaction modes.
    """
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

    logging.info(f"Connecting to authservice gRPC server at {AUTH_GRPC_ADDRESS}...")
    try:
        with create_secure_channel(AUTH_GRPC_ADDRESS) as channel:
            auth_stub = auth_pb_grpc.AuthServiceStub(channel)
            jwt = receive_jwt(auth_stub, config)

            if not jwt:
                logging.error("Could not acquire JWT token. Exiting.")
                sys.exit(1)

            logging.info(f"Authenticated successfully. Token: {jwt[:5]}...{jwt[-5:]}")
    except Exception as e:
        logging.error(f"Failed to authenticate with AuthService: {e}")
        sys.exit(1)

    if connection_type == "mqtt":
        mqtt_execution(config, jwt)
        return
    elif connection_type == "grpc":
        grpc_execution(config, action, grpc_address, jwt, config.get("caller-id", ""))
        return


if __name__ == "__main__":
    main()
