import { useRef, type FormEvent, type KeyboardEvent } from 'react'
import { PRECISION_MAX, PRECISION_MIN } from '@/api/constants'
import { isBinary } from '@/api/operations'
import { InputField } from '@/form/form'
import { StatusKind } from '@/form/status'
import { useCalculator } from '@/hooks/useCalculator'
import { operationText, t } from '@/i18n'
import { cx } from '@/utils/dom'
import { FULL_PRECISION } from '@/utils/validation'
import styles from '@/components/calculator/Calculator.module.css'
import { Field } from '@/components/field/Field'
import { OperationPicker } from '@/components/operation-picker/OperationPicker'
import { ResultDisplay } from '@/components/result-display/ResultDisplay'

/** `KeyboardEvent.key` of the "=" key. */
const EQUALS_KEY = '='

const PRECISION_OPTIONS = Array.from({ length: PRECISION_MAX - PRECISION_MIN + 1 }, (_, i) => PRECISION_MIN + i)

type Control = HTMLInputElement | HTMLSelectElement

/** The calculator: a display, the operation picker, operand inputs and actions. */
export function Calculator() {
  const { operation, values, fieldErrors, status, setOperation, setValue, reset, submit } = useCalculator()
  const controls = useRef<Partial<Record<InputField, Control | null>>>({})
  const text = operationText(operation)
  // The second input's label, or null when the operation takes one operand.
  const rightLabel = isBinary(operation) ? operationText(operation).rightLabel : null
  const loading = status.kind === StatusKind.Loading

  const bind = (input: InputField) => (element: Control | null) => {
    controls.current[input] = element
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const next = await submit()
    if (next !== null) controls.current[next]?.focus()
  }

  // "=" is never part of a number, so in an operand it means "calculate", like Enter.
  function handleOperandKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key !== EQUALS_KEY) return
    event.preventDefault()
    event.currentTarget.form?.requestSubmit()
  }

  function handleClear() {
    reset()
    controls.current[InputField.Left]?.focus()
  }

  function operandField(input: InputField, label: string) {
    return (
      <Field label={label} error={fieldErrors[input]}>
        {(control) => (
          <input
            {...control}
            ref={bind(input)}
            type="text"
            autoComplete="off"
            autoCorrect="off"
            spellCheck={false}
            enterKeyHint="go"
            placeholder={t.form.operandPlaceholder}
            value={values[input]}
            onChange={(event) => setValue(input, event.target.value)}
            onKeyDown={handleOperandKeyDown}
          />
        )}
      </Field>
    )
  }

  return (
    <div className={styles.card}>
      <ResultDisplay status={status} />
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <OperationPicker value={operation} onChange={setOperation} />

        <div className={cx(styles.operands, rightLabel !== null && styles.binary)}>
          {operandField(InputField.Left, text.leftLabel)}
          {rightLabel !== null && (
            <>
              <span className={styles.symbol} aria-hidden="true">
                {text.symbol}
              </span>
              {operandField(InputField.Right, rightLabel)}
            </>
          )}
        </div>

        <div className={styles.footer}>
          <div className={styles.precision}>
            <Field label={t.form.precisionLabel} hint={t.form.precisionHint} error={fieldErrors[InputField.Precision]}>
              {(control) => (
                <select
                  {...control}
                  ref={bind(InputField.Precision)}
                  value={values[InputField.Precision]}
                  onChange={(event) => setValue(InputField.Precision, event.target.value)}
                >
                  <option value={FULL_PRECISION}>{t.form.precisionFull}</option>
                  {PRECISION_OPTIONS.map((places) => (
                    <option key={places} value={places.toString()}>
                      {t.form.precisionOption(places)}
                    </option>
                  ))}
                </select>
              )}
            </Field>
          </div>

          <div className={styles.actions}>
            <button type="button" className={cx(styles.button, styles.secondary)} onClick={handleClear}>
              {t.form.clear}
            </button>
            {/* Shows "=" like a calculator key. The hidden word makes the accessible
                name "= Calculate": it starts with the visible label, so a speech-control
                user can say "click equals" (WCAG 2.5.3 Label in Name). Don't move the
                name back into aria-label. The title tooltip explains "=" to mouse users;
                some screen readers also read it as the description, which is accepted. */}
            <button
              type="submit"
              className={cx(styles.button, styles.primary, styles.equals)}
              title={t.form.calculate}
              aria-busy={loading}
            >
              {t.form.equals} <span className={styles.visuallyHidden}>{t.form.calculate}</span>
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}
