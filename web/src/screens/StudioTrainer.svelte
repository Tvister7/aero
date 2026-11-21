<script lang="ts">
  import { getPublicTrainer } from '../api'
  import Notice from '../lib/Notice.svelte'

  let { trainerId }: { trainerId: string } = $props()
  let card = $derived(getPublicTrainer(trainerId))
</script>

{#if card.status === 'pending'}
  <Notice kind="pending" />
{:else if card.status === 'error'}
  <Notice kind="error" message={card.message} reload={card.reload} />
{:else}
  <h2 class="day">{card.data.name}</h2>
  <article class="card">
    {#if card.data.about}<p class="guide">{card.data.about}</p>{:else}<p class="muted">О тренере пока ничего не написано.</p>{/if}
    {#if card.data.phone}<p><a href={'tel:' + card.data.phone}>{card.data.phone}</a></p>{/if}
    {#if card.data.classes.length}
      <h3>Занятия</h3>
      {#each card.data.classes as title (title)}<p>{title}</p>{/each}
    {/if}
  </article>
{/if}
