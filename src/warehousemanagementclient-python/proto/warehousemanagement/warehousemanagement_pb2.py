"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'warehousemanagement/warehousemanagement.proto')
_sym_db = _symbol_database.Default()
from ..inventory import inventory_pb2 as inventory_dot_inventory__pb2
from ..productcatalog import productcatalog_pb2 as productcatalog_dot_productcatalog__pb2
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n-warehousemanagement/warehousemanagement.proto\x12\x14warehouse.management\x1a\x19inventory/inventory.proto\x1a#productcatalog/productcatalog.proto\x1a\x13common/common.proto"\x94\x01\n\x1dCreateWarehouseProductRequest\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\x13\n\x0bdescription\x18\x02 \x01(\t\x12%\n\tprice_usd\x18\x03 \x01(\x0b2\x12.hipstershop.Money\x12\x12\n\ncategories\x18\x04 \x03(\t\x12\x15\n\rinitial_stock\x18\x05 \x01(\x03"G\n\x1eCreateWarehouseProductResponse\x12%\n\x07product\x18\x01 \x01(\x0b2\x14.hipstershop.Product2\xfa\x01\n\x13WarehouseManagement\x12d\n\x12UpdateProductStock\x12/.hipstershop.ChangeInventoryProductStockRequest\x1a\x1d.hipstershop.InventoryProduct\x12}\n\x10CreateNewProduct\x123.warehouse.management.CreateWarehouseProductRequest\x1a4.warehouse.management.CreateWarehouseProductResponseBVZTgithub.com/turt1z/microservices-demo/proto/warehousemanagement/warehousemanagementpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'warehousemanagement.warehousemanagement_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'ZTgithub.com/turt1z/microservices-demo/proto/warehousemanagement/warehousemanagementpb'
    _globals['_CREATEWAREHOUSEPRODUCTREQUEST']._serialized_start = 157
    _globals['_CREATEWAREHOUSEPRODUCTREQUEST']._serialized_end = 305
    _globals['_CREATEWAREHOUSEPRODUCTRESPONSE']._serialized_start = 307
    _globals['_CREATEWAREHOUSEPRODUCTRESPONSE']._serialized_end = 378
    _globals['_WAREHOUSEMANAGEMENT']._serialized_start = 381
    _globals['_WAREHOUSEMANAGEMENT']._serialized_end = 631