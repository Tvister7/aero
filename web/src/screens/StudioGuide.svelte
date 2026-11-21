<script lang="ts">
  import { getStudio } from '../api'

  let studio = $derived(getStudio().data)
</script>

<h2 class="day">Как нас найти</h2>
{#if studio.guide}
  <article class="card guide">{studio.guide}</article>
{:else}
  <p class="card muted">Памятка пока не заполнена.</p>
{/if}
