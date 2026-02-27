// gRPC client for FlowService
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';
import fs from 'fs';
import os from 'os';
import { fileURLToPath } from 'url';
import { getLogger } from '../../lib/logger.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Embedded proto file contents for standalone binary support
const COMMON_PROTO = `syntax = "proto3";
package bytebrew.v1;
message Error {
  string code = 1;
  string message = 2;
  map<string, string> metadata = 3;
}
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_SUCCESS = 1;
  STATUS_IN_PROGRESS = 2;
  STATUS_FAILED = 3;
  STATUS_CANCELLED = 4;
}`;

const FLOW_SERVICE_PROTO = `syntax = "proto3";
package bytebrew.v1;
import "common.proto";
service FlowService {
  rpc ExecuteFlow(stream FlowRequest) returns (stream FlowResponse);
}
message FlowRequest {
  string session_id = 1;
  string user_id = 2;
  string project_key = 3;
  string task = 4;
  map<string, string> context = 5;
  bool is_first_message = 6;
  bool cancel = 7;
  PingRequest ping = 8;
  ToolResult tool_result = 9;
}
message PingRequest {
  int64 timestamp = 1;
}
message ToolResult {
  string call_id = 1;
  string result = 2;
  Error error = 3;
  repeated SubResult sub_results = 4;
  string summary = 5;
}
message SubQuery {
  string type = 1;
  string query = 2;
  int32 limit = 3;
}
message SubResult {
  string type = 1;
  string result = 2;
  int32 count = 3;
  string error = 4;
}
message FlowResponse {
  string session_id = 1;
  ResponseType type = 2;
  string content = 3;
  ToolCall tool_call = 4;
  ThoughtStep thought = 5;
  ReasoningContent reasoning = 6;
  Error error = 7;
  bool is_final = 8;
  PongResponse pong = 9;
  int32 step = 10;
  ToolResult tool_result = 11;
  string agent_id = 12;
}
message PongResponse {
  string status = 1;
  int64 timestamp = 2;
}
enum ResponseType {
  RESPONSE_TYPE_UNSPECIFIED = 0;
  RESPONSE_TYPE_ANSWER = 1;
  RESPONSE_TYPE_REASONING = 2;
  RESPONSE_TYPE_TOOL_CALL = 3;
  RESPONSE_TYPE_TOOL_RESULT = 4;
  RESPONSE_TYPE_ANSWER_CHUNK = 5;
  RESPONSE_TYPE_ERROR = 6;
}
message ToolCall {
  string tool_name = 1;
  map<string, string> arguments = 2;
  string call_id = 3;
  repeated SubQuery sub_queries = 4;
}
message ThoughtStep {
  string content = 1;
  string action = 2;
}
message ReasoningContent {
  string thinking = 1;
  bool is_complete = 2;
}`;

// Lazy-loaded proto
let bytebrewProto: grpc.GrpcObject | null = null;

/**
 * Ensures proto files exist on disk (needed by @grpc/proto-loader).
 * First tries to find them in standard locations; if not found,
 * writes embedded proto strings to a temp directory.
 */
function ensureProtoFiles(): string {
  const logger = getLogger();

  // Try standard locations first
  const possiblePaths = [
    path.resolve(__dirname, '../../proto/flow_service.proto'),
    path.resolve(process.cwd(), 'proto/flow_service.proto'),
    path.resolve(process.cwd(), 'bytebrew-cli/proto/flow_service.proto'),
  ];

  for (const p of possiblePaths) {
    try {
      fs.accessSync(p);
      logger.debug('Found proto file on disk', { path: p });
      return p;
    } catch {
      // Not found, try next
    }
  }

  // No proto files on disk — write embedded protos to temp dir
  const tmpDir = path.join(os.tmpdir(), 'bytebrew-cli-proto');
  if (!fs.existsSync(tmpDir)) {
    fs.mkdirSync(tmpDir, { recursive: true });
  }

  const commonPath = path.join(tmpDir, 'common.proto');
  const flowPath = path.join(tmpDir, 'flow_service.proto');

  fs.writeFileSync(commonPath, COMMON_PROTO, 'utf-8');
  fs.writeFileSync(flowPath, FLOW_SERVICE_PROTO, 'utf-8');

  logger.debug('Wrote embedded proto files to temp dir', { dir: tmpDir });
  return flowPath;
}

