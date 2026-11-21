<script lang="ts">
  import { cancelBooking, getSchedule, myBookings, rescheduleBooking } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { addDays, formatWhen, hoursUntil, todayKey } from '../lib/time'

  let list = $derived(myBookings())
  let schedule = $derived(getSchedule(todayKey(), addDays(todayKey(), 21)))
  let options = $derived(schedule.status === 'ready' ? schedule.data : [])
  let error = $state('')
  let moving = $state('')
  let target = $state('')

  async function cancel(id: string) {
    if (!confirm('Снять запись? Сообщение никому не уйдёт. Баланс не изменится.')) return
    const response = await cancelBooking(id)
    error = response.ok ? '' : response.message
  }

  async function move(id: string) {
    const response = await rescheduleBooking(id, target)
    error = response.ok ? '' : response.message
    if (response.ok) moving = ''
  }
</script>

<h2 class="day">Мои записи</h2>
{#if list.status === 'pending'}
  <Notice kind="pending" />
{:else if list.status === 'error'}
  <Notice kind="error" message={list.message} reload={list.reload} />
{:else}
{#if error}<Notice kind="error" message={error} testid="form-error" />{/if}
{#if list.data.length === 0}
  <p class="card muted">Будущих записей нет.</p>
{/if}
{#each list.data as item (item.booking_id)}
  {@const hours = hoursUntil(item.class.starts_at)}
  {@const locked = hours <= 6}
  <article class="card">
    <strong>{item.class.title}</strong>
    <p class="meta">{formatWhen(item.class.starts_at)} · {item.class.trainer.name}</p>
    <div class="actions">
      <button
        class="danger"
        type="button"
        disabled={locked}
        data-testid={hours > 6 ? 'cancel-open' : 'cancel-late'}
        onclick={() => cancel(item.booking_id)}
      >Отменить</button>
      {#if !item.class.template_id && item.class.kind === 'group'}
        <button class="ghost" type="button" disabled={locked} data-testid="reschedule" onclick={() => (moving = moving === item.booking_id ? '' : item.booking_id)}>Перенести</button>
      {/if}
    </div>
    {#if locked}
      <p class="muted" data-testid="cutoff-reason">{hours <= 0 ? 'Занятие уже началось' : 'До начала меньше 6 часов'}</p>
    {/if}
    {#if moving === item.booking_id}
      <label for="target">Новое занятие</label>
      <select id="target" bind:value={target}>
        <option value="">Выберите слот</option>
        {#each options.filter((option) => option.training_type_id === item.class.training_type_id && option.id !== item.class.id && new Date(option.starts_at) > new Date()) as option (option.id)}
          <option value={option.id}>{option.title}, {formatWhen(option.starts_at)}</option>
        {/each}
      </select>
      <button class="primary" type="button" onclick={() => move(item.booking_id)}>Подтвердить перенос</button>
    {/if}
  </article>
{/each}
{/if}
