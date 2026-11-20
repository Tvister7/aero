<script lang="ts">
  import { createClass, createHold, createIndividual, listClients, listTrainers, listTypes } from '../api'

  let trainers = $derived(listTrainers().data.filter((item) => item.active))
  let types = $derived(listTypes().data.filter((item) => item.active))
  let trainerId = $state('')
  let typeId = $state('')
  let starts = $state('')
  let capacity = $state('')
  let notice = $state('')
  let individualTrainer = $state('')
  let clientQuery = $state('')
  let clientId = $state('')
  let individualStart = $state('')
  let individualMinutes = $state('60')
  let individualNotice = $state('')
  let holdTitle = $state('')
  let holdStart = $state('')
  let holdMinutes = $state('60')
  let holdNotice = $state('')
  let clients = $derived(clientQuery.trim() ? listClients(clientQuery).data : [])

  $effect(() => {
    if (!trainerId && trainers[0]) trainerId = trainers[0].id
    if (!individualTrainer && trainers[0]) individualTrainer = trainers[0].id
    if (!typeId && types[0]) typeId = types[0].id
  })

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    const iso = starts.length === 16 ? `${starts}:00+04:00` : starts
    const response = await createClass({
      trainer_id: trainerId,
      training_type_id: typeId,
      starts_at: iso,
      capacity: capacity ? Number(capacity) : undefined,
    })
    notice = response.ok ? 'Разовое занятие создано. Само оно не повторится.' : response.message
  }

  function iso(value: string) {
    return value.length === 16 ? `${value}:00+04:00` : value
  }

  async function submitIndividual(event: SubmitEvent) {
    event.preventDefault()
    const response = await createIndividual({
      trainer_id: individualTrainer || trainerId,
      client_id: clientId,
      starts_at: iso(individualStart),
      duration_minutes: Number(individualMinutes || 60),
    })
    individualNotice = response.ok ? 'Индивидуальное создано.' : response.message
  }

  async function submitHold(event: SubmitEvent) {
    event.preventDefault()
    const response = await createHold({
      title: holdTitle,
      starts_at: iso(holdStart),
      duration_minutes: Number(holdMinutes || 60),
    })
    holdNotice = response.ok ? 'Время зарезервировано.' : response.message
  }
</script>

<form class="card" onsubmit={submit}>
  <h2>Разовое занятие</h2>
  <label for="new-trainer">Тренер</label>
  <select id="new-trainer" bind:value={trainerId}>
    {#each trainers as trainer (trainer.id)}<option value={trainer.id}>{trainer.name}</option>{/each}
  </select>
  <label for="new-type">Тип</label>
  <select id="new-type" bind:value={typeId}>
    {#each types as type (type.id)}<option value={type.id}>{type.title}</option>{/each}
  </select>
  <label for="new-start">Начало, время Тбилиси</label>
  <input id="new-start" type="datetime-local" bind:value={starts} required />
  <label for="new-capacity">Вместимость, если отличается</label>
  <input id="new-capacity" type="number" min="1" bind:value={capacity} />
  <button class="primary" type="submit">Создать</button>
  {#if notice}<p>{notice}</p>{/if}
</form>

<form class="card" onsubmit={submitIndividual}>
  <h2>Индивидуальное</h2>
  <label for="individual-trainer">Тренер</label>
  <select id="individual-trainer" bind:value={individualTrainer}>
    {#each trainers as trainer (trainer.id)}<option value={trainer.id}>{trainer.name}</option>{/each}
  </select>
  <label for="client-search">Клиент</label>
  <input id="client-search" bind:value={clientQuery} placeholder="Имя" />
  {#each clients as client (client.id)}
    <button class="ghost" type="button" aria-pressed={clientId === client.id} onclick={() => (clientId = client.id)}>
      {client.name || 'без имени'}
    </button>
  {/each}
  <label for="individual-start">Начало, время Тбилиси</label>
  <input id="individual-start" type="datetime-local" bind:value={individualStart} required />
  <label for="individual-minutes">Длительность, минуты</label>
  <input id="individual-minutes" type="number" min="1" max="600" bind:value={individualMinutes} />
  <button class="primary" type="submit">Назначить</button>
  {#if individualNotice}<p>{individualNotice}</p>{/if}
</form>

<form class="card" onsubmit={submitHold}>
  <h2>Резерв</h2>
  <label for="hold-title">Название</label>
  <input id="hold-title" bind:value={holdTitle} required />
  <label for="hold-start">Начало, время Тбилиси</label>
  <input id="hold-start" type="datetime-local" bind:value={holdStart} required />
  <label for="hold-minutes">Длительность, минуты</label>
  <input id="hold-minutes" type="number" min="1" max="600" bind:value={holdMinutes} />
  <button class="primary" type="submit">Занять зал</button>
  {#if holdNotice}<p>{holdNotice}</p>{/if}
</form>
