<script lang="ts">
  import { onMount } from 'svelte'
  import { getMe, ping, roleSwitchVisible, setPreviewRole } from './api'
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
  import StudioGuide from './screens/StudioGuide.svelte'
  import StudioPrices from './screens/StudioPrices.svelte'
  import StudioTrainer from './screens/StudioTrainer.svelte'
  import StudioTrainers from './screens/StudioTrainers.svelte'
  import Templates from './screens/Templates.svelte'
  import TrainerCard from './screens/TrainerCard.svelte'
  import TrainerHome from './screens/TrainerHome.svelte'
  import Trainers from './screens/Trainers.svelte'
  import Types from './screens/Types.svelte'
  import Visits from './screens/Visits.svelte'

  let hash = $state(window.location.hash || '#/')

  let backTarget = ''
  let inTelegram = $state(false)

  onMount(() => {
    ping()
    inTelegram = Boolean(window.Telegram?.WebApp?.initData)
    const goBack = () => {
      if (backTarget) location.hash = backTarget
    }
    if (inTelegram) window.Telegram?.WebApp?.BackButton?.onClick(goBack)
    const sync = () => {
      hash = window.location.hash || '#/'
    }
    window.addEventListener('hashchange', sync)
    return () => {
      if (inTelegram) window.Telegram?.WebApp?.BackButton?.offClick(goBack)
      window.removeEventListener('hashchange', sync)
    }
  })

  type Role = 'client' | 'trainer' | 'admin'

  function parse(value: string): { name: string; id?: string; area?: 'admin' | 'trainer' } {
    const parts = value.replace(/^#/, '').split('/').filter(Boolean)
    const head = parts[0] ?? ''
    if (!head) return { name: 'schedule' }
    if (head === 'class' && parts[1]) return { name: 'class', id: parts[1] }
    if (head === 'bookings') return { name: 'bookings' }
    if (head === 'profile') return { name: 'profile' }
    if (head === 'visits') return { name: 'visits' }
    if (head === 'studio' && parts[1] === 'guide') return { name: 'studio-guide' }
    if (head === 'studio' && parts[1] === 'prices') return { name: 'studio-prices' }
    if (head === 'studio' && parts[1] === 'trainers' && parts[2]) return { name: 'studio-trainer', id: parts[2] }
    if (head === 'studio' && parts[1] === 'trainers') return { name: 'studio-trainers' }
    if (head === 'studio') return { name: 'studio' }
    if (head === 'trainer') return { name: 'trainer', area: 'trainer' }
    if (head !== 'admin') return { name: 'schedule' }
    const section = parts[1] ?? ''
    if (!section) return { name: 'admin', area: 'admin' }
    if (section === 'class' && parts[2]) return { name: 'admin-class', id: parts[2], area: 'admin' }
    if (section === 'clients' && parts[2]) return { name: 'client', id: parts[2], area: 'admin' }
    if (section === 'trainers' && parts[2]) return { name: 'admin-trainer', id: parts[2], area: 'admin' }
    return { name: `admin-${section}`, area: 'admin' }
  }

  let route = $derived(parse(hash))
  let me = $derived(getMe())
  let serverAdmin = $derived(Boolean(me.data?.is_admin))
  let serverTrainer = $derived(Boolean(me.data?.is_trainer))
  let preview = roleSwitchVisible()
  let picked = $state<Role | ''>('')
  let role = $derived.by((): Role => {
    if (preview) {
      if (serverAdmin) return 'admin'
      if (serverTrainer) return 'trainer'
      return 'client'
    }
    if (picked === 'admin' && serverAdmin) return 'admin'
    if (picked === 'trainer' && serverTrainer) return 'trainer'
    if (picked === 'client') return 'client'
    if (serverAdmin) return 'admin'
    return 'client'
  })
  let showClient = $derived(preview || serverAdmin || serverTrainer)
  let showTrainer = $derived(preview || serverTrainer)
  let showAdmin = $derived(preview || serverAdmin)
  let showSwitch = $derived(showTrainer || showAdmin)

  let header = $state<HTMLElement | undefined>(undefined)

  $effect(() => {
    const node = header
    if (!node) return
    const apply = () => node.parentElement?.style.setProperty('--top-h', `${node.offsetHeight}px`)
    apply()
    const observer = new ResizeObserver(apply)
    observer.observe(node)
    return () => observer.disconnect()
  })

  $effect(() => {
    if (me.pending) return
    if (route.area === 'admin' && role !== 'admin') location.hash = role === 'trainer' ? '#/trainer' : '#/'
    if (route.area === 'trainer' && role !== 'trainer') location.hash = role === 'admin' ? '#/admin' : '#/'
  })

  let back = $derived.by(() => {
    if (route.name === 'visits') return '#/profile'
    if (route.name === 'class') return '#/'
    if (route.name === 'studio-guide' || route.name === 'studio-prices' || route.name === 'studio-trainers') return '#/studio'
    if (route.name === 'studio-trainer') return '#/studio/trainers'
    if (route.name === 'admin-class') return '#/admin/schedule'
    if (route.name === 'client') return '#/admin/clients'
    if (route.name === 'admin-trainer') return '#/admin/trainers'
    if (route.name.startsWith('admin-')) return '#/admin'
    return ''
  })

  $effect(() => {
    backTarget = back
    const button = window.Telegram?.WebApp?.BackButton
    if (!inTelegram || !button) return
    if (back) button.show()
    else button.hide()
  })

  function choose(next: Role) {
    if (preview) setPreviewRole(next)
    else picked = next
    location.hash = next === 'admin' ? '#/admin' : next === 'trainer' ? '#/trainer' : '#/'
  }
</script>

<div class="shell">
  <header class="top" bind:this={header}>
    <p class="brand">aero.ge</p>
    {#if showSwitch}
      <div class="switch">
        {#if showClient}<button type="button" aria-pressed={role === 'client'} data-testid="role-client" onclick={() => choose('client')}>Клиент</button>{/if}
        {#if showTrainer}<button type="button" aria-pressed={role === 'trainer'} data-testid="role-trainer" onclick={() => choose('trainer')}>Тренер</button>{/if}
        {#if showAdmin}<button type="button" aria-pressed={role === 'admin'} data-testid="role-admin" onclick={() => choose('admin')}>Админ</button>{/if}
      </div>
    {/if}
  </header>

  <main class="page">
    {#if back && !inTelegram}<a class="page-back" href={back}>Назад</a>{/if}
    {#if route.name === 'schedule'}
      <Schedule />
    {:else if route.name === 'class'}
      <ClassDetail classId={route.id ?? ''} />
    {:else if route.name === 'bookings'}
      <Bookings />
    {:else if route.name === 'profile'}
      <Profile />
    {:else if route.name === 'visits'}
      <Visits />
    {:else if route.name === 'studio'}
      <Studio />
    {:else if route.name === 'studio-guide'}
      <StudioGuide />
    {:else if route.name === 'studio-prices'}
      <StudioPrices />
    {:else if route.name === 'studio-trainers'}
      <StudioTrainers />
    {:else if route.name === 'studio-trainer'}
      <StudioTrainer trainerId={route.id ?? ''} />
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
    {:else if route.name === 'trainer'}
      <TrainerHome />
    {:else if route.name === 'admin-trainer'}
      <TrainerCard trainerId={route.id ?? ''} />
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
    {#if role === 'admin'}
      <a href="#/admin/roster" class:active={route.name === 'admin-roster'}>Журнал</a>
      <a href="#/admin/schedule" class:active={route.name === 'admin-schedule'}>Неделя</a>
      <a href="#/admin/clients" class:active={route.name === 'admin-clients' || route.name === 'client'}>Клиенты</a>
      <a href="#/admin" class:active={route.name === 'admin'}>Ещё</a>
    {:else if role === 'trainer'}
      <a href="#/trainer" class:active={route.name === 'trainer'}>Занятия</a>
    {:else}
      <a href="#/" class:active={route.name === 'schedule' || route.name === 'class'}>Расписание</a>
      <a href="#/bookings" class:active={route.name === 'bookings'}>Записи</a>
      <a href="#/profile" class:active={route.name === 'profile'}>Профиль</a>
      <a href="#/studio" class:active={route.name === 'studio' || route.name.startsWith('studio-')}>Студия</a>
    {/if}
  </nav>
</div>
