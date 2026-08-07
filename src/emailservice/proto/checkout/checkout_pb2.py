"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'checkout/checkout.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
from ..payment import payment_pb2 as payment_dot_payment__pb2
from ..cart import cart_pb2 as cart_dot_cart__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x17checkout/checkout.proto\x12\x0bhipstershop\x1a\x13common/common.proto\x1a\x15payment/payment.proto\x1a\x0fcart/cart.proto"R\n\tOrderItem\x12#\n\x04item\x18\x01 \x01(\x0b2\x15.hipstershop.CartItem\x12 \n\x04cost\x18\x02 \x01(\x0b2\x12.hipstershop.Money"\xbf\x01\n\x0bOrderResult\x12\x10\n\x08order_id\x18\x01 \x01(\t\x12\x1c\n\x14shipping_tracking_id\x18\x02 \x01(\t\x12)\n\rshipping_cost\x18\x03 \x01(\x0b2\x12.hipstershop.Money\x12.\n\x10shipping_address\x18\x04 \x01(\x0b2\x14.hipstershop.Address\x12%\n\x05items\x18\x05 \x03(\x0b2\x16.hipstershop.OrderItem"\xa3\x01\n\x11PlaceOrderRequest\x12\x0f\n\x07user_id\x18\x01 \x01(\t\x12\x15\n\ruser_currency\x18\x02 \x01(\t\x12%\n\x07address\x18\x03 \x01(\x0b2\x14.hipstershop.Address\x12\r\n\x05email\x18\x05 \x01(\t\x120\n\x0bcredit_card\x18\x06 \x01(\x0b2\x1b.hipstershop.CreditCardInfo"=\n\x12PlaceOrderResponse\x12\'\n\x05order\x18\x01 \x01(\x0b2\x18.hipstershop.OrderResult2b\n\x0fCheckoutService\x12O\n\nPlaceOrder\x12\x1e.hipstershop.PlaceOrderRequest\x1a\x1f.hipstershop.PlaceOrderResponse"\x00B=Z;github.com/netd-tud/ds-onlineshop/proto/checkout;checkoutpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'checkout.checkout_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z;github.com/netd-tud/ds-onlineshop/proto/checkout;checkoutpb'
    _globals['_ORDERITEM']._serialized_start = 101
    _globals['_ORDERITEM']._serialized_end = 183
    _globals['_ORDERRESULT']._serialized_start = 186
    _globals['_ORDERRESULT']._serialized_end = 377
    _globals['_PLACEORDERREQUEST']._serialized_start = 380
    _globals['_PLACEORDERREQUEST']._serialized_end = 543
    _globals['_PLACEORDERRESPONSE']._serialized_start = 545
    _globals['_PLACEORDERRESPONSE']._serialized_end = 606
    _globals['_CHECKOUTSERVICE']._serialized_start = 608
    _globals['_CHECKOUTSERVICE']._serialized_end = 706