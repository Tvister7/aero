<script lang="ts">
  import { getTrainerRoster, getTrainerSchedule, setTrainerAttendance } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { attendanceEarlyText, attendanceTooEarly, formatWhen, todayKey } from '../lib/time'

  let schedule = $derived(getTrainerSchedule(todayKey(), todayKey()))
  let classes = $derived(
    schedule.status === 'ready' ? schedule.data.filter((item) => item.kind !== 'hold' && item.status !== 'cancelled') : [],
  )
  let openId = $state('')
  let roster = $derived(openId ? getTrainerRoster(openId) : null)
  let notice = $state('')
  let failed = $state(false)

  async function mark(bookingId: string, arrived: boolean) {
    if (!arrived && !confirm('Снять явку? Сообщение не уйдёт. Если занятие было списано, оно вернётся на баланс.')) return
    const response = await setTrainerAttendance(bookingId, arrived)
    failed = !response.ok
    notice = response.ok ? '' : response.message
  }
</script>

<h2 class="day">Журнал</h2>
{#if notice}<Notice kind={failed ? 'error' : 'ok'} message={notice} />{/if}
{#if schedule.status === 'pending'}
  <Notice kind="pending" />
{:else if schedule.status === 'error'}
  <Notice kind="error" message={schedule.message} reload={schedule.reload} />
{:else if classes.length === 0}
  <p class="card muted">Сегодня занятий нет.</p>
{:else}
  {#each classes as item (item.id)}
    {@const open = openId === item.id}
    {@const early = attendanceTooEarly(item.starts_at)}
    <article class="card">
      <button class="fold" type="button" aria-expanded={open} onclick={() => (openId = open ? '' : item.id)}>
        <strong>{item.title}</strong>
        <p class="meta">{formatWhen(item.starts_at)}</p>
      </button>
      {#if open}
        {#if roster?.status === 'pending'}
          <Notice kind="pending" />
        {:else if roster?.status === 'error'}
          <Notice kind="error" message={roster.message} reload={roster.reload} />
        {:else if roster?.status === 'ready'}
          {#each roster.data.bookings as booking (booking.booking_id)}
            <p>{booking.name}</p>
            {#if booking.attendance}
              <p class="meta">пришёл, {booking.attendance.payment === 'paid' ? 'занятие списано' : 'без покрытия'}</p>
              <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
            {:else}
              <button class="primary" type="button" disabled={early} onclick={() => mark(booking.booking_id, true)}>Пришёл</button>
              {#if early}<p class="meta">{attendanceEarlyText}</p>{/if}
            {/if}
          {:else}
            <p class="muted">Записей нет.</p>
          {/each}
        {/if}
      {/if}
    </article>
  {/each}
{/if}
