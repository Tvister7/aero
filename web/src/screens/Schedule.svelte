<script lang="ts">
  import { onDestroy, tick } from 'svelte'
  import { getAdminSchedule, getSchedule } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { shownColor } from '../lib/color'
  import { singlePrice } from '../lib/text'
  import {
    addDays,
    dateKeyOfIso,
    formatDay,
    formatWeekRange,
    formatWhen,
    hoursUntil,
    mondayKey,
    monthBounds,
    monthCells,
    monthKeyOf,
    monthTitle,
    shiftMonth,
    todayKey,
  } from '../lib/time'

  let { admin = false }: { admin?: boolean } = $props()
  let mode = $state<'week' | 'month'>('week')
  let start = $state(mondayKey(todayKey()))
  let month = $state(monthKeyOf(todayKey()))
  let end = $derived(addDays(start, 6))
  let cells = $derived(monthCells(month))
  let bounds = $derived(monthBounds(month))
  let range = $derived(
    mode === 'week' ? { from: start, to: end } : { from: bounds.start, to: bounds.end },
  )
  let result = $derived(admin ? getAdminSchedule(range.from, range.to) : getSchedule(range.from, range.to))
  let classes = $derived(result.status === 'ready' ? result.data : [])
  let bookableId = $derived(classes.find((item) => item.free_places > 0 && hoursUntil(item.starts_at) > 6)?.id)
  let byDay = $derived.by(() => {
    const map = new Map<string, typeof classes>()
    for (const item of classes) {
      const key = dateKeyOfIso(item.starts_at)
      const list = map.get(key) ?? []
      list.push(item)
      map.set(key, list)
    }
    return map
  })
  let weekDays = $derived(Array.from({ length: 7 }, (_, index) => addDays(start, index)))
  let legend = $derived(dayMarks(classes))
  let focusDay = $state(todayKey())
  let focusTick = $state(1)
  let scrolled = 0
  let origin: { x: number; y: number; id: number } | null = null
  let swiped = false

  function dayMarks(items: Array<{ title: string; color?: string }>) {
    const seen = new Map<string, string>()
    for (const item of items) {
      if (!seen.has(item.title)) seen.set(item.title, shownColor(item.color, item.title))
    }
    return [...seen.entries()].map(([title, color]) => ({ title, color }))
  }

  function step(direction: 1 | -1) {
    if (mode === 'week') {
      focusDay = ''
      focusTick += 1
      start = addDays(start, direction * 7)
      return
    }
    month = shiftMonth(month, direction)
  }

  function clearGesture() {
    origin = null
    window.removeEventListener('pointerup', finishGesture)
    window.removeEventListener('pointercancel', clearGesture)
  }

  function beginGesture(event: PointerEvent) {
    if (event.button !== 0 || !event.isPrimary || origin) return
    swiped = false
    origin = { x: event.clientX, y: event.clientY, id: event.pointerId }
    window.addEventListener('pointerup', finishGesture)
    window.addEventListener('pointercancel', clearGesture)
  }

  function finishGesture(event: PointerEvent) {
    if (!origin || event.pointerId !== origin.id) return
    const dx = event.clientX - origin.x
    const dy = event.clientY - origin.y
    clearGesture()
    if (Math.abs(dx) < 64 || Math.abs(dx) < Math.abs(dy)) return
    swiped = true
    step(dx < 0 ? 1 : -1)
  }

  function swallowClick(event: MouseEvent) {
    if (!swiped) return
    swiped = false
    event.preventDefault()
    event.stopPropagation()
  }

  onDestroy(clearGesture)

  function showWeek() {
    if (mode === 'week') return
    mode = 'week'
    if (focusDay) focusTick += 1
  }

  function openDay(day: string) {
    focusDay = day
    focusTick += 1
    start = mondayKey(day)
    mode = 'week'
  }

  function scrollToDay(day: string) {
    const heading = document.getElementById(`day-${day}`)
    const bar = document.querySelector('.weekbar')
    if (!heading || !(bar instanceof HTMLElement)) return false
    const stick = Number.parseFloat(getComputedStyle(bar).top) + bar.getBoundingClientRect().height
    const top = heading.getBoundingClientRect().top + window.scrollY - stick - 8
    window.scrollTo({ top: Math.max(0, top), behavior: 'auto' })
    return true
  }

  $effect(() => {
    const token = focusTick
    const day = focusDay
    if (mode !== 'week' || result.status !== 'ready' || scrolled === token) return
    if (!day || day < start || day > end) {
      scrolled = token
      return
    }
    let cancel = false
    void tick().then(() => {
      if (cancel || token !== focusTick) return
      if (scrollToDay(day)) scrolled = token
    })
    return () => {
      cancel = true
    }
  })
