<script setup>
import { computed, ref } from 'vue'
import {
  filterWorkspaceRecords,
  orderWorkspaceRecords,
  workspaceFilterStatusValue,
  workspaceMonthValue,
  workspaceTypeLabel,
  workspaceYearLabel,
} from '../workspace-search.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  workspaces: { type: Array, default: () => [] },
  label: { type: String, required: true },
  placeholder: { type: String, default: 'Busca por servicio, período, estado o ID' },
  icon: { type: String, default: 'mdi-folder-clock-outline' },
  disabled: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  allowTypeFilter: { type: Boolean, default: true },
  allowStatusFilter: { type: Boolean, default: true },
  serviceLabel: { type: Function, default: value => value || 'Servicio' },
  monthLabel: { type: Function, default: value => value || '' },
  statusLabel: { type: Function, default: value => value || 'Estado desconocido' },
  receivedAt: { type: Function, default: () => '' },
  emptyMessage: { type: String, default: 'No hay espacios disponibles para este selector.' },
})

const emit = defineEmits(['update:modelValue'])
const search = ref('')
const service = ref('ALL')
const year = ref('ALL')
const type = ref('ALL')
const status = ref('ALL')

const orderedWorkspaces = computed(() => orderWorkspaceRecords(props.workspaces))
const serviceOptions = computed(() => {
  const services = [...new Set(orderedWorkspaces.value.map(workspace => String(workspace.tipo_servicio || '').trim()).filter(Boolean))]
    .sort((left, right) => props.serviceLabel(left).localeCompare(props.serviceLabel(right), 'es'))
  return [{ title: 'Todos los servicios', value: 'ALL' }, ...services.map(value => ({ title: props.serviceLabel(value), value }))]
})
const yearOptions = computed(() => {
  const years = [...new Set(orderedWorkspaces.value.map(workspaceYearLabel).filter(value => /^\d{4}$/.test(value)))].sort((left, right) => Number(right) - Number(left))
  return [{ title: 'Todos los años', value: 'ALL' }, ...years.map(value => ({ title: value, value }))]
})
const typeOptions = computed(() => {
  const types = [...new Set(orderedWorkspaces.value.map(workspace => workspace.es_objeciones ? 'OBJECIONES' : 'RECEPCION'))]
  return [
    { title: 'Todos los tipos', value: 'ALL' },
    ...types.map(value => ({ title: value === 'OBJECIONES' ? 'Objeciones · espacio separado' : 'Recepción de planillas', value })),
  ]
})
const statusOptions = computed(() => {
  const statuses = new Map()
  for (const workspace of orderedWorkspaces.value) {
    const value = workspaceFilterStatusValue(workspace)
    if (!value) continue
    statuses.set(value, props.statusLabel(workspace.status, workspace))
  }
  return [{ title: 'Todos los estados', value: 'ALL' }, ...[...statuses].map(([value, title]) => ({ title, value }))]
})
const typeFilterVisible = computed(() => props.allowTypeFilter && typeOptions.value.length > 2)
const filteredWorkspaces = computed(() => filterWorkspaceRecords(props.workspaces, {
  search: search.value,
  service: service.value,
  year: year.value,
  type: typeFilterVisible.value ? type.value : 'ALL',
  status: status.value,
}, {
  typeLabel: workspaceTypeLabel,
  serviceLabel: props.serviceLabel,
  monthLabel: props.monthLabel,
  statusLabel: props.statusLabel,
  receivedAt: props.receivedAt,
}))
const selectedWorkspaceOutsideFilters = computed(() => Boolean(props.modelValue)
  && orderedWorkspaces.value.some(workspace => workspace.job_id === props.modelValue)
  && !filteredWorkspaces.value.some(workspace => workspace.job_id === props.modelValue))
const displayedModelValue = computed(() => selectedWorkspaceOutsideFilters.value ? '' : props.modelValue)
const selectedWorkspaceSummary = computed(() => {
  const workspace = orderedWorkspaces.value.find(item => item.job_id === props.modelValue)
  if (!workspace) return ''
  return `${workspaceTypeLabel(workspace)} · ${props.serviceLabel(workspace.tipo_servicio)} · ${props.monthLabel(workspaceMonthValue(workspace))} ${workspaceYearLabel(workspace)} · ${props.statusLabel(workspace.status, workspace)} · ${workspace.job_id}`
})
function workspaceDetailLabel(workspace) {
  const pending = workspace.delivery_missing?.length ? workspace.delivery_missing : workspace.missing_documents || []
  if (!pending.length) return workspace.message || 'Espacio disponible para continuar.'
  const shown = pending.slice(0, 3).join('; ')
  const remaining = pending.length - 3
  return `Pendiente: ${shown}${remaining > 0 ? `; y ${remaining} más` : ''}`
}
const pickerItems = computed(() => filteredWorkspaces.value.map(workspace => {
  const typeName = workspaceTypeLabel(workspace)
  const serviceName = props.serviceLabel(workspace.tipo_servicio)
  const periodName = `${props.monthLabel(workspaceMonthValue(workspace))} ${workspaceYearLabel(workspace)}`.trim()
  const statusName = props.statusLabel(workspace.status, workspace)
  return {
    value: workspace.job_id,
    title: `${typeName} · ${serviceName} · ${periodName} · ${statusName} · ${workspace.job_id}`,
    typeName,
    serviceName,
    periodName,
    statusName,
    receivedLabel: props.receivedAt(workspace.received_at),
    userLabel: workspace.creado_por || 'Usuario anterior',
    detailLabel: workspaceDetailLabel(workspace),
    workspace,
  }
}))
const hasFilters = computed(() => Boolean(String(search.value ?? '').trim())
  || service.value !== 'ALL'
  || year.value !== 'ALL'
  || (typeFilterVisible.value && type.value !== 'ALL')
  || status.value !== 'ALL')

