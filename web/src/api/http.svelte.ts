import '../telegram'
import type {
  ApiError,
  ClassView,
  ClientName,
  ClientSummary,
  ErrorCode,
  Me,
  RosterData,
  Stats,
  PublicTrainer,
  StudioProfile,
  TrainingType,
  Trainer,
  ClassTemplate,
  EnrollmentOption,
  TrainerDetail,
  Visit,
  TrainerStats,
} from './types'
import { addDays, todayKey } from '../lib/time'
import { type Load, ready } from './load'

const CODES = new Set<ErrorCode>([
  'UNAUTHORIZED',
  'FORBIDDEN',
  'VALIDATION',
  'CLASS_NOT_FOUND',
  'NEW_CLASS_NOT_FOUND',
  'BOOKING_NOT_FOUND',
  'CLASS_STARTED',
  'NO_PLACES',
  'ALREADY_BOOKED',
  'CUTOFF_EXCEEDED',
  'SAME_CLASS',
  'NO_CREDITS',
  'ATTENDANCE_TOO_EARLY',
  'STUDIO_BUSY',
])

export const revision = $state({ n: 0 })
const store = new Map<string, unknown>()
const inflight = new Set<string>()

const emptyMe: Me = {
  name: '',
  phone: '',
  username: '',
  balance: 0,
  policy: 'postpay',
  is_admin: false,
  is_trainer: false,
  visited: false,
}

const emptyStudio: StudioProfile = {
  address: '',
  phone: '',
  instagram_url: '',
  contact_url: '',
  map_url: '',
  guide: '',
  pricing: '',
}

