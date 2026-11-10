import type { ErrorCode as ErrorCodeValue, FieldErrorCode as FieldErrorCodeValue } from './types'

/** Calculator operations offered by the UI, one backend endpoint each. */
export const Operation = {
  Add: 'add',
  Subtract: 'subtract',
  Multiply: 'multiply',
  Divide: 'divide',
  Power: 'power',
  Sqrt: 'sqrt',
  Percentage: 'percentage',
} as const
export type Operation = (typeof Operation)[keyof typeof Operation]

/** Operations in display order. */
export const OPERATIONS: readonly Operation[] = Object.values(Operation)

/** Endpoint of each operation (`POST`). */
export const ApiPath = {
  [Operation.Add]: '/api/v1/add',
  [Operation.Subtract]: '/api/v1/subtract',
  [Operation.Multiply]: '/api/v1/multiply',
  [Operation.Divide]: '/api/v1/divide',
  [Operation.Power]: '/api/v1/power',
  [Operation.Sqrt]: '/api/v1/sqrt',
  [Operation.Percentage]: '/api/v1/percentage',
} as const satisfies Record<Operation, string>

/** Swagger UI page served by the backend (proxied in dev and in Docker). */
export const SWAGGER_PATH = '/swagger/'

/** JSON property names of the request bodies, as reported in field errors. */
export const RequestField = {
  LeftOperand: 'leftOperand',
  RightOperand: 'rightOperand',
  Operand: 'operand',
  Precision: 'precision',
} as const
export type RequestField = (typeof RequestField)[keyof typeof RequestField]

/** Top-level error codes of the API (the spec's `ErrorCode` enum). */
export const ErrorCode = {
  InvalidJSON: 'error.request.invalid.json',
  NotFound: 'error.request.not.found',
  MethodNotAllowed: 'error.request.method.not.allowed',
  BodyTooLarge: 'error.request.body.too.large',
  UnsupportedMediaType: 'error.request.unsupported.media.type',
  DivisionByZero: 'error.calculation.division.by.zero',
  NegativeSquareRoot: 'error.calculation.negative.square.root',
  ResultOverflow: 'error.calculation.result.overflow',
  ResultUndefined: 'error.calculation.result.undefined',
  Internal: 'error.internal',
} as const satisfies Record<string, ErrorCodeValue>

/** Per-field error codes (the spec's `FieldErrorCode` enum), shared with client-side validation. */
export const FieldErrorCode = {
  Required: 'error.field.required',
  InvalidType: 'error.field.invalid.type',
  OutOfRange: 'error.field.out.of.range',
  Unknown: 'error.field.unknown',
} as const satisfies Record<string, FieldErrorCodeValue>

/** Failures detected by the client itself; never sent by the API. */
export const ClientErrorCode = {
  Network: 'error.client.network',
  Unavailable: 'error.client.unavailable',
  UnexpectedResponse: 'error.client.unexpected.response',
} as const
export type ClientErrorCode = (typeof ClientErrorCode)[keyof typeof ClientErrorCode]

/** Valid range of the optional `precision` field (decimal places). */
export const PRECISION_MIN = 0
export const PRECISION_MAX = 15

export const HttpMethod = { Post: 'POST' } as const
export const Header = {
  Accept: 'Accept',
  ContentType: 'Content-Type',
  RequestID: 'X-Request-ID',
} as const
export const MediaType = { JSON: 'application/json' } as const

/** `DOMException.name` of a request cancelled through an `AbortSignal`. */
export const ABORT_ERROR_NAME = 'AbortError'
