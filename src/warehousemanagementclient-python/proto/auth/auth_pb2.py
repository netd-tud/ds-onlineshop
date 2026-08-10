"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'auth/auth.proto')
_sym_db = _symbol_database.Default()
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x0fauth/auth.proto\x12\x04auth"2\n\x0cLoginRequest\x12\x10\n\x08username\x18\x01 \x01(\t\x12\x10\n\x08password\x18\x02 \x01(\t"\x1e\n\rLoginResponse\x12\r\n\x05token\x18\x01 \x01(\t2?\n\x0bAuthService\x120\n\x05Login\x12\x12.auth.LoginRequest\x1a\x13.auth.LoginResponseB5Z3github.com/netd-tud/ds-onlineshop/proto/auth;authpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'auth.auth_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'Z3github.com/netd-tud/ds-onlineshop/proto/auth;authpb'
    _globals['_LOGINREQUEST']._serialized_start = 25
    _globals['_LOGINREQUEST']._serialized_end = 75
    _globals['_LOGINRESPONSE']._serialized_start = 77
    _globals['_LOGINRESPONSE']._serialized_end = 107
    _globals['_AUTHSERVICE']._serialized_start = 109
    _globals['_AUTHSERVICE']._serialized_end = 172