<script lang="ts">
  let {
    kind,
    message = '',
    reload,
    testid,
  }: {
    kind: 'pending' | 'ok' | 'error'
    message?: string
    reload?: () => void
    testid?: string
  } = $props()
</script>

<div class="notice" class:error={kind === 'error'} class:ok={kind === 'ok'} data-testid={testid}>
  <p>{kind === 'pending' ? 'Загрузка…' : message}</p>
  {#if kind === 'error' && reload}
    <button class="ghost" type="button" onclick={reload}>Повторить</button>
  {/if}
</div>
