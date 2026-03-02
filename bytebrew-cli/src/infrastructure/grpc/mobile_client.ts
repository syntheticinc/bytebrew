// gRPC client for MobileService
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';
import fs from 'fs';
import os from 'os';
import { getLogger } from '../../lib/logger.js';

// Embedded proto file contents for standalone binary support
const MOBILE_SERVICE_PROTO = `syntax = "proto3";
package bytebrew.v1;
service MobileService {
  rpc GeneratePairingToken(GeneratePairingTokenRequest) returns (GeneratePairingTokenResponse);
  rpc Pair(PairRequest) returns (PairResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc SubscribeSession(SubscribeSessionRequest) returns (stream SessionEvent);
  rpc SendCommand(SendCommandRequest) returns (SendCommandResponse);
  rpc ListDevices(ListDevicesRequest) returns (ListDevicesResponse);
  rpc RevokeDevice(RevokeDeviceRequest) returns (RevokeDeviceResponse);
  rpc Ping(MobilePingRequest) returns (MobilePongResponse);
  rpc WaitForPairing(WaitForPairingRequest) returns (stream WaitForPairingEvent);
}
message GeneratePairingTokenRequest {}
message GeneratePairingTokenResponse {
  string short_code = 1;
  string token = 2;
  int64 expires_at = 3;
  string server_name = 4;
  string server_id = 5;
  int32 server_port = 6;
  bytes server_public_key = 7;
}
message PairRequest {
  string pairing_token = 1;
  string device_name = 2;
}
message PairResponse {
  string device_id = 1;
  string device_token = 2;
  string server_name = 3;
  string server_id = 4;
}
message ListSessionsRequest {}
message ListSessionsResponse {
  repeated MobileSession sessions = 1;
  string server_name = 2;
  string server_id = 3;
}
message MobileSession {
  string session_id = 1;
  string project_key = 2;
  string project_root = 3;
  SessionState status = 4;
  string current_task = 5;
  int64 started_at = 6;
  int64 last_activity_at = 7;
  bool has_ask_user = 8;
  string platform = 9;
}
enum SessionState {
  SESSION_STATE_UNSPECIFIED = 0;
  SESSION_STATE_ACTIVE = 1;
  SESSION_STATE_IDLE = 2;
  SESSION_STATE_NEEDS_ATTENTION = 3;
  SESSION_STATE_COMPLETED = 4;
  SESSION_STATE_FAILED = 5;
}
message SubscribeSessionRequest {
  string session_id = 1;
  string last_event_id = 2;
}
message SessionEvent {
  string event_id = 1;
  string session_id = 2;
  SessionEventType type = 3;
  int64 timestamp = 4;
  string agent_id = 5;
  int32 step = 6;
  oneof payload {
    AgentMessageEvent agent_message = 10;
    ToolCallStartEvent tool_call_start = 11;
    ToolCallEndEvent tool_call_end = 12;
    ReasoningEvent reasoning = 13;
    AskUserEvent ask_user = 14;
    PlanEvent plan = 15;
    SessionStatusEvent session_status = 16;
    ErrorEvent error_event = 17;
  }
}
enum SessionEventType {
  SESSION_EVENT_TYPE_UNSPECIFIED = 0;
  SESSION_EVENT_TYPE_AGENT_MESSAGE = 1;
  SESSION_EVENT_TYPE_TOOL_CALL_START = 2;
  SESSION_EVENT_TYPE_TOOL_CALL_END = 3;
  SESSION_EVENT_TYPE_REASONING = 4;
  SESSION_EVENT_TYPE_ASK_USER = 5;
  SESSION_EVENT_TYPE_PLAN_UPDATE = 6;
  SESSION_EVENT_TYPE_SESSION_STATUS = 7;
  SESSION_EVENT_TYPE_ERROR = 8;
  SESSION_EVENT_TYPE_ANSWER_CHUNK = 9;
}
message AgentMessageEvent {
  string content = 1;
  bool is_complete = 2;
}
message ToolCallStartEvent {
  string call_id = 1;
  string tool_name = 2;
  map<string, string> arguments = 3;
}
message ToolCallEndEvent {
  string call_id = 1;
  string tool_name = 2;
  string result_summary = 3;
  bool has_error = 4;
}
message ReasoningEvent {
  string content = 1;
  bool is_complete = 2;
}
message AskUserEvent {
  string question = 1;
  repeated string options = 2;
  bool is_answered = 3;
}
message PlanEvent {
  string plan_name = 1;
  repeated PlanStepEvent steps = 2;
}
message PlanStepEvent {
  string title = 1;
  PlanStepStatus status = 2;
}
enum PlanStepStatus {
  PLAN_STEP_STATUS_UNSPECIFIED = 0;
  PLAN_STEP_STATUS_PENDING = 1;
  PLAN_STEP_STATUS_IN_PROGRESS = 2;
  PLAN_STEP_STATUS_COMPLETED = 3;
  PLAN_STEP_STATUS_FAILED = 4;
}
message SessionStatusEvent {
  SessionState state = 1;
  string message = 2;
}
message ErrorEvent {
  string code = 1;
  string message = 2;
}
message SendCommandRequest {
  string session_id = 1;
  oneof command {
    string new_task = 2;
    AskUserResponse ask_user_reply = 3;
    bool cancel = 4;
  }
}
message AskUserResponse {
  string question = 1;
  string answer = 2;
}
message SendCommandResponse {
  bool success = 1;
  string error_message = 2;
}
message ListDevicesRequest {}
message ListDevicesResponse {
  repeated PairedDevice devices = 1;
}
message PairedDevice {
  string device_id = 1;
  string device_name = 2;
  int64 paired_at = 3;
  int64 last_seen_at = 4;
}
message RevokeDeviceRequest {
  string device_id = 1;
}
message RevokeDeviceResponse {
  bool success = 1;
}
message MobilePingRequest {
  int64 timestamp = 1;
}
message MobilePongResponse {
  int64 timestamp = 1;
  string server_name = 2;
  string server_id = 3;
}
message WaitForPairingRequest {
  string token = 1;
}
message WaitForPairingEvent {
  string device_name = 1;
  string device_id = 2;
}`;

