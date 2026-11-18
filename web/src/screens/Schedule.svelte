<script lang="ts">
  import { getAdminSchedule, getSchedule } from '../api'
  import { addDays, dateKeyOfIso, formatDay, formatWhen, hoursUntil, mondayKey, todayKey } from '../lib/time'
  import { priceLabel } from '../lib/text'

  let { admin = false }: { admin?: boolean } = $props()
  let start = $state(mondayKey(todayKey()))
  let end = $derived(addDays(start, 6))
  let result = $derived(admin ? getAdminSchedule(start, end) : getSchedule(start, end))
  let bookableId = $derived(result.data.find((item) => item.free_places > 0 && hoursUntil(item.starts_at) > 6)?.id)
  let groups = $derived.by(() => {
    const map = new Map()
    for (const item of result.data) {
      const key = dateKeyOfIso(item.starts_at)
      if (!map.has(key)) map.set(key, [])
      map.get(key).push(item)
    }
    return [...map.entries()]
  })
</script>

<div class="weekbar">
  <button class="ghost" type="button" onclick={() => (start = addDays(start, -7))}>Назад</button>
  <span class="muted">{formatDay(start)}</span>
  <button class="ghost" type="button" onclick={() => (start = addDays(start, 7))}>Дальше</button>
</div>

{#if groups.length === 0}
  <p class="card muted">На этой неделе занятий нет.</p>
{/if}

{#each groups as [day, items] (day)}
  <h2 class="day">{formatDay(day)}</h2>
  {#each items as item (item.id)}
    <a
      class="row"
      href={admin ? `#/admin/class/${item.id}` : `#/class/${item.id}`}
      data-testid={item.id === bookableId ? 'bookable' : item.free_places === 0 && item.status !== 'cancelled' ? 'class-full' : undefined}
    >
      <strong>{item.title}</strong>
      <span class="meta">{formatWhen(item.starts_at)} · {item.trainer.name}</span>
      <span class="meta">свободно {item.free_places} из {item.capacity}</span>
      {#if priceLabel(item.price_min, item.price_max)}
        <span class="price">{priceLabel(item.price_min, item.price_max)}</span>
      {/if}
      {#if item.status === 'cancelled'}
        <span class="tag">отменено</span>
      {/if}
    </a>
  {/each}
{/each}
