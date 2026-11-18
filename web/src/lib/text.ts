import type { BookingPolicy } from '../api/types'

export function classesWord(count: number): string {
  const abs = Math.abs(count) % 100
  const last = abs % 10
  if (abs > 10 && abs < 20) return 'занятий'
  if (last === 1) return 'занятие'
  if (last >= 2 && last <= 4) return 'занятия'
  return 'занятий'
}

export function balanceText(balance: number): string {
  if (balance < 0) return `долг ${Math.abs(balance)} ${classesWord(balance)}`
  return `${balance} ${classesWord(balance)}`
}

export function policyText(policy: BookingPolicy): string {
  return policy === 'restricted' ? 'абонемент' : 'постоплата'
}

export function priceLabel(min: string, max: string): string {
  const left = Number(min)
  const right = Number(max)
  if (!left && !right) return ''
  const amount = (value: number, raw: string) => (Number.isInteger(value) ? String(value) : raw)
  if (left === right) return `${amount(left, min)} ₾`
  return `${amount(left, min)}–${amount(right, max)} ₾`
}

export function money(value: number | string): string {
  return Number(value).toFixed(2)
}
