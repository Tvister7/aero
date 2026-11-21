<script lang="ts">
  import { listPublicTrainers } from '../api'
  import Notice from '../lib/Notice.svelte'

  let rows = $derived(listPublicTrainers())
</script>

<h2 class="day">Тренеры</h2>
{#if rows.status === 'pending'}
  <Notice kind="pending" />
{:else if rows.status === 'error'}
  <Notice kind="error" message={rows.message} reload={rows.reload} />
{:else if rows.data.length === 0}
  <p class="card muted">Тренеров пока нет.</p>
{:else}
  {#each rows.data as trainer (trainer.id)}
    <article class="card">
      <a href={`#/studio/trainers/${trainer.id}`}><strong>{trainer.name}</strong></a>
      {#if trainer.classes.length}<p class="meta">{trainer.classes.join(', ')}</p>{/if}
    </article>
  {/each}
{/if}
