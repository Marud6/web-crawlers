<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

// --- Containers state ---
const containers = ref([])
const lastUpdated = ref(null)
const isLoadingContainers = ref(false)

// --- Seed state ---
const seed = ref('')
const seedTouched = ref(false)
const isSendingSeed = ref(false)

const isValidUrl = computed(() => {
  try {
    const url = new URL(seed.value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
})

const seedError = computed(() => {
  if (!seedTouched.value || seed.value === '') return ''
  if (!isValidUrl.value) return 'Please enter a valid URL (e.g. https://example.com)'
  return ''
})

const canSendSeed = computed(() => {
  return isValidUrl.value && !isSendingSeed.value
})

// --- Worker actions state ---
const isStartingWorker = ref(false)
const stoppingWorkerIds = ref([])

// --- Toast system ---
const toasts = ref([])
let toastCounter = 0

function addToast(message, type = 'info') {
  const id = ++toastCounter
  toasts.value.push({ id, message, type, visible: false })

  requestAnimationFrame(() => {
    const t = toasts.value.find(t => t.id === id)
    if (t) t.visible = true
  })

  setTimeout(() => removeToast(id), 4000)
}

function removeToast(id) {
  const t = toasts.value.find(t => t.id === id)
  if (!t) return
  t.visible = false
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, 300)
}

// --- Confirm dialog ---
const confirmState = ref(null)

function showConfirm(title, message) {
  return new Promise(resolve => {
    confirmState.value = { title, message, resolve }
  })
}

function confirmAccept() {
  if (confirmState.value) confirmState.value.resolve(true)
  confirmState.value = null
}

function confirmCancel() {
  if (confirmState.value) confirmState.value.resolve(false)
  confirmState.value = null
}

// --- API functions ---
async function fetchContainers(silent = false) {
  if (!silent) isLoadingContainers.value = true
  try {
    const res = await fetch('http://localhost:8080/containers')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    containers.value = await res.json()
    lastUpdated.value = new Date()
  } catch (err) {
    if (!silent) addToast('Failed to fetch containers: ' + err.message, 'error')
  } finally {
    if (!silent) isLoadingContainers.value = false
  }
}

async function startWorker() {
  isStartingWorker.value = true
  try {
    const res = await fetch('http://localhost:8080/start', {
      headers: { 'Content-Type': 'application/json' }
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    containers.value = await res.json()
    lastUpdated.value = new Date()
    addToast('Worker started successfully', 'success')
  } catch (err) {
    addToast('Failed to start worker: ' + err.message, 'error')
  } finally {
    isStartingWorker.value = false
  }
}

async function stopWorker(id) {
  const confirmed = await showConfirm(
    'Stop Worker',
    `Stop container ${id.slice(0, 12)}…? This action cannot be undone.`
  )
  if (!confirmed) return

  stoppingWorkerIds.value.push(id)
  try {
    const res = await fetch('http://localhost:8080/stop', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ container_id: id })
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    containers.value = await res.json()
    lastUpdated.value = new Date()
    addToast('Worker stopped', 'success')
  } catch (err) {
    addToast('Failed to stop worker: ' + err.message, 'error')
  } finally {
    stoppingWorkerIds.value = stoppingWorkerIds.value.filter(wid => wid !== id)
  }
}

async function sendSeed() {
  if (!canSendSeed.value) return
  isSendingSeed.value = true
  try {
    const res = await fetch('http://localhost:8080/seed', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ seed: seed.value })
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    addToast('Seed URL sent successfully', 'success')
    seed.value = ''
    seedTouched.value = false
  } catch (err) {
    addToast('Failed to send seed: ' + err.message, 'error')
  } finally {
    isSendingSeed.value = false
  }
}

// --- Formatting ---
function formatTime(date) {
  if (!date) return '—'
  return date.toLocaleTimeString()
}

function isStoppingWorker(id) {
  return stoppingWorkerIds.value.includes(id)
}

// --- Auto-poll ---
let pollInterval = null

onMounted(() => {
  fetchContainers()
  pollInterval = setInterval(() => fetchContainers(true), 5000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <!-- Toast container -->
  <div class="toast-container">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="toast"
      :class="[`toast-${toast.type}`, { 'toast-visible': toast.visible }]"
    >
      <span class="toast-icon">{{ toast.type === 'error' ? '✕' : '✓' }}</span>
      <span>{{ toast.message }}</span>
    </div>
  </div>

  <!-- Confirm dialog -->
  <div v-if="confirmState" class="dialog-overlay" @click.self="confirmCancel">
    <div class="dialog">
      <h3>{{ confirmState.title }}</h3>
      <p>{{ confirmState.message }}</p>
      <div class="dialog-actions">
        <button class="btn-secondary" @click="confirmCancel">Cancel</button>
        <button class="btn-danger" @click="confirmAccept">Stop</button>
      </div>
    </div>
  </div>

  <!-- Header -->
  <header class="app-header">
    <h1 class="app-title">Crawlers Dashboard</h1>
    <div class="header-status">
      <span class="status-dot" :class="{ active: containers.length > 0 }"></span>
      <span>{{ containers.length }} worker{{ containers.length !== 1 ? 's' : '' }} running</span>
      <span class="text-muted"> · Updated {{ formatTime(lastUpdated) }}</span>
    </div>
  </header>

  <!-- Main content -->
  <main class="main-content">
    <!-- Seed URL section -->
    <section class="section">
      <h2 class="section-header">Seed URL</h2>
      <form class="seed-form" @submit.prevent="sendSeed">
        <div class="input-wrapper">
          <input
            v-model="seed"
            type="text"
            placeholder="https://example.com"
            @blur="seedTouched = true"
          />
          <p v-if="seedError" class="input-error">{{ seedError }}</p>
        </div>
        <button
          class="btn-primary"
          type="submit"
          :disabled="!canSendSeed"
        >
          {{ isSendingSeed ? 'Sending...' : 'Send Seed URL' }}
        </button>
      </form>
    </section>

    <!-- Workers section -->
    <section class="section">
      <div class="section-header-row">
        <h2 class="section-header">Workers</h2>
        <button
          class="btn-primary"
          :disabled="isStartingWorker"
          @click="startWorker"
        >
          {{ isStartingWorker ? 'Starting...' : '+ Start New Worker' }}
        </button>
      </div>

      <!-- Loading state -->
      <div v-if="isLoadingContainers && containers.length === 0" class="empty-state">
        <div class="empty-icon">⟳</div>
        <p>Loading workers...</p>
      </div>

      <!-- Empty state -->
      <div v-else-if="containers.length === 0" class="empty-state">
        <div class="empty-icon">◇</div>
        <p>No workers running</p>
        <p class="text-muted">Click "+ Start New Worker" to launch one</p>
      </div>

      <!-- Worker cards grid -->
      <div v-else class="workers-grid">
        <div
          v-for="container in containers"
          :key="container.id"
          class="worker-card"
        >
          <div class="card-header">
            <span class="status-dot active"></span>
            <span class="card-status-text">Running</span>
          </div>
          <p class="card-id" :title="container.id">{{ container.id.slice(0, 12) }}</p>
          <p class="card-image text-muted">{{ container.image }}</p>
          <button
            class="btn-danger-outline btn-sm"
            :disabled="isStoppingWorker(container.id)"
            @click="stopWorker(container.id)"
          >
            {{ isStoppingWorker(container.id) ? 'Stopping...' : 'Stop' }}
          </button>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
/* --- Header --- */
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 2rem;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.app-title {
  font-size: 1.25rem;
  font-weight: 600;
}

.header-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

/* --- Status dot --- */
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-text-muted);
  flex-shrink: 0;
}

.status-dot.active {
  background: var(--color-success);
  box-shadow: 0 0 6px var(--color-success);
}

/* --- Main content --- */
.main-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

/* --- Sections --- */
.section {
  margin-bottom: 2.5rem;
}

.section-header {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 1rem;
}

.section-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.section-header-row .section-header {
  margin-bottom: 0;
}

/* --- Seed form --- */
.seed-form {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
}

.input-wrapper {
  flex: 1;
}

.input-wrapper input {
  width: 100%;
}

.input-error {
  margin: 0.35rem 0 0;
  font-size: 0.8rem;
  color: var(--color-danger);
}

/* --- Buttons --- */
.btn-primary {
  background: var(--color-accent);
  color: #fff;
  border: 1px solid var(--color-accent);
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-accent-hover);
  border-color: var(--color-accent-hover);
}

.btn-secondary {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  border-color: var(--color-text-muted);
}

.btn-danger {
  background: var(--color-danger);
  color: #fff;
  border: 1px solid var(--color-danger);
}

.btn-danger:hover:not(:disabled) {
  background: var(--color-danger-hover);
  border-color: var(--color-danger-hover);
}

.btn-danger-outline {
  background: transparent;
  color: var(--color-danger);
  border: 1px solid var(--color-danger);
}

.btn-danger-outline:hover:not(:disabled) {
  background: var(--color-danger);
  color: #fff;
}

.btn-sm {
  padding: 0.35em 0.75em;
  font-size: 0.8rem;
}

/* --- Workers grid --- */
.workers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.worker-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 1.25rem;
  transition: border-color 0.2s, transform 0.2s;
}

