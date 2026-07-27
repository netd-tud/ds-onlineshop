"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'payment/payment.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x15payment/payment.proto\x12\x0bhipstershop\x1a\x13common/common.proto"\x90\x01\n\x0eCreditCardInfo\x12\x1a\n\x12credit_card_number\x18\x01 \x01(\t\x12\x17\n\x0fcredit_card_cvv\x18\x02 \x01(\x05\x12#\n\x1bcredit_card_expiration_year\x18\x03 \x01(\x05\x12$\n\x1ccredit_card_expiration_month\x18\x04 \x01(\x05"e\n\rChargeRequest\x12"\n\x06amount\x18\x01 \x01(\x0b2\x12.hipstershop.Money\x120\n\x0bcredit_card\x18\x02 \x01(\x0b2\x1b.hipstershop.CreditCardInfo"(\n\x0eChargeResponse\x12\x16\n\x0etransaction_id\x18\x01 \x01(\t2U\n\x0ePaymentService\x12C\n\x06Charge\x12\x1a.hipstershop.ChargeRequest\x1a\x1b.hipstershop.ChargeResponse"\x00B>Z<github.com/turt1z/microservices-demo/proto/payment;paymentpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'payment.payment_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z<github.com/turt1z/microservices-demo/proto/payment;paymentpb'
    _globals['_CREDITCARDINFO']._serialized_start = 60
    _globals['_CREDITCARDINFO']._serialized_end = 204
    _globals['_CHARGEREQUEST']._serialized_start = 206
    _globals['_CHARGEREQUEST']._serialized_end = 307
    _globals['_CHARGERESPONSE']._serialized_start = 309
    _globals['_CHARGERESPONSE']._serialized_end = 349
    _globals['_PAYMENTSERVICE']._serialized_start = 351
    _globals['_PAYMENTSERVICE']._serialized_end = 436