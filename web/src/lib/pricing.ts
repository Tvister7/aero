import { classesWord } from './text'

const sections = ['base', 'aero', 'kids', 'personal'] as const

const titles: Record<(typeof sections)[number], string> = {
  base: 'Базовый',
  aero: 'Aero',
  kids: 'Детский',
  personal: 'Персональный',
}

export type PriceRow = { label: string; price: string }
export type PriceBlock = { title: string; rows: PriceRow[] }

export function pricingError(raw: string): string {
  const text = raw.trim() || '{}'
  if (text.length > 8000) return 'Прайс слишком длинный'
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return 'Прайс должен быть JSON-объектом'
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return 'Прайс должен быть JSON-объектом'
  const allowed = new Set<string>(sections)
  for (const [key, value] of Object.entries(parsed)) {
    if (!allowed.has(key)) return 'В прайсе можно указать только base, aero, kids и personal'
    if (!value || typeof value !== 'object' || Array.isArray(value)) return 'Раздел прайса должен быть объектом'
    for (const [label, price] of Object.entries(value)) {
      if (!label.trim() || label !== label.trim() || typeof price !== 'number' || !Number.isInteger(price) || price <= 0) {
        return 'В прайсе нужны целые положительные числа'
      }
    }
  }
  return ''
}

export function canonicalPricing(raw: string): string {
  const parsed = JSON.parse(raw.trim() || '{}') as Record<string, Record<string, number>>
  const clean: Record<string, Record<string, number>> = {}
  for (const key of Object.keys(parsed).sort()) {
    const section: Record<string, number> = {}
    for (const label of Object.keys(parsed[key]).sort()) section[label] = parsed[key][label]
    clean[key] = section
  }
  return JSON.stringify(clean)
}

export function pricingBlocks(raw: string): PriceBlock[] {
  if (pricingError(raw)) return []
  const parsed = JSON.parse(raw.trim() || '{}') as Record<string, Record<string, number>>
  const blocks: PriceBlock[] = []
  for (const key of sections) {
    const section = parsed[key]
    if (!section) continue
    const rows = Object.entries(section)
      .map(([label, price]) => ({ label: offerLabel(label), price: `${gel(price)} GEL`, sort: offerSort(label) }))
      .sort((a, b) => a.sort - b.sort || a.label.localeCompare(b.label, 'ru'))
      .map(({ label, price }) => ({ label, price }))
    if (rows.length) blocks.push({ title: titles[key], rows })
  }
  return blocks
}

function gel(value: number): string {
  return Number.isInteger(value) ? String(value) : String(value)
}

function offerLabel(key: string): string {
  if (key === 'single') return 'разовое'
  if (key === 'split') return 'сплит'
  if (/^\d+$/.test(key)) return `${key} ${classesWord(Number(key))}`
  return key
}

function offerSort(key: string): number {
  if (/^\d+$/.test(key)) return Number(key)
  if (key === 'single') return 10000
  if (key === 'split') return 10001
  return 20000
}
