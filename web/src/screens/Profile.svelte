<script lang="ts">
  import { getMe, patchMe } from '../api'
  import { balanceText, policyText } from '../lib/text'

  let me = $derived(getMe())
  let name = $state('')
  let phone = $state('')
  let filled = $state(false)

  $effect(() => {
    if (filled || me.pending || !me.data) return
    name = me.data.name ?? ''
    phone = me.data.phone ?? ''
    filled = true
  })

  async function save(event: SubmitEvent) {
    event.preventDefault()
    await patchMe({ name, phone })
  }
</script>

<article class="card">
  <p class="muted">Баланс</p>
  <p class="balance" data-testid="balance">{balanceText(me.data.balance)}</p>
  {#if me.data.balance < 0}
    <p>Не хватает {Math.abs(me.data.balance)} занятий до нуля.</p>
  {/if}
  <p class="meta">Режим записи: {policyText(me.data.policy)}</p>
  {#if me.data.visited && me.data.balance <= 0}
    <p data-testid="empty-balance">На балансе нет занятий.</p>
  {/if}
  {#if me.data.username}<p class="meta">@{me.data.username}</p>{/if}
</article>

<form class="card" onsubmit={save}>
  <label for="profile-name">Имя</label>
  <input id="profile-name" bind:value={name} />
  <label for="profile-phone">Телефон</label>
  <input id="profile-phone" bind:value={phone} />
  <button class="primary" type="submit">Сохранить</button>
</form>
