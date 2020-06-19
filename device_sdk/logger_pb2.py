# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: logger.proto

from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='logger.proto',
  package='pblogger',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=b'\n\x0clogger.proto\x12\x08pblogger\"/\n\tSingleLog\x12\x0f\n\x07logtype\x18\x01 \x01(\t\x12\x11\n\tlogstring\x18\x02 \x01(\t\"\x07\n\x05\x45mpty2>\n\x03Log\x12\x37\n\rSendSingleLog\x12\x13.pblogger.SingleLog\x1a\x0f.pblogger.Empty\"\x00\x62\x06proto3'
)




_SINGLELOG = _descriptor.Descriptor(
  name='SingleLog',
  full_name='pblogger.SingleLog',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='logtype', full_name='pblogger.SingleLog.logtype', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='logstring', full_name='pblogger.SingleLog.logstring', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=26,
  serialized_end=73,
)


_EMPTY = _descriptor.Descriptor(
  name='Empty',
  full_name='pblogger.Empty',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=75,
  serialized_end=82,
)

DESCRIPTOR.message_types_by_name['SingleLog'] = _SINGLELOG
DESCRIPTOR.message_types_by_name['Empty'] = _EMPTY
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

SingleLog = _reflection.GeneratedProtocolMessageType('SingleLog', (_message.Message,), {
  'DESCRIPTOR' : _SINGLELOG,
  '__module__' : 'logger_pb2'
  # @@protoc_insertion_point(class_scope:pblogger.SingleLog)
  })
_sym_db.RegisterMessage(SingleLog)

Empty = _reflection.GeneratedProtocolMessageType('Empty', (_message.Message,), {
  'DESCRIPTOR' : _EMPTY,
  '__module__' : 'logger_pb2'
  # @@protoc_insertion_point(class_scope:pblogger.Empty)
  })
_sym_db.RegisterMessage(Empty)



_LOG = _descriptor.ServiceDescriptor(
  name='Log',
  full_name='pblogger.Log',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  serialized_start=84,
  serialized_end=146,
  methods=[
  _descriptor.MethodDescriptor(
    name='SendSingleLog',
    full_name='pblogger.Log.SendSingleLog',
    index=0,
    containing_service=None,
    input_type=_SINGLELOG,
    output_type=_EMPTY,
    serialized_options=None,
  ),
])
_sym_db.RegisterServiceDescriptor(_LOG)

DESCRIPTOR.services_by_name['Log'] = _LOG

# @@protoc_insertion_point(module_scope)