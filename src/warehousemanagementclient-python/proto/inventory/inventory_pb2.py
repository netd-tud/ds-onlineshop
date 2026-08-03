"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'inventory/inventory.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x19inventory/inventory.proto\x12\x0bhipstershop\x1a\x13common/common.proto"-\n\x10InventoryProduct\x12\n\n\x02id\x18\x01 \x01(\t\x12\r\n\x05stock\x18\x02 \x01(\x03"H\n\x15ListInventoryResponse\x12/\n\x08products\x18\x01 \x03(\x0b2\x1d.hipstershop.InventoryProduct"(\n\x1aGetInventoryProductRequest\x12\n\n\x02id\x18\x01 \x01(\t"?\n"ChangeInventoryProductStockRequest\x12\n\n\x02id\x18\x01 \x01(\t\x12\r\n\x05delta\x18\x02 \x01(\x03"U\n#ChangeInventoryProductStockResponse\x12.\n\x07product\x18\x01 \x01(\x0b2\x1d.hipstershop.InventoryProduct"@\n\x1fSetInventoryProductStockRequest\x12\n\n\x02id\x18\x01 \x01(\t\x12\x11\n\tnew_stock\x18\x02 \x01(\x03"Y\n\'SetInventoryProductStockRequestResponse\x12.\n\x07product\x18\x01 \x01(\x0b2\x1d.hipstershop.InventoryProduct"E\n CreateNewInventoryProductRequest\x12\n\n\x02id\x18\x01 \x01(\t\x12\x15\n\rinitial_stock\x18\x02 \x01(\x03"S\n!CreateNewInventoryProductResponse\x12.\n\x07product\x18\x01 \x01(\x0b2\x1d.hipstershop.InventoryProduct"+\n\x1dDeleteInventoryProductRequest\x12\n\n\x02id\x18\x01 \x01(\t"P\n\x1eDeleteInventoryProductResponse\x12.\n\x07product\x18\x01 \x01(\x0b2\x1d.hipstershop.InventoryProduct"X\n&XaPrepareCreateInventoryProductRequest\x12\x0b\n\x03gid\x18\x01 \x01(\t\x12\n\n\x02id\x18\x02 \x01(\t\x12\x15\n\rinitial_stock\x18\x03 \x01(\x032\xdb\x08\n\x10InventoryService\x12I\n\rListInventory\x12\x12.hipstershop.Empty\x1a".hipstershop.ListInventoryResponse"\x00\x12_\n\x13GetInventoryProduct\x12\'.hipstershop.GetInventoryProductRequest\x1a\x1d.hipstershop.InventoryProduct"\x00\x12\x82\x01\n\x1bChangeInventoryProductStock\x12/.hipstershop.ChangeInventoryProductStockRequest\x1a0.hipstershop.ChangeInventoryProductStockResponse"\x00\x12\x80\x01\n\x18SetInventoryProductStock\x12,.hipstershop.SetInventoryProductStockRequest\x1a4.hipstershop.SetInventoryProductStockRequestResponse"\x00\x12|\n\x19CreateNewInventoryProduct\x12-.hipstershop.CreateNewInventoryProductRequest\x1a..hipstershop.CreateNewInventoryProductResponse"\x00\x12s\n\x16DeleteInventoryProduct\x12*.hipstershop.DeleteInventoryProductRequest\x1a+.hipstershop.DeleteInventoryProductResponse"\x00\x12\x83\x01\n#CompensateCreateNewInventoryProduct\x12-.hipstershop.CreateNewInventoryProductRequest\x1a+.hipstershop.DeleteInventoryProductResponse"\x00\x12l\n\x1fXaPrepareCreateInventoryProduct\x123.hipstershop.XaPrepareCreateInventoryProductRequest\x1a\x12.hipstershop.Empty"\x00\x12T\n\x1eXaCommitCreateInventoryProduct\x12\x1c.hipstershop.XaBranchRequest\x1a\x12.hipstershop.Empty"\x00\x12V\n XaRollbackCreateInventoryProduct\x12\x1c.hipstershop.XaBranchRequest\x1a\x12.hipstershop.Empty"\x00BBZ@github.com/turt1z/microservices-demo/proto/inventory;inventorypbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'inventory.inventory_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z@github.com/turt1z/microservices-demo/proto/inventory;inventorypb'
    _globals['_INVENTORYPRODUCT']._serialized_start = 63
    _globals['_INVENTORYPRODUCT']._serialized_end = 108
    _globals['_LISTINVENTORYRESPONSE']._serialized_start = 110
    _globals['_LISTINVENTORYRESPONSE']._serialized_end = 182
    _globals['_GETINVENTORYPRODUCTREQUEST']._serialized_start = 184
    _globals['_GETINVENTORYPRODUCTREQUEST']._serialized_end = 224
    _globals['_CHANGEINVENTORYPRODUCTSTOCKREQUEST']._serialized_start = 226
    _globals['_CHANGEINVENTORYPRODUCTSTOCKREQUEST']._serialized_end = 289
    _globals['_CHANGEINVENTORYPRODUCTSTOCKRESPONSE']._serialized_start = 291
    _globals['_CHANGEINVENTORYPRODUCTSTOCKRESPONSE']._serialized_end = 376
    _globals['_SETINVENTORYPRODUCTSTOCKREQUEST']._serialized_start = 378
    _globals['_SETINVENTORYPRODUCTSTOCKREQUEST']._serialized_end = 442
    _globals['_SETINVENTORYPRODUCTSTOCKREQUESTRESPONSE']._serialized_start = 444
    _globals['_SETINVENTORYPRODUCTSTOCKREQUESTRESPONSE']._serialized_end = 533
    _globals['_CREATENEWINVENTORYPRODUCTREQUEST']._serialized_start = 535
    _globals['_CREATENEWINVENTORYPRODUCTREQUEST']._serialized_end = 604
    _globals['_CREATENEWINVENTORYPRODUCTRESPONSE']._serialized_start = 606
    _globals['_CREATENEWINVENTORYPRODUCTRESPONSE']._serialized_end = 689
    _globals['_DELETEINVENTORYPRODUCTREQUEST']._serialized_start = 691
    _globals['_DELETEINVENTORYPRODUCTREQUEST']._serialized_end = 734
    _globals['_DELETEINVENTORYPRODUCTRESPONSE']._serialized_start = 736
    _globals['_DELETEINVENTORYPRODUCTRESPONSE']._serialized_end = 816
    _globals['_XAPREPARECREATEINVENTORYPRODUCTREQUEST']._serialized_start = 818
    _globals['_XAPREPARECREATEINVENTORYPRODUCTREQUEST']._serialized_end = 906
    _globals['_INVENTORYSERVICE']._serialized_start = 909
    _globals['_INVENTORYSERVICE']._serialized_end = 2024