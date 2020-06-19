# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
import grpc

import commands_pb2 as commands__pb2


class RelayCommandStub(object):
    """Interface exported by the server.
    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.Download = channel.unary_unary(
                '/pbcommands.RelayCommand/Download',
                request_serializer=commands__pb2.DownloadParams.SerializeToString,
                response_deserializer=commands__pb2.Response.FromString,
                )
        self.Delete = channel.unary_unary(
                '/pbcommands.RelayCommand/Delete',
                request_serializer=commands__pb2.DeleteParams.SerializeToString,
                response_deserializer=commands__pb2.Response.FromString,
                )


class RelayCommandServicer(object):
    """Interface exported by the server.
    """

    def Download(self, request, context):
        """Missing associated documentation comment in .proto file"""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def Delete(self, request, context):
        """Missing associated documentation comment in .proto file"""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')


def add_RelayCommandServicer_to_server(servicer, server):
    rpc_method_handlers = {
            'Download': grpc.unary_unary_rpc_method_handler(
                    servicer.Download,
                    request_deserializer=commands__pb2.DownloadParams.FromString,
                    response_serializer=commands__pb2.Response.SerializeToString,
            ),
            'Delete': grpc.unary_unary_rpc_method_handler(
                    servicer.Delete,
                    request_deserializer=commands__pb2.DeleteParams.FromString,
                    response_serializer=commands__pb2.Response.SerializeToString,
            ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
            'pbcommands.RelayCommand', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))


 # This class is part of an EXPERIMENTAL API.
class RelayCommand(object):
    """Interface exported by the server.
    """

    @staticmethod
    def Download(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(request, target, '/pbcommands.RelayCommand/Download',
            commands__pb2.DownloadParams.SerializeToString,
            commands__pb2.Response.FromString,
            options, channel_credentials,
            call_credentials, compression, wait_for_ready, timeout, metadata)

    @staticmethod
    def Delete(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(request, target, '/pbcommands.RelayCommand/Delete',
            commands__pb2.DeleteParams.SerializeToString,
            commands__pb2.Response.FromString,
            options, channel_credentials,
            call_credentials, compression, wait_for_ready, timeout, metadata)