.worker-card:hover {
  border-color: var(--color-accent);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.card-status-text {
  font-size: 0.8rem;
  color: var(--color-success);
  font-weight: 500;
}

.card-id {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.9rem;
  margin: 0 0 0.25rem;
  word-break: break-all;
}

.card-image {
  font-size: 0.8rem;
  margin: 0 0 1rem;
}

/* --- Empty state --- */
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
}

.empty-icon {
  font-size: 2.5rem;
  color: var(--color-text-muted);
  margin-bottom: 0.75rem;
}

.empty-state p {
  margin: 0.25rem 0;
}

/* --- Toast --- */
.toast-container {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  transform: translateX(calc(100% + 1rem));
  opacity: 0;
  transition: transform 0.3s ease, opacity 0.3s ease;
  pointer-events: auto;
}

.toast-visible {
  transform: translateX(0);
  opacity: 1;
}

.toast-icon {
  font-weight: 700;
  flex-shrink: 0;
}

.toast-success {
  border-color: var(--color-success);
}

.toast-success .toast-icon {
  color: var(--color-success);
}

.toast-error {
  border-color: var(--color-danger);
}

.toast-error .toast-icon {
  color: var(--color-danger);
}

/* --- Confirm dialog --- */
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.dialog {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.75rem;
  max-width: 400px;
  width: 90%;
}

.dialog h3 {
  margin-bottom: 0.75rem;
}

.dialog p {
  color: var(--color-text-muted);
  margin: 0 0 1.5rem;
  font-size: 0.9rem;
  line-height: 1.5;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

/* --- Utility --- */
.text-muted {
  color: var(--color-text-muted);
}
</style>
