<script lang="ts">
  import { createClass, listTrainers, listTypes } from '../api'

  let trainers = $derived(listTrainers().data.filter((item) => item.active))
  let types = $derived(listTypes().data.filter((item) => item.active))
  let trainerId = $state('')
  let typeId = $state('')
  let starts = $state('')
  let capacity = $state('')
  let notice = $state('')

  $effect(() => {
    if (!trainerId && trainers[0]) trainerId = trainers[0].id
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
