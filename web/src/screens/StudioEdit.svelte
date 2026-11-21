<script lang="ts">
  import { getStudio, putStudio } from '../api'

  let studio = $derived(getStudio())
  let address = $state('')
  let phone = $state('')
  let instagram = $state('')
  let contact = $state('')
  let map = $state('')
  let guide = $state('')
  let pricing = $state('')
  let notice = $state('')
  let failed = $state(false)
  let filled = $state(false)

  function pretty(raw: string) {
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }

  $effect(() => {
    if (filled || studio.pending) return
    address = studio.data.address
    phone = studio.data.phone
    instagram = studio.data.instagram_url
    contact = studio.data.contact_url
    map = studio.data.map_url
    guide = studio.data.guide
    pricing = pretty(studio.data.pricing)
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
      guide,
      pricing,
    })
    failed = !response.ok
    notice = response.ok ? 'Контакты сохранены.' : response.message
    if (response.ok) pricing = pretty(response.data.pricing)
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
  <label for="guide">Памятка</label>
  <textarea id="guide" rows="12" bind:value={guide}></textarea>
  <label for="pricing">Прайс, JSON</label>
  <textarea id="pricing" rows="10" bind:value={pricing}></textarea>
  <button class="primary" type="submit">Сохранить</button>
  {#if notice}<p class:error={failed}>{notice}</p>{/if}
</form>
