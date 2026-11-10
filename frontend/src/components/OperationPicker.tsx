import { useId } from 'react'
import { OPERATIONS, type Operation } from '../api/constants'
import { operationText, t } from '../i18n'
import { cx } from '../utils/dom'
import styles from './OperationPicker.module.css'

interface OperationPickerProps {
  value: Operation
  onChange: (operation: Operation) => void
}

/** A radio group of operations, styled as tiles. Arrow keys move between them. */
export function OperationPicker({ value, onChange }: OperationPickerProps) {
  const name = useId()
  return (
    <fieldset className={styles.fieldset}>
      <legend className={styles.legend}>{t.form.operationLegend}</legend>
      <div className={styles.grid}>
        {OPERATIONS.map((operation) => {
          const text = operationText(operation)
          const selected = operation === value
          return (
            <label key={operation} className={cx(styles.tile, selected && styles.selected)}>
              <input
                className={styles.radio}
                type="radio"
                name={name}
                value={operation}
                checked={selected}
                onChange={() => onChange(operation)}
              />
              <span className={styles.symbol} aria-hidden="true">
                {text.symbol}
              </span>
              <span className={styles.name}>{text.name}</span>
            </label>
          )
        })}
      </div>
    </fieldset>
  )
}
