<script lang="ts">
  import { clientsCsv, stats } from '../api'
  import { todayKey } from '../lib/time'

  let from = $state(`${todayKey().slice(0, 8)}01`)
  let to = $state(todayKey())
  let report = $derived(stats(from, to).data)

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
<article class="card">
  <p>Открыли приложение: {report.unique_users}</p>
  <p>Записей: {report.bookings_created}</p>
  <p>Отмен: {report.bookings_cancelled}</p>
  <p>Явок: {report.arrivals}</p>
  <p>Неявок: {report.no_shows}</p>
  <p>Визитов без покрытия: {report.unpaid_visits}</p>
  <button class="primary" type="button" onclick={download}>Скачать контакты</button>
</article>
