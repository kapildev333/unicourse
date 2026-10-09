/** The score after a late penalty, rounded half up to 2 decimals, in integer hundredths (matches the server). */
export function penalised(score: number, pct: number): string {
  const c = Math.round(score * 100)
  const q = Math.round(pct * 100)
  return (Math.round((c * (10000 - q)) / 10000) / 100).toFixed(2)
}
