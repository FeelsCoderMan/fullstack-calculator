/** Separator of space-separated token lists (`class`, `aria-describedby`). */
const TOKEN_SEPARATOR = ' '
const HINT_ID_SUFFIX = '-hint'
const ERROR_ID_SUFFIX = '-error'

type Token = string | false | null | undefined

function joinTokens(tokens: Token[]): string {
  return tokens.filter(Boolean).join(TOKEN_SEPARATOR)
}

/** Joins the truthy class names, e.g. `cx(styles.tile, selected && styles.selected)`. */
export function cx(...names: Token[]): string {
  return joinTokens(names)
}

/** An `aria-describedby` value from the truthy ids, or `undefined` when there are none. */
export function idRefs(...ids: Token[]): string | undefined {
  return joinTokens(ids) || undefined
}

/** Id of the hint element of the control with `id`. */
export function hintId(id: string): string {
  return id + HINT_ID_SUFFIX
}

/** Id of the error message element of the control with `id`. */
export function errorId(id: string): string {
  return id + ERROR_ID_SUFFIX
}
