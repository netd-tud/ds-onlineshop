"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(_runtime_version.Domain.PUBLIC, 6, 33, 5, '', 'productcatalog/productcatalog.proto')
_sym_db = _symbol_database.Default()
from ..common import common_pb2 as common_dot_common__pb2
DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n#productcatalog/productcatalog.proto\x12\x0bhipstershop\x1a\x13common/common.proto"\x84\x01\n\x07Product\x12\n\n\x02id\x18\x01 \x01(\t\x12\x0c\n\x04name\x18\x02 \x01(\t\x12\x13\n\x0bdescription\x18\x03 \x01(\t\x12\x0f\n\x07picture\x18\x04 \x01(\t\x12%\n\tprice_usd\x18\x05 \x01(\x0b2\x12.hipstershop.Money\x12\x12\n\ncategories\x18\x06 \x03(\t">\n\x14ListProductsResponse\x12&\n\x08products\x18\x01 \x03(\x0b2\x14.hipstershop.Product"\x1f\n\x11GetProductRequest\x12\n\n\x02id\x18\x01 \x01(\t"&\n\x15SearchProductsRequest\x12\r\n\x05query\x18\x01 \x01(\t"?\n\x16SearchProductsResponse\x12%\n\x07results\x18\x01 \x03(\x0b2\x14.hipstershop.Product"\x83\x01\n\x17CreateNewProductRequest\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\x13\n\x0bdescription\x18\x02 \x01(\t\x12%\n\tprice_usd\x18\x03 \x01(\x0b2\x12.hipstershop.Money\x12\x12\n\ncategories\x18\x04 \x03(\t\x12\n\n\x02id\x18\x05 \x01(\t"A\n\x18CreateNewProductResponse\x12%\n\x07product\x18\x01 \x01(\x0b2\x14.hipstershop.Product""\n\x14DeleteProductRequest\x12\n\n\x02id\x18\x01 \x01(\t">\n\x15DeleteProductResponse\x12%\n\x07product\x18\x01 \x01(\x0b2\x14.hipstershop.Product"\x96\x01\n\x1dXaPrepareCreateProductRequest\x12\x0b\n\x03gid\x18\x01 \x01(\t\x12\n\n\x02id\x18\x02 \x01(\t\x12\x0c\n\x04name\x18\x03 \x01(\t\x12\x13\n\x0bdescription\x18\x04 \x01(\t\x12%\n\tprice_usd\x18\x05 \x01(\x0b2\x12.hipstershop.Money\x12\x12\n\ncategories\x18\x06 \x03(\t2\xa2\x06\n\x15ProductCatalogService\x12G\n\x0cListProducts\x12\x12.hipstershop.Empty\x1a!.hipstershop.ListProductsResponse"\x00\x12D\n\nGetProduct\x12\x1e.hipstershop.GetProductRequest\x1a\x14.hipstershop.Product"\x00\x12[\n\x0eSearchProducts\x12".hipstershop.SearchProductsRequest\x1a#.hipstershop.SearchProductsResponse"\x00\x12a\n\x10CreateNewProduct\x12$.hipstershop.CreateNewProductRequest\x1a%.hipstershop.CreateNewProductResponse"\x00\x12X\n\rDeleteProduct\x12!.hipstershop.DeleteProductRequest\x1a".hipstershop.DeleteProductResponse"\x00\x12h\n\x1aCompensateCreateNewProduct\x12$.hipstershop.CreateNewProductRequest\x1a".hipstershop.DeleteProductResponse"\x00\x12Z\n\x16XaPrepareCreateProduct\x12*.hipstershop.XaPrepareCreateProductRequest\x1a\x12.hipstershop.Empty"\x00\x12K\n\x15XaCommitCreateProduct\x12\x1c.hipstershop.XaBranchRequest\x1a\x12.hipstershop.Empty"\x00\x12M\n\x17XaRollbackCreateProduct\x12\x1c.hipstershop.XaBranchRequest\x1a\x12.hipstershop.Empty"\x00BLZJgithub.com/turt1z/microservices-demo/proto/productcatalog;productcatalogpbb\x06proto3')
_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'productcatalog.productcatalog_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
    _globals['DESCRIPTOR']._loaded_options = None
    _globals['DESCRIPTOR']._serialized_options = b'ZJgithub.com/turt1z/microservices-demo/proto/productcatalog;productcatalogpb'
    _globals['_PRODUCT']._serialized_start = 74
    _globals['_PRODUCT']._serialized_end = 206
    _globals['_LISTPRODUCTSRESPONSE']._serialized_start = 208
    _globals['_LISTPRODUCTSRESPONSE']._serialized_end = 270
    _globals['_GETPRODUCTREQUEST']._serialized_start = 272
    _globals['_GETPRODUCTREQUEST']._serialized_end = 303
    _globals['_SEARCHPRODUCTSREQUEST']._serialized_start = 305
    _globals['_SEARCHPRODUCTSREQUEST']._serialized_end = 343
    _globals['_SEARCHPRODUCTSRESPONSE']._serialized_start = 345
    _globals['_SEARCHPRODUCTSRESPONSE']._serialized_end = 408
    _globals['_CREATENEWPRODUCTREQUEST']._serialized_start = 411
    _globals['_CREATENEWPRODUCTREQUEST']._serialized_end = 542
    _globals['_CREATENEWPRODUCTRESPONSE']._serialized_start = 544
    _globals['_CREATENEWPRODUCTRESPONSE']._serialized_end = 609
    _globals['_DELETEPRODUCTREQUEST']._serialized_start = 611
    _globals['_DELETEPRODUCTREQUEST']._serialized_end = 645
    _globals['_DELETEPRODUCTRESPONSE']._serialized_start = 647
    _globals['_DELETEPRODUCTRESPONSE']._serialized_end = 709
    _globals['_XAPREPARECREATEPRODUCTREQUEST']._serialized_start = 712
    _globals['_XAPREPARECREATEPRODUCTREQUEST']._serialized_end = 862
    _globals['_PRODUCTCATALOGSERVICE']._serialized_start = 865
    _globals['_PRODUCTCATALOGSERVICE']._serialized_end = 1667