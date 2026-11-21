<script lang="ts">
  import { listClients } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { balanceText, policyText } from '../lib/text'

  let query = $state('')
  let loaded = $derived(listClients(query))
  let clients = $derived(loaded.status === 'ready' ? loaded.data : [])
</script>

<h2 class="day">Клиенты</h2>
{#if loaded.status === 'pending'}
  <Notice kind="pending" />
{:else if loaded.status === 'error'}
  <Notice kind="error" message={loaded.message} reload={loaded.reload} />
{:else}
<div class="card">
  <label for="q">Поиск</label>
  <input id="q" bind:value={query} placeholder="имя, телефон, ник" />
</div>
{#each clients as client (client.id)}
  <a class="row" href={`#/admin/clients/${client.id}`} data-testid={client.id === 'client-dev' ? 'client-dev' : undefined}>
    <strong>{client.name}</strong>
    <span class="meta">{balanceText(client.balance)} · {policyText(client.policy)}</span>
  </a>
{:else}
  <p class="card muted">Клиентов нет.</p>
{/each}
{/if}
