<script lang="ts">
  import {
    cancelTrainerClass,
    createTrainerClass,
    createTrainerIndividual,
    moveTrainerClass,
    searchTrainerClients,
    getTrainerProfile,
    getTrainerRoster,
    getTrainerSchedule,
    listTrainerTypes,
    patchTrainerProfile,
    setTrainerAttendance,
    trainerStats,
  } from '../api'
  import { addDays, formatWhen, hoursUntil, todayKey } from '../lib/time'

  let profile = $derived(getTrainerProfile())
  let types = $derived(listTrainerTypes().data.filter((item) => item.active))
  let classes = $derived(getTrainerSchedule(todayKey(), addDays(todayKey(), 21)).data)
  let about = $state('')
  let phone = $state('')
  let filled = $state(false)
  let from = $state(`${todayKey().slice(0, 8)}01`)
  let to = $state(todayKey())
  let report = $derived(trainerStats(from, to).data)
  let typeId = $state('')
  let starts = $state('')
  let capacity = $state('')
  let clientQuery = $state('')
  let clientId = $state('')
  let individualStart = $state('')
  let individualMinutes = $state('60')
  let moveStart = $state('')
  let clients = $derived(clientQuery.trim() ? searchTrainerClients(clientQuery).data : [])
  let openId = $state('')
  let notice = $state('')
  let failed = $state(false)

  $effect(() => {
    if (!profile.ok || filled || profile.pending || !profile.data.id) return
    about = profile.data.about
    phone = profile.data.phone
    filled = true
  })

  $effect(() => {
    if (!typeId && types[0]) typeId = types[0].id
  })

  let roster = $derived(openId ? getTrainerRoster(openId) : null)

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const response = await patchTrainerProfile({ about, phone })
    failed = !response.ok
    notice = response.ok ? '' : response.message
  }

  async function create(event: SubmitEvent) {
    event.preventDefault()
    const iso = starts.length === 16 ? `${starts}:00+04:00` : starts
    const response = await createTrainerClass({
      training_type_id: typeId,
      starts_at: iso,
      capacity: capacity ? Number(capacity) : undefined,
    })
    failed = !response.ok
    notice = response.ok ? 'Разовое занятие создано.' : response.message
    if (response.ok) starts = ''
  }

  function iso(value: string) {
    return value.length === 16 ? `${value}:00+04:00` : value
  }

  async function assign(event: SubmitEvent) {
    event.preventDefault()
    const response = await createTrainerIndividual({
      client_id: clientId,
      starts_at: iso(individualStart),
      duration_minutes: Number(individualMinutes || 60),
    })
    failed = !response.ok
    notice = response.ok ? 'Индивидуальное создано.' : response.message
    if (response.ok) {
      clientId = ''
      individualStart = ''
    }
  }

  async function move(id: string) {
    const response = await moveTrainerClass(id, iso(moveStart))
    failed = !response.ok
    notice = response.ok ? 'Время перенесено.' : response.message
  }

  async function cancel(id: string) {
    const response = await cancelTrainerClass(id)
    failed = !response.ok
    notice = response.ok ? '' : response.message
  }

  async function mark(bookingId: string, arrived: boolean) {
    const response = await setTrainerAttendance(bookingId, arrived)
    failed = !response.ok
    notice = response.ok ? '' : response.message
  }
</script>

<h2 class="day">{profile.ok ? profile.data.name || 'Тренер' : 'Тренер'}</h2>
{#if notice}<p class="card" class:error={failed}>{notice}</p>{/if}

<form class="card" onsubmit={save}>
  <label for="own-about">О тренере</label>
  <textarea id="own-about" rows="3" bind:value={about}></textarea>
  <label for="own-phone">Телефон</label>
  <input id="own-phone" bind:value={phone} />
  <button class="primary" type="submit">Сохранить</button>
</form>

<div class="card">
  <h3>Сводка</h3>
  <label for="trainer-from">С</label>
  <input id="trainer-from" type="date" bind:value={from} />
  <label for="trainer-to">По</label>
  <input id="trainer-to" type="date" bind:value={to} />
  <p>Клиентов с явкой: {report.clients}</p>
  <p>Списаний: {report.attends}</p>
</div>

<h2 class="day">Мои занятия</h2>
{#each classes as item (item.id)}
  <article class="card">
    <strong>{item.title}</strong>
    <p class="meta">{formatWhen(item.starts_at)}{#if item.status === 'cancelled'} · отменено{/if}</p>
    {#if item.kind !== 'hold' && item.status !== 'cancelled' && new Date(item.starts_at) > new Date()}
      <button class="danger" type="button" onclick={() => cancel(item.id)}>Отменить</button>
    {/if}
    {#if item.kind === 'individual' && item.status !== 'cancelled'}
      <label for="move-{item.id}">Новое начало</label>
      <input id="move-{item.id}" type="datetime-local" bind:value={moveStart} />
      <button class="ghost" type="button" onclick={() => move(item.id)}>Перенести</button>
    {/if}
    {#if item.kind !== 'hold' && item.status !== 'cancelled'}
      <button class="ghost" type="button" onclick={() => (openId = openId === item.id ? '' : item.id)}>Журнал</button>
    {/if}
    {#if openId === item.id && roster?.ok}
      {#each roster.data.bookings as booking (booking.booking_id)}
        <p>{booking.name}</p>
        {#if booking.attendance}
          <p class="meta">пришёл, {booking.attendance.payment === 'paid' ? 'занятие списано' : 'без покрытия'}</p>
          <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
        {:else}
          <button class="primary" type="button" disabled={hoursUntil(item.starts_at) > 0.5} onclick={() => mark(booking.booking_id, true)}>Пришёл</button>
        {/if}
      {:else}
        <p class="muted">Записей нет.</p>
      {/each}
    {/if}
  </article>
{/each}

<form class="card" onsubmit={create}>
  <h3>Разовое занятие</h3>
  <label for="own-type">Тип</label>
  <select id="own-type" bind:value={typeId}>
    {#each types as type (type.id)}<option value={type.id}>{type.title}</option>{/each}
  </select>
  <label for="own-start">Начало, время Тбилиси</label>
  <input id="own-start" type="datetime-local" bind:value={starts} required />
  <label for="own-capacity">Вместимость, если отличается</label>
  <input id="own-capacity" type="number" min="1" bind:value={capacity} />
  <button class="primary" type="submit">Создать</button>
</form>

<form class="card" onsubmit={assign}>
  <h3>Индивидуальное</h3>
  <label for="own-client">Клиент</label>
  <input id="own-client" bind:value={clientQuery} placeholder="Имя" />
  {#each clients as client (client.id)}
    <button class="ghost" type="button" aria-pressed={clientId === client.id} onclick={() => (clientId = client.id)}>
      {client.name || 'без имени'}
    </button>
  {/each}
  <label for="own-individual-start">Начало, время Тбилиси</label>
  <input id="own-individual-start" type="datetime-local" bind:value={individualStart} required />
  <label for="own-individual-minutes">Длительность, минуты</label>
  <input id="own-individual-minutes" type="number" min="1" max="600" bind:value={individualMinutes} />
  <button class="primary" type="submit">Назначить</button>
</form>
