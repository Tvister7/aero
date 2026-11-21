<script lang="ts">
  import { getStudio } from '../api'
  import { pricingBlocks } from '../lib/pricing'

  let blocks = $derived(pricingBlocks(getStudio().data.pricing))
</script>

<h2 class="day">Цены</h2>
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
