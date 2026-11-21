<script lang="ts">
  import { trainerStats } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { todayKey } from '../lib/time'

  let from = $state(`${todayKey().slice(0, 8)}01`)
  let to = $state(todayKey())
  let report = $derived(trainerStats(from, to))
</script>

<h2 class="day">Сводка</h2>
<div class="card">
  <label for="trainer-from">С</label>
  <input id="trainer-from" type="date" bind:value={from} />
  <label for="trainer-to">По</label>
  <input id="trainer-to" type="date" bind:value={to} />
</div>
{#if report.status === 'pending'}
  <Notice kind="pending" />
{:else if report.status === 'error'}
  <Notice kind="error" message={report.message} reload={report.reload} />
{:else}
  <article class="card">
    <p>Клиентов с явкой: {report.data.clients}</p>
    <p>Списаний: {report.data.attends}</p>
  </article>
{/if}