function clearFilters() {
  search.value = ''
  service.value = 'ALL'
  year.value = 'ALL'
  type.value = 'ALL'
  status.value = 'ALL'
}
</script>

<template>
  <section class="workspace-picker" :class="{ 'workspace-picker-fixed-type': !typeFilterVisible }" :aria-label="`Selector: ${label}`">
    <div class="workspace-picker-fields">
      <v-text-field
        v-model="search"
        class="workspace-picker-search"
        label="Buscar períodos"
        :placeholder="placeholder"
        prepend-inner-icon="mdi-magnify"
        variant="outlined"
        density="comfortable"
        hide-details
        clearable
        :disabled="disabled || loading"
      />
      <v-select v-model="service" :items="serviceOptions" label="Servicio" variant="outlined" density="comfortable" hide-details :menu-props="{ contentClass: 'workspace-picker-filter-menu' }" :disabled="disabled || loading" />
      <v-select v-model="year" :items="yearOptions" label="Año" variant="outlined" density="comfortable" hide-details :menu-props="{ contentClass: 'workspace-picker-filter-menu' }" :disabled="disabled || loading" />
      <v-select v-if="typeFilterVisible" v-model="type" :items="typeOptions" label="Tipo de espacio" variant="outlined" density="comfortable" hide-details :menu-props="{ contentClass: 'workspace-picker-filter-menu' }" :disabled="disabled || loading" />
      <v-select v-if="allowStatusFilter" v-model="status" :items="statusOptions" label="Estado" variant="outlined" density="comfortable" hide-details :menu-props="{ contentClass: 'workspace-picker-filter-menu' }" :disabled="disabled || loading" />
    </div>

    <div class="workspace-picker-results" role="status" aria-live="polite">
      <span>{{ filteredWorkspaces.length }} de {{ workspaces.length }} períodos coinciden · el más reciente primero</span>
      <v-btn v-if="hasFilters" type="button" variant="text" prepend-icon="mdi-filter-remove-outline" @click="clearFilters">Limpiar filtros</v-btn>
    </div>
    <div v-if="selectedWorkspaceOutsideFilters" class="workspace-picker-selection-note" role="status">
      <span>La selección actual queda fuera de estos filtros: {{ selectedWorkspaceSummary }}</span>
      <v-btn type="button" variant="text" @click="emit('update:modelValue', '')">Quitar selección</v-btn>
    </div>

    <v-select
      :model-value="displayedModelValue"
      class="workspace-picker-select"
      :items="pickerItems"
      item-title="title"
      item-value="value"
      :label="label"
      :placeholder="placeholder"
      :prepend-inner-icon="icon"
      variant="outlined"
      density="comfortable"
      clearable
      hide-details
      :loading="loading"
      :disabled="disabled || loading"
      :menu-props="{ contentClass: 'workspace-picker-menu', maxHeight: 'min(70vh, 680px)' }"
      @update:model-value="emit('update:modelValue', $event || '')"
    >
      <template #selection="{ item }">
        <span class="workspace-picker-selection">{{ item.raw?.title || item.title }}</span>
      </template>
      <template #item="{ props: itemProps, item }">
        <v-list-item v-bind="itemProps" class="workspace-picker-option">
          <template #title>
            <span class="workspace-picker-option-title">{{ item.raw.typeName }} · {{ item.raw.serviceName }} · {{ item.raw.periodName }}</span>
          </template>
          <template #subtitle>
            <span class="workspace-picker-option-details"><strong>Estado:</strong> {{ item.raw.statusName }} <span aria-hidden="true">·</span> <strong>ID:</strong> {{ item.raw.workspace.job_id }}</span>
            <span class="workspace-picker-option-details"><strong>Usuario:</strong> {{ item.raw.userLabel }}<template v-if="item.raw.receivedLabel"> <span aria-hidden="true">·</span> <strong>Fecha:</strong> {{ item.raw.receivedLabel }}</template></span>
            <span class="workspace-picker-option-details workspace-picker-option-message">{{ item.raw.detailLabel }}</span>
          </template>
        </v-list-item>
      </template>
      <template #no-data>
        <div class="workspace-picker-no-results">
          <strong>{{ workspaces.length ? 'No hay períodos que coincidan' : 'No hay períodos disponibles' }}</strong>
          <span v-if="workspaces.length">Prueba otra búsqueda o pulsa «Limpiar filtros» para volver a ver períodos anteriores, incluidos años antiguos.</span>
          <span v-else>{{ emptyMessage }}</span>
          <v-btn v-if="hasFilters" type="button" variant="text" @click="clearFilters">Limpiar filtros</v-btn>
        </div>
      </template>
    </v-select>
  </section>
</template>
