// Shared gRPC constants

import type { StreamResponse } from '../domain/ports/IStreamGateway.js';

/**
 * Response type enum mapping (numeric to string)
 * Maps protocol buffer enum values to string types
 */
export const ResponseTypeMap: Record<number, StreamResponse['type']> = {
  0: 'UNSPECIFIED',
  1: 'ANSWER',
  2: 'REASONING',
  3: 'TOOL_CALL',
  4: 'TOOL_RESULT',
  5: 'ANSWER_CHUNK',
  6: 'ERROR',
};

/**
 * Convert numeric response type to string
 */
export function getResponseType(numericType: number): StreamResponse['type'] {
  return ResponseTypeMap[numericType] || 'UNSPECIFIED';
}
