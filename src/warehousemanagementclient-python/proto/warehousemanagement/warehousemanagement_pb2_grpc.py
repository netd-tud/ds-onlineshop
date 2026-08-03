"""Client and server classes corresponding to protobuf-defined services."""
import grpc
import warnings
from ..inventory import inventory_pb2 as inventory_dot_inventory__pb2
from ..warehousemanagement import warehousemanagement_pb2 as warehousemanagement_dot_warehousemanagement__pb2
GRPC_GENERATED_VERSION = '1.81.1'
GRPC_VERSION = grpc.__version__
_version_not_supported = False
try:
    from grpc._utilities import first_version_is_lower
    _version_not_supported = first_version_is_lower(GRPC_VERSION, GRPC_GENERATED_VERSION)
except ImportError:
    _version_not_supported = True
if _version_not_supported:
    raise RuntimeError(f'The grpc package installed is at version {GRPC_VERSION},' + ' but the generated code in warehousemanagement/warehousemanagement_pb2_grpc.py depends on' + f' grpcio>={GRPC_GENERATED_VERSION}.' + f' Please upgrade your grpc module to grpcio>={GRPC_GENERATED_VERSION}' + f' or downgrade your generated code using grpcio-tools<={GRPC_VERSION}.')

class WarehouseManagementStub:
    """------------Warehouse management wrapper for students------------------

    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.UpdateProductStock = channel.unary_unary('/warehouse.management.WarehouseManagement/UpdateProductStock', request_serializer=inventory_dot_inventory__pb2.ChangeInventoryProductStockRequest.SerializeToString, response_deserializer=inventory_dot_inventory__pb2.InventoryProduct.FromString, _registered_method=True)
        self.CreateNewProduct = channel.unary_unary('/warehouse.management.WarehouseManagement/CreateNewProduct', request_serializer=warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductRequest.SerializeToString, response_deserializer=warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductResponse.FromString, _registered_method=True)

class WarehouseManagementServicer:
    """------------Warehouse management wrapper for students------------------

    """

    def UpdateProductStock(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def CreateNewProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

def add_WarehouseManagementServicer_to_server(servicer, server):
    rpc_method_handlers = {'UpdateProductStock': grpc.unary_unary_rpc_method_handler(servicer.UpdateProductStock, request_deserializer=inventory_dot_inventory__pb2.ChangeInventoryProductStockRequest.FromString, response_serializer=inventory_dot_inventory__pb2.InventoryProduct.SerializeToString), 'CreateNewProduct': grpc.unary_unary_rpc_method_handler(servicer.CreateNewProduct, request_deserializer=warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductRequest.FromString, response_serializer=warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductResponse.SerializeToString)}
    generic_handler = grpc.method_handlers_generic_handler('warehouse.management.WarehouseManagement', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('warehouse.management.WarehouseManagement', rpc_method_handlers)

class WarehouseManagement:
    """------------Warehouse management wrapper for students------------------

    """

    @staticmethod
    def UpdateProductStock(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/warehouse.management.WarehouseManagement/UpdateProductStock', inventory_dot_inventory__pb2.ChangeInventoryProductStockRequest.SerializeToString, inventory_dot_inventory__pb2.InventoryProduct.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def CreateNewProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/warehouse.management.WarehouseManagement/CreateNewProduct', warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductRequest.SerializeToString, warehousemanagement_dot_warehousemanagement__pb2.CreateWarehouseProductResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)