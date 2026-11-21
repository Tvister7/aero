-- Копирует снимок SQLite из схемы legacy в нативные таблицы public.
-- Сначала aero migrate, затем pgloader. Повтор затирает строки public этим снимком.

begin;
set local search_path to public;

do $$
begin
  if to_regclass('legacy.trainers') is null then
    raise exception 'schema legacy is missing';
  end if;
end
$$;

create function pg_temp.flag(value text) returns boolean
language sql immutable as $$
  select lower(value) in ('1', 't', 'true')
$$;

create function pg_temp.uuid_or_null(value text) returns uuid
language sql immutable as $$
  select nullif(btrim(value), '')::uuid
$$;

truncate table
  balance_notices,
  enrollments,
  attendances,
  credit_entries,
  booking_events,
  bookings,
  app_opens,
  classes,
  class_templates,
  trainer_absences,
  clients,
  messages,
  training_types,
  trainers,
  studio
restart identity cascade;

insert into trainers (id, name, active, telegram_user_id, about, phone, created_at)
select id::text::uuid,
       name,
       coalesce(pg_temp.flag(active::text), false),
       nullif(btrim(telegram_user_id::text), ''),
       coalesce(about::text, ''),
       coalesce(phone::text, ''),
       created_at::text::timestamptz
from legacy.trainers;

insert into training_types (
  id, title, duration_minutes, capacity, price_min, price_max, active, color, created_at
)
select id::text::uuid,
       title,
       duration_minutes::int,
       capacity::int,
       price_min,
       price_max,
       coalesce(pg_temp.flag(active::text), false),
       coalesce(color::text, ''),
       created_at::text::timestamptz
from legacy.training_types;

insert into class_templates (
  id, trainer_id, training_type_id, title, weekday, start_time, capacity, active, created_at
)
select id::text::uuid,
       trainer_id::text::uuid,
       training_type_id::text::uuid,
       title,
       weekday::int,
       start_time::text::time,
       capacity::int,
       coalesce(pg_temp.flag(active::text), false),
       created_at::text::timestamptz
from legacy.class_templates;

insert into classes (
  id, template_id, trainer_id, training_type_id, title, starts_at, ends_at,
  capacity, price_min, price_max, status, kind, created_at
)
select id::text::uuid,
       pg_temp.uuid_or_null(template_id::text),
       pg_temp.uuid_or_null(trainer_id::text),
       pg_temp.uuid_or_null(training_type_id::text),
       title,
       starts_at::text::timestamptz,
       ends_at::text::timestamptz,
       capacity::int,
       price_min,
       price_max,
       status::text::class_status,
       coalesce(nullif(kind::text, ''), 'group')::class_kind,
       created_at::text::timestamptz
from legacy.classes;

insert into clients (
  id, messenger_platform, messenger_user_id, messenger_username, name, phone, policy, created_at, updated_at
)
select id::text::uuid,
       coalesce(nullif(messenger_platform::text, ''), 'telegram'),
       messenger_user_id,
       coalesce(messenger_username::text, ''),
       coalesce(name::text, ''),
       coalesce(phone::text, ''),
       coalesce(nullif(policy::text, ''), 'postpay')::client_policy,
       created_at::text::timestamptz,
       updated_at::text::timestamptz
from legacy.clients;

insert into bookings (
  id, class_id, client_id, status, reminder_enabled, reminder_at, reminder_state, created_at, updated_at
)
select id::text::uuid,
       class_id::text::uuid,
       client_id::text::uuid,
       status::text::booking_status,
       coalesce(pg_temp.flag(reminder_enabled::text), false),
       reminder_at::text::timestamptz,
       reminder_state::text::reminder_state,
       created_at::text::timestamptz,
       updated_at::text::timestamptz
from legacy.bookings;

insert into booking_events (id, booking_id, event_type, old_class_id, new_class_id, created_at)
select id::text::uuid,
       booking_id::text::uuid,
       event_type::text::booking_event_type,
       pg_temp.uuid_or_null(old_class_id::text),
       pg_temp.uuid_or_null(new_class_id::text),
       created_at::text::timestamptz
from legacy.booking_events;

insert into credit_entries (
  id, client_id, delta, reason, booking_id, note, actor_telegram_id, created_at
)
select id::text::uuid,
       client_id::text::uuid,
       delta::int,
       reason::text::credit_reason,
       pg_temp.uuid_or_null(booking_id::text),
       coalesce(note::text, ''),
       actor_telegram_id,
       created_at::text::timestamptz
from legacy.credit_entries;

insert into attendances (
  id, booking_id, active, payment, actor_role, actor_telegram_id, marked_at, ledger_entry_id
)
select id::text::uuid,
       booking_id::text::uuid,
       coalesce(pg_temp.flag(active::text), false),
       payment::text::payment_status,
       actor_role::text::actor_role,
       actor_telegram_id,
       marked_at::text::timestamptz,
       ledger_entry_id::text::uuid
from legacy.attendances;

insert into app_opens (id, messenger_user_id, opened_at)
select id::text::uuid,
       messenger_user_id,
       opened_at::text::timestamptz
from legacy.app_opens;

insert into studio (id, address, phone, instagram_url, contact_url, map_url, guide, pricing)
select id::int,
       coalesce(address::text, ''),
       coalesce(phone::text, ''),
       coalesce(instagram_url::text, ''),
       coalesce(contact_url::text, ''),
       coalesce(map_url::text, ''),
       coalesce(guide::text, ''),
       case when btrim(coalesce(pricing::text, '')) = '' then '{}'::jsonb else pricing::text::jsonb end
from legacy.studio;

insert into studio (id, address, phone, instagram_url, contact_url, map_url)
values (1, '', '', '', '', '')
on conflict (id) do nothing;

insert into messages (key, body)
select key, body
from legacy.messages;

insert into enrollments (id, client_id, template_id, active, created_at)
select id::text::uuid,
       client_id::text::uuid,
       template_id::text::uuid,
       coalesce(pg_temp.flag(active::text), false),
       created_at::text::timestamptz
from legacy.enrollments;

insert into trainer_absences (id, trainer_id, starts_on, ends_on, created_at)
select id::text::uuid,
       trainer_id::text::uuid,
       starts_on::text::date,
       ends_on::text::date,
       created_at::text::timestamptz
from legacy.trainer_absences;

insert into balance_notices (id, client_id, class_id, kind, state, notice_at, created_at)
select id::text::uuid,
       client_id::text::uuid,
       class_id::text::uuid,
       kind::text::notice_kind,
       state::text::notice_state,
       notice_at::text::timestamptz,
       created_at::text::timestamptz
from legacy.balance_notices;

commit;
