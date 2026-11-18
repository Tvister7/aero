<script lang="ts">
  import { onMount } from 'svelte'
  import { getMe, ping, roleSwitchVisible, setPreviewAdmin } from './api'
  import AdminClass from './screens/AdminClass.svelte'
  import AdminMenu from './screens/AdminMenu.svelte'
  import Bookings from './screens/Bookings.svelte'
  import ClassDetail from './screens/ClassDetail.svelte'
  import ClientCard from './screens/ClientCard.svelte'
  import Clients from './screens/Clients.svelte'
  import NewClass from './screens/NewClass.svelte'
  import Profile from './screens/Profile.svelte'
  import Roster from './screens/Roster.svelte'
  import Schedule from './screens/Schedule.svelte'
  import Stats from './screens/Stats.svelte'
  import Studio from './screens/Studio.svelte'
  import StudioEdit from './screens/StudioEdit.svelte'
  import Templates from './screens/Templates.svelte'
  import Trainers from './screens/Trainers.svelte'
  import Types from './screens/Types.svelte'

  let hash = $state(window.location.hash || '#/')

  onMount(() => {
    ping()
    const sync = () => {
      hash = window.location.hash || '#/'
    }
    window.addEventListener('hashchange', sync)
    return () => window.removeEventListener('hashchange', sync)
  })

  function parse(value: string): { name: string; id?: string; area?: 'admin' } {
    const parts = value.replace(/^#/, '').split('/').filter(Boolean)
    const head = parts[0] ?? ''
    if (!head) return { name: 'schedule' }
    if (head === 'class' && parts[1]) return { name: 'class', id: parts[1] }
    if (head === 'bookings') return { name: 'bookings' }
    if (head === 'profile') return { name: 'profile' }
    if (head === 'studio') return { name: 'studio' }
    if (head !== 'admin') return { name: 'schedule' }
    const section = parts[1] ?? ''
    if (!section) return { name: 'admin', area: 'admin' }
    if (section === 'class' && parts[2]) return { name: 'admin-class', id: parts[2], area: 'admin' }
    if (section === 'clients' && parts[2]) return { name: 'client', id: parts[2], area: 'admin' }
    return { name: `admin-${section}`, area: 'admin' }
  }

  let route = $derived(parse(hash))
  let me = $derived(getMe())
  let admin = $derived(Boolean(me.data?.is_admin))
  let showSwitch = roleSwitchVisible()

  $effect(() => {
    if (route.area === 'admin' && !admin) location.hash = '#/'
  })

  let back = $derived.by(() => {
    if (route.name === 'class') return '#/'
    if (route.name === 'admin-class') return '#/admin/schedule'
    if (route.name === 'client') return '#/admin/clients'
    if (route.name.startsWith('admin-') && route.name !== 'admin-roster' && route.name !== 'admin-schedule' && route.name !== 'admin-clients') {
      return '#/admin'
    }
    return ''
  })

  function choose(asAdmin: boolean) {
    setPreviewAdmin(asAdmin)
    location.hash = asAdmin ? '#/admin' : '#/'
  }
</script>

<div class="shell">
  <header class="top">
    {#if back}
      <a class="back" href={back}>Назад</a>
    {:else}
      <p class="brand">aero.ge</p>
    {/if}
    {#if showSwitch}
      <div class="switch">
        <button type="button" aria-pressed={!admin} data-testid="role-client" onclick={() => choose(false)}>Клиент</button>
        <button type="button" aria-pressed={admin} data-testid="role-admin" onclick={() => choose(true)}>Админ</button>
      </div>
    {/if}
  </header>

  <main class="page">
    {#if route.name === 'schedule'}
      <Schedule />
    {:else if route.name === 'class'}
      <ClassDetail classId={route.id ?? ''} />
    {:else if route.name === 'bookings'}
      <Bookings />
    {:else if route.name === 'profile'}
      <Profile />
    {:else if route.name === 'studio'}
      <Studio />
    {:else if route.name === 'admin'}
      <AdminMenu />
    {:else if route.name === 'admin-roster'}
      <Roster />
    {:else if route.name === 'admin-schedule'}
      <Schedule admin />
    {:else if route.name === 'admin-class'}
      <AdminClass classId={route.id ?? ''} />
    {:else if route.name === 'admin-templates'}
      <Templates />
    {:else if route.name === 'admin-new'}
      <NewClass />
    {:else if route.name === 'admin-trainers'}
      <Trainers />
    {:else if route.name === 'admin-types'}
      <Types />
    {:else if route.name === 'admin-clients'}
      <Clients />
    {:else if route.name === 'client'}
      <ClientCard clientId={route.id ?? ''} />
    {:else if route.name === 'admin-stats'}
      <Stats />
    {:else if route.name === 'admin-studio'}
      <StudioEdit />
    {/if}
  </main>

  <nav class="nav">
    {#if admin}
      <a href="#/admin/roster" class:active={route.name === 'admin-roster'}>Журнал</a>
      <a href="#/admin/schedule" class:active={route.name === 'admin-schedule'}>Неделя</a>
      <a href="#/admin/clients" class:active={route.name === 'admin-clients' || route.name === 'client'}>Клиенты</a>
      <a href="#/admin" class:active={route.name === 'admin'}>Ещё</a>
    {:else}
      <a href="#/" class:active={route.name === 'schedule' || route.name === 'class'}>Расписание</a>
      <a href="#/bookings" class:active={route.name === 'bookings'}>Записи</a>
      <a href="#/profile" class:active={route.name === 'profile'}>Профиль</a>
      <a href="#/studio" class:active={route.name === 'studio'}>Студия</a>
    {/if}
  </nav>
</div>
