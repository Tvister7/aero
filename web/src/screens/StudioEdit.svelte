<script lang="ts">
  import { getStudio, putStudio } from '../api'

  let studio = getStudio().data
  let address = $state(studio.address)
  let phone = $state(studio.phone)
  let instagram = $state(studio.instagram_url)
  let contact = $state(studio.contact_url)
  let map = $state(studio.map_url)
  let notice = $state('')

  function save(event: SubmitEvent) {
    event.preventDefault()
    const response = putStudio({
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