// Lazy-loaded proto
let mobileProto: grpc.GrpcObject | null = null;

/**
 * Ensures the mobile proto file exists on disk (needed by @grpc/proto-loader).
 * Writes embedded proto string to a temp directory.
 */
function ensureMobileProtoFile(): string {
  const logger = getLogger();

  const tmpDir = path.join(os.tmpdir(), 'bytebrew-cli-proto');
  if (!fs.existsSync(tmpDir)) {
    fs.mkdirSync(tmpDir, { recursive: true });
  }

  const mobilePath = path.join(tmpDir, 'mobile_service.proto');
  fs.writeFileSync(mobilePath, MOBILE_SERVICE_PROTO, 'utf-8');

  logger.debug('Wrote mobile proto file to temp dir', { dir: tmpDir });
  return mobilePath;
}

function getMobileProto(): grpc.GrpcObject {
  if (mobileProto) return mobileProto;

  const logger = getLogger();

  const protoPath = ensureMobileProtoFile();
  const protoDir = path.dirname(protoPath);

  const packageDefinition = protoLoader.loadSync(protoPath, {
    keepCase: false,
    longs: String,
    enums: Number,
    defaults: true,
    oneofs: true,
    includeDirs: [protoDir],
  });

  const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
  const bytebrewPackage = protoDescriptor.bytebrew as grpc.GrpcObject;
  mobileProto = bytebrewPackage.v1 as grpc.GrpcObject;

  logger.debug('Mobile proto loaded successfully');
  return mobileProto;
}

// Types based on proto definitions
export interface GeneratePairingTokenResponse {
  shortCode: string;
  token: string;
  expiresAt: string; // int64 as string
  serverName: string;
  serverId: string;
  serverPort: number;
  serverPublicKey: Buffer; // bytes field → Buffer in gRPC-js
}

export interface PairedDevice {
  deviceId: string;
  deviceName: string;
  pairedAt: string; // int64 as string
  lastSeenAt: string; // int64 as string
}

export interface ListDevicesResponse {
  devices: PairedDevice[];
}

export interface RevokeDeviceResponse {
  success: boolean;
}

export interface PingResponse {
  timestamp: string; // int64 as string
  serverName: string;
  serverId: string;
}

export interface WaitForPairingEvent {
  deviceName: string;
  deviceId: string;
}

