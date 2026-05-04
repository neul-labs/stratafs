<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { GetStatus, StartAgentFS, StopAgentFS, RestartAgentFS, GetQueueStats, GetConfig, Search, AddSource, RemoveSource, ExportSource, InitConfig, OpenConfigDir } from '../wailsjs/go/main/App'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

// State
const status = ref({ running: false, apiHealthy: false, version: '', configDir: '' })
const queueStats = ref(null)
const config = ref(null)
const searchQuery = ref('')
const searchMode = ref('hybrid')
const searchResults = ref(null)
const activeTab = ref('dashboard')
const loading = ref(false)
const error = ref('')
const newSourceName = ref('')
const newSourcePath = ref('')
const exportSourceName = ref('')
const exportPath = ref('')

let statusInterval = null

// Lifecycle
onMounted(async () => {
  await refreshStatus()
  statusInterval = setInterval(refreshStatus, 5000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
})

// Methods
async function refreshStatus() {
  try {
    status.value = await GetStatus()
    if (status.value.apiHealthy) {
      const stats = await GetQueueStats()
      queueStats.value = stats.queue_stats
    }
  } catch (e) {
    console.error('Failed to get status:', e)
  }
}

async function loadConfig() {
  try {
    config.value = await GetConfig()
    error.value = ''
  } catch (e) {
    error.value = e
    config.value = null
  }
}

async function startService() {
  loading.value = true
  error.value = ''
  try {
    await StartAgentFS()
    await refreshStatus()
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function stopService() {
  loading.value = true
  error.value = ''
  try {
    await StopAgentFS()
    await refreshStatus()
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function restartService() {
  loading.value = true
  error.value = ''
  try {
    await RestartAgentFS()
    await refreshStatus()
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function performSearch() {
  if (!searchQuery.value.trim()) return
  loading.value = true
  error.value = ''
  try {
    searchResults.value = await Search(searchQuery.value, searchMode.value, 20)
  } catch (e) {
    error.value = e
    searchResults.value = null
  }
  loading.value = false
}

async function addNewSource() {
  if (!newSourceName.value.trim() || !newSourcePath.value.trim()) {
    error.value = 'Please enter both name and path'
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
    error.value = e
  }
  loading.value = false
}

async function removeSourceByName(name) {
  if (!confirm(`Remove source "${name}"?`)) return
  loading.value = true
  error.value = ''
  try {
    await RemoveSource(name)
    await loadConfig()
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function exportSourceData() {
  if (!exportSourceName.value || !exportPath.value) {
    error.value = 'Please select source and output path'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await ExportSource(exportSourceName.value, exportPath.value)
    alert('Export completed successfully!')
    exportSourceName.value = ''
    exportPath.value = ''
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function initializeConfig() {
  loading.value = true
  error.value = ''
  try {
    await InitConfig()
    await loadConfig()
    alert('Configuration initialized!')
  } catch (e) {
    error.value = e
  }
  loading.value = false
}

async function openConfig() {
  try {
    await OpenConfigDir()
  } catch (e) {
    error.value = e
  }
}

function openDocs() {
  BrowserOpenURL('http://localhost:8080/docs')
}

function switchTab(tab) {
  activeTab.value = tab
  error.value = ''
  if (tab === 'sources' || tab === 'settings') {
    loadConfig()
  }
}
</script>

<template>
  <div class="app">
    <!-- Header -->
    <header class="header">
      <div class="logo">
        <span class="logo-icon">⚡</span>
        <span class="logo-text">AgentFS</span>
      </div>
      <div class="status-indicator" :class="{ healthy: status.apiHealthy, stopped: !status.running }">
        {{ status.apiHealthy ? 'Running' : status.running ? 'Starting...' : 'Stopped' }}
      </div>
    </header>

    <!-- Navigation -->
    <nav class="nav">
      <button
        v-for="tab in ['dashboard', 'search', 'sources', 'queue', 'export', 'settings']"
        :key="tab"
        :class="{ active: activeTab === tab }"
        @click="switchTab(tab)"
      >
        {{ tab.charAt(0).toUpperCase() + tab.slice(1) }}
      </button>
    </nav>

    <!-- Main Content -->
    <main class="content">
      <!-- Error Display -->
      <div v-if="error" class="error-banner">
        {{ error }}
        <button @click="error = ''" class="close-btn">×</button>
      </div>

      <!-- Dashboard Tab -->
      <div v-if="activeTab === 'dashboard'" class="tab-content">
        <h2>Dashboard</h2>

        <div class="status-cards">
          <div class="card">
            <h3>Service Status</h3>
            <div class="status-badge" :class="{ running: status.running }">
              {{ status.running ? 'Running' : 'Stopped' }}
            </div>
            <p v-if="status.version">Version: {{ status.version }}</p>
            <p>Config: {{ status.configDir }}</p>
          </div>

          <div class="card" v-if="queueStats">
            <h3>Queue Overview</h3>
            <div class="stats-grid">
              <div class="stat">
                <span class="stat-value">{{ queueStats.pending }}</span>
                <span class="stat-label">Pending</span>
              </div>
              <div class="stat">
                <span class="stat-value">{{ queueStats.processing }}</span>
                <span class="stat-label">Processing</span>
              </div>
              <div class="stat">
                <span class="stat-value">{{ queueStats.completed }}</span>
                <span class="stat-label">Completed</span>
              </div>
              <div class="stat">
                <span class="stat-value">{{ queueStats.failed }}</span>
                <span class="stat-label">Failed</span>
              </div>
            </div>
          </div>
        </div>

        <div class="actions">
          <button @click="startService" :disabled="status.running || loading" class="btn primary">
            Start
          </button>
          <button @click="stopService" :disabled="!status.running || loading" class="btn danger">
            Stop
          </button>
          <button @click="restartService" :disabled="loading" class="btn secondary">
            Restart
          </button>
          <button @click="openDocs" :disabled="!status.apiHealthy" class="btn">
            API Docs
          </button>
        </div>
      </div>

      <!-- Search Tab -->
      <div v-if="activeTab === 'search'" class="tab-content">
        <h2>Search</h2>

        <div class="search-box">
          <input
            v-model="searchQuery"
            placeholder="Enter search query..."
            @keyup.enter="performSearch"
            :disabled="!status.apiHealthy"
          />
          <select v-model="searchMode">
            <option value="hybrid">Hybrid</option>
            <option value="fulltext">Full Text</option>
            <option value="vector">Semantic</option>
          </select>
          <button @click="performSearch" :disabled="!status.apiHealthy || loading" class="btn primary">
            Search
          </button>
        </div>

        <div v-if="searchResults" class="search-results">
          <div class="results-header">
            <span>{{ searchResults.total }} results</span>
            <span>{{ searchResults.time_taken }}</span>
          </div>

          <div v-for="result in searchResults.results" :key="result.id" class="result-item">
            <div class="result-path">{{ result.file_path }}</div>
            <div class="result-score">Score: {{ result.score.toFixed(3) }}</div>
            <div class="result-content" v-if="result.content">
              {{ result.content.substring(0, 300) }}{{ result.content.length > 300 ? '...' : '' }}
            </div>
          </div>
        </div>

        <div v-if="!status.apiHealthy" class="warning">
          Start AgentFS to enable search
        </div>
      </div>

      <!-- Sources Tab -->
      <div v-if="activeTab === 'sources'" class="tab-content">
        <h2>Storage Sources</h2>

        <div v-if="config && config.sources" class="sources-list">
          <div v-for="source in config.sources" :key="source.name" class="source-item">
            <div class="source-info">
              <strong>{{ source.name }}</strong>
              <span class="source-type">{{ source.type }}</span>
              <div class="source-path">{{ source.path }}</div>
            </div>
            <button @click="removeSourceByName(source.name)" class="btn danger small">
              Remove
            </button>
          </div>

          <div v-if="config.sources.length === 0" class="empty-state">
            No sources configured
          </div>
        </div>

        <div class="add-source">
          <h3>Add New Source</h3>
          <div class="form-row">
            <input v-model="newSourceName" placeholder="Source name" />
            <input v-model="newSourcePath" placeholder="Directory path" />
            <button @click="addNewSource" :disabled="loading" class="btn primary">
              Add
            </button>
          </div>
        </div>
      </div>

      <!-- Queue Tab -->
      <div v-if="activeTab === 'queue'" class="tab-content">
        <h2>Processing Queue</h2>

        <div v-if="queueStats" class="queue-stats">
          <div class="stat-row">
            <span class="label">Pending Jobs:</span>
            <span class="value">{{ queueStats.pending }}</span>
          </div>
          <div class="stat-row">
            <span class="label">Processing:</span>
            <span class="value">{{ queueStats.processing }}</span>
          </div>
          <div class="stat-row">
            <span class="label">Completed:</span>
            <span class="value">{{ queueStats.completed }}</span>
          </div>
          <div class="stat-row">
            <span class="label">Failed:</span>
            <span class="value">{{ queueStats.failed }}</span>
          </div>
          <div class="stat-row total">
            <span class="label">Total:</span>
            <span class="value">{{ queueStats.total }}</span>
          </div>
        </div>

        <div v-else class="warning">
          Start AgentFS to view queue statistics
        </div>

        <button @click="refreshStatus" class="btn secondary">
          Refresh
        </button>
      </div>

      <!-- Export Tab -->
      <div v-if="activeTab === 'export'" class="tab-content">
        <h2>Export Metadata</h2>

        <p class="description">
          Export a source's semantic metadata to a directory for inspection by traditional tools.
        </p>

        <div class="export-form">
          <div class="form-group">
            <label>Source:</label>
            <select v-model="exportSourceName">
              <option value="">Select source...</option>
              <option v-for="source in (config?.sources || [])" :key="source.name" :value="source.name">
                {{ source.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Output Path:</label>
            <input v-model="exportPath" placeholder="/path/to/export" />
          </div>

          <button @click="exportSourceData" :disabled="loading" class="btn primary">
            Export
          </button>
        </div>
      </div>

      <!-- Settings Tab -->
      <div v-if="activeTab === 'settings'" class="tab-content">
        <h2>Settings</h2>

        <div v-if="config" class="settings-info">
          <div class="setting-row">
            <span class="label">Version:</span>
            <span class="value">{{ config.version || 'N/A' }}</span>
          </div>
          <div class="setting-row">
            <span class="label">API Port:</span>
            <span class="value">{{ config.api_port || 8080 }}</span>
          </div>
          <div class="setting-row">
            <span class="label">MCP Port:</span>
            <span class="value">{{ config.mcp_port || 8081 }}</span>
          </div>
        </div>

        <div class="actions">
          <button @click="openConfig" class="btn secondary">
            Open Config Directory
          </button>
          <button @click="initializeConfig" :disabled="loading" class="btn">
            Reset Config
          </button>
        </div>
      </div>
    </main>

    <!-- Loading Overlay -->
    <div v-if="loading" class="loading-overlay">
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
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
  background: #1a1a2e;
  color: #e0e0e0;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: #16213e;
  border-bottom: 1px solid #0f3460;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo-icon {
  font-size: 1.5rem;
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 600;
}

.status-indicator {
  padding: 0.25rem 0.75rem;
  border-radius: 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  background: #e94560;
}

.status-indicator.healthy {
  background: #4caf50;
}

.status-indicator.stopped {
  background: #666;
}

.nav {
  display: flex;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  background: #16213e;
  border-bottom: 1px solid #0f3460;
  overflow-x: auto;
}

.nav button {
  padding: 0.5rem 1rem;
  border: none;
  background: transparent;
  color: #a0a0a0;
  cursor: pointer;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  white-space: nowrap;
}

.nav button:hover {
  background: #0f3460;
  color: #e0e0e0;
}

.nav button.active {
  background: #0f3460;
  color: #fff;
}

.content {
  flex: 1;
  padding: 1.5rem;
  overflow-y: auto;
}

.tab-content {
  max-width: 800px;
}

h2 {
  margin-bottom: 1.5rem;
  color: #fff;
}

h3 {
  margin-bottom: 1rem;
  font-size: 1rem;
  color: #a0a0a0;
}

.error-banner {
  background: #e94560;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.close-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 1.25rem;
  cursor: pointer;
}

.status-cards {
  display: grid;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.card {
  background: #16213e;
  padding: 1.25rem;
  border-radius: 0.5rem;
  border: 1px solid #0f3460;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  background: #e94560;
  margin-bottom: 0.5rem;
}

.status-badge.running {
  background: #4caf50;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.stat {
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: 600;
  color: #fff;
}

.stat-label {
  font-size: 0.75rem;
  color: #a0a0a0;
}

.actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.btn {
  padding: 0.5rem 1rem;
  border: 1px solid #0f3460;
  background: #16213e;
  color: #e0e0e0;
  border-radius: 0.25rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.btn:hover:not(:disabled) {
  background: #0f3460;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn.primary {
  background: #0f3460;
  border-color: #0f3460;
}

.btn.primary:hover:not(:disabled) {
  background: #1a4b8c;
}

.btn.secondary {
  background: transparent;
}

.btn.danger {
  background: #e94560;
  border-color: #e94560;
}

.btn.danger:hover:not(:disabled) {
  background: #d13652;
}

.btn.small {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.search-box {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}

.search-box input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #0f3460;
  background: #16213e;
  color: #e0e0e0;
  border-radius: 0.25rem;
}

.search-box select {
  padding: 0.5rem;
  border: 1px solid #0f3460;
  background: #16213e;
  color: #e0e0e0;
  border-radius: 0.25rem;
}

.search-results {
  background: #16213e;
  border-radius: 0.5rem;
  border: 1px solid #0f3460;
}

.results-header {
  display: flex;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #0f3460;
  font-size: 0.875rem;
  color: #a0a0a0;
}

.result-item {
  padding: 1rem;
  border-bottom: 1px solid #0f3460;
}

.result-item:last-child {
  border-bottom: none;
}

.result-path {
  font-weight: 500;
  color: #4caf50;
  margin-bottom: 0.25rem;
  word-break: break-all;
}

.result-score {
  font-size: 0.75rem;
  color: #a0a0a0;
  margin-bottom: 0.5rem;
}

.result-content {
  font-size: 0.875rem;
  color: #ccc;
  line-height: 1.4;
}

.warning {
  background: #ff9800;
  color: #000;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  margin: 1rem 0;
}

.sources-list {
  margin-bottom: 1.5rem;
}

.source-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: #16213e;
  border: 1px solid #0f3460;
  border-radius: 0.5rem;
  margin-bottom: 0.5rem;
}

.source-info strong {
  display: block;
  margin-bottom: 0.25rem;
}

.source-type {
  font-size: 0.75rem;
  background: #0f3460;
  padding: 0.125rem 0.5rem;
  border-radius: 0.25rem;
  margin-left: 0.5rem;
}

.source-path {
  font-size: 0.875rem;
  color: #a0a0a0;
  margin-top: 0.25rem;
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: #666;
}

.add-source, .export-form {
  background: #16213e;
  padding: 1.25rem;
  border-radius: 0.5rem;
  border: 1px solid #0f3460;
}

.form-row {
  display: flex;
  gap: 0.5rem;
}

.form-row input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #0f3460;
  background: #1a1a2e;
  color: #e0e0e0;
  border-radius: 0.25rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #0f3460;
  background: #1a1a2e;
  color: #e0e0e0;
  border-radius: 0.25rem;
}

.description {
  color: #a0a0a0;
  margin-bottom: 1.5rem;
  font-size: 0.875rem;
}

.queue-stats, .settings-info {
  background: #16213e;
  padding: 1.25rem;
  border-radius: 0.5rem;
  border: 1px solid #0f3460;
  margin-bottom: 1.5rem;
}

.stat-row, .setting-row {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid #0f3460;
}

.stat-row:last-child, .setting-row:last-child {
  border-bottom: none;
}

.stat-row.total {
  font-weight: 600;
  margin-top: 0.5rem;
  padding-top: 1rem;
}

.label {
  color: #a0a0a0;
}

.value {
  font-weight: 500;
}

.loading-overlay {
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
  width: 40px;
  height: 40px;
  border: 3px solid #0f3460;
  border-top-color: #4caf50;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (min-width: 640px) {
  .status-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .stats-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}
</style>
