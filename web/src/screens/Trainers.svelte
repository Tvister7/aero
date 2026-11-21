<script lang="ts">
  import { createTrainer, listTrainers, patchTrainer } from '../api'
  import { focusField } from '../lib/focus'
  import Notice from '../lib/Notice.svelte'

  let loaded = $derived(listTrainers())
  let trainers = $derived(loaded.status === 'ready' ? loaded.data : [])
  let name = $state('')
  let error = $state('')

  async function add(event: SubmitEvent) {
    event.preventDefault()
    const response = await createTrainer(name)
    error = response.ok ? '' : response.message
    if (response.ok) name = ''
  }
</script>

<h2 class="day">Тренеры</h2>
{#if loaded.status === 'pending'}
  <Notice kind="pending" />
{:else if loaded.status === 'error'}
  <Notice kind="error" message={loaded.message} reload={loaded.reload} />
{:else}
<button class="ghost add" type="button" data-testid="add-trainer" onclick={() => focusField('trainer-name')}>Добавить</button>
{#each trainers as trainer (trainer.id)}
  <article class="card">
    <a href={`#/admin/trainers/${trainer.id}`}><strong>{trainer.name}</strong></a>
    {#if !trainer.active}<p class="tag">выключен</p>{/if}
    <button class="ghost" class:off={!trainer.active} type="button" onclick={() => { void patchTrainer(trainer.id, { active: !trainer.active }) }}>
      {trainer.active ? 'Выключить' : 'Включить'}
    </button>
  </article>
{/each}
<form class="card composer" onsubmit={add}>
  <h3>Новый тренер</h3>
  <label for="trainer-name">Имя</label>
  <input id="trainer-name" bind:value={name} />
  <button class="primary" type="submit">Добавить</button>
  {#if error}<Notice kind="error" message={error} />{/if}
</form>
{/if}
