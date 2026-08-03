"""Client and server classes corresponding to protobuf-defined services."""
import grpc
import warnings
from ..common import common_pb2 as common_dot_common__pb2
from ..productcatalog import productcatalog_pb2 as productcatalog_dot_productcatalog__pb2
GRPC_GENERATED_VERSION = '1.81.1'
GRPC_VERSION = grpc.__version__
_version_not_supported = False
try:
    from grpc._utilities import first_version_is_lower
    _version_not_supported = first_version_is_lower(GRPC_VERSION, GRPC_GENERATED_VERSION)
except ImportError:
    _version_not_supported = True
if _version_not_supported:
    raise RuntimeError(f'The grpc package installed is at version {GRPC_VERSION},' + ' but the generated code in productcatalog/productcatalog_pb2_grpc.py depends on' + f' grpcio>={GRPC_GENERATED_VERSION}.' + f' Please upgrade your grpc module to grpcio>={GRPC_GENERATED_VERSION}' + f' or downgrade your generated code using grpcio-tools<={GRPC_VERSION}.')

class ProductCatalogServiceStub:
    """---------------Product Catalog----------------

    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.ListProducts = channel.unary_unary('/hipstershop.ProductCatalogService/ListProducts', request_serializer=common_dot_common__pb2.Empty.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.ListProductsResponse.FromString, _registered_method=True)
        self.GetProduct = channel.unary_unary('/hipstershop.ProductCatalogService/GetProduct', request_serializer=productcatalog_dot_productcatalog__pb2.GetProductRequest.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.Product.FromString, _registered_method=True)
        self.SearchProducts = channel.unary_unary('/hipstershop.ProductCatalogService/SearchProducts', request_serializer=productcatalog_dot_productcatalog__pb2.SearchProductsRequest.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.SearchProductsResponse.FromString, _registered_method=True)
        self.CreateNewProduct = channel.unary_unary('/hipstershop.ProductCatalogService/CreateNewProduct', request_serializer=productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.CreateNewProductResponse.FromString, _registered_method=True)
        self.DeleteProduct = channel.unary_unary('/hipstershop.ProductCatalogService/DeleteProduct', request_serializer=productcatalog_dot_productcatalog__pb2.DeleteProductRequest.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.DeleteProductResponse.FromString, _registered_method=True)
        self.CompensateCreateNewProduct = channel.unary_unary('/hipstershop.ProductCatalogService/CompensateCreateNewProduct', request_serializer=productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.SerializeToString, response_deserializer=productcatalog_dot_productcatalog__pb2.DeleteProductResponse.FromString, _registered_method=True)
        self.XaPrepareCreateProduct = channel.unary_unary('/hipstershop.ProductCatalogService/XaPrepareCreateProduct', request_serializer=productcatalog_dot_productcatalog__pb2.XaPrepareCreateProductRequest.SerializeToString, response_deserializer=common_dot_common__pb2.Empty.FromString, _registered_method=True)
        self.XaCommitCreateProduct = channel.unary_unary('/hipstershop.ProductCatalogService/XaCommitCreateProduct', request_serializer=common_dot_common__pb2.XaBranchRequest.SerializeToString, response_deserializer=common_dot_common__pb2.Empty.FromString, _registered_method=True)
        self.XaRollbackCreateProduct = channel.unary_unary('/hipstershop.ProductCatalogService/XaRollbackCreateProduct', request_serializer=common_dot_common__pb2.XaBranchRequest.SerializeToString, response_deserializer=common_dot_common__pb2.Empty.FromString, _registered_method=True)

class ProductCatalogServiceServicer:
    """---------------Product Catalog----------------

    """

    def ListProducts(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def GetProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def SearchProducts(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def CreateNewProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def DeleteProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def CompensateCreateNewProduct(self, request, context):
        """SAGA compensation adapter for CreateNewProduct. Accepts the same payload as CreateNewProduct
        because DTM resends the original action's request bytes when invoking the compensating transaction.
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def XaPrepareCreateProduct(self, request, context):
        """XA
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def XaCommitCreateProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def XaRollbackCreateProduct(self, request, context):
        """Missing associated documentation comment in .proto file."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