</script>

<div class="weekbar">
  <div class="tabs" role="group" aria-label="Вид расписания">
    <button type="button" aria-pressed={mode === 'week'} onclick={showWeek}>Неделя</button>
    <button type="button" aria-pressed={mode === 'month'} data-testid="tab-month" onclick={() => (mode = 'month')}>Месяц</button>
  </div>
  {#if mode === 'week'}
    <div class="pager">
      <button class="ghost" type="button" aria-label="Предыдущая неделя" onclick={() => step(-1)}>←</button>
      <span class="muted" data-testid="week-range">{formatWeekRange(start, end)}</span>
      <button class="ghost" type="button" aria-label="Следующая неделя" onclick={() => step(1)}>→</button>
    </div>
  {:else}
    <div class="pager">
      <button class="ghost" type="button" aria-label="Предыдущий месяц" onclick={() => step(-1)}>←</button>
      <span class="muted" data-testid="month-title">{monthTitle(month)}</span>
      <button class="ghost" type="button" aria-label="Следующий месяц" onclick={() => step(1)}>→</button>
    </div>
  {/if}
</div>

<div class="schedule" role="region" aria-label="Расписание" onpointerdown={beginGesture} onclickcapture={swallowClick}>
{#if result.status === 'pending'}
  <Notice kind="pending" />
{:else if result.status === 'error'}
  <Notice kind="error" message={result.message} reload={result.reload} />
{:else if mode === 'week'}
  {#if classes.length === 0 && (!focusDay || focusDay < start || focusDay > end)}
    <p class="card muted">На этой неделе занятий нет.</p>
  {/if}
  {#each weekDays as day (day)}
    {@const items = byDay.get(day) ?? []}
    {#if items.length > 0 || day === focusDay}
    <h2 class="day" id="day-{day}">{formatDay(day)}</h2>
    {#each items as item (item.id)}
      {@const begun = hoursUntil(item.starts_at) <= 0}
      {#if admin || (item.kind !== 'hold' && !begun)}
        <a
          class="row accent"
          style:border-left-color={shownColor(item.color, item.title)}
          href={admin ? `#/admin/class/${item.id}` : `#/class/${item.id}`}
          data-testid={item.id === bookableId ? 'bookable' : item.free_places === 0 && item.status !== 'cancelled' ? 'class-full' : undefined}
        >
          {@render slot(item)}
        </a>
      {:else}
        <article class="row accent" class:past={begun} style:border-left-color={shownColor(item.color, item.title)}>
          {@render slot(item)}
        </article>
      {/if}
    {:else}
      <p class="card muted">В этот день занятий нет.</p>
    {/each}
    {/if}
  {/each}
{:else}
  <div class="month" data-testid="month-grid">
    {#each ['пн', 'вт', 'ср', 'чт', 'пт', 'сб', 'вс'] as name}
      <p>{name}</p>
    {/each}
    {#each cells as day (day)}
      {@const items = byDay.get(day) ?? []}
      {@const marks = dayMarks(items)}
      <button
        class="cell"
        class:outside={!day.startsWith(month)}
        type="button"
        aria-label={marks.length ? `${formatDay(day)}: ${marks.map((mark) => mark.title).join(', ')}` : `${formatDay(day)}, занятий нет`}
        onclick={() => openDay(day)}
      >
        <span class="num">{Number(day.slice(8))}</span>
        {#if marks.length}
          <span class="marks" aria-hidden="true">
            {#each marks as mark (mark.title)}<i style:background={mark.color}></i>{/each}
          </span>
        {/if}
      </button>
    {/each}
  </div>
  {#if legend.length}
    <ul class="legend">
      {#each legend as item (item.title)}
        <li><i style:background={item.color}></i>{item.title}</li>
      {/each}
    </ul>
  {/if}
{/if}
</div>

{#snippet slot(item: (typeof classes)[number])}
  <strong>{item.title}</strong>
  <span class="meta">{formatWhen(item.starts_at)}{#if item.trainer.name} · {item.trainer.name}{/if}</span>
  {#if item.kind !== 'hold'}
    <span class="meta">свободно {item.free_places} из {item.capacity}</span>
  {/if}
  {#if singlePrice(item.price_max)}
    <span class="price">{singlePrice(item.price_max)}</span>
  {/if}
  {#if item.status === 'cancelled'}
    <span class="tag">отменено</span>
  {/if}
{/snippet}
