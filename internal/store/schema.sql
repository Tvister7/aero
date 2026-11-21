-- Перечисления. Повторный запуск пропускает уже созданные типы.

create type class_status as enum ('scheduled', 'cancelled');

create type class_kind as enum ('group', 'individual', 'hold');

create type client_policy as enum ('postpay', 'restricted');

create type booking_status as enum ('booked', 'cancelled');

create type reminder_state as enum ('pending', 'sent', 'cancelled', 'skipped');

create type booking_event_type as enum ('created', 'cancelled', 'rescheduled');

create type credit_reason as enum ('grant', 'attend', 'reversal');

create type payment_status as enum ('paid', 'unpaid');

create type actor_role as enum ('admin', 'trainer');

create type notice_kind as enum ('booked_empty', 'skipped_empty');

create type notice_state as enum ('pending', 'sent', 'skipped');

create table if not exists trainers (
  id uuid primary key,
  name text not null unique,
  active boolean not null,
  telegram_user_id text unique,
  about text not null default '',
  phone text not null default '',
  created_at timestamptz not null
);

create table if not exists training_types (
  id uuid primary key,
  title text not null unique,
  duration_minutes integer not null check (duration_minutes between 1 and 600),
  capacity integer not null check (capacity > 0),
  price_min text not null,
  price_max text not null,
  active boolean not null,
  color text not null default '',
  created_at timestamptz not null
);

create table if not exists class_templates (
  id uuid primary key,
  trainer_id uuid not null references trainers(id),
  training_type_id uuid not null references training_types(id),
  title text not null,
  weekday integer not null check (weekday between 1 and 7),
  start_time time not null,
  capacity integer not null check (capacity > 0),
  active boolean not null,
  created_at timestamptz not null,
  unique (trainer_id, title, weekday, start_time)
);

create table if not exists classes (
  id uuid primary key,
  template_id uuid references class_templates(id),
  trainer_id uuid references trainers(id),
  training_type_id uuid references training_types(id),
  title text not null,
  starts_at timestamptz not null,
  ends_at timestamptz not null,
  capacity integer not null check (capacity >= 0),
  price_min text not null,
  price_max text not null,
  status class_status not null,
  kind class_kind not null default 'group',
  created_at timestamptz not null,
  check (ends_at > starts_at),
  check (
    (kind = 'group' and trainer_id is not null and training_type_id is not null and capacity > 0)
    or (kind = 'individual' and trainer_id is not null and training_type_id is null and capacity = 1)
    or (kind = 'hold' and trainer_id is null and training_type_id is null and capacity = 0)
  ),
  unique (trainer_id, starts_at)
);

create index if not exists classes_starts_at on classes(starts_at);

create table if not exists clients (
  id uuid primary key,
  messenger_platform text not null default 'telegram',
  messenger_user_id text not null,
  messenger_username text not null default '',
  name text not null default '',
  phone text not null default '',
  policy client_policy not null default 'postpay',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  unique (messenger_platform, messenger_user_id)
);

create table if not exists bookings (
  id uuid primary key,
  class_id uuid not null references classes(id),
  client_id uuid not null references clients(id),
  status booking_status not null,
  reminder_enabled boolean not null,
  reminder_at timestamptz not null,
  reminder_state reminder_state not null,
  created_at timestamptz not null,
  updated_at timestamptz not null
);

create unique index if not exists bookings_one_active on bookings(class_id, client_id) where status = 'booked';

create index if not exists bookings_client on bookings(client_id, status);

create table if not exists booking_events (
  id uuid primary key,
  booking_id uuid not null references bookings(id),
  event_type booking_event_type not null,
  old_class_id uuid,
  new_class_id uuid,
  created_at timestamptz not null
);

create table if not exists credit_entries (
  id uuid primary key,
  client_id uuid not null references clients(id),
  delta integer not null check (delta != 0),
  reason credit_reason not null,
  booking_id uuid references bookings(id),
  note text not null default '',
  actor_telegram_id text not null,
  created_at timestamptz not null
);

create index if not exists credit_client on credit_entries(client_id);

create table if not exists attendances (
  id uuid primary key,
  booking_id uuid not null references bookings(id),
  active boolean not null,
  payment payment_status not null,
  actor_role actor_role not null,
  actor_telegram_id text not null,
  marked_at timestamptz not null,
  ledger_entry_id uuid not null references credit_entries(id)
);

create unique index if not exists attendances_one_active on attendances(booking_id) where active;

create table if not exists app_opens (
  id uuid primary key,
  messenger_user_id text not null,
  opened_at timestamptz not null
);

-- Прайс студии: объект разделов base, aero, kids, personal. Цены — целые числа больше нуля.

create or replace function studio_pricing_ok(doc jsonb) returns boolean
language sql immutable as $$
  select case
    when jsonb_typeof(doc) is distinct from 'object' then false
    when exists (
      select 1 from jsonb_object_keys(doc) as section
      where section not in ('base', 'aero', 'kids', 'personal')
    ) then false
    when exists (
      select 1 from jsonb_each(doc) as section(name, body)
      where jsonb_typeof(body) is distinct from 'object'
         or exists (
           select 1 from jsonb_each(body) as offer(label, price)
           where btrim(label) = ''
              or label <> btrim(label)
              or jsonb_typeof(price) is distinct from 'number'
              or price::text !~ '^[0-9]+$'
              or price::text::numeric <= 0
         )
    ) then false
    else true
  end
$$;

create table if not exists studio (
  id integer primary key check (id = 1),
  address text not null default '',
  phone text not null default '',
  instagram_url text not null default '',
  contact_url text not null default '',
  map_url text not null default '',
  guide text not null default '',
  pricing jsonb not null default '{}'::jsonb,
  constraint studio_pricing_check check (studio_pricing_ok(pricing))
);

insert into studio (id, address, phone, instagram_url, contact_url, map_url)
values (1, '', '', '', '', '')
on conflict (id) do nothing;

create table if not exists messages (
  key text primary key,
  body text not null
);

create table if not exists enrollments (
  id uuid primary key,
  client_id uuid not null references clients(id),
  template_id uuid not null references class_templates(id),
  active boolean not null,
  created_at timestamptz not null,
  unique (client_id, template_id)
);

create table if not exists trainer_absences (
  id uuid primary key,
  trainer_id uuid not null references trainers(id),
  starts_on date not null,
  ends_on date not null,
  created_at timestamptz not null,
  check (ends_on >= starts_on)
);

create table if not exists balance_notices (
  id uuid primary key,
  client_id uuid not null references clients(id),
  class_id uuid not null references classes(id),
  kind notice_kind not null,
  state notice_state not null,
  notice_at timestamptz not null,
  created_at timestamptz not null,
  unique (client_id, class_id, kind)
);
