import type { Operands } from '@/api/operations'

export const StatusKind = {
  Idle: 'idle',
  Loading: 'loading',
  Success: 'success',
  Failure: 'failure',
} as const

/** What the display shows: nothing yet, a request in flight, a result, or an error. */
export type Status =
  | { kind: typeof StatusKind.Idle }
  | { kind: typeof StatusKind.Loading }
  | { kind: typeof StatusKind.Success; operands: Operands; result: number }
  | { kind: typeof StatusKind.Failure; code: string; requestId: string | null }

export const IDLE: Status = { kind: StatusKind.Idle }
