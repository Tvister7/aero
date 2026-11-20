import type {
  ApiError,
  ClassView,
  ClientName,
  ClientSummary,
  ErrorCode,
  Me,
  RosterData,
  Stats,
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

function readList<T>(key: string, path: string): { ok: true; data: T[]; pending: boolean } {
  const value = cached(key, () => request('GET', path))
  if (value === null) return { ok: true, data: [], pending: true }
  return { ok: true, data: listOf<T>(value) ?? [], pending: false }
}

export function roleSwitchVisible() {
  return false
}

export function setPreviewAdmin(_value: boolean): void {}

export function setPreviewRole(_role: 'client' | 'trainer' | 'admin'): void {}

export function getStudio() {
  const value = cached('studio', () => request('GET', '/studio'))
  const data = dataOf<StudioProfile>(value)
  if (data) return { ok: true as const, data, pending: false }
  return { ok: true as const, data: emptyStudio, pending: value === null }
}

export function getSchedule(from: string, to: string) {
  return readList<ClassView>('schedule:' + from + ':' + to, `/schedule?from=${from}&to=${to}`)
}

export function getAdminSchedule(from: string, to: string) {
  return readList<ClassView>('admin-schedule:' + from + ':' + to, `/admin/schedule?from=${from}&to=${to}`)
}

export function getClass(id: string) {
  const value = cached('class:' + id, () => request('GET', `/classes/${id}`))
  const data = dataOf<ClassView>(value)
  if (data) return { ok: true as const, data }
  if (value && !succeeded(value)) return asError(value)
  return { ok: false as const, error_code: 'VALIDATION' as const, message: '' }
}

export function getAdminClass(id: string) {
  const from = addDays(todayKey(), -120)
  const to = addDays(todayKey(), 240)
  const schedule = getAdminSchedule(from, to)
  const found = schedule.data.find((item) => item.id === id)
  if (found) return { ok: true as const, data: found }
  if (schedule.pending) return { ok: false as const, error_code: 'VALIDATION' as const, message: '' }
  return { ok: false as const, error_code: 'CLASS_NOT_FOUND' as const, message: 'Занятие не найдено' }
}

export async function bookClass(id: string, body: { name?: string; phone?: string }) {
  return send('POST', `/classes/${id}/bookings`, body) as Promise<
    ApiError | { ok: true; booking_id: string; balance: number; class: ClassView }
  >
}

export function getMe() {
  const value = cached('me', () => request('GET', '/me'))
  const data = dataOf<Me>(value)
  if (!data) return { ok: true as const, data: emptyMe, pending: value === null }
  return {
    ok: true as const,
    pending: false,
    data: { ...emptyMe, ...data, visited: Boolean(data.visited) },
  }
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

const emptyTrainer: TrainerDetail = {
  id: '',
  name: '',
  active: true,
  about: '',
  phone: '',
  telegram_user_id: null,
  absences: [],
}

export function getTrainer(id: string) {
  const value = cached('trainer:' + id, () => request('GET', `/admin/trainers/${id}`))
  const data = dataOf<TrainerDetail>(value)
  if (data) return { ok: true as const, data: { ...data, absences: data.absences ?? [] }, pending: false }
  if (value && !succeeded(value)) return asError(value)
  return { ok: true as const, data: emptyTrainer, pending: value === null }
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
  const value = cached('trainer-me', () => request('GET', '/trainer'))
  const data = dataOf<Trainer>(value)
  if (data) return { ok: true as const, data, pending: false }
  if (value && !succeeded(value)) return asError(value)
  return {
    ok: true as const,
    pending: value === null,
    data: { id: '', name: '', active: true, about: '', phone: '', telegram_user_id: null },
  }
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
  const value = cached('trainer-roster:' + id, () => request('GET', `/trainer/classes/${id}/roster`))
  const data = dataOf<RosterData>(value)
  if (data) return { ok: true as const, data }
  if (value && !succeeded(value)) return asError(value)
  return { ok: false as const, error_code: 'VALIDATION' as const, message: '' }
}

export async function setTrainerAttendance(bookingId: string, arrived: boolean) {
  return send('POST', `/trainer/bookings/${bookingId}/attendance`, { arrived }) as Promise<
    ApiError | { ok: true; booking_id: string; attendance: { payment: string; marked_at: string } | null; balance: number }
  >
}

export function trainerStats(from: string, to: string): { ok: true; data: TrainerStats } {
  const value = cached('trainer-stats:' + from + ':' + to, () => request('GET', `/trainer/stats?from=${from}&to=${to}`))
  const data = dataOf<TrainerStats>(value)
  return { ok: true, data: data ?? { from, to, clients: 0, attends: 0 } }
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
  const value = cached('roster:' + id, () => request('GET', `/admin/classes/${id}/roster`))
  const data = dataOf<RosterData>(value)
  if (data) return { ok: true as const, data }
  if (value && !succeeded(value)) return asError(value)
  return { ok: false as const, error_code: 'VALIDATION' as const, message: '' }
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
  const value = cached('client:' + id, () => request('GET', `/admin/clients/${id}`))
  const data = dataOf<ClientSummary & { ledger: Array<{ id: string; delta: number; reason: 'grant' | 'attend' | 'reversal'; note: string; booking_id: string | null; created_at: string }> }>(value)
  if (data) return { ok: true as const, data }
  if (value && !succeeded(value)) return asError(value)
  return { ok: false as const, error_code: 'VALIDATION' as const, message: '' }
}

export async function patchClient(id: string, policy: 'postpay' | 'restricted') {
  return send('PATCH', `/admin/clients/${id}`, { policy }) as Promise<ApiError | { ok: true; data: ClientSummary }>
}

export async function grantCredits(id: string, delta: number, note?: string) {
  return send('POST', `/admin/clients/${id}/credits`, { delta, note: note ?? '' }) as Promise<
    ApiError | { ok: true; data: ClientSummary }
  >
}

export function stats(from: string, to: string): { ok: true; data: Stats } {
  const value = cached('stats:' + from + ':' + to, () => request('GET', `/admin/stats?from=${from}&to=${to}`))
  const data = dataOf<Stats>(value)
  return {
    ok: true,
    data: data ?? {
      from,
      to,
      unique_users: 0,
      bookings_created: 0,
      bookings_cancelled: 0,
      arrivals: 0,
      no_shows: 0,
      unpaid_visits: 0,
    },
  }
}

export async function clientsCsv() {
  const value = await request('GET', '/admin/clients.csv')
  return typeof value === 'string' ? value : ''
}

export async function putStudio(body: Partial<StudioProfile>) {
  return send('PUT', '/admin/studio', body) as Promise<ApiError | { ok: true; data: StudioProfile }>
}

export function todayRoster(): RosterData[] {
  const today = todayKey()
  const schedule = getAdminSchedule(today, today)
  return schedule.data.flatMap((item) => {
    if (item.status === 'cancelled' || item.kind === 'hold') return []
    const roster = getRoster(item.id)
    return roster.ok ? [roster.data] : []
  })
}

declare global {
  interface Window {
    Telegram?: { WebApp?: { initData?: string; ready?: () => void; disableVerticalSwipes?: () => void } }
  }
}

window.Telegram?.WebApp?.ready?.()
window.Telegram?.WebApp?.disableVerticalSwipes?.()
