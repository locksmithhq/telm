<template>
  <v-container fluid class="pa-3">
    <!-- Header -->
    <div class="d-flex align-center gap-2 mb-3 flex-wrap">
      <v-btn icon size="small" variant="text" @click="$router.push('/dashboards')">
        <v-icon size="16">mdi-arrow-left</v-icon>
      </v-btn>

      <template v-if="!editingName">
        <span class="text-body-2 font-weight-medium">{{ dashboard.name }}</span>
        <v-btn icon size="x-small" variant="text" color="grey" @click="startRename">
          <v-icon size="12">mdi-pencil-outline</v-icon>
        </v-btn>
      </template>
      <template v-else>
        <v-text-field
          v-model="nameInput"
          hide-details density="compact" style="max-width: 260px"
          autofocus
          @keyup.enter="confirmRename"
          @keyup.esc="editingName = false"
          @blur="confirmRename"
        />
      </template>

      <v-spacer />

      <!-- Time range picker — Datadog style -->
      <div class="range-picker">
        <button
          v-for="r in RANGE_VALUES"
          :key="r"
          class="range-btn"
          :class="{ active: dashRange === r }"
          @click="dashRange = r"
        >{{ r }}</button>
      </div>

      <v-btn size="small" color="primary" variant="flat" prepend-icon="mdi-plus" @click="openEditor(null)">
        Add panel
      </v-btn>
    </div>

    <!-- Empty state -->
    <div v-if="!panels.length" class="empty-state d-flex flex-column align-center justify-center">
      <v-icon size="44" color="grey-darken-2">mdi-chart-box-plus-outline</v-icon>
      <div class="text-body-2 text-disabled mt-3">No panels yet</div>
      <div class="text-caption text-disabled mb-4">
        Click <strong>Add panel</strong> and use the builder to select data source, filters and visualization
      </div>
      <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="openEditor(null)">
        Add panel
      </v-btn>
    </div>

    <!-- Panels grid -->
    <VueDraggable
      v-else
      v-model="panels"
      class="panel-grid"
      handle=".drag-handle"
      :animation="150"
      ghost-class="panel-ghost"
      chosen-class="panel-chosen"
      @end="syncAndPersist"
    >
      <div
        v-for="p in panels"
        :key="p.id"
        class="panel-item"
        :style="panelStyle(p)"
      >
        <PanelCard
          :panel="p"
          :dash-range="dashRange"
          @edit="openEditor(p)"
          @remove="removePanel(p)"
        />
        <!-- Width handle — right edge, snaps to size classes -->
        <div class="handle-w" @mousedown.prevent="startWidthResize($event, p)" />
        <!-- Height handle — bottom edge, free pixel drag -->
        <div class="handle-h" @mousedown.prevent="startHeightResize($event, p)" />
        <!-- Corner handle — both at once -->
        <div class="handle-corner" @mousedown.prevent="startCornerResize($event, p)" />
      </div>
    </VueDraggable>

    <!-- Panel editor -->
    <PanelEditor v-model="editorOpen" :panel="editingPanel" @apply="onEditorApply" />
  </v-container>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { VueDraggable } from 'vue-draggable-plus'
import PanelCard from './PanelCard.vue'
import PanelEditor from './PanelEditor.vue'
import { loadAll, update, newPanel } from './store.js'
import { RANGE_VALUES } from './tql.js'
import api from '@/plugins/axios'

const route = useRoute()

const dashboard    = ref({ name: '', panels: [] })
const panels       = ref([])
const editingName  = ref(false)
const nameInput    = ref('')
const editorOpen   = ref(false)
const editingPanel = ref(null)
const dashRange    = ref('1h')

async function load() {
  try {
    const { data } = await api.get(`/dashboards/${route.params.id}`)
    dashboard.value = data
    panels.value = data.panels || []
  } catch (e) {
    console.error('Failed to load dashboard:', e)
  }
}

async function persist() {
  try {
    await update(dashboard.value)
  } catch (e) {
    console.error('Failed to save dashboard:', e)
  }
}

function syncAndPersist() {
  dashboard.value.panels = panels.value
  persist()
}

function openEditor(panel) {
  editingPanel.value = panel
  editorOpen.value   = true
}

function onEditorApply(changes) {
  if (editingPanel.value) {
    const found = panels.value.find((p) => p.id === editingPanel.value.id)
    if (found) Object.assign(found, changes)
  } else {
    panels.value.push({ ...newPanel(changes.query), ...changes })
  }
  syncAndPersist()
}

function removePanel(p) {
  const idx = panels.value.findIndex((x) => x.id === p.id)
  if (idx !== -1) panels.value.splice(idx, 1)
  syncAndPersist()
}

const GRID_COLS = 4
const ROW_H     = 80
const GAP       = 10

const resizingId   = ref(null)
const resizingCols = ref(2)
const resizingRows = ref(3)

function panelCols(p) { return p.cols ?? { sm: 1, md: 2, lg: 3, full: 4 }[p.size] ?? 2 }
function panelRows(p) { return p.rows ?? Math.max(2, Math.ceil((p.height || 240) / ROW_H)) }

function panelStyle(p) {
  const c = resizingId.value === p.id ? resizingCols.value : panelCols(p)
  const r = resizingId.value === p.id ? resizingRows.value : panelRows(p)
  return `grid-column: span ${c}; grid-row: span ${r}`
}

