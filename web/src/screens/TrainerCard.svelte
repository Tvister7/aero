<script lang="ts">
  import { addAbsence, cancelTrainerClasses, deleteAbsence, deleteTrainer, getTrainer, listClients, patchTrainer } from '../api'
  import { formatShortDate } from '../lib/time'

  let { trainerId }: { trainerId: string } = $props()
  let loaded = $derived(getTrainer(trainerId))
  let clients = $derived(listClients().data)
  let name = $state('')
  let about = $state('')
  let phone = $state('')
  let telegram = $state('')
  let startsOn = $state('')
  let endsOn = $state('')
  let error = $state('')
  let filled = $state('')

  $effect(() => {
    if (!loaded.ok || loaded.pending || !loaded.data.id || filled === loaded.data.id) return
    name = loaded.data.name
    about = loaded.data.about
    phone = loaded.data.phone
    telegram = loaded.data.telegram_user_id ?? ''
    filled = loaded.data.id
  })

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const response = await patchTrainer(trainerId, { name, about, phone, telegram_user_id: telegram })
    error = response.ok ? '' : response.message
  }

  async function addLeave(event: SubmitEvent) {
    event.preventDefault()
    const response = await addAbsence(trainerId, startsOn, endsOn)
    error = response.ok ? '' : response.message
    if (response.ok) {
      startsOn = ''
      endsOn = ''
    }
  }

  async function removeLeave(id: string) {
    const response = await deleteAbsence(trainerId, id)
    error = response.ok ? '' : response.message
  }

  async function cancelFuture() {
    if (!confirm('Снять все будущие занятия этого тренера?')) return
    const response = await cancelTrainerClasses(trainerId)
    error = response.ok ? '' : response.message
  }

  async function remove() {
    if (!confirm('Удалить тренера? Будущие занятия снимутся, история останется.')) return
    const response = await deleteTrainer(trainerId)
    if (response.ok) location.hash = '#/admin/trainers'
    else error = response.message
  }
</script>

{#if loaded.ok && loaded.data.id}
  <h2 class="day">{loaded.data.name}</h2>
  {#if !loaded.data.active}<p class="card tag">выключен</p>{/if}
  <form class="card" onsubmit={save}>
    <label for="trainer-card-name">Имя</label>
    <input id="trainer-card-name" bind:value={name} />
    <label for="trainer-about">О тренере</label>
    <textarea id="trainer-about" rows="3" bind:value={about}></textarea>
    <label for="trainer-phone">Телефон</label>
    <input id="trainer-phone" bind:value={phone} />
    <label for="trainer-telegram">Telegram</label>
    <select id="trainer-telegram" bind:value={telegram}>
      <option value="">Не привязан</option>
      {#each clients as client (client.id)}
        <option value={client.telegram_id}>{client.name || client.username || client.telegram_id}</option>
      {/each}
    </select>
    <button class="primary" type="submit">Сохранить</button>
  </form>

  <form class="card" onsubmit={addLeave}>
    <h3>Отпуск</h3>
    {#each loaded.data.absences as absence (absence.id)}
      <p>
        {formatShortDate(absence.starts_on)}–{formatShortDate(absence.ends_on)}
        <button class="ghost" type="button" onclick={() => removeLeave(absence.id)}>Убрать</button>
      </p>
    {/each}
    <label for="leave-from">С</label>
    <input id="leave-from" type="date" bind:value={startsOn} required />
    <label for="leave-to">По</label>
    <input id="leave-to" type="date" bind:value={endsOn} required />
    <button class="primary" type="submit">Добавить отпуск</button>
  </form>

  <div class="card actions">
    <button class="danger" type="button" onclick={cancelFuture}>Снять будущие занятия</button>
    <button class="ghost" type="button" onclick={remove}>Удалить</button>
  </div>
  {#if error}<p class="card error">{error}</p>{/if}
{:else if !loaded.ok}
  <p class="card error">{loaded.message}</p>
{/if}
