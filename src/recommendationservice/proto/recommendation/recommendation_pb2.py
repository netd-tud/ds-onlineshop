"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'recommendation/recommendation.proto')
_sym_db = _symbol_database.Default()
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n#recommendation/recommendation.proto\x12\x0bhipstershop"B\n\x1aListRecommendationsRequest\x12\x0f\n\x07user_id\x18\x01 \x01(\t\x12\x13\n\x0bproduct_ids\x18\x02 \x03(\t"2\n\x1bListRecommendationsResponse\x12\x13\n\x0bproduct_ids\x18\x01 \x03(\t2\x83\x01\n\x15RecommendationService\x12j\n\x13ListRecommendations\x12\'.hipstershop.ListRecommendationsRequest\x1a(.hipstershop.ListRecommendationsResponse"\x00BLZJgithub.com/turt1z/microservices-demo/proto/recommendation;recommendationpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'recommendation.recommendation_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'ZJgithub.com/turt1z/microservices-demo/proto/recommendation;recommendationpb'
    _globals['_LISTRECOMMENDATIONSREQUEST']._serialized_start = 52
    _globals['_LISTRECOMMENDATIONSREQUEST']._serialized_end = 118
    _globals['_LISTRECOMMENDATIONSRESPONSE']._serialized_start = 120
    _globals['_LISTRECOMMENDATIONSRESPONSE']._serialized_end = 170
    _globals['_RECOMMENDATIONSERVICE']._serialized_start = 173
    _globals['_RECOMMENDATIONSERVICE']._serialized_end = 304