// Service client interface - grpc-js generates client methods at runtime
interface MobileServiceMethods {
  waitForReady(deadline: number, callback: (error?: Error) => void): void;
  generatePairingToken(
    request: Record<string, never>,
    callback: (err: grpc.ServiceError | null, response: GeneratePairingTokenResponse) => void,
  ): void;
  listDevices(
    request: Record<string, never>,
    callback: (err: grpc.ServiceError | null, response: ListDevicesResponse) => void,
  ): void;
  revokeDevice(
    request: { deviceId: string },
    callback: (err: grpc.ServiceError | null, response: RevokeDeviceResponse) => void,
  ): void;
  ping(
    request: { timestamp: string },
    callback: (err: grpc.ServiceError | null, response: PingResponse) => void,
  ): void;
  waitForPairing(
    request: { token: string },
  ): grpc.ClientReadableStream<WaitForPairingEvent>;
  close(): void;
}

export class MobileServiceClient {
  private client: MobileServiceMethods;
  private address: string;

  constructor(address: string) {
    this.address = address;
    const proto = getMobileProto();
    const MobileService = proto.MobileService as grpc.ServiceClientConstructor;
    // gRPC dynamically creates methods at runtime from proto definitions
    this.client = new MobileService(
      address,
      grpc.credentials.createInsecure(),
    ) as unknown as MobileServiceMethods;
  }

  /**
   * Wait for the channel to be ready
   */
  async waitForReady(timeoutMs: number = 5000): Promise<void> {
    return new Promise((resolve, reject) => {
      const deadline = Date.now() + timeoutMs;
      this.client.waitForReady(deadline, (error: Error | undefined) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });
  }

  /**
   * Generate a pairing token for mobile device pairing.
   * Called from localhost CLI — no device-token auth required.
   */
  async generatePairingToken(): Promise<GeneratePairingTokenResponse> {
    return new Promise((resolve, reject) => {
      this.client.generatePairingToken({}, (err: grpc.ServiceError | null, response: GeneratePairingTokenResponse) => {
        if (err) reject(err);
        else resolve(response);
      });
    });
  }

  /**
   * List all paired mobile devices.
   */
  async listDevices(): Promise<ListDevicesResponse> {
    return new Promise((resolve, reject) => {
      this.client.listDevices({}, (err: grpc.ServiceError | null, response: ListDevicesResponse) => {
        if (err) reject(err);
        else resolve(response);
      });
    });
  }

  /**
   * Revoke a paired device's access.
   */
  async revokeDevice(deviceId: string): Promise<RevokeDeviceResponse> {
    return new Promise((resolve, reject) => {
      this.client.revokeDevice({ deviceId }, (err: grpc.ServiceError | null, response: RevokeDeviceResponse) => {
        if (err) reject(err);
        else resolve(response);
      });
    });
  }

  /**
   * Ping the server to check connectivity.
   */
  async ping(): Promise<PingResponse> {
    return new Promise((resolve, reject) => {
      const timestamp = String(Date.now());
      this.client.ping({ timestamp }, (err: grpc.ServiceError | null, response: PingResponse) => {
        if (err) reject(err);
        else resolve(response);
      });
    });
  }

  /**
   * Wait for a pairing token to be consumed by a mobile device.
   * Returns a promise that resolves when pairing succeeds, or rejects on error/timeout.
   */
  waitForPairing(token: string, timeoutMs: number = 300000): Promise<WaitForPairingEvent> {
    return new Promise((resolve, reject) => {
      const stream = this.client.waitForPairing({ token });

      const timer = setTimeout(() => {
        stream.cancel();
        reject(new Error('Pairing timed out. Please run mobile-pair again.'));
      }, timeoutMs);

      stream.on('data', (event: WaitForPairingEvent) => {
        clearTimeout(timer);
        resolve(event);
      });

      stream.on('error', (err: grpc.ServiceError) => {
        clearTimeout(timer);
        // CANCELLED = client timeout or ctrl+c, not a real error
        if (err.code === grpc.status.CANCELLED) {
          return; // already rejected by timer
        }
        reject(err);
      });

      stream.on('end', () => {
        clearTimeout(timer);
        // If we get 'end' without 'data', token expired or was not found
      });
    });
  }

  /**
   * Close the client connection.
   */
  close(): void {
    this.client.close();
  }

  /**
   * Get the server address.
   */
  getAddress(): string {
    return this.address;
  }
}
