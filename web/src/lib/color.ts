const PALETTE = ['#3a7ca5', '#d46a3a', '#2f8f6b', '#7a5ea7', '#c43d5a', '#b8860b', '#3d6b8c', '#5c7a3a']

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
