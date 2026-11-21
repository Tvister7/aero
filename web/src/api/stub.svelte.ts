import type {
  ApiError,
  Attendance,
  Booking,
  BookingPolicy,
  ClassRow,
  ClassView,
  Client,
  ClientSummary,
  Database,
  ErrorCode,
  RosterBooking,
  LedgerEntry,
  RosterData,
  Stats,
  PublicTrainer,
  StudioProfile,
  TrainingType,
} from './types'
import { nameError, phoneError } from '../lib/contact'
import { nextTypeColor } from '../lib/color'
import { canonicalPricing, pricingError } from '../lib/pricing'
import { defaultGuide, defaultPricing } from '../lib/studioDefaults'
import { money } from '../lib/text'
import {
  addDays,
  dateKeyOfIso,
  hoursUntil,
  mondayKey,
  todayKey,
  toTbilisiIso,
} from '../lib/time'

const MESSAGES: Record<ErrorCode, string> = {
  UNAUTHORIZED: 'Откройте приложение заново из Telegram',
  FORBIDDEN: 'Недостаточно прав',
  VALIDATION: 'Проверьте поля',
  CLASS_NOT_FOUND: 'Занятие не найдено',
  NEW_CLASS_NOT_FOUND: 'Новое занятие недоступно',
  BOOKING_NOT_FOUND: 'Запись не найдена',
  CLASS_STARTED: 'Занятие уже началось',
  NO_PLACES: 'Свободных мест нет',
  ALREADY_BOOKED: 'Вы уже записаны',
  CUTOFF_EXCEEDED: 'Отмена и перенос доступны не позднее чем за 6 часов',
  SAME_CLASS: 'Это то же занятие',
  NO_CREDITS: 'Занятия на балансе закончились',
  ATTENDANCE_TOO_EARLY: 'Отметить явку можно за 30 минут до начала',
  STUDIO_BUSY: 'Это время уже занято',
}

export const revision = $state({ n: 0 })
const ui = $state({ role: 'client' as 'client' | 'trainer' | 'admin', enrolled: [] as string[] })
const STUB_TRAINER = 'tr-zhanna'

function touch() {
  revision.n += 1
}

function fail(code: ErrorCode): ApiError {
  return { ok: false, error_code: code, message: MESSAGES[code] }
}

function failMessage(message: string): ApiError {
  return { ok: false, error_code: 'VALIDATION', message }
}

function uid(prefix: string): string {
  return `${prefix}-${crypto.randomUUID()}`
}

const db = $state<Database>({
  trainers: [],
  types: [],
  templates: [],
  classes: [],
  clients: [],
  bookings: [],
  events: [],
  ledger: [],
  attendances: [],
  opens: [],
  studio: {
    address: 'Тбилиси, адрес студии',
    phone: '+995 32 000 00 00',
    instagram_url: 'https://instagram.com/aero.ge',
    contact_url: 'https://t.me/aeroge',
    map_url: 'https://maps.google.com/?q=Tbilisi',
    guide: defaultGuide,
    pricing: defaultPricing,
  },
  absences: [],
})

const DEV_ID = 'client-dev'

export function roleSwitchVisible() {
  return !import.meta.env.VITE_API_URL
}

export function setPreviewAdmin(value: boolean): void {
  setPreviewRole(value ? 'admin' : 'client')
}

export function setPreviewRole(role: 'client' | 'trainer' | 'admin'): void {
  ui.role = role
  touch()
}

function devClient(): Client {
  const client = db.clients.find((item) => item.id === DEV_ID)
  if (!client) throw new Error('dev client')
  return client
}

function balanceOf(clientId: string): number {
  return db.ledger.filter((row) => row.client_id === clientId).reduce((sum, row) => sum + row.delta, 0)
}

function bookedCount(classId: string): number {
  return db.bookings.filter((row) => row.class_id === classId && row.status === 'booked').length
}

function activeAttendance(bookingId: string): Attendance | undefined {
  return db.attendances.find((row) => row.booking_id === bookingId && row.active)
}

function classKind(row: ClassRow) {
  return row.kind ?? 'group'
}

function studioBusy(start: string, end: string, ignoreId = '') {
  return db.classes.some(
    (row) =>
      row.status === 'scheduled' &&
      row.id !== ignoreId &&
      new Date(row.starts_at) < new Date(end) &&
      new Date(row.ends_at) > new Date(start),
  )
}

function classView(row: ClassRow, maskHold = false): ClassView {
  const trainer = row.trainer_id ? db.trainers.find((item) => item.id === row.trainer_id) : undefined
  const kind = classKind(row)
  const booked = bookedCount(row.id)
  const type = db.types.find((item) => item.id === row.training_type_id)
  return {
    id: row.id,
    title: maskHold && kind === 'hold' ? 'Студия занята' : row.title,
    trainer: { id: trainer?.id ?? '', name: trainer?.name ?? '' },
    starts_at: row.starts_at,
    ends_at: row.ends_at,
    capacity: row.capacity,
    booked,
    free_places: Math.max(0, row.capacity - booked),
    price_min: maskHold && kind === 'hold' ? '' : row.price_min,
    price_max: maskHold && kind === 'hold' ? '' : row.price_max,
    color: type?.color || (kind === 'hold' ? '#9aa3b2' : kind === 'individual' ? '#7a5ea7' : ''),
    training_type_id: row.training_type_id,
    template_id: row.template_id,
    status: row.status,
    kind,
  }
}

function watch() {
  void revision.n
}

function inRange(iso: string, from: string, to: string): boolean {
  const key = dateKeyOfIso(iso)
  return key >= from && key <= to
}

function addEvent(bookingId: string, eventType: 'created' | 'cancelled' | 'rescheduled', oldClassId: string | null, newClassId: string | null): void {
  db.events.push({
    id: uid('event'),
    booking_id: bookingId,
    event_type: eventType,
    old_class_id: oldClassId ?? null,
    new_class_id: newClassId ?? null,
    created_at: toTbilisiIso(new Date()),
  })
}

