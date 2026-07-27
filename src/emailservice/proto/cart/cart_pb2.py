"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'cart/cart.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x0fcart/cart.proto\x12\x0bhipstershop\x1a\x13common/common.proto"0\n\x08CartItem\x12\x12\n\nproduct_id\x18\x01 \x01(\t\x12\x10\n\x08quantity\x18\x02 \x01(\x05"F\n\x0eAddItemRequest\x12\x0f\n\x07user_id\x18\x01 \x01(\t\x12#\n\x04item\x18\x02 \x01(\x0b2\x15.hipstershop.CartItem"#\n\x10EmptyCartRequest\x12\x0f\n\x07user_id\x18\x01 \x01(\t"!\n\x0eGetCartRequest\x12\x0f\n\x07user_id\x18\x01 \x01(\t"=\n\x04Cart\x12\x0f\n\x07user_id\x18\x01 \x01(\t\x12$\n\x05items\x18\x02 \x03(\x0b2\x15.hipstershop.CartItem2\xca\x01\n\x0bCartService\x12<\n\x07AddItem\x12\x1b.hipstershop.AddItemRequest\x1a\x12.hipstershop.Empty"\x00\x12;\n\x07GetCart\x12\x1b.hipstershop.GetCartRequest\x1a\x11.hipstershop.Cart"\x00\x12@\n\tEmptyCart\x12\x1d.hipstershop.EmptyCartRequest\x1a\x12.hipstershop.Empty"\x00B8Z6github.com/turt1z/microservices-demo/proto/cart;cartpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'cart.cart_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z6github.com/turt1z/microservices-demo/proto/cart;cartpb'
    _globals['_CARTITEM']._serialized_start = 53
    _globals['_CARTITEM']._serialized_end = 101
    _globals['_ADDITEMREQUEST']._serialized_start = 103
    _globals['_ADDITEMREQUEST']._serialized_end = 173
    _globals['_EMPTYCARTREQUEST']._serialized_start = 175
    _globals['_EMPTYCARTREQUEST']._serialized_end = 210
    _globals['_GETCARTREQUEST']._serialized_start = 212
    _globals['_GETCARTREQUEST']._serialized_end = 245
    _globals['_CART']._serialized_start = 247
    _globals['_CART']._serialized_end = 308
    _globals['_CARTSERVICE']._serialized_start = 311
    _globals['_CARTSERVICE']._serialized_end = 513