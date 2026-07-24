"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'email/email.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x11email/email.proto\x12\x0bhipstershop\x1a\x13common/common.proto"V\n\x1cSendOrderConfirmationRequest\x12\r\n\x05email\x18\x01 \x01(\t\x12\'\n\x05order\x18\x02 \x01(\x0b2\x18.hipstershop.OrderResult2h\n\x0cEmailService\x12X\n\x15SendOrderConfirmation\x12).hipstershop.SendOrderConfirmationRequest\x1a\x12.hipstershop.Empty"\x00B:Z8github.com/turt1z/microservices-demo/proto/email;emailpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'email.email_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z8github.com/turt1z/microservices-demo/proto/email;emailpb'
    _globals['_SENDORDERCONFIRMATIONREQUEST']._serialized_start = 55
    _globals['_SENDORDERCONFIRMATIONREQUEST']._serialized_end = 141
    _globals['_EMAILSERVICE']._serialized_start = 143
    _globals['_EMAILSERVICE']._serialized_end = 247