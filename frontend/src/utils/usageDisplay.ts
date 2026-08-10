export function readUsageNumber(
  source: object,
  keys: readonly string[],
  fallback = 0,
): number {
  const values = source as Record<string, unknown>
  for (const key of keys) {
    const raw = values[key]
    if (raw === null || raw === undefined || raw === '') continue
    const value = typeof raw === 'number' ? raw : Number(raw)
    if (Number.isFinite(value)) return value
  }
  return fallback
}