function getProto(): grpc.GrpcObject {
  if (bytebrewProto) return bytebrewProto;

  const logger = getLogger();

  const protoPath = ensureProtoFiles();
  const protoDir = path.dirname(protoPath);

  // Load proto definition
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
  bytebrewProto = bytebrewPackage.v1 as grpc.GrpcObject;

  logger.debug('Proto loaded successfully');
  return bytebrewProto;
}

// Types based on proto definitions
// SubResult for grouped tool operations
export interface SubResult {
  type: string;    // "vector" | "grep" | "symbol"
  result: string;  // Result data (text format)
  count: number;   // Number of matches found
  error?: string;  // Error message if failed
}

export interface FlowRequest {
  sessionId: string;
  userId: string;
  projectKey: string;
  task?: string;
  context?: Record<string, string>;
  isFirstMessage?: boolean;
  cancel?: boolean;
  ping?: { timestamp: string };
  toolResult?: {
    callId: string;
    result: string;
    error?: { code: string; message: string };
    subResults?: SubResult[];  // Results from sub-queries
  };
}

// SubQuery for grouped tool operations
export interface SubQuery {
  type: string;    // "vector" | "grep" | "symbol"
  query: string;   // Search query or pattern
  limit: number;   // Max results
}

export interface FlowResponse {
  sessionId: string;
  type: number;
  content: string;
  toolCall?: {
    toolName: string;
    arguments: Record<string, string>;
    callId: string;
    subQueries?: SubQuery[];  // Sub-queries for grouped operations
  };
  thought?: {
    content: string;
    action: string;
  };
  reasoning?: {
    thinking: string;
    isComplete: boolean;
  };
  error?: {
    code: string;
    message: string;
  };
  toolResult?: {
    callId: string;
    result: string;
    error?: {
      message: string;
    };
    summary?: string;  // Server-computed display summary
  };
  isFinal: boolean;
  pong?: {
    status: string;
    timestamp: string;
  };
  step: number;
  agentId?: string;
}

export type FlowStream = grpc.ClientDuplexStream<FlowRequest, FlowResponse>;

// Service client interface - grpc-js generates client methods at runtime
interface FlowServiceMethods {
  waitForReady(deadline: number, callback: (error?: Error) => void): void;
  executeFlow(metadata?: grpc.Metadata): FlowStream;
  getChannel(): grpc.Channel;
  close(): void;
}

export class FlowServiceClient {
  private client: FlowServiceMethods;
  private address: string;

  constructor(address: string) {
    this.address = address;
    const proto = getProto();
    const FlowService = proto.FlowService as grpc.ServiceClientConstructor;
    // gRPC dynamically creates methods at runtime from proto definitions
    this.client = new FlowService(
      address,
      grpc.credentials.createInsecure()
    ) as unknown as FlowServiceMethods;
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
   * Creates a bidirectional stream for ExecuteFlow.
   * Optionally attaches gRPC metadata (e.g. client version).
   */
  createStream(metadata?: grpc.Metadata): FlowStream {
    return this.client.executeFlow(metadata) as FlowStream;
  }

  /**
   * Check if the channel is connected
   */
  isConnected(): boolean {
    const channel = this.client.getChannel();
    const state = channel.getConnectivityState(false);
    return state === grpc.connectivityState.READY;
  }

  /**
   * Close the client connection
   */
  close(): void {
    this.client.close();
  }

  /**
   * Get the server address
   */
  getAddress(): string {
    return this.address;
  }
}
