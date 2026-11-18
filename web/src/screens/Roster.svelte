<script lang="ts">
  import { setAttendance, todayRoster } from '../api'
  import { formatWhen, hoursUntil } from '../lib/time'

  let rows = $derived(todayRoster())
  let error = $state('')

  async function mark(bookingId: string, arrived: boolean) {
    const response = await setAttendance(bookingId, arrived)
    error = response.ok ? '' : response.message
  }
</script>

<h2 class="day">Журнал на сегодня</h2>
{#if error}<p class="card error" data-testid="form-error">{error}</p>{/if}
{#if rows.length === 0}
  <p class="card muted">Сегодня занятий нет.</p>
{/if}
{#each rows as row (row.class.id)}
  <article class="card">
    <strong>{row.class.title}</strong>
    <p class="meta">{formatWhen(row.class.starts_at)} · {row.class.trainer.name}</p>
    {#each row.bookings as booking (booking.booking_id)}
      <p>{booking.name}</p>
      {#if booking.attendance}
        <p class="meta">пришёл, {booking.attendance.payment === 'paid' ? 'занятие списано' : 'без покрытия'}</p>
        <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
      {:else}
        <button
          class="primary"
          type="button"
          data-testid={hoursUntil(row.class.starts_at) <= 0.5 ? 'mark-arrived' : 'mark-early'}
          onclick={() => mark(booking.booking_id, true)}
        >Пришёл</button>
      {/if}
    {:else}
      <p class="muted">Записей нет.</p>
    {/each}
  </article>
{/each}
