# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: commands.proto

from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='commands.proto',
  package='pbcommands',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=b'\n\x0e\x63ommands.proto\x12\npbcommands\"2\n\x04\x46ile\x12\x0c\n\x04name\x18\x01 \x01(\t\x12\x0b\n\x03\x63\x64n\x18\x02 \x01(\t\x12\x0f\n\x07hashsum\x18\x03 \x01(\t\"\x1e\n\x07\x43hannel\x12\x13\n\x0b\x63hannelname\x18\x01 \x01(\t\"\xc2\x01\n\x0e\x44ownloadParams\x12\x12\n\nfolderpath\x18\x01 \x01(\t\x12\'\n\rmetadatafiles\x18\x02 \x03(\x0b\x32\x10.pbcommands.File\x12#\n\tbulkfiles\x18\x03 \x03(\x0b\x32\x10.pbcommands.File\x12%\n\x08\x63hannels\x18\x04 \x03(\x0b\x32\x13.pbcommands.Channel\x12\x10\n\x08\x64\x65\x61\x64line\x18\x05 \x01(\x03\x12\x15\n\raddtoexisting\x18\x06 \x01(\x08\"J\n\x0c\x44\x65leteParams\x12\x12\n\nfolderpath\x18\x01 \x01(\t\x12\x11\n\trecursive\x18\x02 \x01(\x08\x12\x13\n\x0b\x64\x65leteafter\x18\x03 \x01(\x05\"#\n\x08Response\x12\x17\n\x0fresponsemessage\x18\x01 \x01(\t2\x8a\x01\n\x0cRelayCommand\x12>\n\x08\x44ownload\x12\x1a.pbcommands.DownloadParams\x1a\x14.pbcommands.Response\"\x00\x12:\n\x06\x44\x65lete\x12\x18.pbcommands.DeleteParams\x1a\x14.pbcommands.Response\"\x00\x62\x06proto3'
)




_FILE = _descriptor.Descriptor(
  name='File',
  full_name='pbcommands.File',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='name', full_name='pbcommands.File.name', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='cdn', full_name='pbcommands.File.cdn', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='hashsum', full_name='pbcommands.File.hashsum', index=2,
      number=3, type=9, cpp_type=9, label=1,
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
  serialized_start=30,
  serialized_end=80,
)


_CHANNEL = _descriptor.Descriptor(
  name='Channel',
  full_name='pbcommands.Channel',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='channelname', full_name='pbcommands.Channel.channelname', index=0,
      number=1, type=9, cpp_type=9, label=1,
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
  serialized_start=82,
  serialized_end=112,
)


_DOWNLOADPARAMS = _descriptor.Descriptor(
  name='DownloadParams',
  full_name='pbcommands.DownloadParams',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='folderpath', full_name='pbcommands.DownloadParams.folderpath', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='metadatafiles', full_name='pbcommands.DownloadParams.metadatafiles', index=1,
      number=2, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='bulkfiles', full_name='pbcommands.DownloadParams.bulkfiles', index=2,
      number=3, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='channels', full_name='pbcommands.DownloadParams.channels', index=3,
      number=4, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='deadline', full_name='pbcommands.DownloadParams.deadline', index=4,
      number=5, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='addtoexisting', full_name='pbcommands.DownloadParams.addtoexisting', index=5,
      number=6, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
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
  serialized_start=115,
  serialized_end=309,
)


_DELETEPARAMS = _descriptor.Descriptor(
  name='DeleteParams',
  full_name='pbcommands.DeleteParams',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='folderpath', full_name='pbcommands.DeleteParams.folderpath', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='recursive', full_name='pbcommands.DeleteParams.recursive', index=1,
      number=2, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='deleteafter', full_name='pbcommands.DeleteParams.deleteafter', index=2,
      number=3, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
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
  serialized_start=311,
  serialized_end=385,
)


_RESPONSE = _descriptor.Descriptor(
  name='Response',
  full_name='pbcommands.Response',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='responsemessage', full_name='pbcommands.Response.responsemessage', index=0,
      number=1, type=9, cpp_type=9, label=1,
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
  serialized_start=387,
  serialized_end=422,
)

_DOWNLOADPARAMS.fields_by_name['metadatafiles'].message_type = _FILE
_DOWNLOADPARAMS.fields_by_name['bulkfiles'].message_type = _FILE
_DOWNLOADPARAMS.fields_by_name['channels'].message_type = _CHANNEL
DESCRIPTOR.message_types_by_name['File'] = _FILE
DESCRIPTOR.message_types_by_name['Channel'] = _CHANNEL
DESCRIPTOR.message_types_by_name['DownloadParams'] = _DOWNLOADPARAMS
DESCRIPTOR.message_types_by_name['DeleteParams'] = _DELETEPARAMS
DESCRIPTOR.message_types_by_name['Response'] = _RESPONSE
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

File = _reflection.GeneratedProtocolMessageType('File', (_message.Message,), {
  'DESCRIPTOR' : _FILE,
  '__module__' : 'commands_pb2'
  # @@protoc_insertion_point(class_scope:pbcommands.File)
  })
_sym_db.RegisterMessage(File)

Channel = _reflection.GeneratedProtocolMessageType('Channel', (_message.Message,), {
  'DESCRIPTOR' : _CHANNEL,
  '__module__' : 'commands_pb2'
  # @@protoc_insertion_point(class_scope:pbcommands.Channel)
  })
_sym_db.RegisterMessage(Channel)

DownloadParams = _reflection.GeneratedProtocolMessageType('DownloadParams', (_message.Message,), {
  'DESCRIPTOR' : _DOWNLOADPARAMS,
  '__module__' : 'commands_pb2'
  # @@protoc_insertion_point(class_scope:pbcommands.DownloadParams)
  })
_sym_db.RegisterMessage(DownloadParams)

DeleteParams = _reflection.GeneratedProtocolMessageType('DeleteParams', (_message.Message,), {
  'DESCRIPTOR' : _DELETEPARAMS,
  '__module__' : 'commands_pb2'
  # @@protoc_insertion_point(class_scope:pbcommands.DeleteParams)
  })
_sym_db.RegisterMessage(DeleteParams)

Response = _reflection.GeneratedProtocolMessageType('Response', (_message.Message,), {
  'DESCRIPTOR' : _RESPONSE,
  '__module__' : 'commands_pb2'
  # @@protoc_insertion_point(class_scope:pbcommands.Response)
  })
_sym_db.RegisterMessage(Response)



_RELAYCOMMAND = _descriptor.ServiceDescriptor(
  name='RelayCommand',
  full_name='pbcommands.RelayCommand',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  serialized_start=425,
  serialized_end=563,
  methods=[
  _descriptor.MethodDescriptor(
    name='Download',
    full_name='pbcommands.RelayCommand.Download',
    index=0,
    containing_service=None,
    input_type=_DOWNLOADPARAMS,
    output_type=_RESPONSE,
    serialized_options=None,
  ),
  _descriptor.MethodDescriptor(
    name='Delete',
    full_name='pbcommands.RelayCommand.Delete',
    index=1,
    containing_service=None,
    input_type=_DELETEPARAMS,
    output_type=_RESPONSE,
    serialized_options=None,
  ),
])
_sym_db.RegisterServiceDescriptor(_RELAYCOMMAND)

DESCRIPTOR.services_by_name['RelayCommand'] = _RELAYCOMMAND

# @@protoc_insertion_point(module_scope)
