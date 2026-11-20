create table if not exists trainers (
  id text primary key,
  name text not null unique,
  active integer not null check (active in (0, 1)),
  telegram_user_id text unique,
  about text not null default '',
  phone text not null default '',
  created_at text not null
);

create table if not exists training_types (
  id text primary key,
  title text not null unique,
  duration_minutes integer not null check (duration_minutes between 1 and 600),
  capacity integer not null check (capacity > 0),
  price_min text not null,
  price_max text not null,
  active integer not null check (active in (0, 1)),
  color text not null default '',
  created_at text not null
);

create table if not exists class_templates (
  id text primary key,
  trainer_id text not null references trainers(id),
  training_type_id text not null references training_types(id),
  title text not null,
  weekday integer not null check (weekday between 1 and 7),
  start_time text not null,
  capacity integer not null check (capacity > 0),
  active integer not null check (active in (0, 1)),
  created_at text not null,
  unique (trainer_id, title, weekday, start_time)
);

create table if not exists classes (
  id text primary key,
  template_id text references class_templates(id),
  trainer_id text references trainers(id),
  training_type_id text references training_types(id),
  title text not null,
  starts_at text not null,
  ends_at text not null,
  capacity integer not null check (capacity >= 0),
  price_min text not null,
  price_max text not null,
  status text not null check (status in ('scheduled', 'cancelled')),
  kind text not null default 'group' check (kind in ('group', 'individual', 'hold')),
  created_at text not null,
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
  id text primary key,
  messenger_platform text not null default 'telegram',
  messenger_user_id text not null,
  messenger_username text not null default '',
  name text not null default '',
  phone text not null default '',
  policy text not null default 'postpay' check (policy in ('postpay', 'restricted')),
  created_at text not null,
  updated_at text not null,
  unique (messenger_platform, messenger_user_id)
);

create table if not exists bookings (
  id text primary key,
  class_id text not null references classes(id),
  client_id text not null references clients(id),
  status text not null check (status in ('booked', 'cancelled')),
  reminder_enabled integer not null check (reminder_enabled in (0, 1)),
  reminder_at text not null,
  reminder_state text not null check (reminder_state in ('pending', 'sent', 'cancelled', 'skipped')),
  created_at text not null,
  updated_at text not null
);

create unique index if not exists bookings_one_active on bookings(class_id, client_id) where status = 'booked';

create index if not exists bookings_client on bookings(client_id, status);

create table if not exists booking_events (
  id text primary key,
  booking_id text not null references bookings(id),
  event_type text not null check (event_type in ('created', 'cancelled', 'rescheduled')),
  old_class_id text,
  new_class_id text,
  created_at text not null
);

create table if not exists credit_entries (
  id text primary key,
  client_id text not null references clients(id),
  delta integer not null check (delta != 0),
  reason text not null check (reason in ('grant', 'attend', 'reversal')),
  booking_id text references bookings(id),
  note text not null default '',
  actor_telegram_id text not null,
  created_at text not null
);

create index if not exists credit_client on credit_entries(client_id);

create table if not exists attendances (
  id text primary key,
  booking_id text not null references bookings(id),
  active integer not null check (active in (0, 1)),
  payment text not null check (payment in ('paid', 'unpaid')),
  actor_role text not null check (actor_role in ('admin', 'trainer')),
  actor_telegram_id text not null,
  marked_at text not null,
  ledger_entry_id text not null references credit_entries(id)
);

create unique index if not exists attendances_one_active on attendances(booking_id) where active = 1;

create table if not exists app_opens (
  id text primary key,
  messenger_user_id text not null,
  opened_at text not null
);

create table if not exists studio (
  id integer primary key check (id = 1),
  address text not null default '',
  phone text not null default '',
  instagram_url text not null default '',
  contact_url text not null default '',
  map_url text not null default ''
);

insert into studio (id, address, phone, instagram_url, contact_url, map_url)
values (1, '', '', '', '', '')
on conflict (id) do nothing;

create table if not exists enrollments (
  id text primary key,
  client_id text not null references clients(id),
  template_id text not null references class_templates(id),
  active integer not null check (active in (0, 1)),
  created_at text not null,
  unique (client_id, template_id)
);

create table if not exists trainer_absences (
  id text primary key,
  trainer_id text not null references trainers(id),
  starts_on text not null,
  ends_on text not null,
  created_at text not null,
  check (ends_on >= starts_on)
);

create table if not exists balance_notices (
  id text primary key,
  client_id text not null references clients(id),
  class_id text not null references classes(id),
  kind text not null check (kind in ('booked_empty', 'skipped_empty')),
  state text not null check (state in ('pending', 'sent', 'skipped')),
  notice_at text not null,
  created_at text not null,
  unique (client_id, class_id, kind)
);
