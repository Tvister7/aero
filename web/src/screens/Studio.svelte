<script lang="ts">
  import { getStudio } from '../api'
  import Notice from '../lib/Notice.svelte'

  let studio = $derived(getStudio())
</script>

{#if studio.status === 'pending'}
  <Notice kind="pending" />
{:else if studio.status === 'error'}
  <Notice kind="error" message={studio.message} reload={studio.reload} />
{:else}
  <article class="card">
    <h2>Студия</h2>
    {#if studio.data.address}<p>{studio.data.address}</p>{:else}<p class="muted">Адрес пока не заполнен.</p>{/if}
    {#if studio.data.phone}<p><a href={'tel:' + studio.data.phone}>{studio.data.phone}</a></p>{/if}
    {#if studio.data.instagram_url}<p><a href={studio.data.instagram_url}>Instagram</a></p>{/if}
    {#if studio.data.contact_url}<p><a href={studio.data.contact_url}>Контакт</a></p>{/if}
    {#if studio.data.map_url}<p><a href={studio.data.map_url}>Карта</a></p>{/if}
  </article>
  <a class="row" href="#/studio/guide">Как нас найти</a>
  <a class="row" href="#/studio/prices">Цены</a>
  <a class="row" href="#/studio/trainers">Тренеры</a>
{/if}
