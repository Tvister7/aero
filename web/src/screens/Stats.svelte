<script lang="ts">
  import { clientsCsv, stats } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { todayKey } from '../lib/time'

  let from = $state(`${todayKey().slice(0, 8)}01`)
  let to = $state(todayKey())
  let report = $derived(stats(from, to))

  async function download() {
    const blob = new Blob([await clientsCsv()], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'clients.csv'
    link.click()
    URL.revokeObjectURL(url)
  }
</script>

<div class="card">
  <label for="from">С</label>
  <input id="from" type="date" bind:value={from} />
  <label for="to">По</label>
  <input id="to" type="date" bind:value={to} />
</div>
{#if report.status === 'pending'}
  <Notice kind="pending" />
{:else if report.status === 'error'}
  <Notice kind="error" message={report.message} reload={report.reload} />
{:else}
  <article class="card">
    <p>Открыли приложение: {report.data.unique_users}</p>
    <p>Записей: {report.data.bookings_created}</p>
    <p>Отмен: {report.data.bookings_cancelled}</p>
    <p>Явок: {report.data.arrivals}</p>
    <p>Неявок: {report.data.no_shows}</p>
    <p>Визитов без покрытия: {report.data.unpaid_visits}</p>
    <button class="primary" type="button" onclick={download}>Скачать контакты</button>
  </article>
{/if}
