"""Client and server classes corresponding to protobuf-defined services."""
import grpc
import warnings
from ..cart import cart_pb2 as cart_dot_cart__pb2
from ..common import common_pb2 as common_dot_common__pb2
GRPC_GENERATED_VERSION = '1.81.1'
GRPC_VERSION = grpc.__version__
_version_not_supported = False
try:
    from grpc._utilities import first_version_is_lower
    _version_not_supported = first_version_is_lower(GRPC_VERSION, GRPC_GENERATED_VERSION)
except ImportError:
    _version_not_supported = True
if _version_not_supported:
    raise RuntimeError(f'The grpc package installed is at version {GRPC_VERSION},' + ' but the generated code in cart/cart_pb2_grpc.py depends on' + f' grpcio>={GRPC_GENERATED_VERSION}.' + f' Please upgrade your grpc module to grpcio>={GRPC_GENERATED_VERSION}' + f' or downgrade your generated code using grpcio-tools<={GRPC_VERSION}.')

class CartServiceStub:
    """-----------------Cart service-----------------

    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.AddItem = channel.unary_unary('/hipstershop.CartService/AddItem', request_serializer=cart_dot_cart__pb2.AddItemRequest.SerializeToString, response_deserializer=common_dot_common__pb2.Empty.FromString, _registered_method=True)
        self.GetCart = channel.unary_unary('/hipstershop.CartService/GetCart', request_serializer=cart_dot_cart__pb2.GetCartRequest.SerializeToString, response_deserializer=cart_dot_cart__pb2.Cart.FromString, _registered_method=True)
        self.EmptyCart = channel.unary_unary('/hipstershop.CartService/EmptyCart', request_serializer=cart_dot_cart__pb2.EmptyCartRequest.SerializeToString, response_deserializer=common_dot_common__pb2.Empty.FromString, _registered_method=True)

class CartServiceServicer:
    """-----------------Cart service-----------------

    """

    def AddItem(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def GetCart(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def EmptyCart(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

def add_CartServiceServicer_to_server(servicer, server):
    rpc_method_handlers = {'AddItem': grpc.unary_unary_rpc_method_handler(servicer.AddItem, request_deserializer=cart_dot_cart__pb2.AddItemRequest.FromString, response_serializer=common_dot_common__pb2.Empty.SerializeToString), 'GetCart': grpc.unary_unary_rpc_method_handler(servicer.GetCart, request_deserializer=cart_dot_cart__pb2.GetCartRequest.FromString, response_serializer=cart_dot_cart__pb2.Cart.SerializeToString), 'EmptyCart': grpc.unary_unary_rpc_method_handler(servicer.EmptyCart, request_deserializer=cart_dot_cart__pb2.EmptyCartRequest.FromString, response_serializer=common_dot_common__pb2.Empty.SerializeToString)}
    generic_handler = grpc.method_handlers_generic_handler('hipstershop.CartService', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('hipstershop.CartService', rpc_method_handlers)

class CartService:
    """-----------------Cart service-----------------

    """

    @staticmethod
    def AddItem(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.CartService/AddItem', cart_dot_cart__pb2.AddItemRequest.SerializeToString, common_dot_common__pb2.Empty.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def GetCart(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.CartService/GetCart', cart_dot_cart__pb2.GetCartRequest.SerializeToString, cart_dot_cart__pb2.Cart.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def EmptyCart(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.CartService/EmptyCart', cart_dot_cart__pb2.EmptyCartRequest.SerializeToString, common_dot_common__pb2.Empty.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)