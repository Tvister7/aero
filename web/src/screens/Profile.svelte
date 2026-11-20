<script lang="ts">
  import { getMe, listEnrollments, patchMe, putEnrollments } from '../api'
  import { balanceText, policyText, weekdayName } from '../lib/text'

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

  let options = $derived(listEnrollments().data)
  let query = $state('')
  let enrolled = $derived(options.filter((item) => item.enrolled))
  let found = $derived.by(() => {
    const needle = query.trim().toLowerCase()
    if (!needle) return []
    return options.filter((item) => {
      if (item.enrolled) return false
      return item.title.toLowerCase().includes(needle) || item.trainer_name.toLowerCase().includes(needle)
    })
  })
  let enrollError = $state('')

  async function save(event: SubmitEvent) {
    event.preventDefault()
    await patchMe({ name, phone })
  }

  async function setEnrolled(id: string, on: boolean) {
    const ids = options.filter((item) => item.enrolled && item.template_id !== id).map((item) => item.template_id)
    if (on) ids.push(id)
    const response = await putEnrollments(ids)
    enrollError = response.ok ? '' : response.message
  }
</script>

<article class="card">
  <p class="muted">Баланс</p>
  <p class="balance" data-testid="balance">{balanceText(me.data.balance)}</p>
  {#if me.data.balance < 0}
    <p>Не хватает {Math.abs(me.data.balance)} занятий до нуля.</p>
  {/if}
  <p class="meta">Режим записи: {policyText(me.data.policy)}</p>
  <p><a href="#/visits">История посещений</a></p>
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

<section>
  <h2 class="day">Постоянные записи</h2>
  {#if enrollError}<p class="card error">{enrollError}</p>{/if}
  {#if options.length === 0}
    <p class="card muted">Активных шаблонов нет.</p>
  {:else if enrolled.length === 0}
    <p class="card muted">Постоянных записей нет.</p>
  {/if}
  {#each enrolled as slot (slot.template_id)}
    <article class="card">
      <div class="slot">
        <div>
          <strong>{slot.title}</strong>
          <p class="meta">{slot.trainer_name}, {weekdayName(slot.weekday)} {slot.start_time}</p>
        </div>
        <button class="ghost" type="button" onclick={() => setEnrolled(slot.template_id, false)}>Записан</button>
      </div>
    </article>
  {/each}
  <form class="card" onsubmit={(event) => event.preventDefault()}>
    <label for="enroll-search">Найти занятие</label>
    <input id="enroll-search" bind:value={query} placeholder="Тип или тренер" />
  </form>
  {#each found as slot (slot.template_id)}
    <article class="card">
      <div class="slot">
        <div>
          <strong>{slot.title}</strong>
          <p class="meta">{slot.trainer_name}, {weekdayName(slot.weekday)} {slot.start_time}</p>
        </div>
        <button class="primary" type="button" onclick={() => setEnrolled(slot.template_id, true)}>Записаться</button>
      </div>
    </article>
  {/each}
</section>