function startWidthResize(event, panel) {
  const item       = event.currentTarget.parentElement
  const container  = item.parentElement
  const startX     = event.clientX
  const startCols  = panelCols(panel)
  const colW       = container.getBoundingClientRect().width / GRID_COLS

  resizingId.value   = panel.id
  resizingCols.value = startCols
  resizingRows.value = panelRows(panel)

  const onMove = (e) => {
    resizingCols.value = Math.min(GRID_COLS, Math.max(1, Math.round(startCols + (e.clientX - startX) / colW)))
  }
  const onUp = () => {
    panel.cols = resizingCols.value
    resizingId.value = null
    persist()
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup',   onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup',   onUp)
}

function startHeightResize(event, panel) {
  const startY    = event.clientY
  const startRows = panelRows(panel)

  resizingId.value   = panel.id
  resizingCols.value = panelCols(panel)
  resizingRows.value = startRows

  const onMove = (e) => {
    resizingRows.value = Math.max(1, Math.round(startRows + (e.clientY - startY) / (ROW_H + GAP)))
  }
  const onUp = () => {
    panel.rows = resizingRows.value
    resizingId.value = null
    persist()
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup',   onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup',   onUp)
}

function startCornerResize(event, panel) {
  const item      = event.currentTarget.parentElement
  const container = item.parentElement
  const startX    = event.clientX
  const startY    = event.clientY
  const startCols = panelCols(panel)
  const startRows = panelRows(panel)
  const colW      = container.getBoundingClientRect().width / GRID_COLS

  resizingId.value   = panel.id
  resizingCols.value = startCols
  resizingRows.value = startRows

  const onMove = (e) => {
    resizingCols.value = Math.min(GRID_COLS, Math.max(1, Math.round(startCols + (e.clientX - startX) / colW)))
    resizingRows.value = Math.max(1, Math.round(startRows + (e.clientY - startY) / (ROW_H + GAP)))
  }
  const onUp = () => {
    panel.cols = resizingCols.value
    panel.rows = resizingRows.value
    resizingId.value = null
    persist()
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup',   onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup',   onUp)
}

function startRename() {
  nameInput.value  = dashboard.value.name
  editingName.value = true
}
async function confirmRename() {
  if (nameInput.value.trim()) dashboard.value.name = nameInput.value.trim()
  editingName.value = false
  await persist()
}

onMounted(load)
watch(() => route.params.id, load)
</script>

<style scoped>
/* Gallery grid — 4 columns, each row 80px, dense packing fills gaps */
.panel-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-auto-rows: 80px;
  grid-auto-flow: row dense;
  gap: 10px;
}

/* Each draggable item — fills its grid area */
.panel-item {
  position: relative;
  min-width: 0;
  min-height: 0;
}

@media (max-width: 860px) {
  .panel-grid {
    grid-template-columns: 1fr;
    grid-auto-rows: 60px;
  }
  .panel-item { grid-column: span 1 !important; }
}

/* Drag states */
.panel-ghost  { opacity: 0.25; outline: 2px dashed rgba(99,102,241,0.5); border-radius: 8px; }
.panel-chosen { outline: 2px solid rgba(99,102,241,0.7); border-radius: 8px; }

/* ─── Resize handles ───────────────────────────────────────────────────── */
.handle-w,
.handle-h,
.handle-corner {
  position: absolute;
  z-index: 50;
  opacity: 0;
  transition: opacity 0.15s;
}
.panel-item:hover .handle-w,
.panel-item:hover .handle-h,
.panel-item:hover .handle-corner { opacity: 1; }

/* Right edge — width */
.handle-w {
  top: 8px; right: -6px;
  width: 12px; height: calc(100% - 24px);
  cursor: ew-resize;
  display: flex; align-items: center; justify-content: center;
}
.handle-w::after {
  content: '';
  width: 4px; height: 36px; border-radius: 3px;
  background: rgba(99,102,241,0.55);
}
.handle-w:hover::after { background: rgba(99,102,241,1); }

/* Bottom edge — height */
.handle-h {
  bottom: -6px; left: 8px;
  height: 12px; width: calc(100% - 24px);
  cursor: ns-resize;
  display: flex; align-items: center; justify-content: center;
}
.handle-h::after {
  content: '';
  height: 4px; width: 36px; border-radius: 3px;
  background: rgba(99,102,241,0.55);
}
.handle-h:hover::after { background: rgba(99,102,241,1); }

/* Bottom-right corner — width + height */
.handle-corner {
  bottom: -6px; right: -6px;
  width: 16px; height: 16px;
  cursor: nwse-resize;
  display: flex; align-items: center; justify-content: center;
}
.handle-corner::after {
  content: '';
  width: 8px; height: 8px; border-radius: 2px;
  background: rgba(99,102,241,0.8);
  box-shadow: 0 0 4px rgba(99,102,241,0.5);
}
.handle-corner:hover::after { background: rgba(99,102,241,1); }

/* Empty state */
.empty-state { min-height: 280px; text-align: center; }

/* ─── Time range picker ─────────────────────────────────────────────────── */
.range-picker {
  display: flex;
  align-items: center;
  background: var(--telm-bg-row);
  border: 1px solid var(--telm-border);
  border-radius: 6px;
  padding: 2px;
  gap: 1px;
}

.range-btn {
  font-size: 11px;
  font-weight: 500;
  line-height: 1;
  padding: 4px 8px;
  border-radius: 4px;
  border: none;
  background: transparent;
  color: var(--telm-text-3);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  white-space: nowrap;
}
.range-btn:hover {
  background: var(--telm-bg-hover);
  color: var(--telm-text-1);
}
.range-btn.active {
  background: #6366f1;
  color: #fff;
}
</style>