function placeBooking(classRow: ClassRow, client: Client, { seed = false }: { seed?: boolean } = {}): Booking {
  const booking = {
    id: uid('booking'),
    class_id: classRow.id,
    client_id: client.id,
    status: 'booked' as const,
    reminder_state: 'pending' as const,
    reminder_at: toTbilisiIso(new Date(new Date(classRow.starts_at).getTime() - 24 * 3600000)),
    created_at: toTbilisiIso(new Date()),
  }
  db.bookings.push(booking)
  if (!seed) addEvent(booking.id, 'created', null, classRow.id)
  return booking
}

export function getStudio() {
  watch()
  return { ok: true as const, data: { ...db.studio }, pending: false }
}

export function getSchedule(from: string, to: string) {
  watch()
  const data = db.classes
    .filter((row) => row.status === 'scheduled' && classKind(row) !== 'individual' && inRange(row.starts_at, from, to))
    .sort((a, b) => a.starts_at.localeCompare(b.starts_at))
    .map((row) => classView(row, true))
  return { ok: true as const, data, pending: false }
}

export function getAdminSchedule(from: string, to: string) {
  watch()
  const data = db.classes
    .filter((row) => inRange(row.starts_at, from, to))
    .sort((a, b) => a.starts_at.localeCompare(b.starts_at))
    .map((row) => classView(row))
  return { ok: true as const, data, pending: false }
}

export function getClass(id: string) {
  watch()
  const row = db.classes.find((item) => item.id === id && item.status === 'scheduled')
  if (!row || classKind(row) === 'individual') return fail('CLASS_NOT_FOUND')
  return { ok: true as const, data: classView(row, true) }
}

export function getAdminClass(id: string) {
  watch()
  const row = db.classes.find((item) => item.id === id)
  if (!row) return fail('CLASS_NOT_FOUND')
  return { ok: true as const, data: classView(row) }
}

export function bookClass(id: string, body: { name?: string; phone?: string }) {
  const row = db.classes.find((item) => item.id === id && item.status === 'scheduled')
  if (!row || classKind(row) !== 'group') return fail('CLASS_NOT_FOUND')
  if (new Date(row.starts_at) <= new Date()) return fail('CLASS_STARTED')
  const nameMessage = nameError(body.name ?? '')
  if (body.name?.trim() && nameMessage) return failMessage(nameMessage)
  const phoneMessage = phoneError(body.phone ?? '')
  if (body.phone?.trim() && phoneMessage) return failMessage(phoneMessage)
  const client = devClient()
  const name = body.name?.trim() || client.name
  const phone = body.phone?.trim() || client.phone
  if (!name || !phone) return fail('VALIDATION')
  if (bookedCount(row.id) >= row.capacity) return fail('NO_PLACES')
  if (db.bookings.some((item) => item.class_id === row.id && item.client_id === client.id && item.status === 'booked')) {
    return fail('ALREADY_BOOKED')
  }
  if (client.policy === 'restricted' && balanceOf(client.id) <= 0) return fail('NO_CREDITS')
  if (body.name?.trim()) client.name = body.name.trim()
  if (body.phone?.trim()) client.phone = body.phone.trim()
  const booking = placeBooking(row, client)
  touch()
  return { ok: true as const, booking_id: booking.id, balance: balanceOf(client.id), class: classView(row) }
}

function visitedOf(clientId: string): boolean {
  const ids = new Set(db.bookings.filter((row) => row.client_id === clientId).map((row) => row.id))
  return db.attendances.some((row) => ids.has(row.booking_id))
}

export function getMe() {
  watch()
  const client = devClient()
  return {
    ok: true as const,
    pending: false,
    data: {
      name: client.name,
      phone: client.phone,
      username: client.username,
      balance: balanceOf(client.id),
      policy: client.policy,
      is_admin: ui.role === 'admin',
      is_trainer: ui.role === 'trainer',
      visited: visitedOf(client.id),
    },
  }
}

export function patchMe(body: { name?: string; phone?: string }) {
  const name = body.name ?? ''
  const phone = body.phone ?? ''
  const nameMessage = nameError(name)
  if (nameMessage) return failMessage(nameMessage)
  const phoneMessage = phoneError(phone)
  if (phoneMessage) return failMessage(phoneMessage)
  const client = devClient()
  client.name = name.trim()
  client.phone = phone.trim()
  touch()
  return getMe()
}

export function ping() {
  db.opens.push({
    id: uid('open'),
    messenger_user_id: devClient().telegram_id,
    opened_at: toTbilisiIso(new Date()),
  })
  touch()
  return { ok: true as const }
}

export function myBookings() {
  watch()
  const data = db.bookings
    .filter((row) => row.client_id === DEV_ID && row.status === 'booked')
    .map((row) => ({ booking: row, classRow: db.classes.find((item) => item.id === row.class_id) }))
    .filter((row): row is { booking: Booking; classRow: ClassRow } => {
      const classRow = row.classRow
      return Boolean(classRow && classRow.status === 'scheduled' && new Date(classRow.starts_at) > new Date())
    })
    .sort((a, b) => a.classRow.starts_at.localeCompare(b.classRow.starts_at))
    .map((row) => ({ booking_id: row.booking.id, class: classView(row.classRow) }))
  return { ok: true as const, data }
}

export function myVisits() {
  watch()
  const data = db.attendances
    .filter((row) => row.active)
    .flatMap((row) => {
      const booking = db.bookings.find((item) => item.id === row.booking_id && item.client_id === DEV_ID)
      const classRow = booking ? db.classes.find((item) => item.id === booking.class_id) : undefined
      if (!classRow) return []
      const trainer = db.trainers.find((item) => item.id === classRow.trainer_id)
      return [{ title: classRow.title, trainer_name: trainer?.name ?? '', starts_at: classRow.starts_at, payment: row.payment }]
    })
    .sort((a, b) => b.starts_at.localeCompare(a.starts_at) || b.title.localeCompare(a.title))
  return { ok: true as const, data, pending: false as const }
}

function ownBooking(id: string) {
  return db.bookings.find((row) => row.id === id && row.client_id === DEV_ID && row.status === 'booked')
}

export function cancelBooking(id: string) {
  const booking = ownBooking(id)
  if (!booking) return fail('BOOKING_NOT_FOUND')
  const row = db.classes.find((item) => item.id === booking.class_id)
  if (!row || hoursUntil(row.starts_at) <= 6) return fail('CUTOFF_EXCEEDED')
  booking.status = 'cancelled'
  booking.reminder_state = 'cancelled'
  if (classKind(row) === 'individual') row.status = 'cancelled'
  addEvent(booking.id, 'cancelled', booking.class_id, null)
  touch()
  return { ok: true as const, booking_id: booking.id }
}

