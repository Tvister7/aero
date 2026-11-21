const PALETTE = ['#1d4ed8', '#c2410c', '#047857', '#7c3aed', '#be123c', '#a16207', '#0e7490', '#4d7c0f']

function hash(title: string): number {
  let value = 0
  for (const char of title) value = (value * 31 + char.charCodeAt(0)) >>> 0
  return value
}

export function shownColor(color: string | undefined, title: string): string {
  if (color) return color
  return PALETTE[hash(title) % PALETTE.length] ?? PALETTE[0]
}

export function nextTypeColor(used: Iterable<string>): string {
  const taken = new Set(used)
  const free = PALETTE.find((color) => !taken.has(color))
  if (free) return free
  return PALETTE[taken.size % PALETTE.length] ?? PALETTE[0]
}
