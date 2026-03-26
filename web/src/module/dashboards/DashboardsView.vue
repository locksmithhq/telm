<template>
  <v-container fluid class="pa-4">
    <!-- Header -->
    <div class="d-flex align-center gap-3 mb-4">
      <div>
        <div class="text-body-1 font-weight-medium">Dashboards</div>
        <div class="text-caption text-disabled">Build custom views with any metric, trace or log data</div>
      </div>
      <v-spacer />
      <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="createDialog = true">
        New dashboard
      </v-btn>
    </div>

    <!-- Empty state -->
    <div v-if="!dashboards.length" class="empty-state d-flex flex-column align-center justify-center">
      <v-icon size="52" color="grey-darken-2">mdi-monitor-dashboard</v-icon>
      <div class="text-body-2 text-disabled mt-3">No dashboards yet</div>
      <div class="text-caption text-disabled mb-4">Create your first dashboard to start building custom views</div>
      <v-btn color="primary" variant="flat" prepend-icon="mdi-plus" @click="createDialog = true">
        New dashboard
      </v-btn>
    </div>

    <!-- Dashboard list -->
    <v-row v-else dense>
      <v-col
        v-for="d in dashboards"
        :key="d.id"
        cols="12" sm="6" md="4" lg="3"
      >
        <v-card
          class="dashboard-card"
          hover
          @click="$router.push(`/dashboards/${d.id}`)"
        >
          <v-card-text class="pa-4">
            <div class="d-flex align-start justify-space-between">
              <div class="overflow-hidden flex-grow-1 mr-2">
                <div class="text-body-2 font-weight-medium text-truncate">{{ d.name }}</div>
                <div class="text-caption text-disabled mt-1">
                  {{ d.panels.length }} panel{{ d.panels.length !== 1 ? 's' : '' }}
                </div>
                <div class="text-caption text-disabled" style="font-size: 10px; margin-top: 2px">
                  Created {{ relDate(d.createdAt) }}
                </div>
              </div>
              <v-menu>
                <template #activator="{ props: mp }">
                  <v-btn
                    v-bind="mp"
                    icon size="x-small" variant="text" color="grey"
                    @click.stop
                  >
                    <v-icon size="14">mdi-dots-vertical</v-icon>
                  </v-btn>
                </template>
                <v-list density="compact">
                  <v-list-item prepend-icon="mdi-pencil-outline" @click.stop="startRename(d)">Rename</v-list-item>
                  <v-list-item prepend-icon="mdi-delete-outline" base-color="error" @click.stop="deleteDashboard(d)">Delete</v-list-item>
                </v-list>
              </v-menu>
            </div>

            <!-- Panel query preview chips -->
            <div v-if="d.panels.length" class="d-flex flex-wrap gap-1 mt-2">
              <v-chip
                v-for="p in d.panels.slice(0, 4)"
                :key="p.id"
                size="x-small"
                color="primary"
                variant="tonal"
                class="mono"
                style="font-size: 9px"
              >
                {{ firstToken(p.query) }}
              </v-chip>
              <v-chip v-if="d.panels.length > 4" size="x-small" variant="text" style="font-size: 9px">
                +{{ d.panels.length - 4 }} more
              </v-chip>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create dialog -->
    <v-dialog v-model="createDialog" max-width="400">
      <v-card>
        <v-card-title class="text-body-2 pa-3">New Dashboard</v-card-title>
        <v-card-text class="pa-3 pt-0">
          <v-text-field
            v-model="newName"
            label="Name"
            hide-details
            density="compact"
            autofocus
            @keyup.enter="create"
          />
        </v-card-text>
        <v-card-actions class="pa-3 pt-0">
          <v-spacer />
          <v-btn size="small" variant="text" @click="createDialog = false">Cancel</v-btn>
          <v-btn size="small" color="primary" variant="flat" :disabled="!newName.trim()" @click="create">Create</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Rename dialog -->
    <v-dialog v-model="renameDialog" max-width="400">
      <v-card>
        <v-card-title class="text-body-2 pa-3">Rename Dashboard</v-card-title>
        <v-card-text class="pa-3 pt-0">
          <v-text-field
            v-model="renameInput"
            hide-details density="compact" autofocus
            @keyup.enter="confirmRename"
          />
        </v-card-text>
        <v-card-actions class="pa-3 pt-0">
          <v-spacer />
          <v-btn size="small" variant="text" @click="renameDialog = false">Cancel</v-btn>
          <v-btn size="small" color="primary" variant="flat" :disabled="!renameInput.trim()" @click="confirmRename">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { loadAll, saveAll, newDashboard } from './store.js'

const router = useRouter()

const dashboards = ref([])
const createDialog = ref(false)
const newName = ref('')
const renameDialog = ref(false)
const renameInput = ref('')
let renamingId = null

function load() { dashboards.value = loadAll() }
function persist() { saveAll(dashboards.value) }

function create() {
  if (!newName.value.trim()) return
  const d = newDashboard(newName.value.trim())
  dashboards.value.push(d)
  persist()
  newName.value = ''
  createDialog.value = false
  router.push(`/dashboards/${d.id}`)
}

function startRename(d) {
  renamingId = d.id
  renameInput.value = d.name
  renameDialog.value = true
}

function confirmRename() {
  if (!renameInput.value.trim()) return
  const d = dashboards.value.find((x) => x.id === renamingId)
  if (d) { d.name = renameInput.value.trim(); persist() }
  renameDialog.value = false
}

function deleteDashboard(d) {
  dashboards.value = dashboards.value.filter((x) => x.id !== d.id)
  persist()
}

function firstToken(query) {
  return (query || '').trim().split(/\s+/)[0] || '?'
}

function relDate(iso) {
  if (!iso) return ''
  const s = Math.floor((Date.now() - new Date(iso)) / 1000)
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

onMounted(load)
</script>

<style scoped>
.mono { font-family: ui-monospace, 'JetBrains Mono', monospace; }
.dashboard-card { cursor: pointer; transition: box-shadow 0.15s; }
.empty-state { min-height: 300px; text-align: center; }
</style>