def add_ProductCatalogServiceServicer_to_server(servicer, server):
    rpc_method_handlers = {'ListProducts': grpc.unary_unary_rpc_method_handler(servicer.ListProducts, request_deserializer=common_dot_common__pb2.Empty.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.ListProductsResponse.SerializeToString), 'GetProduct': grpc.unary_unary_rpc_method_handler(servicer.GetProduct, request_deserializer=productcatalog_dot_productcatalog__pb2.GetProductRequest.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.Product.SerializeToString), 'SearchProducts': grpc.unary_unary_rpc_method_handler(servicer.SearchProducts, request_deserializer=productcatalog_dot_productcatalog__pb2.SearchProductsRequest.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.SearchProductsResponse.SerializeToString), 'CreateNewProduct': grpc.unary_unary_rpc_method_handler(servicer.CreateNewProduct, request_deserializer=productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.CreateNewProductResponse.SerializeToString), 'DeleteProduct': grpc.unary_unary_rpc_method_handler(servicer.DeleteProduct, request_deserializer=productcatalog_dot_productcatalog__pb2.DeleteProductRequest.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.DeleteProductResponse.SerializeToString), 'CompensateCreateNewProduct': grpc.unary_unary_rpc_method_handler(servicer.CompensateCreateNewProduct, request_deserializer=productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.FromString, response_serializer=productcatalog_dot_productcatalog__pb2.DeleteProductResponse.SerializeToString), 'XaPrepareCreateProduct': grpc.unary_unary_rpc_method_handler(servicer.XaPrepareCreateProduct, request_deserializer=productcatalog_dot_productcatalog__pb2.XaPrepareCreateProductRequest.FromString, response_serializer=common_dot_common__pb2.Empty.SerializeToString), 'XaCommitCreateProduct': grpc.unary_unary_rpc_method_handler(servicer.XaCommitCreateProduct, request_deserializer=common_dot_common__pb2.XaBranchRequest.FromString, response_serializer=common_dot_common__pb2.Empty.SerializeToString), 'XaRollbackCreateProduct': grpc.unary_unary_rpc_method_handler(servicer.XaRollbackCreateProduct, request_deserializer=common_dot_common__pb2.XaBranchRequest.FromString, response_serializer=common_dot_common__pb2.Empty.SerializeToString)}
    generic_handler = grpc.method_handlers_generic_handler('hipstershop.ProductCatalogService', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('hipstershop.ProductCatalogService', rpc_method_handlers)

class ProductCatalogService:
    """---------------Product Catalog----------------

    """

    @staticmethod
    def ListProducts(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/ListProducts', common_dot_common__pb2.Empty.SerializeToString, productcatalog_dot_productcatalog__pb2.ListProductsResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def GetProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/GetProduct', productcatalog_dot_productcatalog__pb2.GetProductRequest.SerializeToString, productcatalog_dot_productcatalog__pb2.Product.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def SearchProducts(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/SearchProducts', productcatalog_dot_productcatalog__pb2.SearchProductsRequest.SerializeToString, productcatalog_dot_productcatalog__pb2.SearchProductsResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def CreateNewProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/CreateNewProduct', productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.SerializeToString, productcatalog_dot_productcatalog__pb2.CreateNewProductResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def DeleteProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/DeleteProduct', productcatalog_dot_productcatalog__pb2.DeleteProductRequest.SerializeToString, productcatalog_dot_productcatalog__pb2.DeleteProductResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def CompensateCreateNewProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/CompensateCreateNewProduct', productcatalog_dot_productcatalog__pb2.CreateNewProductRequest.SerializeToString, productcatalog_dot_productcatalog__pb2.DeleteProductResponse.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def XaPrepareCreateProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/XaPrepareCreateProduct', productcatalog_dot_productcatalog__pb2.XaPrepareCreateProductRequest.SerializeToString, common_dot_common__pb2.Empty.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def XaCommitCreateProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/XaCommitCreateProduct', common_dot_common__pb2.XaBranchRequest.SerializeToString, common_dot_common__pb2.Empty.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)

    @staticmethod
    def XaRollbackCreateProduct(request, target, options=(), channel_credentials=None, call_credentials=None, insecure=False, compression=None, wait_for_ready=None, timeout=None, metadata=None):
        return grpc.experimental.unary_unary(request, target, '/hipstershop.ProductCatalogService/XaRollbackCreateProduct', common_dot_common__pb2.XaBranchRequest.SerializeToString, common_dot_common__pb2.Empty.FromString, options, channel_credentials, insecure, call_credentials, compression, wait_for_ready, timeout, metadata, _registered_method=True)