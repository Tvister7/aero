<script lang="ts">
  import { createTrainer, listTrainers, patchTrainer } from '../api'

  let trainers = $derived(listTrainers().data)
  let name = $state('')
  let error = $state('')

  function add(event: SubmitEvent) {
    event.preventDefault()
    const response = createTrainer(name)
    error = response.ok ? '' : response.message
    if (response.ok) name = ''
  }
</script>

<h2 class="day">Тренеры</h2>
{#each trainers as trainer (trainer.id)}
  <article class="card">
    <strong>{trainer.name}</strong>
    {#if !trainer.active}<p class="tag">выключен</p>{/if}
    <button class="ghost" type="button" onclick={() => patchTrainer(trainer.id, { active: !trainer.active })}>
      {trainer.active ? 'Выключить' : 'Включить'}
    </button>
  </article>
{/each}
<form class="card" onsubmit={add}>
  <label for="trainer-name">Имя</label>
  <input id="trainer-name" bind:value={name} />
  <button class="primary" type="submit">Добавить</button>
  {#if error}<p class="error">{error}</p>{/if}
</form>