export function rescheduleBooking(id: string, classId: string) {
  const booking = ownBooking(id)
  if (!booking) return fail('BOOKING_NOT_FOUND')
  const current = db.classes.find((item) => item.id === booking.class_id)
  if (!current || hoursUntil(current.starts_at) <= 6) return fail('CUTOFF_EXCEEDED')
  if (booking.class_id === classId) return fail('SAME_CLASS')
  if (current.template_id || classKind(current) !== 'group') return fail('VALIDATION')
  const next = db.classes.find((item) => item.id === classId && item.status === 'scheduled' && new Date(item.starts_at) > new Date())
  if (next && next.training_type_id !== current.training_type_id) return fail('NEW_CLASS_NOT_FOUND')
  if (!next) return fail('NEW_CLASS_NOT_FOUND')
  if (bookedCount(next.id) >= next.capacity) return fail('NO_PLACES')
  if (db.bookings.some((item) => item.class_id === next.id && item.client_id === DEV_ID && item.status === 'booked')) {
    return fail('ALREADY_BOOKED')
  }
  const oldId = booking.class_id
  booking.class_id = next.id
  booking.reminder_state = 'pending'
  booking.reminder_at = toTbilisiIso(new Date(new Date(next.starts_at).getTime() - 24 * 3600000))
  addEvent(booking.id, 'rescheduled', oldId, next.id)
  touch()
  return { ok: true as const, booking_id: booking.id, class: classView(next) }
}

function trainerCard(row: (typeof db.trainers)[number]) {
  return {
    id: row.id,
    name: row.name,
    active: row.active,
    about: row.about ?? '',
    phone: row.phone ?? '',
    telegram_user_id: row.telegram_user_id,
  }
}

export function listTrainers() {
  watch()
  return { ok: true as const, data: db.trainers.map(trainerCard) }
}

export function createTrainer(name: string) {
  const trimmed = name?.trim()
  if (!trimmed || db.trainers.some((row) => row.name === trimmed)) return fail('VALIDATION')
  const row = { id: uid('trainer'), name: trimmed, active: true, about: '', phone: '', telegram_user_id: null }
  db.trainers.push(row)
  touch()
  return { ok: true as const, data: trainerCard(row) }
}

export function patchTrainer(
  id: string,
  body: { name?: string; active?: boolean; about?: string; phone?: string; telegram_user_id?: string },
) {
  const row = db.trainers.find((item) => item.id === id)
  if (!row) return fail('VALIDATION')
  if (body.name !== undefined) {
    const trimmed = body.name.trim()
    if (!trimmed || db.trainers.some((item) => item.name === trimmed && item.id !== id)) return fail('VALIDATION')
    row.name = trimmed
  }
  if (typeof body.active === 'boolean') row.active = body.active
  if (body.about !== undefined) row.about = body.about.trim()
  if (body.phone !== undefined) {
    const message = phoneError(body.phone)
    if (message) return failMessage(message)
    row.phone = body.phone.trim()
  }
  if (body.telegram_user_id !== undefined) {
    const telegram = body.telegram_user_id.trim()
    if (telegram && !db.clients.some((item) => item.telegram_id === telegram)) return fail('VALIDATION')
    if (telegram && db.trainers.some((item) => item.telegram_user_id === telegram && item.id !== id)) return fail('VALIDATION')
    row.telegram_user_id = telegram || null
  }
  touch()
  return { ok: true as const, data: trainerCard(row) }
}

function trainerDetail(id: string) {
  const row = db.trainers.find((item) => item.id === id)
  if (!row) return null
  return {
    ...trainerCard(row),
    absences: db.absences.filter((item) => item.trainer_id === id).map((item) => ({ id: item.id, starts_on: item.starts_on, ends_on: item.ends_on })),
  }
}

function cancelFuture(trainerId: string) {
  for (const row of db.classes) {
    if (row.trainer_id !== trainerId || row.status !== 'scheduled' || new Date(row.starts_at) <= new Date()) continue
    row.status = 'cancelled'
    for (const booking of db.bookings) {
      if (booking.class_id === row.id && booking.status === 'booked') booking.status = 'cancelled'
    }
  }
}

export function getTrainer(id: string) {
  watch()
  const data = trainerDetail(id)
  if (!data) return fail('VALIDATION')
  return { ok: true as const, data, pending: false }
}

export function addAbsence(id: string, startsOn: string, endsOn: string) {
  if (!db.trainers.some((item) => item.id === id) || !startsOn || !endsOn || endsOn < startsOn) return fail('VALIDATION')
  db.absences.push({ id: uid('absence'), trainer_id: id, starts_on: startsOn, ends_on: endsOn })
  for (const row of db.classes) {
    if (row.trainer_id !== id || row.status !== 'scheduled') continue
    const day = row.starts_at.slice(0, 10)
    if (day >= startsOn && day <= endsOn) {
      row.status = 'cancelled'
      for (const booking of db.bookings) {
        if (booking.class_id === row.id && booking.status === 'booked') booking.status = 'cancelled'
      }
    }
  }
  touch()
  const data = trainerDetail(id)
  if (!data) return fail('VALIDATION')
  return { ok: true as const, data }
}

export function deleteAbsence(id: string, absenceId: string) {
  const index = db.absences.findIndex((item) => item.id === absenceId && item.trainer_id === id)
  if (index < 0) return fail('VALIDATION')
  db.absences.splice(index, 1)
  touch()
  const data = trainerDetail(id)
  if (!data) return fail('VALIDATION')
  return { ok: true as const, data }
}

export function cancelTrainerClasses(id: string) {
  if (!db.trainers.some((item) => item.id === id)) return fail('VALIDATION')
  cancelFuture(id)
  touch()
  return { ok: true as const }
}