function base(): string {
  return (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
}

function transportError(): ApiError {
  return { ok: false, error_code: 'VALIDATION', message: 'Не удалось выполнить запрос' }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function asError(value: unknown): ApiError {
  if (isRecord(value) && typeof value.message === 'string') {
    const code = typeof value.error_code === 'string' && CODES.has(value.error_code as ErrorCode)
      ? (value.error_code as ErrorCode)
      : 'VALIDATION'
    return { ok: false, error_code: code, message: value.message }
  }
  return transportError()
}

function succeeded(value: unknown): boolean {
  return isRecord(value) && value.ok === true
}

function dataOf<T>(value: unknown): T | null {
  if (succeeded(value) && isRecord(value) && 'data' in value) return value.data as T
  return null
}

function listOf<T>(value: unknown): T[] | null {
  const data = dataOf<unknown>(value)
  return Array.isArray(data) ? (data as T[]) : null
}

function cached(key: string, load: () => Promise<unknown>): unknown {
  void revision.n
  if (store.has(key)) return store.get(key)
  if (!inflight.has(key)) {
    inflight.add(key)
    void load()
      .then((value) => {
        store.set(key, value)
      })
      .catch(() => {
        store.set(key, transportError())
      })
      .finally(() => {
        inflight.delete(key)
        revision.n += 1
      })
  }
  return null
}

function refresh() {
  store.clear()
  revision.n += 1
}

function forget(key: string) {
  store.delete(key)
  revision.n += 1
}

function read<T>(key: string, path: string, pick: (value: unknown) => T | null): Load<T> {
  const value = cached(key, () => request('GET', path))
  if (value === null) return { status: 'pending' }
  if (!succeeded(value)) return { status: 'error', message: asError(value).message, reload: () => forget(key) }
  const data = pick(value)
  if (data === null) return { status: 'error', message: transportError().message, reload: () => forget(key) }
  return ready(data)
}

let lastMe: Me | null = null

export function knownMe(): Me | null {
  void revision.n
  return lastMe
}

async function request(method: string, path: string, body?: unknown): Promise<unknown> {
  const headers: Record<string, string> = {}
  const initData = window.Telegram?.WebApp?.initData
  if (initData) headers.Authorization = `tma ${initData}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  let response: Response
  try {
    response = await fetch(`${base()}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    return transportError()
  }
  if (path.endsWith('.csv')) {
    if (!response.ok) return asError(await response.json().catch(() => null))
    return response.text()
  }
  const parsed = await response.json().catch(() => null)
  if (!response.ok) return asError(parsed)
  return parsed
}

async function send(method: string, path: string, body?: unknown): Promise<unknown> {
  const value = await request(method, path, body)
  if (succeeded(value)) refresh()
  return value
}

function readList<T>(key: string, path: string): Load<T[]> {
  return read(key, path, (value) => listOf<T>(value))
}

export function roleSwitchVisible() {
  return false
}

export function setPreviewAdmin(_value: boolean): void {}

export function setPreviewRole(_role: 'client' | 'trainer' | 'admin'): void {}

export function listPublicTrainers() {
  return readList<PublicTrainer>('public-trainers', '/trainers')
}

export function getPublicTrainer(id: string) {
  return read<PublicTrainer>('public-trainer:' + id, `/trainers/${id}`, (value) => dataOf<PublicTrainer>(value))
}

export function getStudio() {
  return read<StudioProfile>('studio', '/studio', (value) => dataOf<StudioProfile>(value) ?? (succeeded(value) ? emptyStudio : null))
}

export function getSchedule(from: string, to: string) {
  return readList<ClassView>('schedule:' + from + ':' + to, `/schedule?from=${from}&to=${to}`)
}

export function getAdminSchedule(from: string, to: string) {
  return readList<ClassView>('admin-schedule:' + from + ':' + to, `/admin/schedule?from=${from}&to=${to}`)
}

export function getClass(id: string) {
  return read<ClassView>('class:' + id, `/classes/${id}`, (value) => dataOf<ClassView>(value))
}

export function getAdminClass(id: string): Load<ClassView> {
  const from = addDays(todayKey(), -120)
  const to = addDays(todayKey(), 240)
  const schedule = getAdminSchedule(from, to)
  if (schedule.status !== 'ready') return schedule
  const found = schedule.data.find((item) => item.id === id)
  if (found) return ready(found)
  return { status: 'error', message: 'Занятие не найдено', reload: () => forget('admin-schedule:' + from + ':' + to) }
}

export async function bookClass(id: string, body: { name?: string; phone?: string }) {
  return send('POST', `/classes/${id}/bookings`, body) as Promise<
    ApiError | { ok: true; booking_id: string; balance: number; class: ClassView }
  >
}

export function getMe(): Load<Me> {
  const value = cached('me', async () => {
    const payload = await request('GET', '/me')
    const data = dataOf<Me>(payload)
    if (data) lastMe = { ...emptyMe, ...data, visited: Boolean(data.visited) }
    return payload
  })
  if (value === null) return { status: 'pending' }
  const data = dataOf<Me>(value)
  if (!data) return { status: 'error', message: asError(value).message, reload: () => forget('me') }
  lastMe = { ...emptyMe, ...data, visited: Boolean(data.visited) }
  return ready(lastMe)
}

export async function patchMe(body: { name?: string; phone?: string }) {
  return send('PATCH', '/me', body) as Promise<ApiError | { ok: true; data: Me }>
}

export async function ping() {
  await request('POST', '/me/ping')
  return { ok: true as const }
}

export function myBookings() {
  return readList<{ booking_id: string; class: ClassView }>('bookings', '/me/bookings')
}

export function myVisits() {
  return readList<Visit>('visits', '/me/visits')
}

export async function cancelBooking(id: string) {
  return send('POST', `/bookings/${id}/cancel`) as Promise<ApiError | { ok: true; booking_id: string }>
}

export async function rescheduleBooking(id: string, classId: string) {
  return send('POST', `/bookings/${id}/reschedule`, { class_id: classId }) as Promise<
    ApiError | { ok: true; booking_id: string; class: ClassView }
  >
}

export function listTrainers() {
  return readList<Pick<Trainer, 'id' | 'name' | 'active'>>('trainers', '/admin/trainers')
}

export async function createTrainer(name: string) {
  return send('POST', '/admin/trainers', { name }) as Promise<ApiError | { ok: true; data: Trainer }>
}

export async function patchTrainer(
  id: string,
  body: { name?: string; active?: boolean; about?: string; phone?: string; telegram_user_id?: string },
) {
  return send('PATCH', `/admin/trainers/${id}`, body) as Promise<ApiError | { ok: true; data: Trainer }>
}

export function getTrainer(id: string) {
  return read<TrainerDetail>('trainer:' + id, `/admin/trainers/${id}`, (value) => {
    const data = dataOf<TrainerDetail>(value)
    return data ? { ...data, absences: data.absences ?? [] } : null
  })
}

export async function addAbsence(id: string, startsOn: string, endsOn: string) {
  return send('POST', `/admin/trainers/${id}/absences`, { starts_on: startsOn, ends_on: endsOn }) as Promise<
    ApiError | { ok: true; data: TrainerDetail }
  >
}

export async function deleteAbsence(id: string, absenceId: string) {
  return send('DELETE', `/admin/trainers/${id}/absences/${absenceId}`) as Promise<ApiError | { ok: true; data: TrainerDetail }>
}

export async function cancelTrainerClasses(id: string) {
  return send('POST', `/admin/trainers/${id}/cancel-upcoming`) as Promise<ApiError | { ok: true }>
}

export async function deleteTrainer(id: string) {
  return send('DELETE', `/admin/trainers/${id}`) as Promise<ApiError | { ok: true }>
}

export function listEnrollments() {
  return readList<EnrollmentOption>('enrollments', '/me/enrollments')
}

export async function putEnrollments(templateIds: string[]) {
  return send('PUT', '/me/enrollments', { template_ids: templateIds }) as Promise<
    ApiError | { ok: true; data: EnrollmentOption[] }
  >
}

export function getTrainerProfile() {
  return read<Trainer>('trainer-me', '/trainer', (value) => dataOf<Trainer>(value))
}

export async function patchTrainerProfile(body: { about: string; phone: string }) {
  return send('PATCH', '/trainer', body) as Promise<ApiError | { ok: true; data: Trainer }>
}

export function getTrainerSchedule(from: string, to: string) {
  return readList<ClassView>('trainer-schedule:' + from + ':' + to, `/trainer/schedule?from=${from}&to=${to}`)
}

export function listTrainerTypes() {
  return readList<TrainingType>('trainer-types', '/trainer/types')
}

export async function createTrainerClass(body: { training_type_id?: string; starts_at?: string; capacity?: number }) {
  return send('POST', '/trainer/classes', body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export function searchTrainerClients(query: string) {
  return readList<ClientName>('trainer-clients:' + query, `/trainer/clients?q=${encodeURIComponent(query)}`)
}

export async function createTrainerIndividual(body: { client_id: string; starts_at: string; duration_minutes?: number }) {
  return send('POST', '/trainer/individuals', body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export async function moveTrainerClass(id: string, startsAt: string) {
  return send('PATCH', `/trainer/classes/${id}`, { starts_at: startsAt }) as Promise<ApiError | { ok: true; data: ClassView }>
}

export async function cancelTrainerClass(id: string) {
  return send('POST', `/trainer/classes/${id}/cancel`) as Promise<ApiError | { ok: true; data: ClassView }>
}

export function getTrainerRoster(id: string) {
  return read<RosterData>('trainer-roster:' + id, `/trainer/classes/${id}/roster`, (value) => dataOf<RosterData>(value))
}

export async function setTrainerAttendance(bookingId: string, arrived: boolean) {
  return send('POST', `/trainer/bookings/${bookingId}/attendance`, { arrived }) as Promise<
    ApiError | { ok: true; booking_id: string; attendance: { payment: string; marked_at: string } | null; balance: number }
  >
}

export function trainerStats(from: string, to: string): Load<TrainerStats> {
  return read('trainer-stats:' + from + ':' + to, `/trainer/stats?from=${from}&to=${to}`, (value) => dataOf<TrainerStats>(value))
}

export function listTypes() {
  return readList<TrainingType>('types', '/admin/training-types')
}

export async function createType(body: {
  title?: string
  duration_minutes?: number
  capacity?: number
  price_min?: number | string
  price_max?: number | string
}) {
  return send('POST', '/admin/training-types', body) as Promise<ApiError | { ok: true }>
}

export async function patchType(
  id: string,
  body: {
    title?: string
    duration_minutes?: number
    capacity?: number
    price_min?: number | string
    price_max?: number | string
    active?: boolean
  },
) {
  return send('PATCH', `/admin/training-types/${id}`, body) as Promise<ApiError | { ok: true }>
}

export function listTemplates() {
  return readList<ClassTemplate>('templates', '/admin/templates')
}

export async function createTemplate(body: {
  trainer_id?: string
  training_type_id?: string
  weekday?: number
  start_time?: string
  capacity?: number
}) {
  return send('POST', '/admin/templates', body) as Promise<ApiError | { ok: true }>
}

export async function patchTemplate(
  id: string,
  body: {
    weekday?: number
    start_time?: string
    capacity?: number
    trainer_id?: string
    training_type_id?: string
    active?: boolean
  },
) {
  return send('PATCH', `/admin/templates/${id}`, body) as Promise<ApiError | { ok: true }>
}

export async function generateWeek(weekStart?: string) {
  return send('POST', '/admin/weeks', { week_start: weekStart ?? '' }) as Promise<
    ApiError | { ok: true; week_start: string; week_end: string; created: number; updated: number }
  >
}

export async function createHold(body: { title: string; starts_at: string; duration_minutes?: number }) {
  return send('POST', '/admin/holds', body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export async function createIndividual(body: {
  trainer_id: string
  client_id: string
  starts_at: string
  duration_minutes?: number
}) {
  return send('POST', '/admin/individuals', body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export async function createClass(body: {
  trainer_id?: string
  training_type_id?: string
  starts_at?: string
  capacity?: number
}) {
  return send('POST', '/admin/classes', body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export async function patchClass(
  id: string,
  body: { status?: 'cancelled'; starts_at?: string; trainer_id?: string; capacity?: number; title?: string },
) {
  return send('PATCH', `/admin/classes/${id}`, body) as Promise<ApiError | { ok: true; data: ClassView }>
}

export function getRoster(id: string) {
  return read<RosterData>('roster:' + id, `/admin/classes/${id}/roster`, (value) => dataOf<RosterData>(value))
}

export async function setAttendance(bookingId: string, arrived: boolean) {
  return send('POST', `/admin/bookings/${bookingId}/attendance`, { arrived }) as Promise<
    ApiError | { ok: true; booking_id: string; attendance: { payment: string; marked_at: string } | null; balance: number }
  >
}

export function listClients(query = '') {
  const suffix = query ? `?q=${encodeURIComponent(query)}` : ''
  return readList<ClientSummary>('clients:' + query, `/admin/clients${suffix}`)
}

export function getClient(id: string) {
  return read('client:' + id, `/admin/clients/${id}`, (value) =>
    dataOf<ClientSummary & { ledger: Array<{ id: string; delta: number; reason: 'grant' | 'attend' | 'reversal'; note: string; booking_id: string | null; created_at: string }> }>(value),
  )
}

export async function patchClient(id: string, policy: 'postpay' | 'restricted') {
  return send('PATCH', `/admin/clients/${id}`, { policy }) as Promise<ApiError | { ok: true; data: ClientSummary }>
}

export async function grantCredits(id: string, delta: number, note?: string) {
  return send('POST', `/admin/clients/${id}/credits`, { delta, note: note ?? '' }) as Promise<
    ApiError | { ok: true; data: ClientSummary }
  >
}

export function stats(from: string, to: string): Load<Stats> {
  return read('stats:' + from + ':' + to, `/admin/stats?from=${from}&to=${to}`, (value) => dataOf<Stats>(value))
}

export async function clientsCsv() {
  const value = await request('GET', '/admin/clients.csv')
  return typeof value === 'string' ? value : ''
}

export async function putStudio(body: Partial<StudioProfile>) {
  return send('PUT', '/admin/studio', body) as Promise<ApiError | { ok: true; data: StudioProfile }>
}

export function todayRoster(): Load<RosterData[]> {
  const today = todayKey()
  const schedule = getAdminSchedule(today, today)
  if (schedule.status !== 'ready') return schedule
  const rows: RosterData[] = []
  for (const item of schedule.data) {
    if (item.status === 'cancelled' || item.kind === 'hold') continue
    const roster = getRoster(item.id)
    if (roster.status !== 'ready') return roster
    rows.push(roster.data)
  }
  return ready(rows)
}
