<script lang="ts">
  import type { task } from '../../wailsjs/go/models.js'
  import { taskStore } from '../stores/tasks.svelte.js'
  import { projectStore } from '../stores/projects.svelte.js'
  import { BOARD_COLUMNS } from '../lib/statuses.js'
  import TaskCard from '../components/TaskCard.svelte'

  interface Props {
    onselect: (id: string) => void
  }

  const { onselect }: Props = $props()

  let dragOverStatus = $state<string | null>(null)
  let addingToColumn = $state<string | null>(null)
  let newTaskTitle = $state('')
  let inputRef = $state<HTMLInputElement | null>(null)

  // Filter state
  let searchQuery = $state('')
  let selectedProjectId = $state('')
  let selectedTags = $state<string[]>([])
  let selectedAgentMode = $state('')
  let showDone = $state(false)

  // Derived: unique tags across all tasks
  const allTags = $derived(
    [...new Set(taskStore.list.flatMap((t: task.Task) => t.tags ?? []))].sort()
  )

  // Derived: filter function
  function filteredByStatuses(statuses: string[]): task.Task[] {
    const query = searchQuery.toLowerCase().trim()
    return taskStore.list.filter((t: task.Task) => {
      if (!statuses.includes(t.status)) return false
      if (query && !t.title.toLowerCase().includes(query)
          && !(t.body ?? '').toLowerCase().includes(query)
          && !(t.issue ?? '').toLowerCase().includes(query)) return false
      if (selectedProjectId && t.projectId !== selectedProjectId) return false
      if (selectedTags.length > 0 && !selectedTags.every(tag => t.tags?.includes(tag))) return false
      if (selectedAgentMode && t.agentMode !== selectedAgentMode) return false
      return true
    })
  }

  const visibleColumns = $derived(
    showDone ? BOARD_COLUMNS : BOARD_COLUMNS.filter(c => c.status !== 'done')
  )

  const hasActiveFilters = $derived(
    searchQuery || selectedProjectId || selectedTags.length > 0 || selectedAgentMode
  )

  function clearFilters() {
    searchQuery = ''
    selectedProjectId = ''
    selectedTags = []
    selectedAgentMode = ''
  }

  function toggleTag(tag: string) {
    selectedTags = selectedTags.includes(tag)
      ? selectedTags.filter(t => t !== tag)
      : [...selectedTags, tag]
  }

  const agentModes = [
    { value: '', label: 'All' },
    { value: 'headless', label: 'Headless' },
    { value: 'interactive', label: 'Interactive' },
  ]

  async function handleDrop(e: DragEvent, targetStatus: string) {
    e.preventDefault()
    dragOverStatus = null
    const taskId = e.dataTransfer?.getData('text/plain')
    if (!taskId) return
    const existing = taskStore.tasks.get(taskId)
    if (!existing || existing.status === targetStatus) return
    await taskStore.update(taskId, { status: targetStatus })
  }

  function openInlineAdd(status: string) {
    addingToColumn = status
    newTaskTitle = ''
    requestAnimationFrame(() => inputRef?.focus())
  }

  function dismissInlineAdd() {
    addingToColumn = null
    newTaskTitle = ''
  }

  async function submitInlineAdd(status: string) {
    const title = newTaskTitle.trim()
    if (!title) return
    newTaskTitle = ''
    const created = await taskStore.create(title, '', 'headless')
    if (status !== 'new') {
      await taskStore.update(created.id, { status })
    }
    requestAnimationFrame(() => inputRef?.focus())
  }

  function handleInputKeydown(e: KeyboardEvent, status: string) {
    if (e.key === 'Enter') {
      e.preventDefault()
      submitInlineAdd(status)
    } else if (e.key === 'Escape') {
      dismissInlineAdd()
    }
  }
</script>

