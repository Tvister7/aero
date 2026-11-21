<script lang="ts">
  import { getClient, grantCredits, patchClient } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { balanceText, policyText } from '../lib/text'
  import { formatWhen } from '../lib/time'

  let { clientId }: { clientId: string } = $props()
  let card = $derived(getClient(clientId))
  let delta = $state('3')
  let note = $state('')
  let error = $state('')

  const reasons = { grant: 'начисление', attend: 'явка', reversal: 'снятие явки' }

  async function grant(event: SubmitEvent) {
    event.preventDefault()
    const response = await grantCredits(clientId, Number(delta), note)
    error = response.ok ? '' : response.message
    if (response.ok) note = ''
  }
</script>

{#if card.status === 'pending'}
  <Notice kind="pending" />
{:else if card.status === 'error'}
  <Notice kind="error" message={card.message} reload={card.reload} />
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
        if (value === 'postpay' || value === 'restricted') void patchClient(clientId, value)
      }}
    >
      <option value="postpay">постоплата</option>
      <option value="restricted">абонемент</option>
    </select>
    <p class="muted">Сейчас: {policyText(card.data.policy)}</p>
  </article>
  <form class="card" onsubmit={grant}>
    <label for="delta">Начислить занятия</label>
    <input id="delta" type="number" min="1" step="1" bind:value={delta} data-testid="grant-delta" />
    <label for="note">Комментарий</label>
    <input id="note" bind:value={note} />
    <button class="primary" type="submit" data-testid="grant-submit">Начислить</button>
    {#if error}<Notice kind="error" message={error} testid="form-error" />{/if}
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
