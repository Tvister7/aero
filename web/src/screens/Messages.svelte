<script lang="ts">
  import { listMessages, putMessage } from '../api'
  import Notice from '../lib/Notice.svelte'

  let list = $derived(listMessages())
  let drafts = $state<Record<string, string>>({})
  let notices = $state<Record<string, { ok: boolean; text: string }>>({})
  let filled = $state<Record<string, boolean>>({})

  $effect(() => {
    if (list.status !== 'ready') return
    for (const item of list.data) {
      if (filled[item.key]) continue
      drafts[item.key] = item.body
      filled[item.key] = true
    }
  })

  async function save(key: string, event: SubmitEvent) {
    event.preventDefault()
    const response = await putMessage(key, drafts[key] ?? '')
    notices[key] = {
      ok: response.ok,
      text: response.ok ? 'Сообщение сохранено.' : response.message,
    }
    if (response.ok) drafts[key] = response.data.body
  }
</script>

{#if list.status === 'pending'}
  <Notice kind="pending" />
{:else if list.status === 'error'}
  <Notice kind="error" message={list.message} reload={list.reload} />
{:else}
  {#each list.data as item (item.key)}
    <form class="card" onsubmit={(event) => save(item.key, event)}>
      <label for={`message-${item.key}`}>{item.title}</label>
      {#if item.key === 'start'}
        <p class="muted">Этот текст бот отвечает на /start. Кнопка «Открыть приложение» добавляется сама, когда у бота есть адрес Mini App.</p>
      {/if}
      <textarea id={`message-${item.key}`} rows="8" bind:value={drafts[item.key]}></textarea>
      <button class="primary" type="submit">Сохранить</button>
      {#if notices[item.key]}
        <Notice kind={notices[item.key].ok ? 'ok' : 'error'} message={notices[item.key].text} />
      {/if}
    </form>
  {/each}
{/if}
