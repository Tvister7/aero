<script lang="ts">
  import { getPublicTrainer } from '../api'

  let { trainerId }: { trainerId: string } = $props()
  let card = $derived(getPublicTrainer(trainerId))
</script>

{#if card.ok}
  <h2 class="day">{card.data.name}</h2>
  <article class="card">
    {#if card.data.about}<p class="guide">{card.data.about}</p>{:else}<p class="muted">О тренере пока ничего не написано.</p>{/if}
    {#if card.data.phone}<p><a href={'tel:' + card.data.phone}>{card.data.phone}</a></p>{/if}
    {#if card.data.classes.length}
      <h3>Занятия</h3>
      {#each card.data.classes as title (title)}<p>{title}</p>{/each}
    {/if}
  </article>
{:else if card.message}
  <p class="card error">{card.message}</p>
{/if}
