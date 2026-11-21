<script lang="ts">
  import { setAttendance, todayRoster } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { attendanceEarlyText, attendanceTooEarly, formatWhen } from '../lib/time'

  let rows = $derived(todayRoster())
  let notice = $state('')
  let failed = $state(false)
  let openId = $state('')

  async function mark(bookingId: string, arrived: boolean) {
    if (!arrived && !confirm('Снять явку? Сообщение не уйдёт. Если занятие было списано, оно вернётся на баланс.')) return
    const response = await setAttendance(bookingId, arrived)
    failed = !response.ok
    notice = response.ok ? '' : response.message
  }
</script>

<h2 class="day">Журнал на сегодня</h2>
{#if notice}<Notice kind={failed ? 'error' : 'ok'} message={notice} testid={failed ? 'form-error' : undefined} />{/if}
{#if rows.status === 'pending'}
  <Notice kind="pending" />
{:else if rows.status === 'error'}
  <Notice kind="error" message={rows.message} reload={rows.reload} />
{:else if rows.data.length === 0}
  <p class="card muted">Сегодня занятий нет.</p>
{:else}
  {#each rows.data as row (row.class.id)}
    {@const arrived = row.bookings.filter((booking) => booking.attendance).length}
    {@const open = openId === row.class.id}
    {@const early = attendanceTooEarly(row.class.starts_at)}
    <article class="card">
      <button class="fold" type="button" aria-expanded={open} onclick={() => (openId = open ? '' : row.class.id)}>
        <strong>{row.class.title}</strong>
        <p class="meta">{formatWhen(row.class.starts_at)} · {row.class.trainer.name}</p>
        <p class="meta">пришло {arrived} из {row.bookings.length}</p>
      </button>
      {#if open}
        {#each row.bookings as booking (booking.booking_id)}
          <p>{booking.name}</p>
          {#if booking.attendance}
            <p class="meta">пришёл, {booking.attendance.payment === 'paid' ? 'занятие списано' : 'без покрытия'}</p>
            <button class="ghost" type="button" onclick={() => mark(booking.booking_id, false)}>Снять отметку</button>
          {:else}
            <button
              class="primary"
              type="button"
              disabled={early}
              data-testid={early ? 'mark-early' : 'mark-arrived'}
              onclick={() => mark(booking.booking_id, true)}
            >Пришёл</button>
            {#if early}<p class="meta">{attendanceEarlyText}</p>{/if}
          {/if}
        {:else}
          <p class="muted">Записей нет.</p>
        {/each}
      {/if}
    </article>
  {/each}
{/if}
