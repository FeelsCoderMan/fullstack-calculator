import { ClientErrorCode, ErrorCode, FieldErrorCode, Operation, PRECISION_MAX, PRECISION_MIN } from '@/api/constants'
import type { UnaryOperation } from '@/api/operations'
import type { ErrorCode as ApiErrorCode, FieldErrorCode as ApiFieldErrorCode } from '@/api/types'

/** How an operation is presented: its name, symbol and the label of its (first) input. */
interface BaseOperationText {
  name: string
  symbol: string
  leftLabel: string
}

/** Texts of a two-operand operation. */
export interface BinaryOperationText extends BaseOperationText {
  rightLabel: string
  /** The calculation as a person would write it, e.g. `3 + 4`. Operands are already formatted. */
  expression: (left: string, right: string) => string
}

/** Texts of a one-operand operation: no second input, no second operand. */
export interface UnaryOperationText extends BaseOperationText {
  /**
   * The calculation as a person would write it, e.g. `√9`. The operand is already
   * formatted. `right?: never` makes a two-argument call on an operation that may be
   * unary a compile error: narrow with `isBinary` first.
   */
  expression: (operand: string, right?: never) => string
}

/**
 * The texts each operation must have, by arity (`UNARY_OPERATIONS` in
 * `src/api/operations.ts`): a two-operand operation without `rightLabel` is a
 * compile error.
 */
export type OperationTexts = {
  [O in Operation]: O extends UnaryOperation ? UnaryOperationText : BinaryOperationText
}

/** English texts. Every user-facing string in the app comes from here. */
export const en = {
  app: {
    title: 'Calculator',
    tagline: 'Pick an operation, enter your numbers, and let the Go API do the math.',
    apiDocs: 'API documentation',
  },
  form: {
    operationLegend: 'Operation',
    precisionLabel: 'Decimal places',
    precisionFull: 'Full precision',
    precisionOption: (places: number) => (places === 1 ? '1 place' : `${places} places`),
    precisionHint: `Rounds the result half away from zero (${PRECISION_MIN}–${PRECISION_MAX}).`,
    operandPlaceholder: 'e.g. 12.5',
    calculate: 'Calculate',
    equals: '=',
    clear: 'Clear',
  },
  result: {
    heading: 'Result',
    idle: 'Your result will appear here.',
    loading: 'Calculating…',
    equals: '=',
    errorHeading: 'Could not calculate',
    requestId: (id: string) => `Reference: ${id}`,
  },
  operations: {
    [Operation.Add]: {
      name: 'Add',
      symbol: '+',
      leftLabel: 'First number',
      rightLabel: 'Second number',
      expression: (l, r) => `${l} + ${r}`,
    },
    [Operation.Subtract]: {
      name: 'Subtract',
      symbol: '−',
      leftLabel: 'Number',
      rightLabel: 'Amount to subtract',
      expression: (l, r) => `${l} − ${r}`,
    },
    [Operation.Multiply]: {
      name: 'Multiply',
      symbol: '×',
      leftLabel: 'First number',
      rightLabel: 'Second number',
      expression: (l, r) => `${l} × ${r}`,
    },
    [Operation.Divide]: {
      name: 'Divide',
      symbol: '÷',
      leftLabel: 'Dividend',
      rightLabel: 'Divisor',
      expression: (l, r) => `${l} ÷ ${r}`,
    },
    [Operation.Power]: {
      name: 'Power',
      symbol: 'xʸ',
      leftLabel: 'Base',
      rightLabel: 'Exponent',
      expression: (l, r) => `${l} ^ ${r}`,
    },
    [Operation.Sqrt]: {
      name: 'Square root',
      symbol: '√',
      leftLabel: 'Number',
      expression: (operand) => `√${operand}`,
    },
    [Operation.Percentage]: {
      name: 'Percentage',
      symbol: '%',
      leftLabel: 'Percent',
      rightLabel: 'Of',
      expression: (l, r) => `${l}% of ${r}`,
    },
  } satisfies OperationTexts,
  errors: {
    [ErrorCode.InvalidJSON]: 'The request could not be read. Please try again.',
    [ErrorCode.NotFound]: 'The server doesn’t support this request. Try reloading the page.',
    [ErrorCode.MethodNotAllowed]: 'The server does not accept this request.',
    [ErrorCode.BodyTooLarge]: 'The numbers are too long to send.',
    [ErrorCode.UnsupportedMediaType]: 'The server could not understand the request format.',
    [ErrorCode.DivisionByZero]: 'You can’t divide by zero.',
    [ErrorCode.ZeroNegativePower]: 'Zero can’t be raised to a negative power: that divides by zero.',
    [ErrorCode.NegativeSquareRoot]: 'A negative number has no real square root.',
    [ErrorCode.ResultOverflow]: 'The result is too large to represent.',
    [ErrorCode.ResultUndefined]: 'The result is not a real number.',
    [ErrorCode.Internal]: 'Something went wrong on the server. Please try again.',
    [ClientErrorCode.Network]: 'Can’t reach the server. Check your connection and try again.',
    [ClientErrorCode.Unavailable]: 'The calculator service is unavailable. Please try again shortly.',
    [ClientErrorCode.UnexpectedResponse]: 'The server sent an unexpected response.',
  } satisfies Record<ApiErrorCode | ClientErrorCode, string>,
  fieldErrors: {
    [FieldErrorCode.Required]: 'Enter a number.',
    [FieldErrorCode.InvalidType]: 'Enter a valid number, like 12.5 or -3e4.',
    [FieldErrorCode.OutOfRange]: 'This number is too large.',
    [FieldErrorCode.Unknown]: 'The server did not expect this value.',
  } satisfies Record<ApiFieldErrorCode, string>,
  /** Shown for any code this client has no text for. */
  genericError: 'Something went wrong. Please try again.',
}
