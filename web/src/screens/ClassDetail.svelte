<script lang="ts">
  import { bookClass, getClass, getMe } from '../api'
  import Notice from '../lib/Notice.svelte'
  import { formatWhen, hoursUntil } from '../lib/time'
  import { balanceText, singlePrice } from '../lib/text'

  let { classId }: { classId: string } = $props()
  let card = $derived(getClass(classId))
  let me = $derived(getMe())
  let name = $state('')
  let phone = $state('')
  let filled = $state(false)
  let error = $state('')
  let done = $state('')

  $effect(() => {
    if (filled || me.status !== 'ready') return
    name = me.data.name ?? ''
    phone = me.data.phone ?? ''
    filled = true
  })

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    const response = await bookClass(classId, { name, phone })
    if (!response.ok) {
      error = response.message
      done = ''
      return
    }
    error = ''
    done = `Вы записаны. На балансе ${balanceText(response.balance)}.`
  }
</script>

{#if card.status === 'pending'}
  <Notice kind="pending" />
{:else if card.status === 'error'}
  <Notice kind="error" message={card.message} reload={card.reload} />
{:else}
  <article class="card">
    <h2>{card.data.title}</h2>
    <p class="meta">{formatWhen(card.data.starts_at)} – {formatWhen(card.data.ends_at)}</p>
    {#if card.data.kind === 'hold'}<p class="muted">Запись на это время не открыта.</p>{/if}
    {#if card.data.trainer.name}<p class="meta">{card.data.trainer.name}</p>{/if}
    {#if card.data.kind !== 'hold'}<p>свободно {card.data.free_places} из {card.data.capacity}</p>{/if}
    {#if singlePrice(card.data.price_max)}
      <p class="price">стоимость {singlePrice(card.data.price_max)}</p>
    {/if}
    {#if me.status === 'ready' && me.data.visited && me.data.balance <= 0}
      <p data-testid="empty-balance">На балансе нет занятий.</p>
    {/if}
  </article>

  {#if card.data.kind === 'group' && hoursUntil(card.data.starts_at) <= 0}
    <p class="card muted">Занятие уже началось.</p>
  {:else if card.data.kind === 'group'}
  <form class="card" onsubmit={submit}>
    <label for="name">Имя</label>
    <input id="name" bind:value={name} autocomplete="name" />
    <label for="phone">Телефон</label>
    <input id="phone" bind:value={phone} autocomplete="tel" />
    <button class="primary" type="submit" data-testid="book-submit">Записаться</button>
    {#if error}<Notice kind="error" message={error} testid="form-error" />{/if}
    {#if done}<p class="notice ok" data-testid="book-done">{done}</p>{/if}
  </form>
  {/if}
{/if}
