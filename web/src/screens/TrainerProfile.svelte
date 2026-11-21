<script lang="ts">
  import { getTrainerProfile, patchTrainerProfile } from '../api'
  import Notice from '../lib/Notice.svelte'

  let profile = $derived(getTrainerProfile())
  let about = $state('')
  let phone = $state('')
  let filled = $state(false)
  let notice = $state('')
  let failed = $state(false)

  $effect(() => {
    if (filled || profile.status !== 'ready' || !profile.data.id) return
    about = profile.data.about
    phone = profile.data.phone
    filled = true
  })

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const response = await patchTrainerProfile({ about, phone })
    failed = !response.ok
    notice = response.ok ? 'Сохранено.' : response.message
  }
</script>

{#if profile.status === 'pending'}
  <Notice kind="pending" />
{:else if profile.status === 'error'}
  <Notice kind="error" message={profile.message} reload={profile.reload} />
{:else}
  <h2 class="day">{profile.data.name || 'Карточка'}</h2>
  <form class="card" onsubmit={save}>
    <label for="own-about">О тренере</label>
    <textarea id="own-about" rows="3" bind:value={about}></textarea>
    <label for="own-phone">Телефон</label>
    <input id="own-phone" bind:value={phone} />
    <button class="primary" type="submit">Сохранить</button>
    {#if notice}<Notice kind={failed ? 'error' : 'ok'} message={notice} />{/if}
  </form>
{/if}
