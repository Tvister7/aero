export type BookingPolicy = 'postpay' | 'restricted'
export type ClassStatus = 'scheduled' | 'cancelled'
export type BookingStatus = 'booked' | 'cancelled'
export type ReminderState = 'pending' | 'sent' | 'cancelled' | 'skipped'
export type LedgerReason = 'grant' | 'attend' | 'reversal'
export type AttendancePayment = 'paid' | 'unpaid'
export type ActorRole = 'admin' | 'trainer'

export type ErrorCode =
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'VALIDATION'
  | 'CLASS_NOT_FOUND'
  | 'NEW_CLASS_NOT_FOUND'
  | 'BOOKING_NOT_FOUND'
  | 'CLASS_STARTED'
  | 'NO_PLACES'
  | 'ALREADY_BOOKED'
  | 'CUTOFF_EXCEEDED'
  | 'SAME_CLASS'
  | 'NO_CREDITS'
  | 'ATTENDANCE_TOO_EARLY'

export type ApiError = { ok: false; error_code: ErrorCode; message: string }

export type StudioProfile = {
  address: string
  phone: string
  instagram_url: string
  contact_url: string
  map_url: string
}

export type Trainer = {
  id: string
  name: string
  active: boolean
  telegram_user_id: string | null
}

export type TrainingType = {
  id: string
  title: string
  duration_minutes: number
  capacity: number
  price_min: string
  price_max: string
  active: boolean
  color: string
}

export type ClassTemplate = {
  id: string
  trainer_id: string
  training_type_id: string
  title: string
  weekday: number
  start_time: string
  capacity: number
  active: boolean
}

export type ClassRow = {
  id: string
  template_id: string | null
  trainer_id: string
  training_type_id: string
  title: string
  starts_at: string
  ends_at: string
  capacity: number
  price_min: string
  price_max: string
  status: ClassStatus
}

export type ClassView = {
  id: string
  title: string
  trainer: { id: string; name: string }
  starts_at: string
  ends_at: string
  capacity: number
  booked: number
  free_places: number
  price_min: string
  price_max: string
  color: string
  status: ClassStatus
}

export type Client = {
  id: string
  telegram_id: string
  username: string
  name: string
  phone: string
  policy: BookingPolicy
}

export type Booking = {
  id: string
  class_id: string
  client_id: string
  status: BookingStatus
  reminder_state: ReminderState
  reminder_at: string
  created_at: string
}

export type BookingEvent = {
  id: string
  booking_id: string
  event_type: 'created' | 'cancelled' | 'rescheduled'
  old_class_id: string | null
  new_class_id: string | null
  created_at: string
}

export type LedgerEntry = {
  id: string
  client_id: string
  delta: number
  reason: LedgerReason
  booking_id: string | null
  note: string
  actor_telegram_id: string
  created_at: string
}

export type Attendance = {
  id: string
  booking_id: string
  active: boolean
  payment: AttendancePayment
  actor_role: ActorRole
  actor_telegram_id: string
  marked_at: string
  ledger_entry_id: string
}

export type AppOpen = {
  id: string
  messenger_user_id: string
  opened_at: string
}

export type Database = {
  trainers: Trainer[]
  types: TrainingType[]
  templates: ClassTemplate[]
  classes: ClassRow[]
  clients: Client[]
  bookings: Booking[]
  events: BookingEvent[]
  ledger: LedgerEntry[]
  attendances: Attendance[]
  opens: AppOpen[]
  studio: StudioProfile
}

export type TrainerView = Pick<Trainer, 'id' | 'name' | 'active'>

export type ClientSummary = {
  id: string
  telegram_id: string
  username: string
  name: string
  phone: string
  policy: BookingPolicy
  balance: number
}

export type LedgerView = {
  id: string
  delta: number
  reason: LedgerReason
  note: string
  booking_id: string | null
  created_at: string
}

export type AttendanceView = { payment: AttendancePayment; marked_at: string } | null

export type RosterBooking = {
  booking_id: string
  client_id: string
  name: string
  phone: string
  username: string
  attendance: AttendanceView
}

export type RosterData = { class: ClassView; bookings: RosterBooking[] }

export type Me = {
  name: string
  phone: string
  username: string
  balance: number
  policy: BookingPolicy
  is_admin: boolean
  visited: boolean
}

export type Stats = {
  from: string
  to: string
  unique_users: number
  bookings_created: number
  bookings_cancelled: number
  arrivals: number
  no_shows: number
  unpaid_visits: number
}