export function deleteTrainer(id: string) {
  const row = db.trainers.find((item) => item.id === id)
  if (!row) return fail('VALIDATION')
  cancelFuture(id)
  for (const template of db.templates) {
    if (template.trainer_id === id) template.active = false
  }
  if (db.classes.some((item) => item.trainer_id === id)) {
    row.active = false
    row.telegram_user_id = null
  } else {
    db.templates.splice(0, db.templates.length, ...db.templates.filter((item) => item.trainer_id !== id))
    db.absences.splice(0, db.absences.length, ...db.absences.filter((item) => item.trainer_id !== id))
    db.trainers.splice(db.trainers.indexOf(row), 1)
  }
  touch()
  return { ok: true as const }
}

function enrollmentRows() {
  return db.templates
    .filter((template) => template.active && db.trainers.some((trainer) => trainer.id === template.trainer_id && trainer.active))
    .map((template) => ({
      template_id: template.id,
      trainer_id: template.trainer_id,
      trainer_name: db.trainers.find((trainer) => trainer.id === template.trainer_id)?.name ?? '',
      title: template.title,
      weekday: template.weekday,
      start_time: template.start_time,
      enrolled: ui.enrolled.includes(template.id),
    }))
    .sort((a, b) => a.trainer_name.localeCompare(b.trainer_name) || a.weekday - b.weekday || a.start_time.localeCompare(b.start_time))
}

export function listEnrollments() {
  watch()
  return { ok: true as const, data: enrollmentRows(), pending: false }
}

export function putEnrollments(templateIds: string[]) {
  const allowed = new Set(enrollmentRows().map((item) => item.template_id))
  if (templateIds.some((id) => !allowed.has(id))) return fail('VALIDATION')
  ui.enrolled = [...new Set(templateIds)]
  touch()
  return { ok: true as const, data: enrollmentRows() }
}

export function getTrainerProfile() {
  watch()
  const row = db.trainers.find((item) => item.id === STUB_TRAINER)
  if (!row) return { ok: true as const, pending: false, data: { id: '', name: '', active: true, about: '', phone: '', telegram_user_id: null } }
  return { ok: true as const, pending: false, data: trainerCard(row) }
}

export function patchTrainerProfile(body: { about: string; phone: string }) {
  const row = db.trainers.find((item) => item.id === STUB_TRAINER)
  if (!row) return fail('FORBIDDEN')
  const message = phoneError(body.phone)
  if (message) return failMessage(message)
  row.about = body.about.trim()
  row.phone = body.phone.trim()
  touch()
  return { ok: true as const, data: trainerCard(row) }
}

export function getTrainerSchedule(from: string, to: string) {
  watch()
  const data = db.classes
    .filter((row) => (row.trainer_id === STUB_TRAINER || (classKind(row) === 'hold' && row.status === 'scheduled')) && inRange(row.starts_at, from, to))
    .sort((a, b) => a.starts_at.localeCompare(b.starts_at))
    .map((row) => classView(row))
  return { ok: true as const, data, pending: false }
}

export function listTrainerTypes() {
  return listTypes()
}

export function createTrainerClass(body: { training_type_id?: string; starts_at?: string; capacity?: number }) {
  return createClass({ ...body, trainer_id: STUB_TRAINER })
}

export function cancelTrainerClass(id: string) {
  const row = db.classes.find((item) => item.id === id && item.trainer_id === STUB_TRAINER)
  if (!row) return fail('CLASS_NOT_FOUND')
  if (new Date(row.starts_at) <= new Date()) return fail('CLASS_STARTED')
  return patchClass(id, { status: 'cancelled' })
}

export function getTrainerRoster(id: string) {
  const row = db.classes.find((item) => item.id === id && item.trainer_id === STUB_TRAINER && item.status === 'scheduled')
  if (!row) return fail('CLASS_NOT_FOUND')
  return getRoster(id)
}

export function setTrainerAttendance(bookingId: string, arrived: boolean) {
  const booking = db.bookings.find((item) => item.id === bookingId)
  const row = booking && db.classes.find((item) => item.id === booking.class_id)
  if (!row || row.trainer_id !== STUB_TRAINER) return fail('BOOKING_NOT_FOUND')
  return setAttendance(bookingId, arrived)
}

export function trainerStats(from: string, to: string) {
  watch()
  const classIds = new Set(db.classes.filter((row) => row.trainer_id === STUB_TRAINER && inRange(row.starts_at, from, to)).map((row) => row.id))
  const bookingIds = db.bookings.filter((row) => classIds.has(row.class_id))
  const clients = new Set(
    db.attendances
      .filter((row) => row.active && bookingIds.some((booking) => booking.id === row.booking_id))
      .map((row) => bookingIds.find((booking) => booking.id === row.booking_id)?.client_id)
      .filter((id): id is string => Boolean(id)),
  )
  const attends = db.ledger.filter((row) => row.reason === 'attend' && row.booking_id && bookingIds.some((booking) => booking.id === row.booking_id)).length
  return { ok: true as const, data: { from, to, clients: clients.size, attends } }
}

export function listTypes() {
  watch()
  return {
    ok: true as const,
    data: db.types.map((row) => ({
      id: row.id,
      title: row.title,
      duration_minutes: row.duration_minutes,
      capacity: row.capacity,
      price_min: row.price_min,
      price_max: row.price_max,
      active: row.active,
      color: row.color,
    })),
  }
}

export function createType(body: { title?: string; duration_minutes?: number; capacity?: number; price_min?: number | string; price_max?: number | string }) {
  const title = body.title?.trim()
  if (!title || db.types.some((row) => row.title === title)) return fail('VALIDATION')
  const duration = body.duration_minutes
  const places = body.capacity
  if (duration == null || places == null || duration <= 0 || places <= 0) return fail('VALIDATION')
  if (body.price_min == null || body.price_max == null) return fail('VALIDATION')
  if (Number(body.price_max) < Number(body.price_min)) return fail('VALIDATION')
  const row = {
    id: uid('type'),
    title,
    duration_minutes: duration,
    capacity: places,
    price_min: money(body.price_min),
    price_max: money(body.price_max),
    active: true,
    color: nextTypeColor(db.types.map((item) => item.color)),
  }
  db.types.push(row)
  touch()
  return { ok: true as const, data: { ...row } }
}

