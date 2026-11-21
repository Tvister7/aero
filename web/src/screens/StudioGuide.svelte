<script lang="ts">
  import { getStudio } from '../api'
  import Notice from '../lib/Notice.svelte'

  let studio = $derived(getStudio())
</script>

<h2 class="day">Как нас найти</h2>
{#if studio.status === 'pending'}
  <Notice kind="pending" />
{:else if studio.status === 'error'}
  <Notice kind="error" message={studio.message} reload={studio.reload} />
{:else if studio.data.guide}
  <article class="card guide">{studio.data.guide}</article>
{:else}
  <p class="card muted">Памятка пока не заполнена.</p>
{/if}
