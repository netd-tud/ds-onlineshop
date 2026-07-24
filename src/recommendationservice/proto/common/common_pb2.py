"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'common/common.proto')
_sym_db = _symbol_database.Default()
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x13common/common.proto\x12\x0bhipstershop"\x07\n\x05Empty"\x1e\n\x0fXaBranchRequest\x12\x0b\n\x03gid\x18\x01 \x01(\t"<\n\x05Money\x12\x15\n\rcurrency_code\x18\x01 \x01(\t\x12\r\n\x05units\x18\x02 \x01(\x03\x12\r\n\x05nanos\x18\x03 \x01(\x05"a\n\x07Address\x12\x16\n\x0estreet_address\x18\x01 \x01(\t\x12\x0c\n\x04city\x18\x02 \x01(\t\x12\r\n\x05state\x18\x03 \x01(\t\x12\x0f\n\x07country\x18\x04 \x01(\t\x12\x10\n\x08zip_code\x18\x05 \x01(\x05"0\n\x08CartItem\x12\x12\n\nproduct_id\x18\x01 \x01(\t\x12\x10\n\x08quantity\x18\x02 \x01(\x05"R\n\tOrderItem\x12#\n\x04item\x18\x01 \x01(\x0b2\x15.hipstershop.CartItem\x12 \n\x04cost\x18\x02 \x01(\x0b2\x12.hipstershop.Money"\xbf\x01\n\x0bOrderResult\x12\x10\n\x08order_id\x18\x01 \x01(\t\x12\x1c\n\x14shipping_tracking_id\x18\x02 \x01(\t\x12)\n\rshipping_cost\x18\x03 \x01(\x0b2\x12.hipstershop.Money\x12.\n\x10shipping_address\x18\x04 \x01(\x0b2\x14.hipstershop.Address\x12%\n\x05items\x18\x05 \x03(\x0b2\x16.hipstershop.OrderItemB<Z:github.com/turt1z/microservices-demo/proto/common;commonpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'common.common_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z:github.com/turt1z/microservices-demo/proto/common;commonpb'
    _globals['_EMPTY']._serialized_start = 36
    _globals['_EMPTY']._serialized_end = 43
    _globals['_XABRANCHREQUEST']._serialized_start = 45
    _globals['_XABRANCHREQUEST']._serialized_end = 75
    _globals['_MONEY']._serialized_start = 77
    _globals['_MONEY']._serialized_end = 137
    _globals['_ADDRESS']._serialized_start = 139
    _globals['_ADDRESS']._serialized_end = 236
    _globals['_CARTITEM']._serialized_start = 238
    _globals['_CARTITEM']._serialized_end = 286
    _globals['_ORDERITEM']._serialized_start = 288
    _globals['_ORDERITEM']._serialized_end = 370
    _globals['_ORDERRESULT']._serialized_start = 373
    _globals['_ORDERRESULT']._serialized_end = 564