export function patchType(id: string, body: { title?: string; duration_minutes?: number; capacity?: number; price_min?: number | string; price_max?: number | string; active?: boolean }) {
  const row = db.types.find((item) => item.id === id)
  if (!row) return fail('VALIDATION')
  if (body.title?.trim()) row.title = body.title.trim()
  if (body.duration_minutes) row.duration_minutes = Number(body.duration_minutes)
  if (body.capacity) row.capacity = Number(body.capacity)
  if (body.price_min != null) row.price_min = money(body.price_min)
  if (body.price_max != null) row.price_max = money(body.price_max)
  if (Number(row.price_max) < Number(row.price_min)) return fail('VALIDATION')
  if (typeof body.active === 'boolean') row.active = body.active
  touch()
  return { ok: true as const, data: { ...row } }
}

export function listTemplates() {
  watch()
  return {
    ok: true as const,
    data: db.templates.map((row) => ({
      id: row.id,
      trainer_id: row.trainer_id,
      training_type_id: row.training_type_id,
      title: row.title,
      weekday: row.weekday,
      start_time: row.start_time,
      capacity: row.capacity,
      active: row.active,
    })),
  }
}

export function createTemplate(body: { trainer_id?: string; training_type_id?: string; weekday?: number; start_time?: string; capacity?: number }) {
  const type = db.types.find((item) => item.id === body.training_type_id)
  const trainer = db.trainers.find((item) => item.id === body.trainer_id)
  if (!type || !trainer || body.weekday == null || !body.start_time) return fail('VALIDATION')
  const capacity = Number(body.capacity || type.capacity)
  const row = {
    id: uid('template'),
    trainer_id: trainer.id,
    training_type_id: type.id,
    title: type.title,
    weekday: body.weekday,
    start_time: body.start_time,
    capacity,
    active: true,
  }
  if (db.templates.some((item) => item.trainer_id === row.trainer_id && item.title === row.title && item.weekday === row.weekday && item.start_time === row.start_time)) {
    return fail('VALIDATION')
  }
  db.templates.push(row)
  touch()
  return { ok: true as const, data: { ...row } }
}

export function patchTemplate(id: string, body: { weekday?: number; start_time?: string; capacity?: number; trainer_id?: string; training_type_id?: string; active?: boolean }) {
  const row = db.templates.find((item) => item.id === id)
  if (!row) return fail('VALIDATION')
  if (body.weekday) row.weekday = Number(body.weekday)
  if (body.start_time) row.start_time = body.start_time
  if (body.capacity) row.capacity = Number(body.capacity)
  if (body.trainer_id) row.trainer_id = body.trainer_id
  if (body.training_type_id) {
    const type = db.types.find((item) => item.id === body.training_type_id)
    if (!type) return fail('VALIDATION')
    row.training_type_id = type.id
    row.title = type.title
  }
  if (typeof body.active === 'boolean') row.active = body.active
  touch()
  return { ok: true as const, data: { ...row } }
}

export function generateWeek(weekStart?: string) {
  const monday = weekStart ? mondayKey(weekStart) : addDays(mondayKey(todayKey()), 7)
  let created = 0
  let updated = 0
  for (const template of db.templates) {
    const trainer = db.trainers.find((item) => item.id === template.trainer_id)
    const type = db.types.find((item) => item.id === template.training_type_id)
    if (!template.active || !trainer?.active || !type) continue
    const date = addDays(monday, template.weekday - 1)
    const starts = `${date}T${template.start_time}:00+04:00`
    if (new Date(starts) <= new Date()) continue
    const ends = toTbilisiIso(new Date(new Date(starts).getTime() + type.duration_minutes * 60000))
    const existing = db.classes.find((item) => item.trainer_id === template.trainer_id && item.starts_at === starts)
    if (!existing) {
      if (studioBusy(starts, ends)) continue
      db.classes.push({
        id: uid('class'),
        template_id: template.id,
        trainer_id: trainer.id,
        training_type_id: type.id,
        title: template.title,
        starts_at: starts,
        ends_at: ends,
        capacity: template.capacity,
        price_min: type.price_min,
        price_max: type.price_max,
        status: 'scheduled' as const,
      })
      created += 1
    } else if (existing.status === 'scheduled' && classKind(existing) === 'group' && !studioBusy(starts, ends, existing.id)) {
      existing.template_id = template.id
      existing.title = template.title
      existing.ends_at = ends
      existing.capacity = Math.max(template.capacity, bookedCount(existing.id))
      existing.price_min = type.price_min
      existing.price_max = type.price_max
      existing.training_type_id = type.id
      updated += 1
    }
  }
  touch()
  return { ok: true as const, week_start: monday, week_end: addDays(monday, 6), created, updated }
}

export function createClass(body: { trainer_id?: string; training_type_id?: string; starts_at?: string; capacity?: number }) {
  const trainer = db.trainers.find((item) => item.id === body.trainer_id)
  const type = db.types.find((item) => item.id === body.training_type_id)
  if (!trainer || !type || !body.starts_at) return fail('VALIDATION')
  if (db.classes.some((item) => item.trainer_id === trainer.id && item.starts_at === body.starts_at)) return fail('VALIDATION')
  const capacity = Number(body.capacity || type.capacity)
  const ends = toTbilisiIso(new Date(new Date(body.starts_at).getTime() + type.duration_minutes * 60000))
  if (studioBusy(body.starts_at, ends)) return fail('STUDIO_BUSY')
  const row = {
    id: uid('class'),
    template_id: null,
    trainer_id: trainer.id,
    training_type_id: type.id,
    title: type.title,
    starts_at: body.starts_at,
    ends_at: ends,
    capacity,
    price_min: type.price_min,
    price_max: type.price_max,
    status: 'scheduled' as const,
  }
  db.classes.push(row)
  touch()
  return { ok: true as const, data: classView(row) }
}

function spanEnd(starts: string, minutes: number) {
  return toTbilisiIso(new Date(new Date(starts).getTime() + minutes * 60000))
}

export function createHold(body: { title: string; starts_at: string; duration_minutes?: number }) {
  const title = body.title.trim()
  const minutes = body.duration_minutes ?? 60
  if (!title || !body.starts_at || minutes < 1 || minutes > 600) return fail('VALIDATION')
  const ends = spanEnd(body.starts_at, minutes)
  if (studioBusy(body.starts_at, ends)) return fail('STUDIO_BUSY')
  const row: ClassRow = {
    id: uid('class'),
    template_id: null,
    trainer_id: '',
    training_type_id: '',
    title,
    starts_at: body.starts_at,
    ends_at: ends,
    capacity: 0,
    price_min: '',
    price_max: '',
    status: 'scheduled',
    kind: 'hold',
  }
  db.classes.push(row)
  touch()
  return { ok: true as const, data: classView(row) }
}

