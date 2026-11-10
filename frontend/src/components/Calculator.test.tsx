import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { calculate, type CalculationOutcome } from '../api/client'
import { Calculator } from './Calculator'

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>()
  return { ...actual, calculate: vi.fn<typeof actual.calculate>() }
})

const calculateMock = vi.mocked(calculate)

function setup() {
  const user = userEvent.setup()
  render(<Calculator />)
  return { user, display: screen.getByRole('status') }
}

beforeEach(() => {
  calculateMock.mockReset()
})

describe('Calculator', () => {
  it('shows the result of a calculation', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 7 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '3')
    await user.type(screen.getByLabelText('Second number'), '4')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(await screen.findByText('7')).toBeInTheDocument()
    expect(display).toHaveTextContent('3 + 4 =7')
    expect(calculateMock).toHaveBeenCalledWith('add', { leftOperand: 3, rightOperand: 4 }, expect.any(AbortSignal))
  })

  it('continues from the result like a pocket calculator', async () => {
    calculateMock
      .mockResolvedValueOnce({ kind: 'success', result: 27 })
      .mockResolvedValueOnce({ kind: 'success', result: 54 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '15')
    await user.type(screen.getByLabelText('Second number'), '12')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await waitFor(() => expect(display).toHaveTextContent('15 + 12 =27'))

    expect(screen.getByLabelText('First number')).toHaveValue('27')
    expect(screen.getByLabelText('Second number')).toHaveValue('')
    expect(screen.getByLabelText('Second number')).toHaveFocus()

    await user.click(screen.getByRole('radio', { name: 'Multiply' }))
    expect(display).toHaveTextContent('15 + 12 =27')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => expect(display).toHaveTextContent('27 × 2 =54'))
    expect(calculateMock).toHaveBeenLastCalledWith(
      'multiply',
      { leftOperand: 27, rightOperand: 2 },
      expect.any(AbortSignal),
    )
    expect(screen.getByLabelText('First number')).toHaveValue('54')
  })

  it('calculates when "=" is typed in a number field', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 27 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '15')
    await user.type(screen.getByLabelText('Second number'), '12=')

    await waitFor(() => expect(display).toHaveTextContent('15 + 12 =27'))
    expect(calculateMock).toHaveBeenCalledWith('add', { leftOperand: 15, rightOperand: 12 }, expect.any(AbortSignal))
  })

  it('carries a square root into the single input', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 3 })
    const { user, display } = setup()

    await user.click(screen.getByRole('radio', { name: 'Square root' }))
    await user.type(screen.getByLabelText('Number'), '9=')

    await waitFor(() => expect(display).toHaveTextContent('√9 =3'))
    expect(screen.getByLabelText('Number')).toHaveValue('3')
    expect(screen.getByLabelText('Number')).toHaveFocus()
  })

  it('keeps the inputs when the calculation fails', async () => {
    calculateMock.mockResolvedValue({ kind: 'failure', code: 'error.calculation.division.by.zero', requestId: 'R1' })
    const { user, display } = setup()

    await user.click(screen.getByRole('radio', { name: 'Divide' }))
    await user.type(screen.getByLabelText('Dividend'), '1')
    await user.type(screen.getByLabelText('Divisor'), '0=')

    await waitFor(() => expect(display).toHaveTextContent('You can’t divide by zero.'))
    expect(screen.getByLabelText('Dividend')).toHaveValue('1')
    expect(screen.getByLabelText('Divisor')).toHaveValue('0')
  })

  it('submits with the Enter key and sends the chosen precision', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 0.33 })
    const { user } = setup()

    await user.click(screen.getByRole('radio', { name: 'Divide' }))
    await user.selectOptions(screen.getByLabelText('Decimal places'), '2 places')
    await user.type(screen.getByLabelText('Dividend'), '1')
    await user.type(screen.getByLabelText('Divisor'), '3{Enter}')

    expect(await screen.findByText('0.33')).toBeInTheDocument()
    expect(calculateMock).toHaveBeenCalledWith(
      'divide',
      { leftOperand: 1, rightOperand: 3, precision: 2 },
      expect.any(AbortSignal),
    )
  })

  it('validates on the client and focuses the first invalid field', async () => {
    const { user } = setup()

    await user.type(screen.getByLabelText('Second number'), 'abc')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    const left = screen.getByLabelText('First number')
    const right = screen.getByLabelText('Second number')
    expect(left).toHaveAttribute('aria-invalid', 'true')
    expect(left).toHaveAccessibleDescription('Enter a number.')
    expect(right).toHaveAttribute('aria-invalid', 'true')
    expect(right).toHaveAccessibleDescription('Enter a valid number, like 12.5 or -3e4.')
    expect(left).toHaveFocus()
    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('clears a field error when the field is edited', async () => {
    const { user } = setup()

    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await user.type(screen.getByLabelText('First number'), '1')

    expect(screen.getByLabelText('First number')).not.toHaveAttribute('aria-invalid')
    expect(screen.getByLabelText('Second number')).toHaveAttribute('aria-invalid', 'true')
  })

  it('shows field errors from the API next to their inputs', async () => {
    calculateMock.mockResolvedValue({
      kind: 'fieldErrors',
      errors: [{ field: 'rightOperand', code: 'error.field.out.of.range' }],
      requestId: 'R1',
    })
    const { user } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    const right = screen.getByLabelText('Second number')
    await waitFor(() => expect(right).toHaveAttribute('aria-invalid', 'true'))
    expect(right).toHaveAccessibleDescription('This number is too large.')
    expect(right).toHaveFocus()
  })

  it('reports a field error the form has no input for as a form error', async () => {
    calculateMock.mockResolvedValue({
      kind: 'fieldErrors',
      errors: [{ field: 'zzz', code: 'error.field.unknown' }],
      requestId: 'R7',
    })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => expect(display).toHaveTextContent('The server did not expect this value.'))
    expect(display).toHaveTextContent('Reference: R7')
  })

  const failures: Array<{ name: string; outcome: CalculationOutcome; want: string }> = [
    {
      name: 'division by zero',
      outcome: { kind: 'failure', code: 'error.calculation.division.by.zero', requestId: 'R1' },
      want: 'You can’t divide by zero.',
    },
    {
      name: 'overflow',
      outcome: { kind: 'failure', code: 'error.calculation.result.overflow', requestId: 'R1' },
      want: 'The result is too large to represent.',
    },
    {
      name: 'operation not on the server yet',
      outcome: { kind: 'failure', code: 'error.request.not.found', requestId: 'R1' },
      want: 'This operation is not available on the server yet.',
    },
    {
      name: 'network failure',
      outcome: { kind: 'failure', code: 'error.client.network', requestId: null },
      want: 'Can’t reach the server. Check your connection and try again.',
    },
    {
      name: 'unknown code',
      outcome: { kind: 'failure', code: 'error.brand.new', requestId: 'R1' },
      want: 'Something went wrong. Please try again.',
    },
  ]

  it.each(failures)('shows the message for $name', async ({ outcome, want }) => {
    calculateMock.mockResolvedValue(outcome)
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '0')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => expect(display).toHaveTextContent(want))
  })

  it('shows the request id of a failed request', async () => {
    calculateMock.mockResolvedValue({ kind: 'failure', code: 'error.internal', requestId: 'ABC-123' })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => expect(display).toHaveTextContent('Reference: ABC-123'))
  })

  it('asks for a single number for square root', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 3 })
    const { user, display } = setup()

    await user.click(screen.getByRole('radio', { name: 'Square root' }))
    expect(screen.queryByLabelText('Second number')).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('Number'), '9')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    await waitFor(() => expect(display).toHaveTextContent('√9 =3'))
    expect(calculateMock).toHaveBeenCalledWith('sqrt', { operand: 9 }, expect.any(AbortSignal))
  })

  it('shows only the answer to the latest request', async () => {
    let resolveFirst: (outcome: CalculationOutcome) => void = () => {}
    calculateMock
      .mockImplementationOnce(() => new Promise((resolve) => (resolveFirst = resolve)))
      .mockResolvedValueOnce({ kind: 'success', result: 20 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '1')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    expect(display).toHaveTextContent('Calculating…')
    await user.clear(screen.getByLabelText('First number'))
    await user.type(screen.getByLabelText('First number'), '19')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await waitFor(() => expect(display).toHaveTextContent('19 + 1 =20'))

    // Let the stale answer settle completely before asserting it was ignored.
    await act(async () => {
      resolveFirst({ kind: 'success', result: 2 })
    })

    expect(display).toHaveTextContent('19 + 1 =20')
    expect(calculateMock.mock.calls[0]?.[2]?.aborted).toBe(true)
  })

  it('drops the request in flight when the operation changes', async () => {
    let resolveFirst: (outcome: CalculationOutcome) => void = () => {}
    calculateMock.mockImplementationOnce(() => new Promise((resolve) => (resolveFirst = resolve)))
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '1')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await user.click(screen.getByRole('radio', { name: 'Multiply' }))
    await act(async () => {
      resolveFirst({ kind: 'success', result: 2 })
    })

    expect(display).toHaveTextContent('Your result will appear here.')
    expect(calculateMock.mock.calls[0]?.[2]?.aborted).toBe(true)
  })

  it('aborts the request in flight on unmount', async () => {
    calculateMock.mockImplementationOnce(() => new Promise(() => {}))
    const user = userEvent.setup()
    const { unmount } = render(<Calculator />)

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '1')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    unmount()

    expect(calculateMock.mock.calls[0]?.[2]?.aborted).toBe(true)
  })

  it('keeps the previous result when client-side validation fails', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 3 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await waitFor(() => expect(display).toHaveTextContent('1 + 2 =3'))
    await user.clear(screen.getByLabelText('Second number'))
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    expect(screen.getByLabelText('Second number')).toHaveAttribute('aria-invalid', 'true')
    expect(display).toHaveTextContent('1 + 2 =3')
  })

  it('shows an API precision error on the decimal places select', async () => {
    calculateMock.mockResolvedValue({
      kind: 'fieldErrors',
      errors: [{ field: 'precision', code: 'error.field.out.of.range' }],
      requestId: 'R1',
    })
    const { user } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.selectOptions(screen.getByLabelText('Decimal places'), '3 places')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))

    const precision = screen.getByLabelText('Decimal places')
    await waitFor(() => expect(precision).toHaveAttribute('aria-invalid', 'true'))
    expect(precision).toHaveFocus()
  })

  it('clears the inputs, errors and result', async () => {
    calculateMock.mockResolvedValue({ kind: 'success', result: 3 })
    const { user, display } = setup()

    await user.type(screen.getByLabelText('First number'), '1')
    await user.type(screen.getByLabelText('Second number'), '2')
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
    await waitFor(() => expect(display).toHaveTextContent('1 + 2 =3'))
    await user.click(screen.getByRole('button', { name: 'Clear' }))

    expect(screen.getByLabelText('First number')).toHaveValue('')
    expect(screen.getByLabelText('Second number')).toHaveValue('')
    expect(display).toHaveTextContent('Your result will appear here.')
    expect(screen.getByLabelText('First number')).toHaveFocus()
  })

  it('labels the inputs for the selected operation', async () => {
    const { user } = setup()

    await user.click(screen.getByRole('radio', { name: 'Percentage' }))

    expect(screen.getByRole('radio', { name: 'Percentage' })).toBeChecked()
    expect(screen.getByLabelText('Percent')).toBeInTheDocument()
    expect(screen.getByLabelText('Of')).toBeInTheDocument()
  })
})
