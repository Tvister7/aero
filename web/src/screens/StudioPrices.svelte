<script lang="ts">
  import { getStudio } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { pricingBlocks } from '../lib/pricing'

  let studio = $derived(getStudio())
  let blocks = $derived(studio.status === 'ready' ? pricingBlocks(studio.data.pricing) : [])
</script>

<h2 class="day">Цены</h2>
{#if studio.status === 'pending'}
  <Notice kind="pending" />
{:else if studio.status === 'error'}
  <Notice kind="error" message={studio.message} reload={studio.reload} />
{:else}
  {#each blocks as block (block.title)}
    <article class="card">
      <h3>{block.title}</h3>
      {#each block.rows as row (row.label)}
        <p class="slot"><span>{row.label}</span><strong>{row.price}</strong></p>
      {/each}
    </article>
  {:else}
    <p class="card muted">Цены пока не заполнены.</p>
  {/each}
{/if}
