<script lang="ts">
  import { createType, listTypes } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { shownColor } from '../lib/color'
  import { focusField } from '../lib/focus'
  import { priceLabel } from '../lib/text'

  let loaded = $derived(listTypes())
  let types = $derived(loaded.status === 'ready' ? loaded.data : [])
  let title = $state('')
  let duration = $state('60')
  let capacity = $state('8')
  let priceMin = $state('17')
  let priceMax = $state('35')
  let error = $state('')

  async function add(event: SubmitEvent) {
    event.preventDefault()
    const response = await createType({
      title,
      duration_minutes: Number(duration),
      capacity: Number(capacity),
      price_min: priceMin,
      price_max: priceMax,
    })
    error = response.ok ? '' : response.message
    if (response.ok) title = ''
  }
</script>

<h2 class="day">Типы занятий</h2>
{#if loaded.status === 'pending'}
  <Notice kind="pending" />
{:else if loaded.status === 'error'}
  <Notice kind="error" message={loaded.message} reload={loaded.reload} />
{:else}
<button class="ghost add" type="button" onclick={() => focusField('type-title')}>Добавить</button>
<p class="card muted">Стоимость здесь только подпись на карточке. Учёт идёт в занятиях.</p>
{#each types as type (type.id)}
  <article class="card">
    <strong><i class="swatch" style:background={shownColor(type.color, type.title)}></i>{type.title}</strong>
    <p class="meta">{type.duration_minutes} мин · мест {type.capacity}</p>
    {#if priceLabel(type.price_min, type.price_max)}<p class="price">{priceLabel(type.price_min, type.price_max)}</p>{/if}
  </article>
{/each}
<form class="card composer" onsubmit={add}>
  <h3>Новый тип</h3>
  <label for="type-title">Название</label>
  <input id="type-title" bind:value={title} />
  <label for="type-duration">Минуты</label>
  <input id="type-duration" type="number" bind:value={duration} />
  <label for="type-capacity">Мест</label>
  <input id="type-capacity" type="number" bind:value={capacity} />
  <label for="type-min">Подпись от</label>
  <input id="type-min" bind:value={priceMin} />
  <label for="type-max">Подпись до</label>
  <input id="type-max" bind:value={priceMax} />
  <button class="primary" type="submit">Добавить</button>
  {#if error}<Notice kind="error" message={error} />{/if}
</form>
{/if}
