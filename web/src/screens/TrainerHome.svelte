<script lang="ts">
  import {
    cancelTrainerClass,
    createTrainerClass,
    createTrainerIndividual,
    getTrainerSchedule,
    listTrainerTypes,
    moveTrainerClass,
    searchTrainerClients,
  } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { addDays, formatWhen, todayKey } from '../lib/time'

  let typeLoad = $derived(listTrainerTypes())
  let schedule = $derived(getTrainerSchedule(todayKey(), addDays(todayKey(), 21)))
  let types = $derived(typeLoad.status === 'ready' ? typeLoad.data.filter((item) => item.active) : [])
  let classes = $derived(schedule.status === 'ready' ? schedule.data : [])
  let typeId = $state('')
  let starts = $state('')
  let capacity = $state('')
  let clientQuery = $state('')
  let clientId = $state('')
  let individualStart = $state('')
  let individualMinutes = $state('60')
  let moveStart = $state('')
  let clientLoad = $derived(clientQuery.trim() ? searchTrainerClients(clientQuery) : null)
  let clients = $derived(clientLoad?.status === 'ready' ? clientLoad.data : [])
  let notice = $state('')
  let failed = $state(false)

  $effect(() => {
    if (!typeId && types[0]) typeId = types[0].id
  })

  function iso(value: string) {
    return value.length === 16 ? `${value}:00+04:00` : value
  }

  function report(ok: boolean, message: string) {
    failed = !ok
    notice = message
  }

  async function create(event: SubmitEvent) {
    event.preventDefault()
    const response = await createTrainerClass({
      training_type_id: typeId,
      starts_at: iso(starts),
      capacity: capacity ? Number(capacity) : undefined,
    })
    report(response.ok, response.ok ? 'Разовое занятие создано.' : response.message)
    if (response.ok) starts = ''
  }

  async function assign(event: SubmitEvent) {
    event.preventDefault()
    const response = await createTrainerIndividual({
      client_id: clientId,
      starts_at: iso(individualStart),
      duration_minutes: Number(individualMinutes || 60),
    })
    report(response.ok, response.ok ? 'Индивидуальное создано.' : response.message)
    if (response.ok) {
      clientId = ''
      individualStart = ''
    }
  }

  async function move(id: string) {
    const response = await moveTrainerClass(id, iso(moveStart))
    report(response.ok, response.ok ? 'Время перенесено.' : response.message)
  }

  async function cancel(id: string) {
    if (!confirm('Отменить занятие? Записанным уйдёт сообщение. Баланс не изменится.')) return
    const response = await cancelTrainerClass(id)
    report(response.ok, response.ok ? 'Занятие отменено. Баланс записей не изменился.' : response.message)
  }
</script>

<h2 class="day">Мои занятия</h2>
{#if typeLoad.status === 'pending' || schedule.status === 'pending'}
  <Notice kind="pending" />
{:else if typeLoad.status === 'error'}
  <Notice kind="error" message={typeLoad.message} reload={typeLoad.reload} />
{:else if schedule.status === 'error'}
  <Notice kind="error" message={schedule.message} reload={schedule.reload} />
{:else}
  {#if notice}<Notice kind={failed ? 'error' : 'ok'} message={notice} />{/if}
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
    </article>
  {:else}
    <p class="card muted">Ближайших занятий нет.</p>
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
    {#if clientLoad?.status === 'pending'}
      <Notice kind="pending" />
    {:else if clientLoad?.status === 'error'}
      <Notice kind="error" message={clientLoad.message} reload={clientLoad.reload} />
    {/if}
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
{/if}
