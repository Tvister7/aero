<script lang="ts">
  import { getStudio } from '../api'

  let studio = $derived(getStudio().data)
</script>

<article class="card">
  <h2>Студия</h2>
  <p>{studio.address}</p>
  {#if studio.phone}<p><a href={'tel:' + studio.phone}>{studio.phone}</a></p>{/if}
  {#if studio.instagram_url}<p><a href={studio.instagram_url}>Instagram</a></p>{/if}
  {#if studio.contact_url}<p><a href={studio.contact_url}>Контакт</a></p>{/if}
  {#if studio.map_url}<p><a href={studio.map_url}>Карта</a></p>{/if}
</article>
<a class="row" href="#/studio/guide">Как нас найти</a>
<a class="row" href="#/studio/prices">Цены</a>
<a class="row" href="#/studio/trainers">Тренеры</a>