export function createIndividual(body: { trainer_id: string; client_id: string; starts_at: string; duration_minutes?: number }) {
  const trainer = db.trainers.find((item) => item.id === body.trainer_id)
  const client = db.clients.find((item) => item.id === body.client_id)
  const minutes = body.duration_minutes ?? 60
  if (!trainer || !client || !body.starts_at || minutes < 1 || minutes > 600) return fail('VALIDATION')
  const ends = spanEnd(body.starts_at, minutes)
  if (studioBusy(body.starts_at, ends)) return fail('STUDIO_BUSY')
  const row: ClassRow = {
    id: uid('class'),
    template_id: null,
    trainer_id: trainer.id,
    training_type_id: '',
    title: 'Индивидуальное',
    starts_at: body.starts_at,
    ends_at: ends,
    capacity: 1,
    price_min: '',
    price_max: '',
    status: 'scheduled',
    kind: 'individual',
  }
  db.classes.push(row)
  placeBooking(row, client)
  touch()
  return { ok: true as const, data: classView(row) }
}

export function searchTrainerClients(query: string) {
  watch()
  const needle = query.trim().toLowerCase()
  if (!needle) return { ok: true as const, data: [] as { id: string; name: string }[], pending: false as const }
  const data = db.clients
    .filter((client) => client.name.toLowerCase().includes(needle))
    .slice(0, 20)
    .map((client) => ({ id: client.id, name: client.name }))
  return { ok: true as const, data, pending: false as const }
}

export function createTrainerIndividual(body: { client_id: string; starts_at: string; duration_minutes?: number }) {
  return createIndividual({ ...body, trainer_id: STUB_TRAINER })
}

export function moveTrainerClass(id: string, startsAt: string) {
  const row = db.classes.find((item) => item.id === id && item.trainer_id === STUB_TRAINER && classKind(item) === 'individual')
  if (!row) return fail('CLASS_NOT_FOUND')
  return patchClass(id, { starts_at: startsAt })
}

export function patchClass(id: string, body: { status?: 'cancelled'; starts_at?: string; trainer_id?: string; capacity?: number; title?: string }) {
  const row = db.classes.find((item) => item.id === id && item.status === 'scheduled')
  if (!row) return fail('CLASS_NOT_FOUND')
  if (body.status === 'cancelled') {
    row.status = 'cancelled'
    for (const booking of db.bookings) {
      if (booking.class_id === row.id && booking.status === 'booked') {
        booking.status = 'cancelled'
        booking.reminder_state = 'cancelled'
        addEvent(booking.id, 'cancelled', row.id, null)
      }
    }
    touch()
    return { ok: true as const, data: classView(row) }
  }
  const booked = bookedCount(row.id)
  if (body.capacity != null && Number(body.capacity) < booked) return fail('VALIDATION')
  if (classKind(row) === 'hold' && (body.capacity != null || body.trainer_id)) return fail('VALIDATION')
  if (classKind(row) === 'individual' && body.capacity != null && Number(body.capacity) !== 1) return fail('VALIDATION')
  const previousStart = row.starts_at
  let nextStart = row.starts_at
  let nextEnd = row.ends_at
  if (body.starts_at) {
    const duration = new Date(row.ends_at).getTime() - new Date(row.starts_at).getTime()
    nextStart = body.starts_at
    nextEnd = toTbilisiIso(new Date(new Date(body.starts_at).getTime() + duration))
  }
  if (studioBusy(nextStart, nextEnd, row.id)) return fail('STUDIO_BUSY')
  row.starts_at = nextStart
  row.ends_at = nextEnd
  if (body.trainer_id) row.trainer_id = body.trainer_id
  if (body.capacity != null) row.capacity = Number(body.capacity)
  if (body.title?.trim()) row.title = body.title.trim()
  if (row.starts_at !== previousStart) {
    for (const booking of db.bookings) {
      if (booking.class_id !== row.id || booking.status !== 'booked') continue
      const remindAt = new Date(new Date(row.starts_at).getTime() - 24 * 3600000)
      booking.reminder_at = toTbilisiIso(remindAt)
      if (booking.reminder_state === 'sent' && remindAt > new Date()) booking.reminder_state = 'pending'
    }
  }
  touch()
  return { ok: true as const, data: classView(row) }
}

export function getRoster(id: string) {
  watch()
  const row = db.classes.find((item) => item.id === id && item.status === 'scheduled')
  if (!row) return fail('CLASS_NOT_FOUND')
  const bookings = db.bookings
    .filter((item) => item.class_id === id && item.status === 'booked')
    .flatMap((item): RosterBooking[] => {
      const client = db.clients.find((person) => person.id === item.client_id)
      if (!client) return []
      const attendance = activeAttendance(item.id)
      return [{
        booking_id: item.id,
        client_id: client.id,
        name: client.name,
        phone: client.phone,
        username: client.username,
        attendance: attendance ? { payment: attendance.payment, marked_at: attendance.marked_at } : null,
      }]
    })
  return { ok: true as const, data: { class: classView(row), bookings } }
}

