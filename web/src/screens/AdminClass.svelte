<script lang="ts">
  import { getAdminClass, getRoster, patchClass, setAttendance } from '../api'
  import { listTrainers } from '../api'
  import { formatWhen, hoursUntil } from '../lib/time'

  let { classId }: { classId: string } = $props()
  let card = $derived(getAdminClass(classId))
  let roster = $derived(card.ok && card.data.status === 'scheduled' ? getRoster(classId) : null)
  let trainers = $derived(listTrainers().data)
  let error = $state('')
  let starts = $state('')
  let capacity = $state('')
  let title = $state('')
  let trainerId = $state('')
  let filled = $state(false)

  $effect(() => {
    if (filled || !card.ok) return
    starts = card.data.starts_at.slice(0, 16)
    capacity = String(card.data.capacity)
    title = card.data.title
    trainerId = card.data.trainer.id
    filled = true
  })

  function localToIso(value: string) {
    if (value.length === 16) return `${value}:00+04:00`
    return value.endsWith('+04:00') ? value : `${value}+04:00`
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const response = await patchClass(classId, {
      starts_at: localToIso(starts),
      capacity: Number(capacity),
      title,
      trainer_id: trainerId,
    })
    error = response.ok ? 'Слот обновлён. Записанным уйдёт сообщение, когда бот будет подключён.' : response.message
  }

  async function cancel() {
    const response = await patchClass(classId, { status: 'cancelled' })
    error = response.ok ? 'Занятие отменено. Баланс записей не изменился.' : response.message
  }

  async function mark(bookingId: string, arrived: boolean) {
    const response = await setAttendance(bookingId, arrived)
    error = response.ok ? '' : response.message
  }
</script>

{#if !card.ok}
  {#if card.message}<p class="card error">{card.message}</p>{/if}
{:else}
  <article class="card">
    <h2>{card.data.title}</h2>
    <p class="meta">{formatWhen(card.data.starts_at)} · {card.data.trainer.name}</p>
    {#if card.data.status === 'cancelled'}<p class="tag">отменено</p>{/if}
  </article>
  {#if error}<p class="card" data-testid="form-error">{error}</p>{/if}
  {#if card.data.status === 'scheduled'}
    {#if roster?.ok}
      {#each roster.data.bookings as booking (booking.booking_id)}
        <article class="card">
          <strong>{booking.name}</strong>
          {#if booking.attendance}
            <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
          {:else}
            <button class="primary" type="button" data-testid={hoursUntil(card.data.starts_at) <= 0.5 ? 'mark-arrived' : 'mark-early'} onclick={() => mark(booking.booking_id, true)}>Пришёл</button>
          {/if}
        </article>
      {/each}
    {/if}
    <form class="card" onsubmit={save}>
      <label for="starts">Начало, время Тбилиси</label>
      <input id="starts" type="datetime-local" bind:value={starts} />
      <label for="capacity">Вместимость</label>
      <input id="capacity" type="number" min="1" bind:value={capacity} />
      <label for="title">Название</label>
      <input id="title" bind:value={title} />
      <label for="trainer">Тренер</label>
      <select id="trainer" bind:value={trainerId}>
        {#each trainers as trainer (trainer.id)}
          <option value={trainer.id}>{trainer.name}</option>
        {/each}
      </select>
      <button class="primary" type="submit">Перенести</button>
      <button class="danger" type="button" onclick={cancel}>Отменить занятие</button>
    </form>
  {/if}
{/if}
