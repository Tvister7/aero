<script lang="ts">
  import { getClient, grantCredits, patchClient } from '../api'
  import { balanceText, policyText } from '../lib/text'
  import { formatWhen } from '../lib/time'

  let { clientId }: { clientId: string } = $props()
  let card = $derived(getClient(clientId))
  let delta = $state('3')
  let note = $state('')
  let error = $state('')

  const reasons = { grant: 'начисление', attend: 'явка', reversal: 'снятие явки' }

  function grant(event: SubmitEvent) {
    event.preventDefault()
    const response = grantCredits(clientId, Number(delta), note)
    error = response.ok ? '' : response.message
    if (response.ok) note = ''
  }
</script>

{#if !card.ok}
  <p class="card error">{card.message}</p>
{:else}
  <article class="card">
    <h2>{card.data.name}</h2>
    <p class="balance" data-testid="client-balance">{balanceText(card.data.balance)}</p>
    <p class="meta">{card.data.phone} · @{card.data.username}</p>
    <label for="policy">Режим записи</label>
    <select
      id="policy"
      value={card.data.policy}
      onchange={(event: Event) => {
        const value = (event.currentTarget as HTMLSelectElement).value
        if (value === 'postpay' || value === 'restricted') patchClient(clientId, value)
      }}
    >
      <option value="postpay">можно в долг</option>
      <option value="restricted">только пакет</option>
    </select>
    <p class="muted">Сейчас: {policyText(card.data.policy)}</p>
  </article>
  <form class="card" onsubmit={grant}>
    <label for="delta">Начислить занятия</label>
    <input id="delta" type="number" min="1" step="1" bind:value={delta} data-testid="grant-delta" />
    <label for="note">Комментарий</label>
    <input id="note" bind:value={note} />
    <button class="primary" type="submit" data-testid="grant-submit">Начислить</button>
    {#if error}<p class="error" data-testid="form-error">{error}</p>{/if}
  </form>
  <h2 class="day">Проводки</h2>
  {#each card.data.ledger as row (row.id)}
    <article class="card">
      <strong>{row.delta > 0 ? '+' : ''}{row.delta}</strong>
      <p class="meta">{reasons[row.reason]}{row.note ? ` · ${row.note}` : ''}</p>
      <p class="meta">{formatWhen(row.created_at)}</p>
    </article>
  {:else}
    <p class="card muted">Проводок пока нет.</p>
  {/each}
{/if}
