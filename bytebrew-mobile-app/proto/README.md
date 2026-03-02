# Proto Code Generation

## Prerequisites

Install the Dart protoc plugin:

```bash
dart pub global activate protoc_plugin
```

Ensure `protoc` (protobuf compiler) is installed:
- macOS: `brew install protobuf`
- Linux: `apt install protobuf-compiler`
- Windows: Download from https://github.com/protocolbuffers/protobuf/releases

## Generate Dart code

From the `bytebrew-mobile-app` directory:

```bash
protoc \
  --proto_path=proto \
  --dart_out=grpc:lib/core/infrastructure/grpc/generated \
  proto/mobile_service.proto
```

This generates:
- `mobile_service.pb.dart` - Message classes
- `mobile_service.pbenum.dart` - Enum classes
- `mobile_service.pbgrpc.dart` - gRPC client/server stubs
- `mobile_service.pbjson.dart` - JSON serialization

## After generation

1. Replace `MobileServiceClient` in `mobile_service_client.dart` with the
   generated `MobileServiceClient` from `mobile_service.pbgrpc.dart`
2. Replace DTO classes with generated protobuf message classes
3. Remove the placeholder `_encodeJson` / `_decodeJson` methods

## Proto source

The proto file is copied from:
`bytebrew-srv/api/proto/mobile_service.proto`

Keep in sync when the server proto changes.
