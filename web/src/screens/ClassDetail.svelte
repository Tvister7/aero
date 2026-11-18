<script lang="ts">
  import { bookClass, getClass, getMe } from '../api'
  import { formatWhen } from '../lib/time'
  import { balanceText, priceLabel } from '../lib/text'

  let { classId }: { classId: string } = $props()
  let card = $derived(getClass(classId))
  let me = $derived(getMe())
  let name = $state('')
  let phone = $state('')
  let filled = $state(false)
  let error = $state('')
  let done = $state('')

  $effect(() => {
    if (filled || me.pending || !me.data) return
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

{#if !card.ok}
  {#if card.message}<p class="card error">{card.message}</p>{/if}
{:else}
  <article class="card">
    <h2>{card.data.title}</h2>
    <p class="meta">{formatWhen(card.data.starts_at)} – {formatWhen(card.data.ends_at)}</p>
    <p class="meta">{card.data.trainer.name}</p>
    <p>свободно {card.data.free_places} из {card.data.capacity}</p>
    {#if priceLabel(card.data.price_min, card.data.price_max)}
      <p class="price">стоимость {priceLabel(card.data.price_min, card.data.price_max)}</p>
    {/if}
    {#if me.data.visited && me.data.balance <= 0}
      <p data-testid="empty-balance">На балансе нет занятий.</p>
    {/if}
  </article>

  <form class="card" onsubmit={submit}>
    <label for="name">Имя</label>
    <input id="name" bind:value={name} autocomplete="name" />
    <label for="phone">Телефон</label>
    <input id="phone" bind:value={phone} autocomplete="tel" />
    <button class="primary" type="submit" data-testid="book-submit">Записаться</button>
    {#if error}<p class="error" data-testid="form-error">{error}</p>{/if}
    {#if done}<p data-testid="book-done">{done}</p>{/if}
  </form>
{/if}
