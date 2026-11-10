import { ClientErrorCode, ErrorCode, FieldErrorCode, Operation, PRECISION_MAX, PRECISION_MIN } from '../api/constants'
import type { ErrorCode as ApiErrorCode, FieldErrorCode as ApiFieldErrorCode } from '../api/types'

/** How an operation is presented: its name, symbol, input labels and written-out expression. */
export interface OperationText {
  name: string
  symbol: string
  leftLabel: string
  /** Absent for one-operand operations. */
  rightLabel?: string
  /**
   * The calculation as a person would write it, e.g. `3 + 4`. Operands are already
   * formatted; `right` is absent for one-operand operations.
   */
  expression: (left: string, right?: string) => string
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
      expression: (l) => `√${l}`,
    },
    [Operation.Percentage]: {
      name: 'Percentage',
      symbol: '%',
      leftLabel: 'Percent',
      rightLabel: 'Of',
      expression: (l, r) => `${l}% of ${r}`,
    },
  } satisfies Record<Operation, OperationText>,
  errors: {
    [ErrorCode.InvalidJSON]: 'The request could not be read. Please try again.',
    [ErrorCode.NotFound]: 'This operation is not available on the server yet.',
    [ErrorCode.MethodNotAllowed]: 'The server does not accept this request.',
    [ErrorCode.BodyTooLarge]: 'The numbers are too long to send.',
    [ErrorCode.UnsupportedMediaType]: 'The server could not understand the request format.',
    [ErrorCode.DivisionByZero]: 'You can’t divide by zero.',
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
