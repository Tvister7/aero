<script lang="ts">
  import { listClients } from '../api'
  import { balanceText, policyText } from '../lib/text'

  let query = $state('')
  let clients = $derived(listClients(query).data)
</script>

<h2 class="day">Клиенты</h2>
<div class="card">
  <label for="q">Поиск</label>
  <input id="q" bind:value={query} placeholder="имя, телефон, ник" />
</div>
{#each clients as client (client.id)}
  <a class="row" href={`#/admin/clients/${client.id}`} data-testid={client.id === 'client-dev' ? 'client-dev' : undefined}>
    <strong>{client.name}</strong>
    <span class="meta">{balanceText(client.balance)} · {policyText(client.policy)}</span>
  </a>
{/each}
