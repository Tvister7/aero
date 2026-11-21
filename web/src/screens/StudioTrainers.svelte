<script lang="ts">
  import { getMe, listPublicTrainers, patchTrainer } from '../api'

  let rows = $derived(listPublicTrainers())
  let admin = $derived(Boolean(getMe().data?.is_admin))
  let error = $state('')

  async function disable(id: string) {
    const response = await patchTrainer(id, { active: false })
    error = response.ok ? '' : response.message
  }
</script>

<h2 class="day">Тренеры</h2>
{#if error}<p class="card error">{error}</p>{/if}
{#each rows.data as trainer (trainer.id)}
  <article class="card">
    <a href={`#/studio/trainers/${trainer.id}`}><strong>{trainer.name}</strong></a>
    {#if trainer.classes.length}<p class="meta">{trainer.classes.join(', ')}</p>{/if}
    {#if admin}
      <button class="ghost" type="button" onclick={() => disable(trainer.id)}>Выключить</button>
    {/if}
  </article>
{:else}
  {#if !rows.pending}<p class="card muted">Тренеров пока нет.</p>{/if}
{/each}
