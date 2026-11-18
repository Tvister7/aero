const TZ = 'Asia/Tbilisi'

export function formatDateKey(date: Date): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

export function todayKey(): string {
  return formatDateKey(new Date())
}

export function addDays(dateKey: string, days: number): string {
  const [y, m, d] = dateKey.split('-').map(Number)
  const utc = new Date(Date.UTC(y, m - 1, d))
  utc.setUTCDate(utc.getUTCDate() + days)
  return utc.toISOString().slice(0, 10)
}

export function isoWeekday(dateKey: string): number {
  const [y, m, d] = dateKey.split('-').map(Number)
  const js = new Date(Date.UTC(y, m - 1, d)).getUTCDay()
  return js === 0 ? 7 : js
}

export function mondayKey(dateKey: string): string {
  return addDays(dateKey, 1 - isoWeekday(dateKey))
}

function part(parts: Intl.DateTimeFormatPart[], type: Intl.DateTimeFormatPartTypes): string {
  const value = parts.find((item) => item.type === type)?.value
  if (!value) throw new Error(type)
  return value
}

export function toTbilisiIso(date: Date): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(date)
  return `${part(parts, 'year')}-${part(parts, 'month')}-${part(parts, 'day')}T${part(parts, 'hour')}:${part(parts, 'minute')}:${part(parts, 'second')}+04:00`
}

export function dateKeyOfIso(iso: string): string {
  return formatDateKey(new Date(iso))
}

export function formatWhen(iso: string): string {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: TZ,
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}

export function formatDay(dateKey: string): string {
  const [y, m, d] = dateKey.split('-').map(Number)
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: TZ,
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  }).format(new Date(Date.UTC(y, m - 1, d, 8)))
}

const WEEKDAYS = ['', 'понедельник', 'вторник', 'среда', 'четверг', 'пятница', 'суббота', 'воскресенье']

export function weekdayName(weekday: number): string {
  return WEEKDAYS[weekday] ?? ''
}

export function hoursUntil(iso: string): number {
  return (new Date(iso).getTime() - Date.now()) / 3600000
}

export function formatShortDate(dateKey: string): string {
  const [, month, day] = dateKey.split('-')
  return `${day}.${month}`
}

export function formatWeekRange(start: string, end: string): string {
  return `${formatShortDate(start)}–${formatShortDate(end)}`
}

export function monthKeyOf(dateKey: string): string {
  return dateKey.slice(0, 7)
}

export function monthBounds(month: string): { start: string; end: string } {
  const [year, monthNumber] = month.split('-').map(Number)
  const last = new Date(Date.UTC(year, monthNumber, 0)).getUTCDate()
  return { start: `${month}-01`, end: `${month}-${String(last).padStart(2, '0')}` }
}

export function shiftMonth(month: string, delta: number): string {
  const [year, monthNumber] = month.split('-').map(Number)
  const date = new Date(Date.UTC(year, monthNumber - 1 + delta, 1))
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`
}

export function monthTitle(month: string): string {
  const [year, monthNumber] = month.split('-').map(Number)
  return new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(
    new Date(Date.UTC(year, monthNumber - 1, 1)),
  )
}

export function monthCells(month: string): string[] {
  const { start, end } = monthBounds(month)
  const gridStart = mondayKey(start)
  const gridEnd = addDays(mondayKey(end), 6)
  const days: string[] = []
  for (let day = gridStart; day <= gridEnd; day = addDays(day, 1)) days.push(day)
  return days
}
