<script lang="ts">
  import { myVisits } from '../api'
  import { formatWhen } from '../lib/time'

  let rows = $derived(myVisits())
</script>

<h2 class="day">История посещений</h2>
{#if rows.pending}
  <p class="card muted">Загрузка…</p>
{:else if rows.data.length === 0}
  <p class="card muted">Посещений пока нет.</p>
{:else}
  {#each rows.data as row (`${row.starts_at}-${row.title}`)}
    <article class="card">
      <strong>{row.title}</strong>
      <p class="meta">{formatWhen(row.starts_at)} · {row.trainer_name}</p>
      <p class="meta">{row.payment === 'paid' ? 'занятие списано' : 'без покрытия'}</p>
    </article>
  {/each}
{/if}