export function setAttendance(bookingId: string, arrived: boolean) {
  const booking = db.bookings.find((item) => item.id === bookingId && item.status === 'booked')
  if (!booking) return fail('BOOKING_NOT_FOUND')
  const row = db.classes.find((item) => item.id === booking.class_id && item.status === 'scheduled')
  if (!row) return fail('BOOKING_NOT_FOUND')
  const current = activeAttendance(booking.id)
  if (arrived && hoursUntil(row.starts_at) > 0.5) return fail('ATTENDANCE_TOO_EARLY')
  if (arrived && current) {
    touch()
    return attendanceResult(booking, current)
  }
  if (!arrived && !current) {
    touch()
    return { ok: true as const, booking_id: booking.id, attendance: null, balance: balanceOf(booking.client_id) }
  }
  if (arrived) {
    const before = balanceOf(booking.client_id)
    const entry: LedgerEntry = {
      id: uid('ledger'),
      client_id: booking.client_id,
      delta: -1,
      reason: 'attend',
      booking_id: booking.id,
      note: '',
      actor_telegram_id: 'admin',
      created_at: toTbilisiIso(new Date()),
    }
    db.ledger.push(entry)
    const attendance: Attendance = {
      id: uid('attendance'),
      booking_id: booking.id,
      active: true,
      payment: before > 0 ? 'paid' : 'unpaid',
      actor_role: 'admin',
      actor_telegram_id: 'admin',
      marked_at: toTbilisiIso(new Date()),
      ledger_entry_id: entry.id,
    }
    db.attendances.push(attendance)
    touch()
    return attendanceResult(booking, attendance)
  }
  if (!current) return fail('BOOKING_NOT_FOUND')
  current.active = false
  db.ledger.push({
    id: uid('ledger'),
    client_id: booking.client_id,
    delta: 1,
    reason: 'reversal' as const,
    booking_id: booking.id,
    note: '',
    actor_telegram_id: 'admin',
    created_at: toTbilisiIso(new Date()),
  })
  touch()
  return { ok: true as const, booking_id: booking.id, attendance: null, balance: balanceOf(booking.client_id) }
}

function attendanceResult(booking: Booking, attendance: Attendance) {
  return {
    ok: true as const,
    booking_id: booking.id,
    attendance: { payment: attendance.payment, marked_at: attendance.marked_at },
    balance: balanceOf(booking.client_id),
  }
}

function clientSummary(client: Client): ClientSummary {
  return {
    id: client.id,
    telegram_id: client.telegram_id,
    username: client.username,
    name: client.name,
    phone: client.phone,
    policy: client.policy,
    balance: balanceOf(client.id),
  }
}

export function listClients(query = ''): { ok: true; data: ClientSummary[] } {
  watch()
  const needle = query.trim().toLowerCase()
  const data = db.clients
    .filter((client) => {
      if (!needle) return true
      return [client.name, client.phone, client.username, client.telegram_id].join(' ').toLowerCase().includes(needle)
    })
    .map(clientSummary)
  return { ok: true as const, data }
}

export function getClient(id: string) {
  watch()
  const client = db.clients.find((item) => item.id === id)
  if (!client) return fail('VALIDATION')
  const ledger = db.ledger
    .filter((row) => row.client_id === id)
    .sort((a, b) => b.created_at.localeCompare(a.created_at))
    .map((row) => ({
      id: row.id,
      delta: row.delta,
      reason: row.reason,
      note: row.note,
      booking_id: row.booking_id,
      created_at: row.created_at,
    }))
  return { ok: true as const, data: { ...clientSummary(client), ledger } }
}

export function patchClient(id: string, policy: BookingPolicy) {
  const client = db.clients.find((item) => item.id === id)
  if (!client) return fail('VALIDATION')
  client.policy = policy
  touch()
  return getClient(id)
}

export function grantCredits(id: string, delta: number, note?: string) {
  const client = db.clients.find((item) => item.id === id)
  const amount = Number(delta)
  if (!client || !Number.isInteger(amount) || amount < 1) return fail('VALIDATION')
  db.ledger.push({
    id: uid('ledger'),
    client_id: id,
    delta: amount,
    reason: 'grant' as const,
    booking_id: null,
    note: note?.trim() ?? '',
    actor_telegram_id: 'admin',
    created_at: toTbilisiIso(new Date()),
  })
  touch()
  return getClient(id)
}

export function stats(from: string, to: string): { ok: true; data: Stats } {
  watch()
  const users = new Set(db.opens.filter((row) => inRange(row.opened_at, from, to)).map((row) => row.messenger_user_id))
  const arrivals = db.attendances.filter((row) => row.active && inRange(row.marked_at, from, to))
  const noShows = db.bookings.filter((booking) => {
    if (booking.status !== 'booked' || activeAttendance(booking.id)) return false
    const row = db.classes.find((item) => item.id === booking.class_id)
    return row && row.status !== 'cancelled' && new Date(row.ends_at) < new Date() && inRange(row.ends_at, from, to)
  })
  return {
    ok: true as const,
    data: {
      from,
      to,
      unique_users: users.size,
      bookings_created: db.events.filter((row) => row.event_type === 'created' && inRange(row.created_at, from, to)).length,
      bookings_cancelled: db.events.filter((row) => row.event_type === 'cancelled' && inRange(row.created_at, from, to)).length,
      arrivals: arrivals.length,
      no_shows: noShows.length,
      unpaid_visits: arrivals.filter((row) => row.payment === 'unpaid').length,
    },
  }
}

export function clientsCsv() {
  watch()
  const lines = ['name,phone,username,telegram_id,balance,policy']
  for (const client of db.clients) {
    const row = clientSummary(client)
    lines.push([row.name, row.phone, row.username, row.telegram_id, row.balance, row.policy].map(csvCell).join(','))
  }
  return lines.join('\n')
}

