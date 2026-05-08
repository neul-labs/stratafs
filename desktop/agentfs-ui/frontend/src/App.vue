<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { GetStatus, StartAgentFS, StopAgentFS, GetConfig, Search, AddSource, RemoveSource, InitConfig, Quit, ShowWindow, GetMountStatus, MountFilesystem, UnmountFilesystem, OpenMountPoint } from '../wailsjs/go/main/App'

// State
const status = ref({ running: false, apiHealthy: false, version: '', configDir: '' })
const config = ref(null)
const mountStatus = ref({ mounted: false, mount_point: '' })
const searchQuery = ref('')
const searchResults = ref(null)
const activeView = ref('files') // 'files' | 'search' | 'settings'
const loading = ref(false)
const error = ref('')
const newSourceName = ref('')
const newSourcePath = ref('')

let statusInterval = null

// Computed
const isReady = computed(() => status.value.apiHealthy)
const statusText = computed(() => {
  if (status.value.apiHealthy) return 'Ready'
  if (status.value.running) return 'Starting...'
  return 'Offline'
})

// Lifecycle
onMounted(async () => {
  await refreshStatus()
  await loadConfig()
  await refreshMountStatus()
  statusInterval = setInterval(async () => {
    await refreshStatus()
    await refreshMountStatus()
  }, 5000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
})

// Methods
async function refreshStatus() {
  try {
    status.value = await GetStatus()
  } catch (e) {
    console.error('Failed to get status:', e)
  }
}

async function loadConfig() {
  try {
    config.value = await GetConfig()
    error.value = ''
  } catch (e) {
    config.value = null
  }
}

async function toggleService() {
  loading.value = true
  error.value = ''
  try {
    if (status.value.running) {
      await StopAgentFS()
    } else {
      await StartAgentFS()
    }
    await refreshStatus()
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}

async function performSearch() {
  if (!searchQuery.value.trim()) return
  if (!isReady.value) {
    error.value = 'Start AgentFS first'
    return
  }
  loading.value = true
  error.value = ''
  try {
    searchResults.value = await Search(searchQuery.value, 'hybrid', 20)
  } catch (e) {
    error.value = String(e)
    searchResults.value = null
  }
  loading.value = false
}

async function addNewSource() {
  if (!newSourceName.value.trim() || !newSourcePath.value.trim()) {
    error.value = 'Enter both name and path'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await AddSource(newSourceName.value, newSourcePath.value)
    newSourceName.value = ''
    newSourcePath.value = ''
    await loadConfig()
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}

async function removeSourceByName(name) {
  if (!confirm(`Remove "${name}"?`)) return
  loading.value = true
  try {
    await RemoveSource(name)
    await loadConfig()
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}

async function initializeConfig() {
  loading.value = true
  try {
    await InitConfig()
    await loadConfig()
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}

function quitApp() {
  Quit()
}

function clearError() {
  error.value = ''
}

async function refreshMountStatus() {
  try {
    mountStatus.value = await GetMountStatus()
  } catch (e) {
    console.error('Failed to get mount status:', e)
  }
}

async function toggleMount() {
  loading.value = true
  error.value = ''
  try {
    if (mountStatus.value.mounted) {
      await UnmountFilesystem()
    } else {
      await MountFilesystem()
    }
    await refreshMountStatus()
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}

async function openFiles() {
  try {
    await OpenMountPoint()
  } catch (e) {
    error.value = String(e)
  }
}
</script>

<template>
  <div class="app">
    <!-- Header -->
    <header class="header">
      <div class="title">AgentFS</div>
      <div class="header-right">
        <span class="status-dot" :class="{ online: isReady }"></span>
        <span class="status-text">{{ statusText }}</span>
      </div>
    </header>

    <!-- Navigation -->
    <nav class="nav">
      <button :class="{ active: activeView === 'files' }" @click="activeView = 'files'; loadConfig()">
        Files
      </button>
      <button :class="{ active: activeView === 'search' }" @click="activeView = 'search'">
        Search
      </button>
      <button :class="{ active: activeView === 'settings' }" @click="activeView = 'settings'; loadConfig()">
        Settings
      </button>
    </nav>

    <!-- Error Banner -->
    <div v-if="error" class="error-banner" @click="clearError">
      {{ error }}
    </div>

    <!-- Main Content -->
    <main class="content">

      <!-- Search View -->
      <div v-if="activeView === 'search'" class="view">
        <div class="search-container">
          <input
            v-model="searchQuery"
            placeholder="Search your files..."
            @keyup.enter="performSearch"
            class="search-input"
            :disabled="!isReady"
          />
          <button @click="performSearch" :disabled="!isReady || loading" class="search-btn">
            Go
          </button>
        </div>

        <div v-if="!isReady" class="offline-notice">
          <p>AgentFS is offline</p>
          <button @click="toggleService" :disabled="loading" class="btn primary">
            {{ loading ? 'Starting...' : 'Start AgentFS' }}
          </button>
        </div>

        <div v-else-if="searchResults" class="results">
          <div class="results-meta">
            {{ searchResults.total }} results
          </div>
          <div v-for="result in searchResults.results" :key="result.id" class="result">
            <div class="result-file">{{ result.file_path.split('/').pop() }}</div>
            <div class="result-path">{{ result.file_path }}</div>
            <div class="result-snippet" v-if="result.content">
              {{ result.content.substring(0, 200) }}{{ result.content.length > 200 ? '...' : '' }}
            </div>
          </div>
          <div v-if="searchResults.results.length === 0" class="empty">
            No results found
          </div>
        </div>

        <div v-else class="placeholder">
          <p>Search across all your indexed files</p>
        </div>
      </div>

      <!-- Files View -->
      <div v-if="activeView === 'files'" class="view">
        <!-- Mount Section -->
        <div class="settings-section">
          <h3>AgentFS Drive</h3>
          <div class="setting-row">
            <span>Status</span>
            <span class="status-badge" :class="{ running: mountStatus.mounted }">
              {{ mountStatus.mounted ? 'Mounted' : 'Not Mounted' }}
            </span>
          </div>
          <div class="setting-row">
            <span>Location</span>
            <span class="path">{{ mountStatus.mount_point }}</span>
          </div>
          <div class="button-row">
            <button @click="toggleMount" :disabled="loading || !isReady" class="btn" :class="mountStatus.mounted ? 'danger' : 'primary'">
              {{ mountStatus.mounted ? 'Unmount' : 'Mount Drive' }}
            </button>
            <button @click="openFiles" :disabled="!mountStatus.mounted" class="btn">
              Open in Files
            </button>
          </div>
          <div v-if="!isReady" class="note">
            Start AgentFS service first
          </div>
        </div>

        <!-- Sources Section -->
        <h2>Indexed Folders</h2>
        <div v-if="config && config.sources && config.sources.length > 0" class="sources-list">
          <div v-for="source in config.sources" :key="source.name" class="source-item">
            <div class="source-info">
              <div class="source-name">{{ source.name }}</div>
              <div class="source-path">{{ source.path }}</div>
            </div>
            <button @click="removeSourceByName(source.name)" class="btn-icon danger">×</button>
          </div>
        </div>
        <div v-else class="empty">
          No folders added yet
        </div>

        <div class="add-source">
          <h3>Add Folder</h3>
          <input v-model="newSourceName" placeholder="Name (e.g., Documents)" />
          <input v-model="newSourcePath" placeholder="Path (e.g., /home/user/Documents)" />
          <button @click="addNewSource" :disabled="loading" class="btn primary">
            Add
          </button>
        </div>
      </div>

      <!-- Settings View -->
      <div v-if="activeView === 'settings'" class="view">
        <h2>Settings</h2>

        <div class="settings-section">
          <h3>Service</h3>
          <div class="setting-row">
            <span>Status</span>
            <span class="status-badge" :class="{ running: status.running }">
              {{ status.running ? 'Running' : 'Stopped' }}
            </span>
          </div>
          <div class="setting-row" v-if="status.version">
            <span>Version</span>
            <span>{{ status.version }}</span>
          </div>
          <button @click="toggleService" :disabled="loading" class="btn" :class="status.running ? 'danger' : 'primary'">
            {{ status.running ? 'Stop Service' : 'Start Service' }}
          </button>
        </div>

        <div class="settings-section">
          <h3>Configuration</h3>
          <div class="setting-row">
            <span>Config Directory</span>
            <span class="path">{{ status.configDir }}</span>
          </div>
          <button @click="initializeConfig" :disabled="loading" class="btn">
            Reset Config
          </button>
        </div>

        <div class="settings-section">
          <h3>Application</h3>
          <button @click="quitApp" class="btn danger">
            Quit AgentFS
          </button>
        </div>
      </div>
    </main>

    <!-- Loading Overlay -->
    <div v-if="loading" class="loading">
      <div class="spinner"></div>
    </div>
  </div>
</template>

<style>
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #1a1a2e;
  color: #e0e0e0;
  font-size: 14px;
  overflow: hidden;
  user-select: none;
}

.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

/* Header */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #16213e;
  border-bottom: 1px solid #0f3460;
  -webkit-app-region: drag;
}

.title {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  -webkit-app-region: no-drag;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #666;
}

.status-dot.online {
  background: #4caf50;
}

.status-text {
  font-size: 12px;
  color: #888;
}

/* Navigation */
.nav {
  display: flex;
  background: #16213e;
  border-bottom: 1px solid #0f3460;
  padding: 0 8px;
}

.nav button {
  flex: 1;
  padding: 10px;
  border: none;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 13px;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.nav button:hover {
  color: #e0e0e0;
}

.nav button.active {
  color: #fff;
  border-bottom-color: #4caf50;
}

/* Error Banner */
.error-banner {
  background: #e94560;
  color: #fff;
  padding: 8px 16px;
  font-size: 12px;
  cursor: pointer;
}

/* Content */
.content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.view {
  height: 100%;
}

h2 {
  font-size: 16px;
  margin-bottom: 16px;
  color: #fff;
}

h3 {
  font-size: 13px;
  margin-bottom: 8px;
  color: #888;
}

/* Search */
.search-container {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.search-input {
  flex: 1;
  padding: 12px;
  border: 1px solid #0f3460;
  background: #16213e;
  color: #fff;
  border-radius: 6px;
  font-size: 14px;
}

.search-input:focus {
  outline: none;
  border-color: #4caf50;
}

.search-input:disabled {
  opacity: 0.5;
}

.search-btn {
  padding: 12px 20px;
  background: #4caf50;
  border: none;
  color: #fff;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
}

.search-btn:hover:not(:disabled) {
  background: #45a049;
}

.search-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Offline Notice */
.offline-notice {
  text-align: center;
  padding: 40px 20px;
}

.offline-notice p {
  margin-bottom: 16px;
  color: #888;
}

/* Results */
.results-meta {
  font-size: 12px;
  color: #888;
  margin-bottom: 12px;
}

.result {
  background: #16213e;
  border: 1px solid #0f3460;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}

.result-file {
  font-weight: 500;
  color: #4caf50;
  margin-bottom: 2px;
}

.result-path {
  font-size: 11px;
  color: #666;
  margin-bottom: 6px;
  word-break: break-all;
}

.result-snippet {
  font-size: 12px;
  color: #aaa;
  line-height: 1.4;
}

/* Placeholder & Empty */
.placeholder, .empty {
  text-align: center;
  padding: 40px 20px;
  color: #666;
}

/* Sources */
.sources-list {
  margin-bottom: 20px;
}

.source-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #16213e;
  border: 1px solid #0f3460;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}

.source-name {
  font-weight: 500;
  color: #fff;
}

.source-path {
  font-size: 11px;
  color: #666;
  margin-top: 2px;
}

.btn-icon {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-icon.danger:hover {
  background: #e94560;
  color: #fff;
}

/* Add Source */
.add-source {
  background: #16213e;
  border: 1px solid #0f3460;
  border-radius: 6px;
  padding: 16px;
}

.add-source input {
  width: 100%;
  padding: 10px;
  border: 1px solid #0f3460;
  background: #1a1a2e;
  color: #fff;
  border-radius: 4px;
  margin-bottom: 8px;
  font-size: 13px;
}

.add-source input:focus {
  outline: none;
  border-color: #4caf50;
}

/* Settings */
.settings-section {
  background: #16213e;
  border: 1px solid #0f3460;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 12px;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #0f3460;
}

.setting-row:last-of-type {
  border-bottom: none;
  margin-bottom: 12px;
}

.setting-row .path {
  font-size: 11px;
  color: #666;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: #666;
}

.status-badge.running {
  background: #4caf50;
}

/* Buttons */
.btn {
  padding: 10px 16px;
  border: 1px solid #0f3460;
  background: #1a1a2e;
  color: #e0e0e0;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  width: 100%;
}

.btn:hover:not(:disabled) {
  background: #0f3460;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn.primary {
  background: #4caf50;
  border-color: #4caf50;
  color: #fff;
}

.btn.primary:hover:not(:disabled) {
  background: #45a049;
}

.btn.danger {
  background: #e94560;
  border-color: #e94560;
  color: #fff;
}

.btn.danger:hover:not(:disabled) {
  background: #d13652;
}

/* Button Row */
.button-row {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.button-row .btn {
  flex: 1;
}

/* Note */
.note {
  font-size: 11px;
  color: #888;
  margin-top: 8px;
  text-align: center;
}

/* Loading */
.loading {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #0f3460;
  border-top-color: #4caf50;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
}

::-webkit-scrollbar-track {
  background: #1a1a2e;
}

::-webkit-scrollbar-thumb {
  background: #0f3460;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #16213e;
}
</style>
