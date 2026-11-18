<script lang="ts">
  import { createTemplate, generateWeek, listTemplates, listTrainers, listTypes, patchTemplate } from '../api'
  import { weekdayName } from '../lib/time'

  let templates = $derived(listTemplates().data)
  let trainers = $derived(listTrainers().data.filter((item) => item.active))
  let types = $derived(listTypes().data.filter((item) => item.active))
  let trainerId = $state('')
  let typeId = $state('')
  let weekday = $state('1')
  let startTime = $state('10:00')
  let weekStart = $state('')
  let notice = $state('')

  $effect(() => {
    if (!trainerId && trainers[0]) trainerId = trainers[0].id
    if (!typeId && types[0]) typeId = types[0].id
  })

  function add(event: SubmitEvent) {
    event.preventDefault()
    const response = createTemplate({
      trainer_id: trainerId,
      training_type_id: typeId,
      weekday: Number(weekday),
      start_time: startTime,
    })
    notice = response.ok ? 'Шаблон добавлен. Текущие занятия не изменились.' : response.message
  }

  function generate() {
    const response = generateWeek(weekStart || undefined)
    notice = `Неделя ${response.week_start}: создано ${response.created}, обновлено ${response.updated}.`
  }
</script>

<h2 class="day">Шаблоны</h2>
{#if notice}<p class="card">{notice}</p>{/if}
<div class="card">
  <label for="week">Понедельник недели</label>
  <input id="week" type="date" bind:value={weekStart} />
  <button class="primary" type="button" onclick={generate}>Собрать неделю</button>
  <p class="muted">Пустая дата собирает следующую неделю. Повтор не плодит слоты.</p>
</div>
{#each templates as template (template.id)}
  <article class="card">
    <strong>{template.title}</strong>
    <p class="meta">{weekdayName(template.weekday)}, {template.start_time}</p>
    <button class="ghost" type="button" onclick={() => patchTemplate(template.id, { active: !template.active })}>
      {template.active ? 'Выключить' : 'Включить'}
    </button>
  </article>
{/each}
<form class="card" onsubmit={add}>
  <h3>Новый шаблон</h3>
  <label for="tpl-trainer">Тренер</label>
  <select id="tpl-trainer" bind:value={trainerId}>
    {#each trainers as trainer (trainer.id)}<option value={trainer.id}>{trainer.name}</option>{/each}
  </select>
  <label for="tpl-type">Тип</label>
  <select id="tpl-type" bind:value={typeId}>
    {#each types as type (type.id)}<option value={type.id}>{type.title}</option>{/each}
  </select>
  <label for="tpl-day">День</label>
  <select id="tpl-day" bind:value={weekday}>
    {#each [1, 2, 3, 4, 5, 6, 7] as day}
      <option value={String(day)}>{weekdayName(day)}</option>
    {/each}
  </select>
  <label for="tpl-time">Время</label>
  <input id="tpl-time" type="time" bind:value={startTime} />
  <button class="primary" type="submit">Добавить шаблон</button>
</form>