function csvCell(value: string | number): string {
  const text = String(value ?? '')
  if (/[",\n]/.test(text)) return `"${text.replaceAll('"', '""')}"`
  return text
}

export function putStudio(body: Partial<StudioProfile>) {
  const keys = ['address', 'phone', 'instagram_url', 'contact_url', 'map_url', 'guide', 'pricing'] as const
  for (const key of keys) {
    if (typeof body[key] !== 'string') return fail('VALIDATION')
  }
  const guide = body.guide?.trim() ?? ''
  if ([...guide].length > 4000) return failMessage('Памятка слишком длинная')
  const message = pricingError(body.pricing ?? '')
  if (message) return failMessage(message)
  db.studio.address = body.address ?? ''
  db.studio.phone = body.phone ?? ''
  db.studio.instagram_url = body.instagram_url ?? ''
  db.studio.contact_url = body.contact_url ?? ''
  db.studio.map_url = body.map_url ?? ''
  db.studio.guide = guide
  db.studio.pricing = canonicalPricing(body.pricing ?? '')
  touch()
  return getStudio()
}

export function listPublicTrainers() {
  watch()
  const data = db.trainers
    .filter((row) => row.active)
    .map(publicTrainer)
    .sort((a, b) => a.name.localeCompare(b.name, 'ru') || a.id.localeCompare(b.id))
  return { ok: true as const, data, pending: false }
}

export function getPublicTrainer(id: string) {
  watch()
  const row = db.trainers.find((item) => item.id === id && item.active)
  if (!row) return { ...failMessage('Тренер не найден'), pending: false }
  return { ok: true as const, data: publicTrainer(row), pending: false }
}

function publicTrainer(row: (typeof db.trainers)[number]): PublicTrainer {
  const classes = [
    ...new Set(db.templates.filter((item) => item.active && item.trainer_id === row.id).map((item) => item.title)),
  ].sort((a, b) => a.localeCompare(b, 'ru'))
  return { id: row.id, name: row.name, about: row.about, phone: row.phone, classes }
}

export function todayRoster(): RosterData[] {
  watch()
  const today = todayKey()
  return db.classes
    .filter((row) => row.status === 'scheduled' && classKind(row) !== 'hold' && dateKeyOfIso(row.starts_at) === today)
    .sort((a, b) => a.starts_at.localeCompare(b.starts_at))
    .flatMap((row) => {
      const roster = getRoster(row.id)
      return roster.ok ? [roster.data] : []
    })
}

function addClient(fields: Partial<Client> & Pick<Client, 'id' | 'telegram_id' | 'name'>): void {
  db.clients.push({
    username: '',
    phone: '',
    policy: 'postpay',
    ...fields,
  })
}

function seed() {
  const trainers: Array<[string, string]> = [
    ['tr-zhanna', 'Жанна'],
    ['tr-lilia', 'Лилия'],
    ['tr-tata', 'Тата'],
    ['tr-tatiana', 'Татьяна'],
    ['tr-taya', 'Тая'],
  ]
  for (const [id, name] of trainers) db.trainers.push({ id, name, active: true, about: '', phone: '', telegram_user_id: null })

  const types: Array<[string, string, number, number, number, number, string]> = [
    ['ty-kids', 'K-POP KIDS (Dance)', 60, 10, 30, 30, '#1d4ed8'],
    ['ty-kpop', 'K-POP 16+', 60, 9, 17, 35, '#7c3aed'],
    ['ty-yoga', 'Aero Yoga', 60, 6, 17, 40, '#4d7c0f'],
    ['ty-stretch', 'DYNAMIC STRETCHING', 55, 9, 17, 35, '#0e7490'],
    ['ty-heels', 'HIGH HEELS', 60, 9, 0, 0, '#c2410c'],
    ['ty-aero', 'Aero Stretching', 60, 6, 17, 40, '#be123c'],
  ]
  for (const [id, title, duration, capacity, min, max, color] of types) {
    db.types.push({
      id,
      title,
      duration_minutes: duration,
      capacity,
      price_min: money(min),
      price_max: money(max),
      active: true,
      color,
    })
  }

  const slots: Array<[string, string, number, string, number]> = [
    ['tr-tatiana', 'ty-aero', 2, '10:00', 6],
    ['tr-taya', 'ty-kids', 2, '17:00', 10],
    ['tr-taya', 'ty-kpop', 2, '18:00', 9],
    ['tr-tata', 'ty-stretch', 2, '20:00', 9],
    ['tr-tatiana', 'ty-aero', 4, '10:00', 6],
    ['tr-taya', 'ty-kids', 4, '17:00', 10],
    ['tr-taya', 'ty-kpop', 4, '18:00', 9],
    ['tr-lilia', 'ty-yoga', 4, '19:00', 6],
    ['tr-tata', 'ty-stretch', 4, '20:00', 9],
    ['tr-zhanna', 'ty-heels', 5, '11:30', 9],
    ['tr-taya', 'ty-kids', 6, '11:00', 10],
    ['tr-lilia', 'ty-yoga', 6, '12:00', 6],
    ['tr-taya', 'ty-kpop', 6, '13:00', 9],
  ]
  for (const [trainerId, typeId, weekday, start, capacity] of slots) {
    const type = db.types.find((item) => item.id === typeId)
    if (!type) throw new Error(typeId)
    db.templates.push({
      id: uid('template'),
      trainer_id: trainerId,
      training_type_id: typeId,
      title: type.title,
      weekday,
      start_time: start,
      capacity,
      active: true,
    })
  }

  addClient({ id: DEV_ID, telegram_id: '1001', username: 'anna', name: 'Анна', phone: '+995555112233', policy: 'restricted' })
  addClient({ id: 'client-maria', telegram_id: '1002', username: 'maria', name: 'Мария', phone: '+995555000002', policy: 'postpay' })
  addClient({ id: 'client-nino', telegram_id: '1003', username: 'nino', name: 'Нино', phone: '+995555000003', policy: 'postpay' })
  db.ledger.push({
    id: uid('ledger'),
    client_id: 'client-nino',
    delta: 4,
    reason: 'grant' as const,
    booking_id: null,
    note: 'пакет',
    actor_telegram_id: 'admin',
    created_at: toTbilisiIso(new Date()),
  })

  const monday = mondayKey(todayKey())
  generateWeek(monday)
  generateWeek(addDays(monday, 7))

  const heels = db.classes.find((row) => row.title === 'HIGH HEELS' && new Date(row.starts_at) > new Date())
  const maria = db.clients.find((item) => item.id === 'client-maria')
  if (heels && maria) {
    heels.capacity = 1
    placeBooking(heels, maria, { seed: true })
  }

  const soon = new Date(Date.now() + 20 * 60 * 1000)
  const mid = new Date(Date.now() + 3 * 60 * 60 * 1000)
  const yoga = db.types.find((item) => item.id === 'ty-yoga')
  const stretch = db.types.find((item) => item.id === 'ty-stretch')
  if (!yoga || !stretch) throw new Error('seed types')
  const demos: Array<[string, TrainingType, Date]> = [
    ['tr-lilia', yoga, soon],
    ['tr-tata', stretch, mid],
  ]
  for (const [trainerId, type, when] of demos) {
    const starts = toTbilisiIso(when)
    const row = {
      id: uid('class'),
      template_id: null,
      trainer_id: trainerId,
      training_type_id: type.id,
      title: type.title,
      starts_at: starts,
      ends_at: toTbilisiIso(new Date(when.getTime() + type.duration_minutes * 60000)),
      capacity: type.capacity,
      price_min: type.price_min,
      price_max: type.price_max,
      status: 'scheduled' as const,
    }
    db.classes.push(row)
    placeBooking(row, devClient(), { seed: true })
  }
}

seed()
