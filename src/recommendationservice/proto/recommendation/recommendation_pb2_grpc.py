"""Client and server classes corresponding to protobuf-defined services."""
import grpc
import warnings
from ..recommendation import recommendation_pb2 as recommendation_dot_recommendation__pb2
GRPC_GENERATED_VERSION = '1.81.1'
GRPC_VERSION = grpc.__version__
_version_not_supported = False
try:
    from grpc._utilities import first_version_is_lower
    _version_not_supported = first_version_is_lower(GRPC_VERSION, GRPC_GENERATED_VERSION)
except ImportError:
    _version_not_supported = True
if _version_not_supported:
    raise RuntimeError(f'The grpc package installed is at version {GRPC_VERSION},' + ' but the generated code in recommendation/recommendation_pb2_grpc.py depends on' + f' grpcio>={GRPC_GENERATED_VERSION}.' + f' Please upgrade your grpc module to grpcio>={GRPC_GENERATED_VERSION}' + f' or downgrade your generated code using grpcio-tools<={GRPC_VERSION}.')

class RecommendationServiceStub:
    """---------------Recommendation service----------

    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.ListRecommendations = channel.unary_unary('/hipstershop.RecommendationService/ListRecommendations', request_serializer=recommendation_dot_recommendation__pb2.ListRecommendationsRequest.SerializeToString, response_deserializer=recommendation_dot_recommendation__pb2.ListRecommendationsResponse.FromString, _registered_method=True)

class RecommendationServiceServicer:
    """---------------Recommendation service----------

    """

    def ListRecommendations(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

def add_RecommendationServiceServicer_to_server(servicer, server):
    rpc_method_handlers = {'ListRecommendations': grpc.unary_unary_rpc_method_handler(servicer.ListRecommendations, request_deserializer=recommendation_dot_recommendation__pb2.ListRecommendationsRequest.FromString, response_serializer=recommendation_dot_recommendation__pb2.ListRecommendationsResponse.SerializeToString)}
    generic_handler = grpc.method_handlers_generic_handler('hipstershop.RecommendationService', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('hipstershop.RecommendationService', rpc_method_handlers)

class RecommendationService:
    """---------------Recommendation service----------

    """

    @staticmethod
    def ListRecommendations(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.RecommendationService/ListRecommendations', recommendation_dot_recommendation__pb2.ListRecommendationsRequest.SerializeToString, recommendation_dot_recommendation__pb2.ListRecommendationsResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)