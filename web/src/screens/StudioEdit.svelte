<script lang="ts">
  import { getStudio, putStudio } from '../api'

  let studio = $derived(getStudio())
  let address = $state('')
  let phone = $state('')
  let instagram = $state('')
  let contact = $state('')
  let map = $state('')
  let notice = $state('')
  let filled = $state(false)

  $effect(() => {
    if (filled || studio.pending) return
    address = studio.data.address
    phone = studio.data.phone
    instagram = studio.data.instagram_url
    contact = studio.data.contact_url
    map = studio.data.map_url
    filled = true
  })

  async function save(event: SubmitEvent) {
    event.preventDefault()
    const response = await putStudio({
      address,
      phone,
      instagram_url: instagram,
      contact_url: contact,
      map_url: map,
    })
    notice = response.ok ? 'Контакты сохранены.' : response.message
  }
</script>

<form class="card" onsubmit={save}>
  <label for="address">Адрес</label>
  <input id="address" bind:value={address} />
  <label for="studio-phone">Телефон</label>
  <input id="studio-phone" bind:value={phone} />
  <label for="instagram">Instagram</label>
  <input id="instagram" bind:value={instagram} />
  <label for="contact">Контакт</label>
  <input id="contact" bind:value={contact} />
  <label for="map">Карта</label>
  <input id="map" bind:value={map} />
  <button class="primary" type="submit">Сохранить</button>
  {#if notice}<p>{notice}</p>{/if}
</form>