<div class="flex h-full flex-col">
  <!-- Filter bar -->
  <div class="flex flex-wrap items-center gap-3 border-b border-surface-200 px-6 py-3 dark:border-surface-700">
    <!-- Search -->
    <div class="relative">
      <svg class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search tasks..."
        class="w-56 rounded-md border border-surface-300 bg-surface-50 py-1.5 pl-8 pr-2 text-sm outline-none focus:border-primary-400 focus:ring-1 focus:ring-primary-400 dark:border-surface-600 dark:bg-surface-800"
      />
    </div>

    <!-- Project filter -->
    {#if projectStore.list.length > 0}
      <select
        bind:value={selectedProjectId}
        class="rounded-md border border-surface-300 bg-surface-50 px-2 py-1.5 text-sm outline-none focus:border-primary-400 focus:ring-1 focus:ring-primary-400 dark:border-surface-600 dark:bg-surface-800"
      >
        <option value="">All projects</option>
        {#each projectStore.list as p}
          <option value={p.id}>{p.owner}/{p.repo}</option>
        {/each}
      </select>
    {/if}

    <!-- Agent mode pills -->
    <div class="flex rounded-md border border-surface-300 dark:border-surface-600">
      {#each agentModes as mode}
        <button
          type="button"
          class="px-2.5 py-1 text-xs font-medium transition-colors first:rounded-l-md last:rounded-r-md {selectedAgentMode === mode.value
            ? 'bg-primary-500 text-white dark:bg-primary-600'
            : 'bg-surface-50 text-surface-600 hover:bg-surface-200 dark:bg-surface-800 dark:text-surface-300 dark:hover:bg-surface-700'}"
          onclick={() => (selectedAgentMode = mode.value)}
        >
          {mode.label}
        </button>
      {/each}
    </div>

    <!-- Tag chips -->
    {#if allTags.length > 0}
      <div class="flex flex-wrap items-center gap-1">
        {#each allTags as tag}
          <button
            type="button"
            class="rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors {selectedTags.includes(tag)
              ? 'bg-primary-500 text-white dark:bg-primary-600'
              : 'bg-surface-200 text-surface-600 hover:bg-surface-300 dark:bg-surface-700 dark:text-surface-300 dark:hover:bg-surface-600'}"
            onclick={() => toggleTag(tag)}
          >
            {tag}
          </button>
        {/each}
      </div>
    {/if}

    <!-- Right side: clear + show done -->
    <div class="ml-auto flex items-center gap-3">
      {#if hasActiveFilters}
        <button
          type="button"
          class="text-xs text-surface-500 underline hover:text-surface-700 dark:hover:text-surface-300"
          onclick={clearFilters}
        >
          Clear filters
        </button>
      {/if}
      <label class="flex items-center gap-1.5 text-xs text-surface-500">
        <input type="checkbox" bind:checked={showDone} class="accent-primary-500" />
        Show done
      </label>
    </div>
  </div>

  <!-- Board columns -->
  <div class="flex min-h-0 flex-1 gap-4 overflow-x-auto p-6">
    {#if taskStore.loading}
      <p class="m-auto text-sm opacity-60">Loading tasks...</p>
    {:else if taskStore.error}
      <p class="m-auto text-sm text-error-500">{taskStore.error}</p>
    {:else}
      {#each visibleColumns as col}
        {@const statuses = col.includes.length > 0 ? col.includes : [col.status]}
        {@const tasks = filteredByStatuses(statuses)}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="flex min-w-[250px] flex-1 flex-col rounded-lg border-t-4 bg-surface-100 transition-shadow dark:bg-surface-900 {col.border} {dragOverStatus === col.status ? 'ring-2 ring-primary-400 dark:ring-primary-500' : ''}"
          ondragover={(e) => { e.preventDefault(); dragOverStatus = col.status }}
          ondragleave={() => { dragOverStatus = null }}
          ondrop={(e) => handleDrop(e, col.status)}
        >
          <div class="flex items-center justify-between px-3 py-2">
            <h2 class="text-sm font-semibold">{col.label}</h2>
            <span class="rounded-full bg-surface-200 px-2 py-0.5 text-xs font-medium dark:bg-surface-700">
              {tasks.length}
            </span>
          </div>
          <div class="flex flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2">
            {#each tasks as t (t.id)}
              <TaskCard task={t} onclick={() => onselect(t.id)} />
            {/each}
          </div>
          <div class="px-2 pb-2">
            {#if addingToColumn === col.status}
              <input
                bind:this={inputRef}
                bind:value={newTaskTitle}
                type="text"
                placeholder="Task title"
                class="w-full rounded-md border border-surface-300 bg-surface-50 px-2 py-1.5 text-sm outline-none focus:border-primary-400 focus:ring-1 focus:ring-primary-400 dark:border-surface-600 dark:bg-surface-800"
                onkeydown={(e) => handleInputKeydown(e, col.status)}
                onblur={() => dismissInlineAdd()}
              />
            {:else}
              <button
                type="button"
                class="flex w-full items-center gap-1 rounded-md px-2 py-1.5 text-sm opacity-50 transition-opacity hover:bg-surface-200 hover:opacity-100 dark:hover:bg-surface-800"
                onclick={() => openInlineAdd(col.status)}
              >
                <span class="text-base leading-none">+</span> Add task
              </button>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>
