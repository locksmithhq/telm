<template>
  <v-card class="panel-card" :class="`size-${panel.size || 'md'}`">
    <!-- Header -->
    <div class="panel-header">
      <span class="drag-handle">
        <v-icon size="14" color="grey">mdi-drag-vertical</v-icon>
      </span>
      <span class="text-caption font-weight-medium text-truncate flex-grow-1 mx-1">
        {{ panel.title || sourceLabel || 'Panel' }}
      </span>
      <v-btn icon size="x-small" variant="text" color="grey" title="Edit" @click="$emit('edit')">
        <v-icon size="12">mdi-pencil-outline</v-icon>
      </v-btn>
      <v-btn icon size="x-small" variant="text" color="grey" title="Remove" @click="$emit('remove')">
        <v-icon size="12">mdi-close</v-icon>
      </v-btn>
    </div>

    <!-- Visualization -->
    <div class="panel-viz">
      <PanelViz :query="panel.query" :dash-range="dashRange" />
    </div>
  </v-card>
</template>

<script setup>
import { computed } from 'vue'
import { parse, SOURCES } from './tql.js'
import PanelViz from './PanelViz.vue'

const props = defineProps({
  panel:     { type: Object, required: true },
  dashRange: { type: String, default: null },
})
defineEmits(['edit', 'remove'])

const sourceLabel = computed(() => {
  const q = (props.panel.query || '').trim()
  if (!q) return ''
  const parsed = parse(q)
  return parsed?.def?.label || ''
})
</script>

<style scoped>
.panel-card {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: var(--telm-bg-row) !important;
}

.panel-header {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-bottom: 1px solid var(--telm-border-light);
  flex-shrink: 0;
  gap: 2px;
}

.drag-handle {
  cursor: grab;
  opacity: 0.45;
  transition: opacity 0.15s;
  display: flex;
  align-items: center;
}
.drag-handle:hover { opacity: 1; }
.drag-handle:active { cursor: grabbing; }

.panel-viz {
  flex: 1;
  padding: 4px 4px 6px;
  min-height: 0;
  overflow: hidden;
}
</style>
