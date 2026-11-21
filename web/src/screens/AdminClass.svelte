<script lang="ts">
  import { getAdminClass, getRoster, listTrainers, patchClass, setAttendance } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { attendanceEarlyText, attendanceTooEarly, formatWhen } from '../lib/time'

  let { classId }: { classId: string } = $props()
  let card = $derived(getAdminClass(classId))
  let roster = $derived(card.status === 'ready' && card.data.status === 'scheduled' ? getRoster(classId) : null)
  let trainerLoad = $derived(listTrainers())
  let trainers = $derived(trainerLoad.status === 'ready' ? trainerLoad.data : [])
  let notice = $state('')
  let failed = $state(false)
  let starts = $state('')
  let capacity = $state('')
  let title = $state('')
  let trainerId = $state('')
  let filled = $state(false)

  $effect(() => {
    if (filled || card.status !== 'ready') return
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

  function report(ok: boolean, message: string) {
    failed = !ok
    notice = message
  }

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const kind = card.status === 'ready' ? card.data.kind : 'group'
    const response = await patchClass(
      classId,
      kind === 'hold'
        ? { starts_at: localToIso(starts), title }
        : kind === 'individual'
          ? { starts_at: localToIso(starts), title }
          : {
              starts_at: localToIso(starts),
              capacity: Number(capacity),
              title,
              trainer_id: trainerId,
            },
    )
    report(response.ok, response.ok ? 'Слот обновлён. Записанным уйдёт сообщение.' : response.message)
  }

  async function cancel() {
    if (!confirm('Отменить занятие? Записанным уйдёт сообщение. Баланс не изменится.')) return
    const response = await patchClass(classId, { status: 'cancelled' })
    report(response.ok, response.ok ? 'Занятие отменено. Баланс записей не изменился.' : response.message)
  }

  async function mark(bookingId: string, arrived: boolean) {
    if (!arrived && !confirm('Снять явку? Сообщение не уйдёт. Если занятие было списано, оно вернётся на баланс.')) return
    const response = await setAttendance(bookingId, arrived)
    report(response.ok, response.ok ? '' : response.message)
  }
</script>

{#if card.status === 'pending'}
  <Notice kind="pending" />
{:else if card.status === 'error'}
  <Notice kind="error" message={card.message} reload={card.reload} />
{:else}
  <article class="card">
    <h2>{card.data.title}</h2>
    <p class="meta">{formatWhen(card.data.starts_at)} · {card.data.trainer.name}</p>
    {#if card.data.status === 'cancelled'}<p class="tag">отменено</p>{/if}
  </article>
  {#if notice}<Notice kind={failed ? 'error' : 'ok'} message={notice} testid={failed ? 'form-error' : undefined} />{/if}
  {#if card.data.status === 'scheduled'}
    {#if roster?.status === 'pending'}
      <Notice kind="pending" />
    {:else if roster?.status === 'error'}
      <Notice kind="error" message={roster.message} reload={roster.reload} />
    {:else if roster?.status === 'ready'}
      {#each roster.data.bookings as booking (booking.booking_id)}
        <article class="card">
          <strong>{booking.name}</strong>
          {#if booking.attendance}
            <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
          {:else}
            {@const early = attendanceTooEarly(card.data.starts_at)}
            <button class="primary" type="button" disabled={early} data-testid={early ? 'mark-early' : 'mark-arrived'} onclick={() => mark(booking.booking_id, true)}>Пришёл</button>
            {#if early}<p class="meta">{attendanceEarlyText}</p>{/if}
          {/if}
        </article>
      {/each}
    {/if}
    <form class="card" onsubmit={save}>
      <label for="starts">Начало, время Тбилиси</label>
      <input id="starts" type="datetime-local" bind:value={starts} />
      {#if card.data.kind === 'group'}
        <label for="capacity">Вместимость</label>
        <input id="capacity" type="number" min="1" bind:value={capacity} />
      {/if}
      <label for="title">Название</label>
      <input id="title" bind:value={title} />
      {#if card.data.kind === 'group'}
        <label for="trainer">Тренер</label>
        {#if trainerLoad.status === 'pending'}
          <Notice kind="pending" />
        {:else if trainerLoad.status === 'error'}
          <Notice kind="error" message={trainerLoad.message} reload={trainerLoad.reload} />
        {/if}
        <select id="trainer" bind:value={trainerId}>
          {#each trainers as trainer (trainer.id)}
            <option value={trainer.id}>{trainer.name}</option>
          {/each}
        </select>
      {/if}
      <button class="primary" type="submit">Перенести</button>
      <button class="danger" type="button" onclick={cancel}>Отменить занятие</button>
    </form>
  {/if}
{/if}
