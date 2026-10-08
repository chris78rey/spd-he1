<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { openDB } from 'idb'
import mspPdfCodes from '../catalogos/codigos_msp.json'
import WorkspacePicker from './components/WorkspacePicker.vue'
import { filterWorkspaceRecords, normalizeWorkspaceSearch, orderWorkspaceRecords, workspaceFilterStatusValue, workspaceMonthValue, workspaceTypeLabel, workspaceYearLabel } from './workspace-search.js'

const dbPromise = openDB('folio-documentos', 1, { upgrade(db) { db.createObjectStore('files', { keyPath: 'id' }) } })
const folders = ['Todos los documentos', 'Favoritos']
const activeFolder = ref(folders[0])
const query = ref('')
const sortBy = ref('Recientes')
const typeFilter = ref('Todo')
const dateFilter = ref('Cualquier fecha')
const view = ref('grid')
const docs = ref([])
const uploadInput = ref(null)
const preview = ref(null)
const previewUrl = ref('')
const snackbar = ref('')
const loading = ref(true)
const customSpaces = ref(JSON.parse(localStorage.getItem('folio-spaces') || '[]'))
const createSpaceDialog = ref(false)
const newSpaceName = ref('')
const dragging = ref(false)
const selectedIds = ref([])
const moveDialog = ref(false)
const moveTarget = ref('Mi espacio')
const activeDoc = ref(null)
const searchInput = ref(null)
const authStatus = ref('checking')
const authError = ref('')
const oracleUser = ref('')
const oraclePassword = ref('')
const showPassword = ref(false)
const signingIn = ref(false)
const currentUser = ref('')
const canDeleteWorkspaces = ref(false)
const displayUser = computed(() => currentUser.value.toLocaleUpperCase('es'))
const activePage = ref('ingesta')
const planillaPage = ref(1)
const planillaData = ref({ columns: [], rows: [], total: 0, totalPages: 1 })
const planillaLoading = ref(false)
const planillaError = ref('')
const ingestMonth = ref('')
const ingestYear = ref(String(new Date().getFullYear()))
const ingestService = ref('')
const ingestServices = [
  { title: 'Ambulatorio', value: 'AMBULATORIO' },
  { title: 'Emergencia', value: 'EMERGENCIA' },
  { title: 'Hospitalización', value: 'HOSPITALIZACION' },
]
const ingestMonths = ['Enero','Febrero','Marzo','Abril','Mayo','Junio','Julio','Agosto','Septiembre','Octubre','Noviembre','Diciembre'].map((title, index) => ({ title, value: String(index + 1).padStart(2, '0') }))
const ingestFileFields = [
  { field: 'zip_file', title: 'Lote', detail: 'Archivo ZIP con las carpetas numéricas de planillas', accept: '.zip', icon: 'mdi-folder-zip-outline' },
  { field: 'matriz_file', title: 'Matriz de planillaje', detail: 'Libro Excel habilitado para macros', accept: '.xlsm', icon: 'mdi-file-excel-outline' },
  { field: 'consolidada_file', title: 'Planilla consolidada', detail: 'PDF firmado por responsables médico y financiero', accept: '.pdf,application/pdf', icon: 'mdi-file-pdf-box' },
  { field: 'oficio_file', title: 'Oficio de pago', detail: 'PDF firmado por representante legal', accept: '.pdf,application/pdf', icon: 'mdi-file-document-outline' },
]
const ingestFiles = ref({ zip_file: null, matriz_file: null, consolidada_file: null, oficio_file: null })
const zipReplacementInput = ref(null)
const zipReplacementSending = ref(false)
const ingestSending = ref(false)
const completionSending = ref(false)
const ingestProcessing = ref(false)
const existingIngestJobId = ref('')
const ingestProgress = ref(0)
const ingestError = ref('')
const ingestResult = ref(null)
const ingestPreview = ref(null)
const ingestPreviewLoading = ref(false)
const ingestPreviewError = ref('')
const ingestPreviewNotice = ref('')
const ingestMappingSelections = ref({})
const ingestMappingSaving = ref('')
const ingestPreviewPage = ref(1)
const ingestPreviewPageSize = 10
const savedWorkspaces = ref([])
const selectedSavedWorkspace = ref('')
const savedWorkspacesLoading = ref(false)
const savedWorkspacesError = ref('')
const showWorkspaceDetails = ref(false)
const savedWorkspaceSearch = ref('')
const savedWorkspaceStatusFilter = ref('ALL')
const savedWorkspaceTypeFilter = ref('ALL')
const savedWorkspaceServiceFilter = ref('ALL')
const savedWorkspaceYearFilter = ref('ALL')
const selectedCoverageWorkspace = ref('')
const coveragePlanillas = ref([])
const coverageSelectedIds = ref([])
const coverageLoading = ref(false)
const coverageGenerating = ref(false)
const coverageProgress = ref('')
const coverageSelectableRows = computed(() => coveragePlanillas.value.filter(row => row.pdi_cobertura !== 'S'))
const coverageAllSelected = computed(() => coverageSelectableRows.value.length > 0 && coverageSelectableRows.value.every(row => coverageSelectedIds.value.includes(row.pdi_id)))
const coverageManualUploadingId = ref(null)
const coverageError = ref('')
const coverageNotice = ref('')
const ingestMode = ref('new')
const ingestModeTouched = ref(false)
const openingSavedWorkspace = ref(false)
const patientDocumentsDialog = ref(false)
const patientDocumentsError = ref('')
const selectedZipWorkspace = ref('')
const zipDownloadType = ref('RECEPCION')
const zipDownloading = ref(false)
const zipDownloadError = ref('')
const zipDownloadNotice = ref('')
const zipDownloadArchiveState = ref('')
const objectionSourceWorkspace = ref('')
const objectionPreview = ref(null)
const objectionSelectedTramites = ref([])
const objectionAddPreview = ref(null)
const objectionAddSelectedTramites = ref([])
const objectionHeaderFiles = ref({})
const objectionCurrentWorkspace = ref(null)
const objectionSelectedWorkspace = ref('')
const objectionBusy = ref(false)
const objectionError = ref('')
const objectionConflictMessage = ref('')
const objectionConflictSpaces = ref([])
const objectionNotice = ref('')
const objectionUploadFiles = ref({})
const objectionPostures = ref({})
const objectionPackagePDFs = ref([])
const objectionPDFLoading = ref(false)
const objectionPDFError = ref('')
const objectionPDFSavingPath = ref('')
const objectionPostureSaving = ref('')
const objectionAnnexTypeByPatient = ref({})
const objectionAnnexNameByPatient = ref({})
const objectionPatientSearch = ref('')
const objectionPatientPage = ref(1)
const objectionPatientPageSize = 10
const objectionAnnexTypes = [
  { title: 'Ficha técnica', value: 'FICHA_TECNICA' },
  { title: 'Factura de dispositivo', value: 'FACTURA_DISPOSITIVO' },
  { title: 'Facturas de respaldo', value: 'FACTURAS_RESPALDO' },
  { title: 'Factura de compra', value: 'FACTURA_COMPRA' },
  { title: 'Protocolos médicos', value: 'PROTOCOLOS_MEDICOS' },
  { title: 'Protocolo detallado', value: 'PROTOCOLO_DETALLADO' },
  { title: 'Examen adicional', value: 'EXAMEN_ADICIONAL' },
  { title: 'Otro justificativo', value: 'OTRO' },
]
const objectionSelectionCandidates = computed(() => objectionPreview.value?.candidatos || [])
const objectionComboItems = computed(() => objectionSelectionCandidates.value.map(row => ({
  title: `${row.paciente} · Trámite ${row.pdi_tramite}${row.pdi_cedula ? ` · Cédula ${row.pdi_cedula}` : ''}`,
  value: row.pdi_tramite,
})))
const objectionSelectedCandidates = computed(() => objectionSelectionCandidates.value.filter(row => objectionSelectedTramites.value.includes(row.pdi_tramite)))
function sameObjectionPeriod(source, workspace) {
  if (!source || !workspace) return false
  const sourceMonth = Number(source.mes)
  const workspaceMonth = Number(workspace.mes)
  const monthsMatch = Number.isFinite(sourceMonth) && Number.isFinite(workspaceMonth)
    ? sourceMonth === workspaceMonth
    : String(source.mes || '').trim() === String(workspace.mes || '').trim()
  return String(source.tipo_servicio || '').trim().toLowerCase() === String(workspace.tipo_servicio || '').trim().toLowerCase()
    && monthsMatch
    && String(source.anio || '').trim() === String(workspace.anio || '').trim()
}
const selectedObjectionSource = computed(() => savedWorkspaces.value.find(workspace => workspace.job_id === objectionSourceWorkspace.value) || null)
const objectionSourcePeriodSpaces = computed(() => savedWorkspaces.value
  .filter(workspace => workspace.es_objeciones && sameObjectionPeriod(selectedObjectionSource.value, workspace))
  .sort((a, b) => String(a.received_at || '').localeCompare(String(b.received_at || '')) || String(a.job_id).localeCompare(String(b.job_id))))
const objectionPeriodSpaces = computed(() => {
  const byID = new Map(objectionSourcePeriodSpaces.value.map(workspace => [workspace.job_id, workspace]))
  for (const workspace of objectionConflictSpaces.value) {
    if (workspace?.job_id && sameObjectionPeriod(selectedObjectionSource.value, workspace)) byID.set(workspace.job_id, workspace)
  }
  return [...byID.values()]
})
const objectionChosenPeriodSpace = computed(() => objectionPeriodSpaces.value.find(workspace => workspace.job_id === objectionSelectedWorkspace.value) || null)
const objectionCanCreate = computed(() => objectionSelectedTramites.value.length > 0 && Boolean(objectionSourceWorkspace.value && objectionPreview.value)
  && (objectionPeriodSpaces.value.length < 2 || Boolean(objectionChosenPeriodSpace.value)))
const objectionCreateLabel = computed(() => objectionPeriodSpaces.value.length > 1
  ? 'Agregar al espacio elegido'
  : objectionSourcePeriodSpaces.value.length === 1
    ? 'Continuar en el espacio existente'
    : 'Crear espacio separado')
const objectionAddCandidateItems = computed(() => {
  const selected = new Set((objectionCurrentWorkspace.value?.objeciones || []).map(row => row.pdi_tramite))
  return (objectionAddPreview.value?.candidatos || [])
    .filter(row => !selected.has(row.pdi_tramite))
    .map(row => ({ title: row.paciente + ' · Trámite ' + row.pdi_tramite + (row.pdi_cedula ? ' · Cédula ' + row.pdi_cedula : ''), value: row.pdi_tramite }))
})
const objectionHeaderOptions = computed(() => objectionCurrentWorkspace.value?.cabeceras_objeciones || [])
const objectionUniquePatients = computed(() => {
  const patients = new Map()
  for (const row of objectionCurrentWorkspace.value?.objeciones || objectionPreview.value?.objeciones || []) {
    const key = row.pdi_tramite
    if (!patients.has(key)) patients.set(key, { tramite: key, patient: row.paciente, folder: row.carpeta_paciente, rows: [] })
    patients.get(key).rows.push(row)
  }
  return [...patients.values()]
})
const objectionPackagePDFsByFolder = computed(() => {
  const grouped = new Map()
  const root = '4. EXPEDIENTES/'
  for (const document of objectionPackagePDFs.value) {
    if (!document.path.startsWith(root)) continue
    const folder = document.path.slice(root.length).split('/')[0]
    if (!folder) continue
    if (!grouped.has(folder)) grouped.set(folder, [])
    grouped.get(folder).push(document)
  }
  return grouped
})
const filteredObjectionPatients = computed(() => {
  const terms = normalizeWorkspaceSearch(objectionPatientSearch.value).split(/\s+/).filter(Boolean)
  if (!terms.length) return objectionUniquePatients.value
  return objectionUniquePatients.value.filter(patient => {
    const searchText = normalizeWorkspaceSearch([
      patient.patient,
      patient.tramite,
      patient.rows.map(row => row.pdi_cedula || '').join(' '),
    ].join(' '))
    return terms.every(term => searchText.includes(term))
  })
})
const objectionPatientPageCount = computed(() => Math.max(1, Math.ceil(filteredObjectionPatients.value.length / objectionPatientPageSize)))
const objectionPatientsPage = computed(() => filteredObjectionPatients.value.slice(
  (objectionPatientPage.value - 1) * objectionPatientPageSize,
  objectionPatientPage.value * objectionPatientPageSize,
))
const objectionRowsPage = computed(() => objectionPatientsPage.value.flatMap(patient => patient.rows))
const objectionPatientRangeStart = computed(() => filteredObjectionPatients.value.length ? (objectionPatientPage.value - 1) * objectionPatientPageSize + 1 : 0)
const objectionPatientRangeEnd = computed(() => Math.min(objectionPatientPage.value * objectionPatientPageSize, filteredObjectionPatients.value.length))
watch(objectionPatientSearch, () => { objectionPatientPage.value = 1 })
watch(() => objectionUniquePatients.value.length, () => { objectionPatientPage.value = 1 })
watch(() => objectionCurrentWorkspace.value?.job_id, () => {
  objectionPatientSearch.value = ''
  objectionPatientPage.value = 1
})
function objectionPatientPDFs(patient) {
  return objectionPackagePDFsByFolder.value.get(patient.folder) || []
}
const workspacePDFs = ref([])
const workspacePDFRevision = ref(0)
const workspacePatients = ref([])
const selectedWorkspacePDF = ref('')
const workspaceRename = ref('')
const workspacePatient = ref('')
const workspaceUploadFiles = ref([])
const workspaceBusy = ref(false)
const workspaceMutationBusy = ref(false)
const workspaceSending = ref(false)
const workspaceNotice = ref('')
const workspacePDFInput = ref(null)
const replacePDFInput = ref(null)
const replacePDFFile = ref(null)
const replacePDFDialog = ref(false)
const replacePDFError = ref('')
const replacePDFTarget = ref('')
const deletePDFTarget = ref('')
const deletePDFDialog = ref(false)
const mergePDFDialog = ref(false)
const mergePDFPaths = ref([])
const mergePreviewPath = ref('')
const mergeDraggingIndex = ref(-1)
const mergeDropIndex = ref(-1)
const mergeSending = ref(false)
const mergeAllConfirmDialog = ref(false)
const mergeAllSending = ref(false)
const mergeAllProgress = ref({ done: 0, total: 0 })
watch(() => ingestResult.value?.job_id, (jobId, previousJobId) => {
  patientDocumentsDialog.value = false
  if (jobId !== previousJobId) workspaceUploadFiles.value = []
})
watch(patientDocumentsDialog, (open, wasOpen) => {
  if (!open && wasOpen) {
    workspaceDocumentsAbortController?.abort()
    mergePDFDialog.value = false
    mergeAllConfirmDialog.value = false
    deletePDFDialog.value = false
    replacePDFDialog.value = false
  }
  const jobId = ingestResult.value?.job_id
  if (!open && wasOpen && ingestResult.value?.es_objeciones && jobId) void loadObjectionPDFSelection(jobId)
})
const deleteWorkspaceDialog = ref(false)
const workspaceDeleteTargetID = ref('')
const workspaceDeleteTargetName = ref('')
const workspaceDeleteInput = ref('')
const workspaceDeleteError = ref('')
const workspaceDeleting = ref(false)
const workspaceDeleteIsObjections = computed(() => savedWorkspaces.value.some(workspace => workspace.job_id === workspaceDeleteTargetID.value && workspace.es_objeciones))
let workspaceDocumentsAbortController = null
const selectedWorkspaceDocument = computed(() => workspacePDFs.value.find(document => document.path === selectedWorkspacePDF.value) || null)
const workspacePatientPDFs = computed(() => {
  const prefix = workspacePatient.value ? `4. EXPEDIENTES/${workspacePatient.value}/` : ''
  return prefix ? workspacePDFs.value.filter(document => document.path.startsWith(prefix)) : []
})
const workspaceFusionGroups = computed(() => {
  const allowed = new Set(mspPdfCodes.map(option => option.value.toLowerCase()))
  const groups = new Map()
  for (const document of workspacePDFs.value) {
    if (!document.path.startsWith('4. EXPEDIENTES/')) continue
    let code = document.name
    if (!allowed.has(code.toLowerCase())) {
      const match = code.match(/^(.*)_([1-9]\d*)\.pdf$/i)
      if (!match) continue
      code = `${match[1]}.pdf`
      if (!allowed.has(code.toLowerCase())) continue
    }
    const key = `${document.path.slice(0, document.path.lastIndexOf('/') + 1)}${code.toLowerCase()}`
    const group = groups.get(key) || { code, paths: [] }
    group.paths.push(document.path)
    groups.set(key, group)
  }
  return [...groups.values()].filter(group => group.paths.length > 1).map(group => {
    const canonical = group.paths.find(path => path.toLowerCase().endsWith(`/${group.code.toLowerCase()}`))
    group.paths.sort((a, b) => a === canonical ? -1 : b === canonical ? 1 : a.localeCompare(b))
    return group
  })
})
const workspacePatientFusionGroups = computed(() => {
  const prefix = workspacePatient.value ? `4. EXPEDIENTES/${workspacePatient.value}/` : ''
  return prefix ? workspaceFusionGroups.value.filter(group => group.paths.some(path => path.startsWith(prefix))) : []
})
const workspaceFusionQueue = computed(() => workspaceFusionGroups.value.map(group => {
  const patient = group.paths[0]?.split('/')[1] || ''
  return { ...group, patient, value: group.paths[0], title: `${patient} · ${group.code} · ${group.paths.length} PDFs` }
}))
const pendingFusionSelection = ref('')
const selectedPendingFusionGroup = computed(() => workspaceFusionQueue.value.find(group => group.value === pendingFusionSelection.value) || null)
const queuedWorkspaceDuplicates = computed(() => {
  const seen = new Set(workspacePatientPDFs.value.map(document => standardCodeForFilename(document.name).toLowerCase()).filter(Boolean))
  let duplicates = 0
  for (const item of workspaceUploadFiles.value) {
    const code = item.code?.toLowerCase()
    if (!code) continue
    if (seen.has(code)) duplicates++
    seen.add(code)
  }
  return duplicates
})
const mspPDFOptions = mspPdfCodes
function standardCodeForFilename(filename = '') {
  const canonical = filename.replace(/_\d+(?=\.pdf$)/i, '').toLowerCase()
  return mspPDFOptions.find(option => option.value.toLowerCase() === canonical)?.value || ''
}
function isPendingWorkspaceDocument(document) {
  return /^PENDIENTE_tmp_/i.test(document?.name || '')
}
const workspacePDFURL = computed(() => selectedWorkspacePDF.value && ingestResult.value?.job_id
  ? `/api/v1/expedientes/documentos/archivo/${encodeURIComponent(ingestResult.value.job_id)}?path=${encodeURIComponent(selectedWorkspacePDF.value)}&v=${workspacePDFRevision.value}`
  : '')
const mergePreviewURL = computed(() => mergePreviewPath.value && ingestResult.value?.job_id
  ? `/api/v1/expedientes/documentos/archivo/${encodeURIComponent(ingestResult.value.job_id)}?path=${encodeURIComponent(mergePreviewPath.value)}&v=${workspacePDFRevision.value}`
  : '')
let dragDepth = 0
const spaces = computed(() => ['Trabajo','Personal','Mi espacio',...customSpaces.value])
async function loadDocuments() {
  try { const db = await dbPromise; docs.value = await db.getAll('files') } catch { snackbar.value = 'No se pudo abrir el almacenamiento local.' }
  loading.value = false
}
async function checkSession() {
  try {
    const response = await fetch('/api/session', { credentials: 'same-origin' })
    if (!response.ok) { canDeleteWorkspaces.value = false; authStatus.value = 'anonymous'; return }
    const data = await response.json()
    currentUser.value = data.username
    canDeleteWorkspaces.value = data.can_delete_workspaces === true
    authStatus.value = 'authenticated'
    await loadDocuments()
    await validateCachedIngest()
    await loadSavedWorkspaces()
  } catch {
    canDeleteWorkspaces.value = false
    authStatus.value = 'anonymous'
    authError.value = 'No se pudo conectar con el servicio de inicio de sesión.'
  }
}
async function signIn() {
  authError.value = ''
  canDeleteWorkspaces.value = false
  signingIn.value = true
  try {
    const response = await fetch('/api/login', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: oracleUser.value.trim(), password: oraclePassword.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo iniciar sesión.')
    currentUser.value = data.username
    canDeleteWorkspaces.value = data.can_delete_workspaces === true
    oraclePassword.value = ''
    authStatus.value = 'authenticated'
    await loadDocuments()
    await validateCachedIngest()
    await loadSavedWorkspaces()
  } catch (error) {
    canDeleteWorkspaces.value = false
    authError.value = error.message || 'No se pudo conectar con Oracle.'
  } finally {
    signingIn.value = false
  }
}
async function signOut() {
  try { await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' }) } catch { /* La sesión local se cierra aunque falle la petición. */ }
  authStatus.value = 'anonymous'
  currentUser.value = ''
  canDeleteWorkspaces.value = false
  oraclePassword.value = ''
}
async function openPlanilla() {
  activePage.value = 'planilla'
  await loadPlanilla(1)
}
async function openCoverageDownloads() {
  activePage.value = 'coberturas'
  coverageError.value = ''
  coverageNotice.value = ''
  await loadSavedWorkspaces()
  const eligible = coverageWorkspaceCandidates.value
  if (!eligible.some(workspace => workspace.job_id === selectedCoverageWorkspace.value)) {
    selectedCoverageWorkspace.value = eligible[0]?.job_id || ''
  }
  if (selectedCoverageWorkspace.value) await loadCoveragePlanillas(selectedCoverageWorkspace.value)
  else coveragePlanillas.value = []
}
async function loadCoveragePlanillas(jobId = selectedCoverageWorkspace.value) {
  selectedCoverageWorkspace.value = jobId || ''
  coveragePlanillas.value = []
  coverageSelectedIds.value = []
  coverageError.value = ''
  coverageNotice.value = ''
  if (!jobId) return
  coverageLoading.value = true
  try {
    const response = await fetch(`/api/v1/coberturas/planillas/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (response.status === 401) { await signOut(); return }
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar las planillas de cobertura.')
    coveragePlanillas.value = data.planillas || []
    coverageNotice.value = `${coverageServiceLabel(data.tipo_servicio)} · ${coverageMonthLabel(data.mes)} ${data.anio}`
  } catch (error) {
    coverageError.value = error.message || 'No se pudieron cargar las planillas de cobertura.'
  } finally { coverageLoading.value = false }
}
function coverageServiceLabel(service) {
  return ingestServices.find(item => item.value === service)?.title || service || 'Servicio'
}
function coverageMonthLabel(month) {
  return ingestMonths.find(item => item.value === String(month).padStart(2, '0'))?.title || month || ''
}
function toggleCoveragePlanilla(row) {
  if (!row || (row.pdi_cobertura === 'S' && !row.hoja_generada)) return
  const selected = new Set(coverageSelectedIds.value)
  if (selected.has(row.pdi_id)) selected.delete(row.pdi_id)
  else selected.add(row.pdi_id)
  coverageSelectedIds.value = [...selected]
}
function toggleAllCoveragePlanillas() {
  const selectable = coverageSelectableRows.value.map(row => row.pdi_id)
  coverageSelectedIds.value = coverageAllSelected.value
    ? coverageSelectedIds.value.filter(id => !selectable.includes(id))
    : [...new Set([...coverageSelectedIds.value, ...selectable])]
}
async function generateCoverageSheets() {
  if (!selectedCoverageWorkspace.value || !coverageSelectedIds.value.length || coverageGenerating.value) return
  const selectedIds = [...coverageSelectedIds.value]
  const jobId = selectedCoverageWorkspace.value
  coverageGenerating.value = true
  coverageError.value = ''
  coverageNotice.value = ''
  try {
    const processed = []
    const failures = []
    for (let offset = 0; offset < selectedIds.length; offset += 10) {
      const batch = selectedIds.slice(offset, offset + 10)
      coverageProgress.value = `Procesando ${Math.min(offset + batch.length, selectedIds.length)} de ${selectedIds.length} planillas…`
      const response = await fetch(`/api/v1/coberturas/generar/${encodeURIComponent(jobId)}`, {
        method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pdi_ids: batch }),
      })
      if (response.status === 401) { await signOut(); return }
      const result = await response.json().catch(() => ({}))
      if (!response.ok) throw new Error(result.error || 'No se pudieron generar las hojas seleccionadas.')
      processed.push(...(result.generadas || []))
      failures.push(...(result.errores || []))
    }
    await loadCoveragePlanillas(jobId)
    coverageSelectedIds.value = selectedIds.filter(id => coveragePlanillas.value.some(row => row.pdi_id === id))
    const manualCount = failures.filter(item => item.manual).length
    const dataErrorCount = failures.length - manualCount
    coverageNotice.value = `${processed.length} planillas revisadas; las hojas disponibles quedaron guardadas en sus expedientes.${manualCount ? ` ${manualCount} requieren descarga manual en el portal MSP.` : ''}${dataErrorCount ? ` ${dataErrorCount} requieren corregir datos o revisar el expediente.` : ''}`
  } catch (error) {
    coverageError.value = error.message || 'No se pudieron generar las hojas seleccionadas.'
  } finally { coverageProgress.value = ''; coverageGenerating.value = false }
}
async function downloadSelectedCoverageSheets() {
  if (!selectedCoverageWorkspace.value || !coverageSelectedIds.value.length || coverageGenerating.value) return
  coverageGenerating.value = true
  coverageError.value = ''
  try {
    const response = await fetch(`/api/v1/coberturas/descargar/${encodeURIComponent(selectedCoverageWorkspace.value)}?pdi_ids=${coverageSelectedIds.value.join(',')}`, { credentials: 'same-origin' })
    if (response.status === 401) { await signOut(); return }
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || 'No se pudo preparar el ZIP de las hojas seleccionadas.')
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const disposition = response.headers.get('Content-Disposition') || ''
    const filename = disposition.match(/filename="?([^";]+)"?/i)?.[1] || 'coberturas.zip'
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    coverageError.value = error.message || 'No se pudo descargar el ZIP.'
  } finally { coverageGenerating.value = false }
}
async function uploadManualCoverageSheets(row, event) {
  const files = [...(event.target.files || [])]
  event.target.value = ''
  if (!files.length || !selectedCoverageWorkspace.value) return
  coverageManualUploadingId.value = row.pdi_id
  coverageError.value = ''
  try {
    const form = new FormData()
    form.append('pdi_id', String(row.pdi_id))
    files.forEach(file => form.append('pdf_files', file, file.name))
    const response = await fetch(`/api/v1/coberturas/manual/${encodeURIComponent(selectedCoverageWorkspace.value)}`, { method: 'POST', credentials: 'same-origin', body: form })
    if (response.status === 401) { await signOut(); return }
    const result = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(result.error || 'No se pudieron adjuntar las hojas manuales.')
    await loadCoveragePlanillas(selectedCoverageWorkspace.value)
    coverageNotice.value = `Se adjuntaron ${result.adjuntadas} hojas descargadas del portal MSP y se actualizaron en Oracle.`
  } catch (error) {
    coverageError.value = error.message || 'No se pudieron adjuntar las hojas manuales.'
  } finally { coverageManualUploadingId.value = null }
}
async function loadPlanilla(page = planillaPage.value) {
  planillaPage.value = page
  planillaLoading.value = true
  planillaError.value = ''
  try {
    const response = await fetch(`/api/planilla-digital?page=${page}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (response.status === 401) { await signOut(); return }
    if (!response.ok) throw new Error(data.error || 'No se pudo cargar planilla_digital.')
    planillaData.value = data
    planillaPage.value = data.page
  } catch (error) {
    planillaError.value = error.message || 'No se pudo cargar planilla_digital.'
  } finally {
    planillaLoading.value = false
  }
}
function formatPlanillaValue(value) {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
const ingestValid = computed(() => {
  const expected = { zip_file: '.zip', matriz_file: '.xlsm', consolidada_file: '.pdf', oficio_file: '.pdf' }
  return /^\d{2}$/.test(ingestMonth.value) && Number(ingestMonth.value) >= 1 && Number(ingestMonth.value) <= 12
    && /^20\d{2}$/.test(ingestYear.value) && Boolean(ingestService.value)
    && ingestFiles.value.zip_file?.size > 0 && ingestFiles.value.zip_file.name.toLowerCase().endsWith(expected.zip_file)
    && Object.entries(expected).filter(([key]) => key !== 'zip_file').every(([key, extension]) => !ingestFiles.value[key] || (ingestFiles.value[key].size > 0 && ingestFiles.value[key].name.toLowerCase().endsWith(extension)))
})
function selectIngestFile(field, event) {
  const file = event.target.files?.[0] || null
  const expected = { zip_file: '.zip', matriz_file: '.xlsm', consolidada_file: '.pdf', oficio_file: '.pdf' }[field]
  ingestError.value = ''
  if (file && (file.size === 0 || !file.name.toLowerCase().endsWith(expected))) {
    ingestFiles.value[field] = null
    ingestError.value = `Selecciona un archivo ${expected} que no esté vacío.`
  } else {
    ingestFiles.value[field] = file
  }
  event.target.value = ''
}
async function loadWorkspaceDocuments(jobId, preferPath = '') {
  if (!jobId) return
  workspaceDocumentsAbortController?.abort()
  const controller = new AbortController()
  workspaceDocumentsAbortController = controller
  let timedOut = false
  const timeoutId = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, 30000)
  workspaceBusy.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/documentos/${encodeURIComponent(jobId)}`, { credentials: 'same-origin', signal: controller.signal })
    if (controller.signal.aborted || workspaceDocumentsAbortController !== controller) return
    const data = await response.json().catch(() => ({}))
    if (controller.signal.aborted || workspaceDocumentsAbortController !== controller) return
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar los PDFs del expediente.')
    if (ingestResult.value?.job_id !== jobId) return
    workspacePDFs.value = data.documents || []
    workspacePDFRevision.value++
    workspacePatients.value = data.patients || []
    workspaceNotice.value = ''
    workspacePatient.value = workspacePatients.value.includes(workspacePatient.value) ? workspacePatient.value : (workspacePatients.value[0] || '')
    const patientPrefix = workspacePatient.value ? `4. EXPEDIENTES/${workspacePatient.value}/` : ''
    const patientDocuments = patientPrefix ? workspacePDFs.value.filter(file => file.path.startsWith(patientPrefix)) : []
    selectedWorkspacePDF.value = patientDocuments.some(file => file.path === preferPath)
      ? preferPath
      : patientDocuments.some(file => file.path === selectedWorkspacePDF.value) ? selectedWorkspacePDF.value : (patientDocuments[0]?.path || '')
    workspaceRename.value = standardCodeForFilename(selectedWorkspaceDocument.value?.name || '')
  } catch (error) {
    if (timedOut) workspaceNotice.value = 'La carga de PDFs tardó demasiado. Puedes cerrar el visor y volver a intentarlo.'
    else if (!controller.signal.aborted) workspaceNotice.value = error.message || 'No se pudieron cargar los PDFs del expediente.'
  } finally {
    window.clearTimeout(timeoutId)
    if (workspaceDocumentsAbortController === controller) {
      workspaceDocumentsAbortController = null
      workspaceBusy.value = false
    }
  }
}
async function fetchWorkspaceJSON(url, options, timeoutMs, operation) {
  const controller = new AbortController()
  let timedOut = false
  const timeoutId = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, timeoutMs)
  try {
    const response = await fetch(url, { ...options, signal: controller.signal })
    const data = await response.json().catch(error => {
      if (timedOut) throw error
      return {}
    })
    return { response, data }
  } catch (error) {
    if (timedOut) {
      throw new Error(`La solicitud de ${operation} superó ${Math.round(timeoutMs / 1000)} segundos. El servidor puede haber completado el cambio; revisa el expediente antes de repetirla.`)
    }
    throw error
  } finally {
    window.clearTimeout(timeoutId)
  }
}
function selectWorkspacePDF(path) {
  selectedWorkspacePDF.value = path
  workspaceRename.value = standardCodeForFilename(selectedWorkspaceDocument.value?.name || '')
}
function selectWorkspacePatient(patient) {
  workspacePatient.value = patient
  const prefix = `4. EXPEDIENTES/${patient}/`
  if (!selectedWorkspacePDF.value.startsWith(prefix)) selectWorkspacePDF(workspacePDFs.value.find(file => file.path.startsWith(prefix))?.path || '')
}
async function renameWorkspacePDF() {
  const jobId = ingestResult.value?.job_id
  const document = selectedWorkspaceDocument.value
  if (!jobId || !document || !workspaceRename.value || workspaceBusy.value) return
  workspaceNotice.value = ''; workspaceBusy.value = true; workspaceMutationBusy.value = true
  try {
    const { response, data } = await fetchWorkspaceJSON(`/api/v1/expedientes/documentos/renombrar/${encodeURIComponent(jobId)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: document.path, name: workspaceRename.value }),
    }, 60000, 'cambiar el nombre del PDF')
    if (!response.ok) throw new Error(data.error || 'No se pudo cambiar el nombre del PDF.')
    const path = data.path
    if (ingestResult.value?.job_id === jobId && patientDocumentsDialog.value) {
      await loadWorkspaceDocuments(jobId, path)
      workspaceNotice.value = `Se guardó como ${data.name}.`
    } else snackbar.value = `Se guardó como ${data.name}. Al volver a abrir, se actualizará la lista de PDFs.`
  } catch (error) {
    const message = error.message || 'No se pudo cambiar el nombre del PDF.'
    if (patientDocumentsDialog.value && ingestResult.value?.job_id === jobId) workspaceNotice.value = message
    else snackbar.value = message
  }
  finally { workspaceBusy.value = false; workspaceMutationBusy.value = false }
}
function openWorkspaceFusion(group) {
  if (!group || mergeAllSending.value) return
  mergePDFPaths.value = [...group.paths]
  mergePreviewPath.value = group.paths[0] || ''
  mergePDFDialog.value = true
}
function reviewPendingFusion() {
  const group = workspaceFusionQueue.value.find(item => item.value === pendingFusionSelection.value)
  if (!group) return
  workspacePatient.value = group.patient
  selectedWorkspacePDF.value = group.paths[0]
  workspaceRename.value = standardCodeForFilename(group.code)
  openWorkspaceFusion(group)
}
function requestMergeAllPending() {
  if (!workspaceFusionQueue.value.length || mergeSending.value || mergeAllSending.value) return
  mergeAllProgress.value = { done: 0, total: workspaceFusionQueue.value.length }
  mergeAllConfirmDialog.value = true
}
async function confirmMergeAllPending() {
  const jobId = ingestResult.value?.job_id
  const groups = workspaceFusionQueue.value.map(group => ({ patient: group.patient, code: group.code, paths: [...group.paths] }))
  if (!jobId || !groups.length || mergeAllSending.value || mergeSending.value) return
  mergeAllSending.value = true
  mergeAllConfirmDialog.value = false
  mergeAllProgress.value = { done: 0, total: groups.length }
  let merged = 0
  let failure = ''
  for (const group of groups) {
    try {
      const { response, data } = await fetchWorkspaceJSON(`/api/v1/expedientes/documentos/fusionar/${encodeURIComponent(jobId)}`, {
        method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rutas: group.paths }),
      }, 60000, 'fusionar PDFs')
      if (!response.ok) throw new Error(data.error || `Falló la fusión de ${group.patient} · ${group.code}.`)
      merged++
      mergeAllProgress.value = { done: merged, total: groups.length }
    } catch (error) {
      failure = error.message || `Falló la fusión de ${group.patient} · ${group.code}.`
      break
    }
  }
  mergeAllSending.value = false
  if (patientDocumentsDialog.value && ingestResult.value?.job_id === jobId) {
    await loadWorkspaceDocuments(jobId, selectedWorkspacePDF.value)
    await loadSavedWorkspaces()
  }
  if (failure) workspaceNotice.value = merged
    ? `Se detuvo después de fusionar ${merged} de ${groups.length} grupos. Los demás siguen pendientes. ${failure}`
    : `No se completó la fusión general. Los grupos siguen pendientes. ${failure}`
  else workspaceNotice.value = `Se fusionaron ${merged} grupos pendientes en el orden actual de cada lista. Las fuentes originales se conservaron.`
  if (!patientDocumentsDialog.value || ingestResult.value?.job_id !== jobId) snackbar.value = workspaceNotice.value
}
function moveWorkspaceFusionPDF(index, offset) {
  const target = index + offset
  if (target < 0 || target >= mergePDFPaths.value.length) return
  const [path] = mergePDFPaths.value.splice(index, 1)
  mergePDFPaths.value.splice(target, 0, path)
}
function startWorkspaceFusionDrag(event, index) {
  mergeDraggingIndex.value = index
  mergeDropIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }
}
function dragOverWorkspaceFusion(event, index) {
  event.preventDefault()
  mergeDropIndex.value = index
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
}
function dropWorkspaceFusion(event, index) {
  event.preventDefault()
  const from = mergeDraggingIndex.value
  if (from >= 0 && from !== index) {
    const ordered = [...mergePDFPaths.value]
    const [path] = ordered.splice(from, 1)
    ordered.splice(index, 0, path)
    mergePDFPaths.value = ordered
  }
  finishWorkspaceFusionDrag()
}
function finishWorkspaceFusionDrag() {
  mergeDraggingIndex.value = -1
  mergeDropIndex.value = -1
}
async function confirmWorkspaceFusion() {
  const jobId = ingestResult.value?.job_id
  if (!jobId || mergePDFPaths.value.length < 2 || mergeSending.value || mergeAllSending.value) return
  mergeSending.value = true
  workspaceNotice.value = ''
  try {
    const { response, data } = await fetchWorkspaceJSON(`/api/v1/expedientes/documentos/fusionar/${encodeURIComponent(jobId)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rutas: mergePDFPaths.value }),
    }, 60000, 'fusionar PDFs')
    if (!response.ok) throw new Error(data.error || 'No se pudieron fusionar los PDFs.')
    mergePDFDialog.value = false
    mergePDFPaths.value = []
    mergePreviewPath.value = ''
    pendingFusionSelection.value = ''
    const message = `${data.merged_count} PDFs fusionados en ${data.fused.split('/').at(-1)}. Las fuentes originales se conservaron.`
    if (ingestResult.value?.job_id === jobId && patientDocumentsDialog.value) {
      await loadWorkspaceDocuments(jobId, data.fused)
      workspaceNotice.value = message
      await loadSavedWorkspaces()
    } else snackbar.value = `${message} Al volver a abrir, se actualizará la lista de PDFs.`
  } catch (error) {
    const message = error.message || 'No se pudieron fusionar los PDFs.'
    if (patientDocumentsDialog.value && ingestResult.value?.job_id === jobId) workspaceNotice.value = message
    else snackbar.value = message
  }
  finally { mergeSending.value = false }
}
function selectWorkspacePDFs(event) {
  const files = [...(event.target.files || [])]
  workspaceNotice.value = ''
  const invalid = files.find(file => !file.name.toLowerCase().endsWith('.pdf') || file.size === 0)
  if (invalid) {
    workspaceUploadFiles.value = []
    workspaceNotice.value = 'Selecciona solamente archivos PDF que no estén vacíos.'
  } else workspaceUploadFiles.value = files.map((file, index) => ({
    id: `${file.name}-${file.size}-${file.lastModified}-${index}`,
    file,
    code: standardCodeForFilename(file.name),
  }))
  event.target.value = ''
}
function removeQueuedWorkspacePDF(index) {
  workspaceUploadFiles.value.splice(index, 1)
}
function selectReplacementPDF(event) {
  replacePDFError.value = ''
  const file = event.target.files?.[0] || null
  replacePDFFile.value = file && file.size > 0 && file.name.toLowerCase().endsWith('.pdf') ? file : null
  workspaceNotice.value = file && !replacePDFFile.value ? 'Selecciona un archivo PDF que no esté vacío.' : ''
  event.target.value = ''
  if (replacePDFFile.value && selectedWorkspaceDocument.value) {
    replacePDFTarget.value = selectedWorkspaceDocument.value.path
    replacePDFDialog.value = true
  }
}
async function addWorkspacePDFs() {
  const jobId = ingestResult.value?.job_id
  const patient = workspacePatient.value
  if (!jobId || !workspaceUploadFiles.value.length || !patient || workspaceSending.value) return
  workspaceNotice.value = ''; workspaceSending.value = true; ingestProgress.value = 0
  const form = new FormData()
  form.append('paciente', patient)
  for (const item of workspaceUploadFiles.value) {
    form.append('pdf_files', item.file)
    form.append('pdf_codes', item.code)
  }
  try {
    const data = await new Promise((resolve, reject) => {
      const request = new XMLHttpRequest()
      request.open('POST', `/api/v1/expedientes/documentos/${encodeURIComponent(jobId)}`)
      request.timeout = 120000
      request.withCredentials = true
      request.upload.onprogress = event => { if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100) }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch { /* La API debe responder JSON. */ }
        if (request.status === 200) resolve(response)
        else reject(new Error(response.error || `No se pudieron añadir los PDFs (HTTP ${request.status}).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante la carga.'))
      request.ontimeout = () => reject(new Error('La carga superó 120 segundos. El servidor puede haber recibido los PDFs; revisa la carpeta antes de volver a subirlos.'))
      request.send(form)
    })
    if (ingestResult.value?.job_id !== jobId) {
      snackbar.value = 'Terminó la carga del período anterior. Abre ese período para revisar sus PDFs.'
      return
    }
    workspaceUploadFiles.value = []
    workspacePDFs.value = data.documents || []
    workspacePDFRevision.value++
    workspacePatients.value = data.patients || []
    const replacedCount = data.replaced_count || 0
    const addedCount = data.added_count ?? (data.added?.length || 0)
    const message = replacedCount
      ? `${addedCount} PDF${addedCount === 1 ? '' : 's'} añadido${addedCount === 1 ? '' : 's'} y ${replacedCount} reemplazado${replacedCount === 1 ? '' : 's'}. Las versiones anteriores se conservan como fuentes.`
      : `${addedCount} PDF${addedCount === 1 ? '' : 's'} añadido${addedCount === 1 ? '' : 's'} al expediente.`
    if (patientDocumentsDialog.value) workspaceNotice.value = message
    else snackbar.value = `${message} Al volver a abrir, se actualizará la lista de PDFs.`
    selectedWorkspacePDF.value = data.added?.at(-1)?.relative_path || selectedWorkspacePDF.value
    workspaceRename.value = standardCodeForFilename(selectedWorkspaceDocument.value?.name || '')
    ingestProgress.value = 100
  } catch (error) {
    const message = error.message || 'No se pudieron añadir los PDFs.'
    if (patientDocumentsDialog.value && ingestResult.value?.job_id === jobId) workspaceNotice.value = message
    else snackbar.value = message
  }
  finally { workspaceSending.value = false }
}
async function confirmReplaceWorkspacePDF() {
  const jobId = ingestResult.value?.job_id
  const current = workspacePDFs.value.find(document => document.path === replacePDFTarget.value)
  if (!jobId || !current || !replacePDFFile.value || workspaceSending.value) return
  replacePDFError.value = ''
  workspaceNotice.value = ''; workspaceSending.value = true; ingestProgress.value = 0
  const form = new FormData()
  const patient = current.path.split('/')[1]
  form.append('paciente', patient)
  form.append('modo', 'reemplazar')
  form.append('ruta', current.path)
  form.append('pdf_files', replacePDFFile.value)
  try {
    const data = await new Promise((resolve, reject) => {
      const request = new XMLHttpRequest()
      request.open('POST', `/api/v1/expedientes/documentos/${encodeURIComponent(jobId)}`)
      request.timeout = 120000
      request.withCredentials = true
      request.upload.onprogress = event => { if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100) }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch { /* La API debe responder JSON. */ }
        if (request.status === 200) resolve(response)
        else reject(new Error(response.error || `No se pudo reemplazar el PDF (HTTP ${request.status}).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante el reemplazo.'))
      request.ontimeout = () => reject(new Error('El reemplazo superó 120 segundos. El servidor puede haber recibido el PDF; revisa la carpeta antes de volver a intentarlo.'))
      request.send(form)
    })
    if (ingestResult.value?.job_id !== jobId) {
      snackbar.value = 'Terminó el reemplazo del período anterior. Abre ese período para revisar sus PDFs.'
      return
    }
    workspacePDFs.value = data.documents || []
    workspacePDFRevision.value++
    replacePDFFile.value = null
    selectedWorkspacePDF.value = current.path
    workspaceNotice.value = 'PDF reemplazado. La versión fuente se conserva en el expediente.'
    ingestProgress.value = 100
    replacePDFDialog.value = false
  } catch (error) {
    const message = error.message || 'No se pudo reemplazar el PDF.'
    if (replacePDFDialog.value) replacePDFError.value = message
    else snackbar.value = message
  }
  finally { workspaceSending.value = false }
}
function requestDeleteWorkspacePDF(path) {
  deletePDFTarget.value = path
  deletePDFDialog.value = true
}
function requestDeleteWorkspace(jobId = ingestResult.value?.job_id) {
  if (!canDeleteWorkspaces.value || !jobId || workspaceDeleting.value) return
  workspaceDeleteTargetID.value = jobId
  workspaceDeleteTargetName.value = jobId === ingestResult.value?.job_id
    ? ingestResult.value.workspace?.split('/').at(-1) || jobId
    : savedWorkspaceItems.value.find(item => item.value === jobId)?.title || jobId
  workspaceDeleteInput.value = ''
  workspaceDeleteError.value = ''
  deleteWorkspaceDialog.value = true
}
function setWorkspaceDeleteInput(value) { workspaceDeleteInput.value = String(value || '').trim().toUpperCase() }
function workspaceDeleteConfirmationMatches() {
  return Boolean(workspaceDeleteTargetID.value) && workspaceDeleteInput.value.toUpperCase() === workspaceDeleteTargetID.value.toUpperCase()
}
function closeWorkspaceDeleteDialog() {
  deleteWorkspaceDialog.value = false
  workspaceDeleteInput.value = ''
  workspaceDeleteError.value = ''
  if (!workspaceDeleting.value) {
    workspaceDeleteTargetID.value = ''
    workspaceDeleteTargetName.value = ''
  }
}
async function deleteWholeWorkspace() {
  const jobId = workspaceDeleteTargetID.value
  const deletedFolder = workspaceDeleteTargetName.value
  if (!canDeleteWorkspaces.value || !workspaceDeleteConfirmationMatches() || workspaceDeleting.value) return
  workspaceDeleteError.value = ''
  workspaceDeleting.value = true
  try {
    const { response, data } = await fetchWorkspaceJSON(`/api/v1/expedientes/eliminar/${encodeURIComponent(jobId)}`, {
      method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ confirmacion: jobId }),
    }, 60000, 'eliminar el espacio')
    if (!response.ok) throw new Error(data.error || 'No se pudo eliminar el espacio completo.')
    if (ingestResult.value?.job_id === jobId) {
      ingestResult.value = null
      ingestPreview.value = null
      ingestPreviewError.value = ''
      workspacePDFs.value = []
      workspacePatients.value = []
      workspacePatient.value = ''
      selectedWorkspacePDF.value = ''
      workspaceUploadFiles.value = []
      localStorage.removeItem('folio-ingest-result')
    }
    if (selectedCoverageWorkspace.value === jobId) {
      selectedCoverageWorkspace.value = ''
      coveragePlanillas.value = []
      coverageSelectedIds.value = []
      coverageNotice.value = ''
      coverageError.value = ''
    }
    if (objectionSelectedWorkspace.value === jobId) {
      objectionSelectedWorkspace.value = ''
      objectionCurrentWorkspace.value = null
      objectionPostures.value = {}
      objectionUploadFiles.value = {}
      objectionAnnexTypeByPatient.value = {}
      objectionAnnexNameByPatient.value = {}
    }
    deleteWorkspaceDialog.value = false
    selectedSavedWorkspace.value = ''
    workspaceDeleting.value = false
    void loadSavedWorkspaces()
    workspaceDeleteTargetID.value = ''
    workspaceDeleteTargetName.value = ''
    workspaceDeleteInput.value = ''
    snackbar.value = data.message || `Se eliminó ${deletedFolder} y sus archivos.`
  } catch (error) {
    const message = error.message || 'No se pudo eliminar el espacio completo.'
    if (deleteWorkspaceDialog.value && workspaceDeleteTargetID.value === jobId) workspaceDeleteError.value = message
    else snackbar.value = message
  }
  finally { workspaceDeleting.value = false }
}
async function deleteWorkspacePDF() {
  const jobId = ingestResult.value?.job_id
  const path = deletePDFTarget.value
  if (!jobId || !path || workspaceBusy.value) return
  workspaceNotice.value = ''; workspaceBusy.value = true; workspaceMutationBusy.value = true
  try {
    const { response, data } = await fetchWorkspaceJSON(`/api/v1/expedientes/documentos/${encodeURIComponent(jobId)}`, {
      method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    }, 60000, 'quitar el PDF')
    if (!response.ok) throw new Error(data.error || 'No se pudo quitar el PDF.')
    if (ingestResult.value?.job_id !== jobId) {
      snackbar.value = 'Terminó la operación en el período anterior. Abre ese período para revisar sus PDFs.'
      return
    }
    workspacePDFs.value = data.documents || []
    const previousPath = path
    deletePDFTarget.value = ''
    if (selectedWorkspacePDF.value === previousPath) selectWorkspacePDF(workspacePatientPDFs.value[0]?.path || '')
    workspaceNotice.value = 'PDF quitado de la carpeta del paciente.'
    deletePDFDialog.value = false
  } catch (error) {
    const message = error.message || 'No se pudo quitar el PDF.'
    if (patientDocumentsDialog.value && ingestResult.value?.job_id === jobId) workspaceNotice.value = message
    else snackbar.value = message
  }
  finally { workspaceBusy.value = false; workspaceMutationBusy.value = false }
}
async function downloadWorkspaceZIP() {
  if (!selectedZipWorkspace.value || zipDownloading.value) return
  zipDownloadError.value = ''
  zipDownloadNotice.value = ''
  zipDownloadArchiveState.value = ''
  zipDownloading.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/descargar/${encodeURIComponent(selectedZipWorkspace.value)}`, { credentials: 'same-origin' })
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `No se pudo descargar el ZIP (HTTP ${response.status}).`)
    }
    const blob = await response.blob()
    zipDownloadArchiveState.value = response.headers.get('X-Folio-Archive-State') || (selectedZipWorkspaceReady.value ? 'READY' : 'INCOMPLETE')
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = response.headers.get('Content-Disposition')?.match(/filename="?([^";]+)"?/i)?.[1] || `${selectedZipWorkspace.value}.zip`
    link.click()
    URL.revokeObjectURL(url)
    const filename = link.download || 'expediente.zip'
    zipDownloadNotice.value = zipDownloadArchiveState.value === 'INCOMPLETE'
      ? `Se descargó ${filename} como avance incompleto. La descarga no cambia el estado ni los documentos guardados.`
      : `Se descargó ${filename}. El expediente está listo para entrega; la descarga no cambia su estado ni sus documentos.`
  } catch (error) { zipDownloadError.value = error.message || 'No se pudo descargar el ZIP.' }
  finally { zipDownloading.value = false }
}
async function submitIngest() {
  if (!ingestValid.value || ingestSending.value) return
  ingestError.value = ''
  ingestResult.value = null
  ingestPreview.value = null
  ingestPreviewError.value = ''
  ingestProgress.value = 0
  ingestSending.value = true
  const form = new FormData()
  for (const [field, file] of Object.entries(ingestFiles.value)) if (file) form.append(field, file)
  form.append('mes', ingestMonth.value)
  form.append('anio', ingestYear.value)
  form.append('tipo_servicio', ingestService.value)
  try {
    ingestResult.value = await new Promise((resolve, reject) => {
      const request = new XMLHttpRequest()
      request.open('POST', '/api/v1/ingesta/lote-dual')
      request.withCredentials = true
      request.upload.onprogress = event => {
        if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100)
      }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch {
          const detail = request.responseText.trim()
          if (detail) response.error = detail.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').slice(0, 180)
        }
        if (request.status === 202 || request.status === 200) resolve(response)
        else if (request.status === 409 && response.existing_job) {
          response.existing_job.reuse_notice = response.error
          resolve(response.existing_job)
        }
        else reject(new Error(response.error || response.message || `El servidor no recibió el lote (HTTP ${request.status}, sin detalle).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante la carga.'))
      request.send(form)
    })
    localStorage.setItem('folio-ingest-result', JSON.stringify(ingestResult.value))
    ingestProgress.value = 100
    if (ingestResult.value?.output) await loadWorkspaceDocuments(ingestResult.value.job_id)
    await loadIngestPreview(ingestResult.value.job_id)
    await loadSavedWorkspaces()
  } catch (error) {
    ingestError.value = error.message || 'No se pudo enviar el lote.'
  } finally {
    ingestSending.value = false
  }
}
async function replaceIngestZIP(event) {
  const file = event.target.files?.[0] || null
  event.target.value = ''
  if (!file || !ingestResult.value?.job_id || zipReplacementSending.value) return
  if (!file.name.toLowerCase().endsWith('.zip') || file.size === 0) {
    ingestError.value = 'Selecciona un archivo ZIP que no esté vacío.'
    return
  }
  ingestError.value = ''
  zipReplacementSending.value = true
  ingestPreview.value = null
  ingestPreviewError.value = ''
  const form = new FormData()
  form.append('zip_file', file)
  try {
    const response = await fetch(`/api/v1/ingesta/reemplazar-zip/${encodeURIComponent(ingestResult.value.job_id)}`, {
      method: 'POST', credentials: 'same-origin', body: form,
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo actualizar el ZIP guardado.')
    ingestResult.value = data
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    if (data.es_objeciones) {
      ingestPreview.value = null
      ingestPreviewError.value = ''
    } else {
      await loadIngestPreview(data.job_id)
    }
    await loadSavedWorkspaces()
  } catch (error) {
    ingestError.value = error.message || 'No se pudo actualizar el ZIP guardado.'
  } finally { zipReplacementSending.value = false }
}
async function processIngest() {
  const jobId = ingestResult.value?.job_id || existingIngestJobId.value.trim()
  if (!jobId || ingestProcessing.value) return
  ingestError.value = ''
  ingestProcessing.value = true
  try {
    if (!ingestResult.value) {
      const statusResponse = await fetch(`/api/v1/ingesta/estado/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
      const statusData = await statusResponse.json().catch(() => ({}))
      if (!statusResponse.ok) throw new Error(statusData.error || 'No se pudo abrir el expediente.')
      ingestResult.value = statusData
      localStorage.setItem('folio-ingest-result', JSON.stringify(statusData))
      if (statusData.output) {
        await loadWorkspaceDocuments(statusData.job_id)
        await loadIngestPreview(statusData.job_id)
        return
      }
    }
    if (!ingestPreview.value) await loadIngestPreview(jobId)
    if (!ingestPreviewReadyToPrepare.value) throw new Error('Revisa la vista previa: corrige las carpetas o el período antes de preparar el expediente.')
    const response = await fetch(`/api/v1/ingesta/procesar/${encodeURIComponent(jobId)}`, { method: 'POST', credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok && response.status === 409) {
      const retry = await fetch(`/api/v1/ingesta/clasificar/${encodeURIComponent(jobId)}`, { method: 'POST', credentials: 'same-origin' })
      const retryData = await retry.json().catch(() => ({}))
      if (!retry.ok) {
        if (retry.status === 404 || retry.status === 410) forgetUnavailableIngest(jobId)
        const error = new Error(retryData.error || data.error || 'No se pudo continuar el lote.')
        error.status = retry.status
        throw error
      }
      ingestResult.value = retryData
      localStorage.setItem('folio-ingest-result', JSON.stringify(retryData))
      await loadWorkspaceDocuments(retryData.job_id)
      await loadIngestPreview(retryData.job_id)
      await loadSavedWorkspaces()
      return
    }
    if (!response.ok) {
      if (response.status === 404) forgetUnavailableIngest(jobId)
      const error = new Error(data.error || 'No se pudo procesar el lote.')
      error.status = response.status
      throw error
    }
    ingestResult.value = data
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    if (data.output) await loadWorkspaceDocuments(data.job_id)
    await loadIngestPreview(data.job_id)
    await loadSavedWorkspaces()
  } catch (error) {
    if (error.status === 404 || error.status === 410) forgetUnavailableIngest(jobId)
    ingestError.value = error.message || 'No se pudo procesar el lote.'
  } finally {
    ingestProcessing.value = false
  }
}
async function classifyIngest() {
  const jobId = ingestResult.value?.job_id || existingIngestJobId.value.trim()
  if (!jobId || ingestProcessing.value) return
  ingestError.value = ''
  ingestProcessing.value = true
  try {
    const response = await fetch(`/api/v1/ingesta/clasificar/${encodeURIComponent(jobId)}`, { method: 'POST', credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      if (response.status === 404 || response.status === 410) forgetUnavailableIngest(jobId)
      const error = new Error(data.error || 'No se pudo clasificar el lote.')
      error.status = response.status
      throw error
    }
    ingestResult.value = data
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    if (data.output) await loadWorkspaceDocuments(data.job_id)
    await loadIngestPreview(data.job_id)
    await loadSavedWorkspaces()
  } catch (error) {
    if (error.status === 404 || error.status === 410) forgetUnavailableIngest(jobId)
    ingestError.value = error.message || 'No se pudo clasificar el lote.'
  } finally {
    ingestProcessing.value = false
  }
}
function forgetUnavailableIngest(jobId) {
  if (ingestResult.value?.job_id === jobId) ingestResult.value = null
  if (existingIngestJobId.value.trim() === jobId) existingIngestJobId.value = ''
  localStorage.removeItem('folio-ingest-result')
}
async function validateCachedIngest() {
  const jobId = ingestResult.value?.job_id
  if (!jobId) return
  try {
    const response = await fetch(`/api/v1/ingesta/estado/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    if (response.status === 404 || response.status === 410) {
      forgetUnavailableIngest(jobId)
      ingestError.value = 'El lote anterior ya no está guardado en el servidor. Para continuar, selecciona otra vez el ZIP clínico y crea un lote nuevo.'
    } else if (response.ok) {
      const data = await response.json()
      ingestResult.value = data
      localStorage.setItem('folio-ingest-result', JSON.stringify(data))
      if (data.output) await loadWorkspaceDocuments(jobId)
      await loadIngestPreview(jobId)
    }
  } catch { /* La vista conserva el lote local si no puede verificar el estado ahora. */ }
}
async function loadIngestPreview(jobId) {
  if (!jobId) return
  ingestPreviewLoading.value = true
  ingestPreview.value = null
  ingestPreviewError.value = ''
  try {
    const response = await fetch(`/api/v1/ingesta/previsualizar/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo inspeccionar el contenido del ZIP.')
    ingestPreview.value = data.vista_previa
    ingestPreviewPage.value = 1
  } catch (error) {
    ingestPreview.value = null
    ingestPreviewError.value = error.message || 'No se pudo inspeccionar el contenido del ZIP.'
  } finally { ingestPreviewLoading.value = false }
}
async function saveIngestTramiteMapping(folder, target = ingestMappingSelections.value[folder.tramite]) {
  const jobId = ingestResult.value?.job_id
  if (!jobId || !target || ingestMappingSaving.value) return
  ingestMappingSaving.value = folder.tramite
  ingestPreviewNotice.value = ''
  try {
    const response = await fetch(`/api/v1/ingesta/vincular-tramite/${encodeURIComponent(jobId)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tramite_zip: folder.tramite, tramite_oracle: target }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo emparejar la carpeta con Oracle.')
    ingestPreviewNotice.value = data.message || 'Se guardó el vínculo manual.'
    ingestMappingSelections.value = { ...ingestMappingSelections.value, [folder.tramite]: '' }
    await loadIngestPreview(jobId)
  } catch (error) {
    ingestPreviewNotice.value = error.message || 'No se pudo emparejar la carpeta con Oracle.'
  } finally { ingestMappingSaving.value = '' }
}
async function clearIngestTramiteMapping(folder) {
  const jobId = ingestResult.value?.job_id
  if (!jobId || ingestMappingSaving.value) return
  ingestMappingSaving.value = folder.tramite
  ingestPreviewNotice.value = ''
  try {
    const response = await fetch(`/api/v1/ingesta/vincular-tramite/${encodeURIComponent(jobId)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tramite_zip: folder.tramite, tramite_oracle: '' }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo quitar el vínculo manual.')
    ingestPreviewNotice.value = data.message || 'Se quitó el vínculo manual.'
    await loadIngestPreview(jobId)
  } catch (error) {
    ingestPreviewNotice.value = error.message || 'No se pudo quitar el vínculo manual.'
  } finally { ingestMappingSaving.value = '' }
}
const orderedSavedWorkspaces = computed(() => orderWorkspaceRecords(savedWorkspaces.value))
const coverageWorkspaceCandidates = computed(() => orderedSavedWorkspaces.value
  .filter(workspace => !workspace.es_objeciones && ['PROCESSED', 'INCOMPLETE'].includes(workspace.status)))
const objectionSourceWorkspaces = computed(() => {
  const seen = new Set()
  return orderedSavedWorkspaces.value
    .filter(workspace => !workspace.es_objeciones && ['PROCESSED', 'INCOMPLETE'].includes(workspace.status))
    .filter(workspace => {
      const id = String(workspace.job_id || '').trim()
      if (!id || seen.has(id)) return false
      seen.add(id)
      return true
    })
})
const objectionSourceItems = computed(() => objectionSourceWorkspaces.value
  .map(workspaceSavedOption))
const objectionWorkspaceWorkspaces = computed(() => orderedSavedWorkspaces.value
  .filter(workspace => workspace.es_objeciones && ['PROCESSED', 'INCOMPLETE'].includes(workspace.status)))
const objectionWorkspaceItems = computed(() => objectionWorkspaceWorkspaces.value
  .map(workspaceSavedOption))
const zipDownloadPreparedWorkspaces = computed(() => orderedSavedWorkspaces.value
  .filter(workspace => ['PROCESSED', 'INCOMPLETE'].includes(workspace.status)))
const zipDownloadWorkspaces = computed(() => zipDownloadPreparedWorkspaces.value
  .filter(workspace => zipDownloadType.value === 'OBJECIONES' ? workspace.es_objeciones : !workspace.es_objeciones))
const zipDownloadTypeCounts = computed(() => ({
  RECEPCION: zipDownloadPreparedWorkspaces.value.filter(workspace => !workspace.es_objeciones).length,
  OBJECIONES: zipDownloadPreparedWorkspaces.value.filter(workspace => workspace.es_objeciones).length,
}))
const selectedZipWorkspaceRecord = computed(() => savedWorkspaces.value.find(workspace => workspace.job_id === selectedZipWorkspace.value) || null)
const selectedZipWorkspaceReady = computed(() => selectedZipWorkspaceRecord.value?.ready_for_delivery === true)
const selectedZipWorkspaceMissing = computed(() => selectedZipWorkspaceRecord.value?.delivery_missing || [])
function workspaceSavedOption(workspace) {
  const monthName = coverageMonthLabel(workspaceMonthValue(workspace))
  const year = workspaceYearLabel(workspace)
  const serviceName = coverageServiceLabel(workspace.tipo_servicio)
  const state = workspaceStatusLabel(workspace.status, workspace)
  return { title: `${workspaceTypeLabel(workspace)} · ${serviceName} · ${monthName} ${year} · ${workspace.job_id} · ${state} · ${workspace.creado_por || 'usuario anterior'}`, value: workspace.job_id }
}
const savedWorkspaceItems = computed(() => orderedSavedWorkspaces.value.map(workspaceSavedOption))
const zipDownloadWorkspaceItems = computed(() => zipDownloadWorkspaces.value
  .map(workspaceSavedOption))
const savedWorkspaceTypeFilters = computed(() => [
  { title: 'Todos los tipos', value: 'ALL' },
  { title: 'Recepción de planillas', value: 'RECEPCION' },
  { title: 'Objeciones · espacio separado', value: 'OBJECIONES' },
])
const savedWorkspaceServiceFilters = computed(() => {
  const services = [...new Set(savedWorkspaces.value.map(workspace => String(workspace.tipo_servicio || '').trim()).filter(Boolean))]
  services.sort((left, right) => coverageServiceLabel(left).localeCompare(coverageServiceLabel(right), 'es'))
  return [{ title: `Todos los servicios (${savedWorkspaces.value.length})`, value: 'ALL' }, ...services.map(service => ({ title: coverageServiceLabel(service), value: service }))]
})
const savedWorkspaceYearFilters = computed(() => {
  const years = [...new Set(savedWorkspaces.value.map(workspaceYearLabel).filter(year => /^\d{4}$/.test(year)))].sort((left, right) => Number(right) - Number(left))
  return [{ title: 'Todos los años', value: 'ALL' }, ...years.map(year => ({ title: year, value: year }))]
})
const filteredSavedWorkspaces = computed(() => {
  return filterWorkspaceRecords(savedWorkspaces.value, {
    search: savedWorkspaceSearch.value,
    type: savedWorkspaceTypeFilter.value,
    service: savedWorkspaceServiceFilter.value,
    year: savedWorkspaceYearFilter.value,
    status: savedWorkspaceStatusFilter.value,
  }, {
    typeLabel: workspaceTypeLabel,
    serviceLabel: coverageServiceLabel,
    monthLabel: coverageMonthLabel,
    statusLabel: workspaceStatusLabel,
    receivedAt: workspaceReceivedAt,
  })
})
const hasSavedWorkspaceFilters = computed(() => Boolean(String(savedWorkspaceSearch.value || '').trim())
  || savedWorkspaceTypeFilter.value !== 'ALL'
  || savedWorkspaceServiceFilter.value !== 'ALL'
  || savedWorkspaceYearFilter.value !== 'ALL'
  || savedWorkspaceStatusFilter.value !== 'ALL')
const savedWorkspaceStatusFilters = computed(() => [
  { title: `Todos (${savedWorkspaces.value.length})`, value: 'ALL' },
  { title: `Listos para entrega (${savedWorkspaces.value.filter(item => workspaceFilterStatusValue(item) === 'READY_FOR_DELIVERY').length})`, value: 'READY_FOR_DELIVERY' },
  { title: `Avances incompletos (${savedWorkspaces.value.filter(item => workspaceFilterStatusValue(item) === 'INCOMPLETE_PACKAGE').length})`, value: 'INCOMPLETE_PACKAGE' },
  { title: `Por preparar (${savedWorkspaces.value.filter(item => item.status === 'STAGED').length})`, value: 'STAGED' },
  { title: `Revisión (${savedWorkspaces.value.filter(item => item.status === 'REQUIERE_REVISION').length})`, value: 'REQUIERE_REVISION' },
  ...[...new Set(savedWorkspaces.value.map(item => item.status).filter(status => status && !['INCOMPLETE', 'PROCESSED', 'STAGED', 'REQUIERE_REVISION'].includes(status)))].sort().map(status => ({ title: `${workspaceStatusLabel(status)} (${savedWorkspaces.value.filter(item => item.status === status).length})`, value: status })),
])
function workspaceStatusLabel(status, workspace = null) {
  if (workspace && (status === 'PROCESSED' || status === 'INCOMPLETE')) {
    if (workspace.ready_for_delivery === true) return 'Listo para entrega'
    if (workspace.ready_for_delivery === false) return 'Avance incompleto'
  }
  return ({ STAGED: 'Pendiente de preparar', INCOMPLETE: 'Incompleto', PROCESSED: 'Preparado', REQUIERE_REVISION: 'Requiere revisión' })[status] || status || 'Estado desconocido'
}
function workspaceStatusColor(status, workspace = null) {
  if (workspace && (status === 'PROCESSED' || status === 'INCOMPLETE')) return workspace.ready_for_delivery === true ? 'success' : 'warning'
  return ({ STAGED: 'info', INCOMPLETE: 'warning', PROCESSED: 'success', REQUIERE_REVISION: 'error' })[status] || 'secondary'
}
function workspaceReceivedAt(value) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('es-EC', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
function clearSavedWorkspaceFilters() {
  savedWorkspaceSearch.value = ''
  savedWorkspaceTypeFilter.value = 'ALL'
  savedWorkspaceServiceFilter.value = 'ALL'
  savedWorkspaceYearFilter.value = 'ALL'
  savedWorkspaceStatusFilter.value = 'ALL'
}
function billedPeriodLabel(period) {
  const [monthValue, year] = String(period || '').split('/')
  const month = ingestMonths.find(item => item.value === monthValue)?.title
  return month && year ? `${month.toLowerCase()} de ${year}` : period
}
async function loadSavedWorkspaces() {
  savedWorkspacesLoading.value = true
  savedWorkspacesError.value = ''
  try {
    const response = await fetch('/api/v1/expedientes', { credentials: 'same-origin' })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar los períodos guardados.')
    savedWorkspaces.value = data.workspaces || []
    if (!ingestModeTouched.value && !ingestResult.value) ingestMode.value = savedWorkspaces.value.length ? 'resume' : 'new'
    if (!savedWorkspaces.value.some(workspace => workspace.job_id === selectedSavedWorkspace.value)) selectedSavedWorkspace.value = savedWorkspaces.value[0]?.job_id || ''
  } catch (error) { savedWorkspacesError.value = error.message || 'No se pudieron cargar los períodos guardados.' }
  finally { savedWorkspacesLoading.value = false }
}
function switchIngestMode(mode) {
  if (!mode || mode === ingestMode.value) return
  ingestModeTouched.value = true
  if (ingestResult.value) startNewIngest()
  ingestMode.value = mode
  if (mode === 'resume') loadSavedWorkspaces()
}
function showSavedWorkspaceList() {
  ingestModeTouched.value = true
  if (ingestResult.value) startNewIngest()
  ingestMode.value = 'resume'
  loadSavedWorkspaces()
}
async function openSavedWorkspace(jobId = selectedSavedWorkspace.value) {
  if (!jobId || openingSavedWorkspace.value) return
  selectedSavedWorkspace.value = jobId
  showWorkspaceDetails.value = false
  openingSavedWorkspace.value = true
  ingestError.value = ''
  try {
    const response = await fetch(`/api/v1/ingesta/estado/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo abrir el espacio guardado.')
    ingestMode.value = 'resume'
    ingestModeTouched.value = true
    ingestResult.value = data
    if (data.es_objeciones) localStorage.removeItem('folio-ingest-result')
    else localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    if (data.output) await loadWorkspaceDocuments(data.job_id)
    else { workspacePDFs.value = []; workspacePatients.value = [] }
    await loadIngestPreview(data.job_id)
  } catch (error) { ingestError.value = error.message || 'No se pudo abrir el espacio guardado.' }
  finally { openingSavedWorkspace.value = false }
}
async function openPatientDocumentsPage() {
  activePage.value = 'patient-documents'
  patientDocumentsDialog.value = false
  patientDocumentsError.value = ''
  await loadSavedWorkspaces()
}
async function openPatientDocumentsWorkspace(jobId = selectedSavedWorkspace.value) {
  if (!jobId || openingSavedWorkspace.value) return
  patientDocumentsError.value = ''
  await openSavedWorkspace(jobId)
  if (ingestError.value) {
    patientDocumentsError.value = ingestError.value
    return
  }
  if (ingestResult.value?.job_id !== jobId || !ingestResult.value?.output) {
    patientDocumentsError.value = 'El período seleccionado todavía no tiene documentos preparados.'
    return
  }
  patientDocumentsDialog.value = true
}
async function openZipDownloadPage() {
  activePage.value = 'zip-download'
  zipDownloadError.value = ''
  zipDownloadNotice.value = ''
  await loadSavedWorkspaces()
  if (!zipDownloadWorkspaceItems.value.some(item => item.value === selectedZipWorkspace.value)) selectedZipWorkspace.value = zipDownloadWorkspaceItems.value[0]?.value || ''
}
function selectZipDownloadType(type) {
  if (!['RECEPCION', 'OBJECIONES'].includes(type) || zipDownloadType.value === type) return
  zipDownloadType.value = type
  selectedZipWorkspace.value = zipDownloadWorkspaceItems.value[0]?.value || ''
  zipDownloadError.value = ''
  zipDownloadNotice.value = ''
  zipDownloadArchiveState.value = ''
}
async function openObjectionsPage() {
  activePage.value = 'objeciones'
  objectionError.value = ''
  objectionConflictMessage.value = ''
  objectionConflictSpaces.value = []
  objectionNotice.value = ''
  await loadSavedWorkspaces()
  if (!objectionSourceItems.value.some(item => item.value === objectionSourceWorkspace.value)) objectionSourceWorkspace.value = objectionSourceItems.value[0]?.value || ''
  if (!objectionWorkspaceItems.value.some(item => item.value === objectionSelectedWorkspace.value)) objectionSelectedWorkspace.value = objectionWorkspaceItems.value[0]?.value || ''
  if (objectionSelectedWorkspace.value) await loadObjectionWorkspace(objectionSelectedWorkspace.value)
}
function objectionFormData() {
  if (!objectionSourceWorkspace.value) return null
  const form = new FormData()
  form.append('expediente_origen', objectionSourceWorkspace.value)
  return form
}
function resetObjectionPreview() {
  objectionPreview.value = null
  objectionSelectedTramites.value = []
  objectionConflictMessage.value = ''
  objectionConflictSpaces.value = []
}
async function chooseObjectionWorkspace(jobId) {
  if (!jobId) return
  objectionSelectedWorkspace.value = jobId
  objectionConflictMessage.value = ''
  await loadObjectionWorkspace(jobId)
}
function setObjectionInput(kind, event) {
  const file = event.target.files?.[0] || null
  event.target.value = ''
  objectionHeaderFiles.value = { ...objectionHeaderFiles.value, [kind]: file }
}
async function previewObjections() {
  if (!objectionSourceWorkspace.value || objectionBusy.value) return
  objectionBusy.value = true
  objectionError.value = ''
  objectionNotice.value = ''
  objectionPreview.value = null
  objectionSelectedTramites.value = []
  try {
    const form = new URLSearchParams({ expediente_origen: objectionSourceWorkspace.value })
    const response = await fetch('/api/v1/objeciones/previsualizar', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' }, body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar los pacientes del período.')
    objectionPreview.value = data
    objectionNotice.value = ''
  } catch (error) { objectionError.value = error.message || 'No se pudieron cargar los pacientes del período.' }
  finally { objectionBusy.value = false }
}
async function createObjectionWorkspace() {
  const form = objectionFormData()
  if (!form || !objectionCanCreate.value || objectionBusy.value) return
  if (objectionPeriodSpaces.value.length > 1 && !objectionChosenPeriodSpace.value) {
    objectionConflictSpaces.value = objectionPeriodSpaces.value
    objectionConflictMessage.value = 'Ya hay varios espacios de Objeciones para este período. Elige uno para continuar; no se borrarán ni fusionarán automáticamente.'
    return
  }
  if (objectionChosenPeriodSpace.value) form.append('reutilizar_espacio', objectionChosenPeriodSpace.value.job_id)
  const selected = new Set(objectionSelectedTramites.value)
  for (const tramite of selected) form.append('tramites_objetados', tramite)
  objectionBusy.value = true
  objectionError.value = ''
  objectionConflictMessage.value = ''
  objectionConflictSpaces.value = []
  objectionNotice.value = ''
  try {
    const response = await fetch('/api/v1/objeciones/crear', { method: 'POST', credentials: 'same-origin', body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (response.status === 409 && Array.isArray(data.espacios_existentes)) {
      objectionConflictSpaces.value = data.espacios_existentes
      objectionConflictMessage.value = data.error || 'Hay varios espacios de Objeciones para este período. Elige uno para continuar.'
      return
    }
    if (!response.ok) throw new Error(data.error || 'No se pudo crear el espacio separado.')
    objectionSelectedWorkspace.value = data.job_id
    objectionCurrentWorkspace.value = data
    objectionHeaderFiles.value = {}
    objectionAddPreview.value = null
    objectionPreview.value = null
    objectionSelectedTramites.value = []
    objectionPostures.value = Object.fromEntries((data.objeciones || []).filter(row => row.postura).map(row => [row.pdi_tramite, row.postura]))
    objectionNotice.value = data.message || 'Se creó el espacio separado de objeciones.'
    objectionConflictMessage.value = ''
    objectionConflictSpaces.value = []
    await loadObjectionPDFSelection(data.job_id)
    await loadSavedWorkspaces()
  } catch (error) { objectionError.value = error.message || 'No se pudo crear el espacio separado.' }
  finally { objectionBusy.value = false }
}
async function loadObjectionWorkspace(jobId = objectionSelectedWorkspace.value) {
  if (!jobId) { objectionCurrentWorkspace.value = null; objectionPackagePDFs.value = []; objectionPDFError.value = ''; return }
  if (objectionCurrentWorkspace.value?.job_id !== jobId) {
    objectionHeaderFiles.value = {}
    objectionAddPreview.value = null
    objectionAddSelectedTramites.value = []
    objectionUploadFiles.value = {}
    objectionPackagePDFs.value = []
  }
  objectionBusy.value = true
  objectionError.value = ''
  objectionSelectedWorkspace.value = jobId
  try {
    const response = await fetch(`/api/v1/ingesta/estado/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo abrir el espacio de objeciones.')
    if (!data.es_objeciones) throw new Error('El espacio seleccionado no corresponde a objeciones.')
    objectionCurrentWorkspace.value = data
    objectionPostures.value = Object.fromEntries((data.objeciones || []).filter(row => row.postura).map(row => [row.pdi_tramite, row.postura]))
    await loadObjectionPDFSelection(jobId)
  } catch (error) { objectionError.value = error.message || 'No se pudo abrir el espacio de objeciones.' }
  finally { objectionBusy.value = false }
}
async function loadObjectionPDFSelection(jobId) {
  if (!jobId) { objectionPackagePDFs.value = []; objectionPDFError.value = ''; return }
  objectionPDFLoading.value = true
  objectionPDFError.value = ''
  objectionPackagePDFs.value = []
  try {
    const response = await fetch(`/api/v1/objeciones/pdfs/seleccion/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar los PDFs del paquete.')
    objectionPackagePDFs.value = data.documents || []
  } catch (error) {
    objectionPDFError.value = error.message || 'No se pudieron cargar los PDFs del paquete.'
  } finally { objectionPDFLoading.value = false }
}
async function setObjectionPDFIncluded(document, included) {
  if (!document || document.obligatorio || !objectionSelectedWorkspace.value || objectionPDFSavingPath.value) return
  objectionPDFSavingPath.value = document.path
  objectionPDFError.value = ''
  objectionNotice.value = ''
  try {
    const response = await fetch(`/api/v1/objeciones/pdfs/seleccion/${encodeURIComponent(objectionSelectedWorkspace.value)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: document.path, incluir: Boolean(included) }),
    })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo guardar la selección del PDF.')
    objectionPackagePDFs.value = data.documents || []
    objectionNotice.value = included ? `${document.name} se incluirá en el ZIP.` : `${document.name} quedará fuera del ZIP.`
  } catch (error) { objectionPDFError.value = error.message || 'No se pudo guardar la selección del PDF.' }
  finally { objectionPDFSavingPath.value = '' }
}
function setObjectionPostureSelection(tramite, value) {
  const key = String(tramite ?? '')
  if (!key) return
  objectionPostures.value = { ...objectionPostures.value, [key]: String(value ?? '') }
}
async function saveObjectionPosture(patient) {
  const tramite = String(patient?.tramite ?? '')
  const jobId = objectionSelectedWorkspace.value
  const posture = objectionPostures.value[tramite]
  if (!patient || !tramite || !['ACEPTA', 'RECHAZA'].includes(posture) || !jobId || objectionPostureSaving.value) return
  objectionPostureSaving.value = tramite
  objectionError.value = ''
  objectionNotice.value = ''
  try {
    const { response, data } = await fetchWorkspaceJSON(`/api/v1/objeciones/postura/${encodeURIComponent(jobId)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pdi_tramite: tramite, postura: posture }),
    }, 30000, 'guardar la postura')
    if (response.status === 401) { await signOut(); return }
    if (!response.ok) throw new Error(data.error || 'No se pudo guardar la postura.')
    if (objectionSelectedWorkspace.value === jobId) {
      objectionCurrentWorkspace.value = data
      objectionPostures.value = Object.fromEntries((data.objeciones || []).filter(row => row.postura).map(row => [String(row.pdi_tramite), row.postura]))
      objectionNotice.value = `Se guardó ${posture} para el trámite ${tramite}.`
    }
  } catch (error) { objectionError.value = error.message || 'No se pudo guardar la postura.' }
  finally { objectionPostureSaving.value = '' }
}
async function previewObjectionAddCandidates() {
  const sourceId = objectionCurrentWorkspace.value?.expediente_origen
  if (!sourceId || objectionBusy.value) return
  objectionBusy.value = true
  objectionError.value = ''
  objectionNotice.value = ''
  try {
    const form = new URLSearchParams({ expediente_origen: sourceId })
    const response = await fetch('/api/v1/objeciones/previsualizar', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' }, body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo volver a cargar la lista del período.')
    objectionAddPreview.value = data
    objectionAddSelectedTramites.value = []
  } catch (error) { objectionError.value = error.message || 'No se pudo volver a cargar la lista del período.' }
  finally { objectionBusy.value = false }
}
async function addObjectionPatients() {
  if (!objectionSelectedWorkspace.value || !objectionAddSelectedTramites.value.length || objectionBusy.value) return
  objectionBusy.value = true
  objectionError.value = ''
  objectionNotice.value = ''
  try {
    const form = new URLSearchParams()
    for (const tramite of objectionAddSelectedTramites.value) form.append('tramites_objetados', tramite)
    const response = await fetch('/api/v1/objeciones/agregar/' + encodeURIComponent(objectionSelectedWorkspace.value), { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' }, body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron agregar los pacientes seleccionados.')
    objectionCurrentWorkspace.value = data
    objectionAddPreview.value = null
    objectionAddSelectedTramites.value = []
    objectionHeaderFiles.value = { ...objectionHeaderFiles.value, matriz: null }
    await loadObjectionPDFSelection(data.job_id)
    objectionNotice.value = data.message || 'Se agregaron los trámites al espacio de objeciones.'
    await loadSavedWorkspaces()
  } catch (error) { objectionError.value = error.message || 'No se pudieron agregar los pacientes seleccionados.' }
  finally { objectionBusy.value = false }
}
async function uploadObjectionHeader(kind) {
  const file = objectionHeaderFiles.value[kind]
  if (!file || !objectionSelectedWorkspace.value || objectionBusy.value) return
  objectionBusy.value = true
  objectionError.value = ''
  objectionNotice.value = ''
  try {
    const form = new FormData()
    form.append('tipo', kind)
    form.append('archivo', file, file.name)
    const response = await fetch('/api/v1/objeciones/cabeceras/' + encodeURIComponent(objectionSelectedWorkspace.value), { method: 'POST', credentials: 'same-origin', body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo guardar el documento de cabecera.')
    objectionHeaderFiles.value = { ...objectionHeaderFiles.value, [kind]: null }
    objectionNotice.value = data.message || 'Documento de cabecera guardado.'
    await loadObjectionWorkspace(objectionSelectedWorkspace.value)
  } catch (error) { objectionError.value = error.message || 'No se pudo guardar el documento de cabecera.' }
  finally { objectionBusy.value = false }
}
function setObjectionUpload(event, tramite, kind) {
  const file = event.target.files?.[0] || null
  event.target.value = ''
  objectionUploadFiles.value = { ...objectionUploadFiles.value, [`${tramite}:${kind}`]: file }
}
async function uploadObjectionDocument(patient, kind) {
  if (kind !== 'anexo') return
  const key = `${patient.tramite}:${kind}`
  const file = objectionUploadFiles.value[key]
  if (!file || !objectionSelectedWorkspace.value || objectionBusy.value) return
  objectionBusy.value = true
  objectionError.value = ''
  objectionNotice.value = ''
  try {
    const form = new FormData()
    form.append('tipo', kind)
    form.append('pdi_tramite', patient.tramite)
    const annexType = objectionAnnexTypeByPatient.value[patient.tramite] || ''
    form.append('tipo_anexo', annexType)
    if (annexType === 'OTRO') form.append('nombre_anexo', objectionAnnexNameByPatient.value[patient.tramite] || '')
    form.append('pdf_file', file, file.name)
    const response = await fetch(`/api/v1/objeciones/documentos/${encodeURIComponent(objectionSelectedWorkspace.value)}`, { method: 'POST', credentials: 'same-origin', body: form })
    if (response.status === 401) { await signOut(); return }
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo guardar el documento.')
    objectionUploadFiles.value = { ...objectionUploadFiles.value, [key]: null }
    objectionNotice.value = data.nombre_archivo
      ? `${data.nombre_archivo} guardado en 5. ANEXOS/${patient.folder}.`
      : data.message || 'Anexo guardado.'
    await loadObjectionWorkspace(objectionSelectedWorkspace.value)
  } catch (error) { objectionError.value = error.message || 'No se pudo guardar el documento.' }
  finally { objectionBusy.value = false }
}
function setObjectionAnnexName(tramite, value) {
  objectionAnnexNameByPatient.value = { ...objectionAnnexNameByPatient.value, [tramite]: value }
}
async function openObjectionPatientDocuments(patient) {
  await openPatientDocumentsWorkspace(objectionSelectedWorkspace.value)
  if (!patientDocumentsDialog.value) return
  workspacePatient.value = patient.folder
  const prefix = `4. EXPEDIENTES/${patient.folder}/`
  const first = workspacePDFs.value.find(document => document.path.startsWith(prefix))
  selectWorkspacePDF(first?.path || '')
}
function startNewIngest() {
  ingestMode.value = 'new'
  ingestModeTouched.value = true
  showWorkspaceDetails.value = false
  ingestResult.value = null
  ingestPreview.value = null
  ingestPreviewError.value = ''
  ingestFiles.value = { zip_file: null, matriz_file: null, consolidada_file: null, oficio_file: null }
  completionFiles.value = { matriz_file: null, consolidada_file: null, oficio_file: null }
  ingestError.value = ''
  ingestProgress.value = 0
  existingIngestJobId.value = ''
  localStorage.removeItem('folio-ingest-result')
}
const completionFields = computed(() => {
  if (!['STAGED', 'PROCESSED', 'INCOMPLETE', 'REQUIERE_REVISION'].includes(ingestResult.value?.status)) return []
  return ingestFileFields.filter(item => item.field !== 'zip_file')
})
function currentCompletionFile(field) { return ingestResult.value?.files?.find(file => file.field === field) || null }
const completionFiles = ref({ matriz_file: null, consolidada_file: null, oficio_file: null })
const completionValid = computed(() => completionFields.value.length > 0 && completionFields.value.every(item => {
  const file = completionFiles.value[item.field]
  return !file || (file.size > 0 && file.name.toLowerCase().endsWith(item.accept.split(',')[0]))
}))
const ingestPreviewPageCount = computed(() => Math.max(1, Math.ceil((ingestPreview.value?.carpetas?.length || 0) / ingestPreviewPageSize)))
const ingestPreviewReadyToPrepare = computed(() => Boolean(ingestPreview.value && ingestPreview.value.pdfs > 0 && ingestPreview.value.carpetas_tramite > 0 && ingestPreview.value.entradas_invalidas === 0 && ingestPreview.value.tramites_sin_oracle === 0 && ingestPreview.value.tramites_sin_paciente_oracle === 0))
const ingestPreviewRows = computed(() => {
  const folders = ingestPreview.value?.carpetas || []
  const start = (ingestPreviewPage.value - 1) * ingestPreviewPageSize
  return folders.slice(start, start + ingestPreviewPageSize)
})
function selectCompletionFile(field, event) {
  const file = event.target.files?.[0] || null
  const expected = { matriz_file: '.xlsm', consolidada_file: '.pdf', oficio_file: '.pdf' }[field]
  ingestError.value = ''
  if (file && (file.size === 0 || !file.name.toLowerCase().endsWith(expected))) {
    completionFiles.value[field] = null
    ingestError.value = `Selecciona un archivo ${expected} que no esté vacío.`
  } else completionFiles.value[field] = file
  event.target.value = ''
}
async function addMissingDocuments() {
  const jobId = ingestResult.value?.job_id
  if (!jobId || !completionValid.value || ingestProcessing.value) return
  const selected = Object.entries(completionFiles.value).filter(([, file]) => file)
  if (!selected.length) { ingestError.value = 'Selecciona al menos uno de los documentos pendientes.'; return }
  ingestError.value = ''; ingestProcessing.value = true; completionSending.value = true; ingestProgress.value = 0
  const form = new FormData()
  for (const [field, file] of selected) form.append(field, file)
  try {
    const data = await new Promise((resolve, reject) => {
      const request = new XMLHttpRequest()
      request.open('POST', `/api/v1/ingesta/completar/${encodeURIComponent(jobId)}`)
      request.withCredentials = true
      request.upload.onprogress = event => {
        if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100)
      }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch { /* La API debe responder JSON. */ }
        if (request.status === 200) resolve(response)
        else reject(new Error(response.error || `No se pudieron añadir los documentos (HTTP ${request.status}).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante la carga.'))
      request.send(form)
    })
    ingestResult.value = data
    completionFiles.value = { matriz_file: null, consolidada_file: null, oficio_file: null }
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    ingestProgress.value = 100
    if (data.output) await loadWorkspaceDocuments(jobId)
    await loadSavedWorkspaces()
  } catch (error) { ingestError.value = error.message || 'No se pudieron añadir los documentos.' }
  finally { ingestProcessing.value = false; completionSending.value = false }
}
onMounted(async () => {
  try {
    const savedIngest = JSON.parse(localStorage.getItem('folio-ingest-result') || 'null')
    if (savedIngest?.job_id) {
      ingestResult.value = savedIngest
      ingestMode.value = 'resume'
      ingestModeTouched.value = true
    }
  } catch { localStorage.removeItem('folio-ingest-result') }
  await checkSession()
  window.addEventListener('keydown', handleShortcut)
})
onUnmounted(() => window.removeEventListener('keydown', handleShortcut))
watch(preview, (doc, previous) => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = doc?.blob ? URL.createObjectURL(doc.blob) : ''
})
const visibleDocs = computed(() => {
  let items = [...docs.value]
  if (activeFolder.value === 'Favoritos') items = items.filter(d => d.favorite)
  else if (activeFolder.value !== 'Todos los documentos') items = items.filter(d => d.folder === activeFolder.value)
  if (query.value.trim()) { const q = query.value.toLowerCase(); items = items.filter(d => (d.name+' '+d.ext+' '+d.folder).toLowerCase().includes(q)) }
  if (typeFilter.value !== 'Todo') items = items.filter(d => d.color === typeFilter.value)
  if (dateFilter.value !== 'Cualquier fecha') { const days = dateFilter.value === 'Últimos 7 días' ? 7 : 30; const since = Date.now()-days*24*60*60*1000; items = items.filter(d => d.addedAt && d.addedAt >= since) }
  if (sortBy.value === 'Nombre') items.sort((a,b) => a.name.localeCompare(b.name))
  else if (sortBy.value === 'Tamaño') items.sort((a,b) => (b.bytes||0)-(a.bytes||0))
  else items.sort((a,b) => (b.addedAt||0)-(a.addedAt||0))
  return items
})
  const title = computed(() => activePage.value === 'planilla' ? 'Planilla digital' : activePage.value === 'coberturas' ? 'Hojas de cobertura' : activePage.value === 'objeciones' ? 'Subsanar objeciones' : activePage.value === 'patient-documents' ? 'Revisar documentos del paciente' : activePage.value === 'zip-download' ? 'Descargar expediente ZIP' : activePage.value === 'ingesta' ? 'Recepción de planillas' : activeFolder.value === 'Todos los documentos' ? 'Mis documentos' : activeFolder.value)
function prettySize(n) { return n < 1024*1024 ? `${Math.max(1, Math.round(n/1024))} KB` : `${(n/1024/1024).toFixed(1)} MB` }
function handleShortcut(event) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); searchInput.value?.focus() }
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'u') { event.preventDefault(); uploadInput.value?.click() }
  if (event.key === 'Escape') { dragging.value = false; selectedIds.value = [] }
}
function onDragEnter(event) { if (activePage.value !== 'documents') return; if (event.dataTransfer?.types?.includes('Files')) { event.preventDefault(); dragDepth++; dragging.value = true } }
function onDragLeave(event) { if (event.dataTransfer?.types?.includes('Files')) { event.preventDefault(); dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) dragging.value = false } }
function onDrop(event) { event.preventDefault(); dragDepth = 0; dragging.value = false; if (activePage.value === 'documents' && event.dataTransfer?.files?.length) upload({ target: { files: event.dataTransfer.files, value: '' } }) }
function toggleSelected(doc) { selectedIds.value = selectedIds.value.includes(doc.id) ? selectedIds.value.filter(id => id !== doc.id) : [...selectedIds.value, doc.id] }
function selectAll() { selectedIds.value = selectedIds.value.length === visibleDocs.value.length ? [] : visibleDocs.value.map(doc => doc.id) }
async function createSpace() {
  const name = newSpaceName.value.trim()
  if (!name) return
  if (spaces.value.some(space => space.toLowerCase() === name.toLowerCase())) { snackbar.value = 'Ya existe un espacio con ese nombre.'; return }
  customSpaces.value.push(name); localStorage.setItem('folio-spaces', JSON.stringify(customSpaces.value)); activeFolder.value = name; newSpaceName.value = ''; createSpaceDialog.value = false; snackbar.value = `Espacio «${name}» creado`
}
function startMove(doc = null) { activeDoc.value = doc; moveTarget.value = spaces.value.find(s => s !== doc?.folder) || 'Mi espacio'; moveDialog.value = true }
async function confirmMove() {
  const moving = activeDoc.value ? [activeDoc.value] : [...docs.value].filter(doc => selectedIds.value.includes(doc.id))
  let count = 0
  for (const doc of moving) { doc.folder = moveTarget.value; await (await dbPromise).put('files', doc); count++ }
  snackbar.value = `${count} ${count === 1 ? 'documento movido' : 'documentos movidos'} a «${moveTarget.value}»`
  moveDialog.value = false; activeDoc.value = null; selectedIds.value = []
}
async function bulkFavorite() { const moving = docs.value.filter(doc => selectedIds.value.includes(doc.id)); for (const doc of moving) { doc.favorite = true; await (await dbPromise).put('files', doc) }; snackbar.value = `${moving.length} ${moving.length === 1 ? 'documento añadido' : 'documentos añadidos'} a favoritos`; selectedIds.value = [] }
async function bulkDelete() { const moving = docs.value.filter(doc => selectedIds.value.includes(doc.id)); for (const doc of moving) await (await dbPromise).delete('files', doc.id); docs.value = docs.value.filter(doc => !selectedIds.value.includes(doc.id)); snackbar.value = `${moving.length} ${moving.length === 1 ? 'documento eliminado' : 'documentos eliminados'}`; selectedIds.value = [] }
async function upload(event) {
  const files = [...(event.target.files || [])]
  if (!files.length) return
  const db = await dbPromise
  for (const file of files) {
    const ext = (file.name.split('.').pop() || 'FILE').toUpperCase()
    const typeMap = { PDF:['pdf','mdi-file-pdf-box'], DOC:['word','mdi-file-word'], DOCX:['word','mdi-file-word'], XLS:['sheet','mdi-file-excel'], XLSX:['sheet','mdi-file-excel'], PNG:['image','mdi-image-outline'], JPG:['image','mdi-image-outline'], JPEG:['image','mdi-image-outline'] }
    const [color, icon] = typeMap[ext] || ['other','mdi-file-outline']
    const doc = { id: crypto.randomUUID(), name:file.name, ext, size:prettySize(file.size), bytes:file.size, date:'Ahora', addedAt:Date.now(), folder:activeFolder.value === 'Todos los documentos' || activeFolder.value === 'Favoritos' ? 'Mi espacio' : activeFolder.value, color, icon, blob:file, mime:file.type, favorite:false }
    await db.put('files', doc)
    docs.value.unshift(doc)
  }
  snackbar.value = `${files.length} ${files.length === 1 ? 'documento añadido' : 'documentos añadidos'} a tu espacio`
  event.target.value = ''
}
async function toggleFavorite(doc) {
  doc.favorite = !doc.favorite; await (await dbPromise).put('files', doc); snackbar.value = doc.favorite ? 'Añadido a favoritos' : 'Quitado de favoritos'
}
function openPreview(doc) { if (!doc.blob) { snackbar.value = 'El archivo no está disponible en este navegador.'; return }; preview.value = doc }
function download(doc) { if (!doc.blob) { snackbar.value = 'El archivo no está disponible en este navegador.'; return }; const url=URL.createObjectURL(doc.blob); const a=document.createElement('a'); a.href=url; a.download=doc.name; a.click(); URL.revokeObjectURL(url) }
async function removeDoc(doc) { await (await dbPromise).delete('files', doc.id); docs.value=docs.value.filter(d=>d.id!==doc.id); preview.value=null; snackbar.value='Documento eliminado' }
</script>

<template>
  <v-app>
    <main v-if="authStatus !== 'authenticated'" class="login-page">
      <div class="login-orbit orbit-one"></div><div class="login-orbit orbit-two"></div>
      <section class="login-card" aria-labelledby="login-title">
        <a class="login-brand" href="#" aria-label="SPD MSP"><span class="brand-mark"><v-icon icon="mdi-book-open-page-variant" size="21" /></span><span>SPD MSP</span></a>
        <div v-if="authStatus === 'checking'" class="login-loading"><v-progress-circular indeterminate color="primary"/><p>Comprobando tu sesión…</p></div>
        <template v-else>
          <div class="login-heading"><span class="login-kicker">GESTIÓN DE EXPEDIENTES DE FACTURACIÓN</span><h1 id="login-title">Bienvenido a SPD MSP</h1><p>Inicia sesión con tu usuario de Oracle. Necesitas el rol <strong>SPD_EXTERNOS</strong>.</p></div>
          <form class="login-form" @submit.prevent="signIn">
            <v-text-field v-model="oracleUser" label="Usuario de Oracle" placeholder="Tu usuario" prepend-inner-icon="mdi-account-outline" autocomplete="username" name="username" required autofocus />
            <v-text-field v-model="oraclePassword" :type="showPassword ? 'text' : 'password'" label="Contraseña" placeholder="Tu contraseña de Oracle" prepend-inner-icon="mdi-lock-outline" :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'" autocomplete="current-password" name="password" required @click:append-inner="showPassword = !showPassword" />
            <p v-if="authError" class="login-error" role="alert"><v-icon icon="mdi-alert-circle-outline" size="17"/>{{ authError }}</p>
            <v-btn class="login-submit" type="submit" color="primary" size="large" block :loading="signingIn" :disabled="!oracleUser.trim() || !oraclePassword">Iniciar sesión <v-icon icon="mdi-arrow-right" end/></v-btn>
          </form>
          <div class="login-security"><v-icon icon="mdi-shield-lock-outline" size="17"/><span>Tus credenciales solo validan el acceso; necesitas el rol SPD_EXTERNOS.</span></div>
        </template>
        <footer class="login-footer">Hecho con cuidado para tus documentos <span>✳</span></footer>
      </section>
    </main>
    <div v-else class="app-shell" @dragenter="onDragEnter" @dragleave="onDragLeave" @dragover.prevent @drop="onDrop">
      <aside class="sidebar">
        <a class="brand" href="#" aria-label="SPD MSP" @click.prevent="activePage='ingesta'"><span class="brand-mark"><v-icon icon="mdi-book-open-page-variant" size="21" /></span><span>SPD MSP</span></a>
        <div class="nav-label">EXPEDIENTES</div>
        <p class="nav-hint">Usa solo las opciones que necesite cada lote.</p>
        <button class="nav-item" :class="{selected:activePage==='ingesta'}" aria-label="Recibir planillas" title="Recibir planillas" @click="activePage='ingesta'"><v-icon icon="mdi-cloud-upload-outline" size="19"/><span>Recibir planillas</span></button>
        <button class="nav-item" :class="{selected:activePage==='patient-documents'}" aria-label="Revisar documentos del paciente" title="Revisar documentos del paciente" @click="openPatientDocumentsPage"><v-icon icon="mdi-folder-account-outline" size="19"/><span>Revisar documentos del paciente</span></button>
        <button class="nav-item" :class="{selected:activePage==='coberturas'}" aria-label="Hojas de cobertura" title="Hojas de cobertura" @click="openCoverageDownloads"><v-icon icon="mdi-file-download-outline" size="19"/><span>Hojas de cobertura</span></button>
        <button class="nav-item" :class="{selected:activePage==='objeciones'}" aria-label="Subsanar objeciones" title="Subsanar objeciones" @click="openObjectionsPage"><v-icon icon="mdi-file-alert-outline" size="19"/><span>Subsanar objeciones</span></button>
        <button class="nav-item" :class="{selected:activePage==='zip-download'}" aria-label="Descargar expediente ZIP" title="Descargar expediente ZIP" @click="openZipDownloadPage"><v-icon icon="mdi-folder-zip-outline" size="19"/><span>Descargar expediente ZIP</span></button>
        <div class="nav-label">BIBLIOTECA LOCAL</div>
        <button class="nav-item" :class="{selected:activePage==='documents'}" aria-label="Mis documentos guardados en este navegador" title="Mis documentos" @click="activePage='documents'"><v-icon icon="mdi-folder-multiple-outline" size="19"/><span>Mis documentos</span></button>
        <div class="nav-label">CONSULTAS</div>
        <button class="nav-item" :class="{selected:activePage==='planilla'}" aria-label="Consultar planilla digital" title="Consultar planilla digital" @click="openPlanilla"><v-icon icon="mdi-table-large" size="19"/><span>Planilla digital</span></button>
        <div class="sidebar-bottom"><button class="profile" @click="signOut"><span class="avatar">{{ displayUser.slice(0,1) }}</span><span class="profile-copy"><b>{{ displayUser }}</b><small>Cerrar sesión</small></span><v-icon icon="mdi-logout" size="18"/></button></div>
      </aside>
      <main class="main-area">
        <header class="topbar"><div class="breadcrumbs"><span>SPD MSP</span><v-icon icon="mdi-chevron-right" size="16"/><b>{{ title }}</b></div><div class="top-actions"><span class="avatar top-avatar">{{ displayUser.slice(0,1) }}</span></div></header>
        <section v-if="activePage==='documents'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">BIBLIOTECA LOCAL</div><h1>{{ title }}<span class="title-period">.</span></h1><p class="subtitle">Busca, organiza y abre documentos guardados en este navegador.</p></div><button class="primary-upload" @click="uploadInput?.click()"><v-icon icon="mdi-upload" size="18"/> Subir documento</button></div>
          <div class="stats-row"><div class="stat-card"><span class="stat-icon green"><v-icon icon="mdi-file-multiple-outline"/></span><div><span class="stat-label">Documentos</span><strong>{{ docs.length }} <small>archivos</small></strong></div></div><div class="stat-card"><span class="stat-icon peach"><v-icon icon="mdi-folder-multiple-outline"/></span><div><span class="stat-label">Espacios</span><strong>{{ spaces.length }} <small>activos</small></strong></div></div></div>
          <section class="recent-section"><div class="section-heading"><div><h2>{{ activeFolder==='Todos los documentos' ? 'Tus archivos' : 'Archivos' }} <span class="muted-count">{{ visibleDocs.length }}</span></h2><p>Organiza y encuentra lo que buscas.</p></div><button class="text-action" @click="activeFolder=folders[0]">Ver todo <v-icon icon="mdi-arrow-right" size="16"/></button></div>
            <div class="toolbar"><div class="search-wrap"><v-icon icon="mdi-magnify" size="19"/><input ref="searchInput" v-model="query" placeholder="Buscar documentos..." aria-label="Buscar documentos"/><kbd>⌘ K</kbd></div><div class="toolbar-right"><v-select v-model="sortBy" :items="['Recientes','Nombre','Tamaño']" prepend-inner-icon="mdi-sort" class="sort-select" aria-label="Ordenar documentos"/><div class="view-switch"><button :class="{active:view==='grid'}" aria-label="Vista de cuadrícula" @click="view='grid'"><v-icon icon="mdi-view-grid-outline" size="18"/></button><button :class="{active:view==='list'}" aria-label="Vista de lista" @click="view='list'"><v-icon icon="mdi-view-list-outline" size="19"/></button></div></div></div>
            <div class="filter-row"><v-chip-group v-model="typeFilter" selected-class="filter-chip-selected" mandatory color="primary"><v-chip value="Todo" size="small" variant="outlined" filter>Todos los tipos</v-chip><v-chip value="pdf" size="small" variant="outlined" prepend-icon="mdi-file-pdf-box">PDF</v-chip><v-chip value="word" size="small" variant="outlined" prepend-icon="mdi-file-word">Word</v-chip><v-chip value="sheet" size="small" variant="outlined" prepend-icon="mdi-file-excel">Hojas de cálculo</v-chip><v-chip value="image" size="small" variant="outlined" prepend-icon="mdi-image-outline">Imágenes</v-chip></v-chip-group><v-select v-model="dateFilter" class="date-filter" :items="['Cualquier fecha','Últimos 7 días','Últimos 30 días']" prepend-inner-icon="mdi-calendar-blank-outline" aria-label="Filtrar por fecha"/><v-chip v-if="query" size="small" closable variant="tonal" color="secondary" @click:close="query=''">“{{ query }}”</v-chip></div>
            <div v-if="selectedIds.length" class="selection-bar"><v-chip size="small" color="primary" variant="tonal" prepend-icon="mdi-check-circle-outline">{{ selectedIds.length }} seleccionados</v-chip><v-btn size="small" variant="text" @click="selectAll">{{ selectedIds.length === visibleDocs.length ? 'Quitar selección' : 'Seleccionar todos' }}</v-btn><v-spacer/><v-tooltip text="Mover a un espacio"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-folder-move-outline" size="small" variant="text" aria-label="Mover seleccionados" @click="startMove()"/></template></v-tooltip><v-tooltip text="Añadir a favoritos"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-star-outline" size="small" variant="text" aria-label="Favoritos" @click="bulkFavorite"/></template></v-tooltip><v-tooltip text="Eliminar"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-delete-outline" size="small" variant="text" color="error" aria-label="Eliminar seleccionados" @click="bulkDelete"/></template></v-tooltip><v-btn size="small" variant="text" @click="selectedIds=[]">Cancelar</v-btn></div>
            <div v-if="loading" class="empty-state">Cargando tus documentos…</div>
            <div v-else-if="!visibleDocs.length" class="empty-state"><span class="empty-icon"><v-icon icon="mdi-folder-search-outline" size="32"/></span><h3>No encontramos documentos</h3><p>Prueba con otra búsqueda o añade un documento a tu espacio.</p><button class="primary-upload" @click="uploadInput?.click()"><v-icon icon="mdi-upload" size="18"/> Subir documento</button></div>
            <div v-else class="document-grid" :class="{'list-view':view==='list'}"><article v-for="doc in visibleDocs" :key="doc.id" class="document-card" :class="{'is-selected':selectedIds.includes(doc.id)}" @click="openPreview(doc)"><div class="doc-cover" :class="'cover-'+doc.color"><div class="paper-sheet"><div class="sheet-header"><span class="file-badge" :class="'badge-'+doc.color">{{ doc.ext }}</span><v-menu location="bottom end"><template #activator="{ props }"><button v-bind="props" class="more-btn" aria-label="Opciones del documento" @click.stop><v-icon icon="mdi-dots-horizontal" size="19"/></button></template><v-list density="compact" min-width="185" class="doc-menu"><v-list-item prepend-icon="mdi-eye-outline" title="Vista previa" @click="openPreview(doc)"/><v-list-item :prepend-icon="doc.favorite ? 'mdi-star' : 'mdi-star-outline'" :title="doc.favorite ? 'Quitar favorito' : 'Añadir a favoritos'" @click="toggleFavorite(doc)"/><v-list-item prepend-icon="mdi-folder-move-outline" title="Mover a…" @click="startMove(doc)"/><v-list-item prepend-icon="mdi-download-outline" title="Descargar" @click="download(doc)"/><v-divider class="my-1"/><v-list-item prepend-icon="mdi-delete-outline" title="Eliminar" class="text-error" @click="removeDoc(doc)"/></v-list></v-menu></div><div v-if="doc.color==='image'" class="image-art"><div></div><span>✳</span><i></i></div><div v-else class="fake-lines"><span class="line-title"></span><span></span><span></span><span class="line-short"></span><span class="line-gap"></span><span></span><span class="line-mid"></span></div><span class="page-corner"></span></div><v-btn class="select-doc" :class="{checked:selectedIds.includes(doc.id)}" :icon="selectedIds.includes(doc.id) ? 'mdi-check-circle' : 'mdi-checkbox-blank-circle-outline'" size="small" variant="flat" aria-label="Seleccionar documento" @click.stop="toggleSelected(doc)"/></div><div class="doc-info"><div class="doc-type-icon" :class="'type-'+doc.color"><v-icon :icon="doc.icon" size="19"/></div><div class="doc-detail"><h3>{{ doc.name }}</h3><p>{{ doc.ext }} <i>·</i> {{ doc.size }} <i>·</i> {{ doc.date }}</p></div><v-tooltip :text="doc.favorite ? 'Favorito' : 'Más opciones'"><template #activator="{ props }"><v-btn v-bind="props" class="favorite-btn" :icon="doc.favorite ? 'mdi-star' : 'mdi-star-outline'" size="small" variant="text" :color="doc.favorite ? 'amber-darken-2' : 'grey'" aria-label="Añadir a favoritos" @click.stop="toggleFavorite(doc)"/></template></v-tooltip></div></article></div>
          </section>
          <footer>Hecho con cuidado para tus documentos <span>✳</span></footer>
        </section>
        <section v-else-if="activePage==='planilla'" class="content-wrap planilla-content">
          <div class="welcome-line"><div><div class="eyebrow">DATOS DE ORACLE</div><h1>Planilla digital<span class="title-period">.</span></h1><p class="subtitle">Registros de la tabla planilla_digital.</p></div><button class="primary-upload" :disabled="planillaLoading" @click="loadPlanilla()"><v-icon icon="mdi-refresh" size="18"/>Actualizar</button></div>
          <div class="planilla-card">
            <div class="planilla-toolbar"><div><h2>Registros</h2><p v-if="planillaData.total">{{ (planillaPage-1)*10+1 }}–{{ Math.min(planillaPage*10, planillaData.total) }} de {{ planillaData.total }} filas</p><p v-else>0 filas</p></div><v-chip size="small" variant="tonal" color="primary">10 por página</v-chip></div>
            <div v-if="planillaLoading" class="planilla-state"><v-progress-circular indeterminate color="primary"/><span>Cargando registros…</span></div>
            <div v-else-if="planillaError" class="planilla-state planilla-error"><v-icon icon="mdi-alert-circle-outline"/><span>{{ planillaError }}</span><v-btn size="small" variant="text" @click="loadPlanilla()">Reintentar</v-btn></div>
            <div v-else-if="!planillaData.rows.length" class="planilla-state"><v-icon icon="mdi-table-search" size="25"/><span>La tabla no tiene registros.</span></div>
            <div v-else class="planilla-table-wrap"><v-table class="planilla-table" density="comfortable" fixed-header height="min(62vh, 620px)"><thead><tr><th v-for="column in planillaData.columns" :key="column">{{ column }}</th></tr></thead><tbody><tr v-for="(row,rowIndex) in planillaData.rows" :key="`${planillaPage}-${rowIndex}`"><td v-for="(value,columnIndex) in row" :key="columnIndex" :title="formatPlanillaValue(value)">{{ formatPlanillaValue(value) }}</td></tr></tbody></v-table></div>
            <div class="planilla-pagination"><span>Página {{ planillaPage }} de {{ planillaData.totalPages }}</span><v-pagination v-if="planillaData.totalPages>1" v-model="planillaPage" :length="planillaData.totalPages" :total-visible="5" density="comfortable" @update:model-value="loadPlanilla"/></div>
          </div>
          <footer>Hecho con cuidado para tus documentos <span>✳</span></footer>
        </section>
        <section v-else-if="activePage==='coberturas'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">COBERTURAS DEL LOTE</div><h1>Hojas de cobertura<span class="title-period">.</span></h1><p class="subtitle">Las hojas se guardan en la carpeta de cada paciente. Puedes descargar una copia ZIP si la necesitas.</p></div><v-btn type="button" variant="tonal" prepend-icon="mdi-refresh" :loading="coverageLoading || savedWorkspacesLoading" @click="openCoverageDownloads">Actualizar</v-btn></div>
          <v-card class="planilla-card coverage-card" rounded="xl" elevation="0">
            <div class="coverage-picker-actions">
              <WorkspacePicker
                :model-value="selectedCoverageWorkspace"
                :workspaces="coverageWorkspaceCandidates"
                label="Servicio y período"
                placeholder="Busca por servicio, período, estado o ID"
                icon="mdi-folder-zip-outline"
                :loading="savedWorkspacesLoading"
                :disabled="coverageLoading || coverageGenerating"
                :allow-type-filter="false"
                :service-label="coverageServiceLabel"
                :month-label="coverageMonthLabel"
                :status-label="workspaceStatusLabel"
                :received-at="workspaceReceivedAt"
                empty-message="Coberturas está disponible para períodos de Recepción preparados o incompletos."
                @update:model-value="loadCoveragePlanillas"
              />
              <div class="coverage-picker-buttons">
                <v-btn color="primary" prepend-icon="mdi-content-save-outline" :loading="coverageGenerating" :disabled="!coverageSelectedIds.length || coverageGenerating" @click="generateCoverageSheets">Generar y guardar ({{ coverageSelectedIds.length }})</v-btn>
                <v-btn variant="tonal" prepend-icon="mdi-download-outline" :loading="coverageGenerating" :disabled="!coverageSelectedIds.length || coverageSelectedIds.length > 500 || coverageGenerating" @click="downloadSelectedCoverageSheets">Descargar ZIP ({{ coverageSelectedIds.length }})</v-btn>
              </div>
            </div>
            <div v-if="coverageProgress" class="coverage-notice"><v-progress-circular indeterminate color="primary" size="18"/><span>{{ coverageProgress }}</span></div>
            <div v-if="coverageNotice" class="coverage-notice"><v-icon icon="mdi-information-outline"/><span>{{ coverageNotice }}</span></div>
            <div v-if="coverageError" class="planilla-state planilla-error"><v-icon icon="mdi-alert-circle-outline"/><span>{{ coverageError }}</span><v-btn size="small" variant="text" @click="loadCoveragePlanillas()">Reintentar</v-btn></div>
            <div v-else-if="coverageLoading" class="planilla-state"><v-progress-circular indeterminate color="primary" size="22"/><span>Consultando planillas MSP del lote…</span></div>
            <div v-else-if="savedWorkspacesError" class="planilla-state planilla-error"><v-icon icon="mdi-alert-circle-outline"/><span>No se pudieron cargar los períodos para Coberturas: {{ savedWorkspacesError }}</span><v-btn type="button" size="small" variant="text" @click="openCoverageDownloads">Reintentar</v-btn></div>
            <div v-else-if="!selectedCoverageWorkspace" class="planilla-state"><v-icon icon="mdi-folder-search-outline"/><span>Prepara primero el ZIP en “Recibir planillas” y selecciona aquí ese mismo servicio y período.</span></div>
            <div v-else-if="!coveragePlanillas.length" class="planilla-state"><v-icon icon="mdi-file-search-outline"/><span>No se encontraron planillas MSP de ese período en las carpetas del ZIP.</span></div>
            <div v-else class="planilla-table-wrap"><v-table class="planilla-table" density="comfortable" fixed-header height="min(62vh, 620px)"><thead><tr><th><label title="Seleccionar todas las pendientes"><input type="checkbox" :checked="coverageAllSelected" :disabled="!coverageSelectableRows.length || coverageGenerating" aria-label="Seleccionar todas las planillas pendientes" @change="toggleAllCoveragePlanillas"/> Todas</label></th><th>Trámite del ZIP</th><th>Paciente</th><th>Fecha hasta</th><th>Estado de cobertura</th><th>Acción manual</th></tr></thead><tbody><tr v-for="row in coveragePlanillas" :key="row.pdi_id"><td><input type="checkbox" :checked="coverageSelectedIds.includes(row.pdi_id)" :disabled="(row.pdi_cobertura === 'S' && !row.hoja_generada) || coverageGenerating" :aria-label="`Seleccionar planilla ${row.pdi_tramite}`" @change="toggleCoveragePlanilla(row)"/></td><td>{{ row.pdi_tramite }}</td><td>{{ row.paciente || '—' }}</td><td>{{ row.fecha_hasta || '—' }}</td><td><v-chip size="small" :color="row.pdi_cobertura === 'S' ? 'success' : row.hoja_generada ? 'info' : (row.descarga_manual || row.motivo_manual) ? 'error' : 'warning'" variant="tonal">{{ row.pdi_cobertura === 'S' ? 'Generada' : row.hoja_generada ? 'PDF en expediente · Oracle pendiente' : row.descarga_manual ? 'Descarga manual' : row.motivo_manual ? 'Revisar datos' : 'Pendiente' }}</v-chip><small v-if="row.motivo_manual" class="coverage-failure">{{ row.motivo_manual }}</small></td><td><div v-if="row.descarga_manual" class="coverage-manual"><a href="https://coberturasalud.msp.gob.ec/" target="_blank" rel="noopener noreferrer">Abrir portal MSP</a><small v-if="row.coberturas_manual?.length">Descarga un PDF por cédula: {{ row.coberturas_manual.map(member => member.cedula).join(', ') }}</small><label class="coverage-manual-upload"><input type="file" accept="application/pdf,.pdf" multiple :disabled="coverageManualUploadingId === row.pdi_id" @change="uploadManualCoverageSheets(row,$event)"/>{{ coverageManualUploadingId === row.pdi_id ? 'Adjuntando…' : 'Adjuntar PDFs descargados' }}</label></div><span v-else>—</span></td></tr></tbody></v-table></div>
            <div v-if="coveragePlanillas.length" class="coverage-footnote">Marca la casilla del encabezado para seleccionar todas las pendientes. Se procesan en grupos de 10; las hojas generadas se guardan en cada expediente. El ZIP admite hasta 500 planillas.</div>
          </v-card>
        </section>
        <section v-else-if="activePage==='objeciones'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">SUBSANACIÓN DE PLANILLAS OBJETADAS</div><h1>Subsanar objeciones<span class="title-period">.</span></h1><p class="subtitle">Parte de un período de primer ingreso preparado y crea un espacio separado para trabajar los trámites objetados.</p></div><v-btn type="button" variant="tonal" prepend-icon="mdi-refresh" :loading="savedWorkspacesLoading || objectionBusy" @click="openObjectionsPage">Actualizar</v-btn></div>
          <v-alert class="mb-4" type="info" variant="tonal" density="comfortable" prepend-icon="mdi-shield-check-outline">Tú defines qué pacientes fueron objetados. Solo los seleccionados se copian al nuevo espacio; el período original permanece intacto y Oracle no se modifica.</v-alert>
          <v-alert v-if="!savedWorkspacesLoading && !objectionSourceItems.length" class="mb-4" type="warning" variant="tonal" density="comfortable">Primero prepara un período de primer ingreso en “Recibir planillas”.</v-alert>
          <v-card class="planilla-card saved-workspaces-card mb-5" rounded="xl" elevation="0">
            <div class="ingest-card-heading"><span class="ingest-step">1</span><div><h2>Cargar pacientes del período</h2><p>Elige el primer ingreso del que salieron las objeciones. Los documentos de cabecera se cargan después de crear el espacio.</p></div></div>
            <div class="objection-source-picker-row">
              <WorkspacePicker
                v-model="objectionSourceWorkspace"
                :workspaces="objectionSourceWorkspaces"
                label="Período de primer ingreso"
                placeholder="Busca por servicio, período, estado o ID"
                icon="mdi-folder-open-outline"
                :disabled="objectionBusy"
                :loading="savedWorkspacesLoading"
                :allow-type-filter="false"
                :service-label="coverageServiceLabel"
                :month-label="coverageMonthLabel"
                :status-label="workspaceStatusLabel"
                :received-at="workspaceReceivedAt"
                empty-message="Selecciona un período de Recepción preparado o incompleto como primer ingreso."
                @update:model-value="resetObjectionPreview"
              />
              <v-btn type="button" variant="tonal" prepend-icon="mdi-account-search-outline" :loading="objectionBusy" :disabled="!objectionSourceWorkspace || objectionBusy" @click="previewObjections">Cargar pacientes</v-btn>
            </div>
            <v-alert v-if="objectionError" class="mt-3" type="error" variant="tonal" density="comfortable">{{ objectionError }}</v-alert>
            <v-alert v-if="objectionNotice" class="mt-3" type="success" variant="tonal" density="comfortable">{{ objectionNotice }}</v-alert>
          </v-card>
          <v-card v-if="objectionPreview" class="planilla-card saved-workspaces-card mb-5" rounded="xl" elevation="0">
            <div class="ingest-card-heading"><span class="ingest-step">2</span><div><h2>Selecciona los objetados</h2><p>Elige uno o varios pacientes de la lista. El informe de liquidación se guarda como respaldo, sin leerlo.</p></div></div>
            <v-autocomplete v-model="objectionSelectedTramites" :items="objectionComboItems" label="Pacientes objetados" placeholder="Busca por nombre, trámite o cédula" prepend-inner-icon="mdi-account-multiple-check-outline" multiple chips closable-chips clearable hide-selected :disabled="objectionBusy || !objectionComboItems.length" no-data-text="No hay pacientes disponibles en este período"/>
            <div v-if="!objectionSelectionCandidates.length" class="planilla-state"><v-icon icon="mdi-account-search-outline"/><span>No hay pacientes con PDFs vinculados al período preparado.</span></div>
            <div v-else-if="objectionSelectedCandidates.length" class="planilla-state mt-2"><v-icon icon="mdi-account-check-outline"/><span>{{ objectionSelectedCandidates.length }} trámite{{ objectionSelectedCandidates.length === 1 ? '' : 's' }} seleccionado{{ objectionSelectedCandidates.length === 1 ? '' : 's' }}. Revisa los datos en las etiquetas antes de crear el espacio.</span></div>
            <v-alert v-if="objectionPeriodSpaces.length===1" class="mt-3" type="info" variant="tonal" density="comfortable">Ya existe {{ objectionPeriodSpaces[0].job_id }} para este servicio y período. Al continuar se reutilizará y solo se agregarán los trámites que falten.</v-alert>
            <v-alert v-if="objectionPeriodSpaces.length>1" class="mt-3" type="warning" variant="tonal" density="comfortable">
              <div>{{ objectionConflictMessage || 'Hay varios espacios de Objeciones para este servicio y período. Elige uno para continuar; no se borrarán ni fusionarán automáticamente.' }}</div>
              <div class="objection-existing-choices">
                <v-btn v-for="workspace in objectionPeriodSpaces" :key="workspace.job_id" type="button" size="small" variant="outlined" :color="objectionSelectedWorkspace===workspace.job_id ? 'primary' : undefined" :loading="objectionBusy && objectionSelectedWorkspace===workspace.job_id" :disabled="objectionBusy" @click="chooseObjectionWorkspace(workspace.job_id)">
                  {{ workspaceTypeLabel(workspace) }} · {{ coverageServiceLabel(workspace.tipo_servicio) }} · {{ coverageMonthLabel(workspaceMonthValue(workspace)) }} {{ workspaceYearLabel(workspace) }} · {{ workspaceStatusLabel(workspace.status, workspace) }} · ID {{ workspace.job_id }} · {{ workspaceReceivedAt(workspace.received_at) || 'sin fecha' }}
                </v-btn>
              </div>
            </v-alert>
            <div class="ingest-card-heading mt-4"><span class="ingest-step">3</span><div><h2>Crear espacio separado</h2><p>Los documentos de cabecera se cargan después; no bloquean la selección ni la creación.</p></div></div>
            <div class="workspace-download-row mt-3"><span>Se copiarán únicamente los trámites seleccionados. Podrás agregar los que hayas omitido.</span><v-btn type="button" color="primary" prepend-icon="mdi-folder-plus-outline" :loading="objectionBusy" :disabled="!objectionCanCreate || objectionBusy" @click="createObjectionWorkspace">{{ objectionCreateLabel }}</v-btn></div>
          </v-card>
          <v-card class="planilla-card saved-workspaces-card" rounded="xl" elevation="0">
            <div class="planilla-toolbar"><div><h2>Espacios existentes</h2><p>Los anexos y las correcciones se guardan solo en el espacio derivado.</p></div></div>
            <div class="objection-workspace-picker-row">
              <WorkspacePicker
                v-model="objectionSelectedWorkspace"
                :workspaces="objectionWorkspaceWorkspaces"
                label="Espacio de Objeciones"
                placeholder="Busca por servicio, período, estado o ID"
                icon="mdi-folder-alert-outline"
                :disabled="objectionBusy || workspaceDeleting"
                :loading="savedWorkspacesLoading"
                :allow-type-filter="false"
                :service-label="coverageServiceLabel"
                :month-label="coverageMonthLabel"
                :status-label="workspaceStatusLabel"
                :received-at="workspaceReceivedAt"
                empty-message="Aún no hay espacios separados de Objeciones preparados o incompletos."
                @update:model-value="loadObjectionWorkspace"
              />
              <div class="objection-workspace-picker-buttons">
                <v-btn type="button" color="primary" variant="tonal" prepend-icon="mdi-folder-open-outline" :disabled="!objectionSelectedWorkspace || objectionBusy || workspaceDeleting" @click="loadObjectionWorkspace(objectionSelectedWorkspace)">Abrir</v-btn>
                <v-btn v-if="canDeleteWorkspaces" type="button" color="error" variant="tonal" prepend-icon="mdi-delete-outline" :disabled="!objectionSelectedWorkspace || objectionBusy || workspaceDeleting" @click="requestDeleteWorkspace(objectionSelectedWorkspace)">Eliminar</v-btn>
              </div>
            </div>
            <div v-if="!objectionWorkspaceItems.length && !savedWorkspacesLoading" class="planilla-state"><v-icon icon="mdi-folder-search-outline" size="24"/><span>Aún no hay espacios de objeciones.</span></div>
            <template v-if="objectionCurrentWorkspace">
              <v-alert class="mt-3" type="info" variant="tonal" density="comfortable">El ZIP se llamará {{ objectionCurrentWorkspace.job_id }}_OBJECIONES. Los PDFs del primer ingreso se copiaron a este espacio; marca aquí cuáles enviar. P_INDIVIDUAL.pdf y C_COBERTURA.pdf son obligatorios. Para reemplazar o renombrar documentos, abre «Revisar y corregir expediente». Los anexos son opcionales.</v-alert>
              <v-card class="planilla-card mt-4" rounded="xl" elevation="0">
                <div class="planilla-toolbar"><div><h3>Documentos para completar el ZIP</h3><p>No hacen falta para crear, abrir ni continuar el espacio. Quedan pendientes y puedes cargarlos después; se solicitan al preparar la descarga final.</p></div></div>
                <div v-for="header in objectionHeaderOptions" :key="header.tipo" class="coverage-toolbar objection-header-row">
                  <label class="coverage-manual-upload">{{ header.nombre }}<input type="file" :accept="header.tipo === 'matriz' ? '.xlsm,application/vnd.ms-excel.sheet.macroEnabled.12' : '.pdf,application/pdf'" :disabled="objectionBusy" @change="setObjectionInput(header.tipo,$event)"/><small>{{ objectionHeaderFiles[header.tipo]?.name || (header.cargado ? 'Cargado; selecciona para reemplazar' : 'Pendiente de carga') }}</small></label>
                  <v-btn type="button" size="small" variant="tonal" :loading="objectionBusy" :disabled="!objectionHeaderFiles[header.tipo] || objectionBusy" @click="uploadObjectionHeader(header.tipo)">{{ header.cargado ? 'Reemplazar' : 'Guardar' }}</v-btn>
                  <v-chip size="small" :color="header.cargado ? 'success' : 'warning'" variant="tonal">{{ header.cargado ? 'Listo' : 'Pendiente' }}</v-chip>
                </div>
                <small class="objection-scan-hint">La matriz oficial .xlsm debe detallar por trámite el valor objetado, código, motivo y respuesta técnica. Folio no genera ni interpreta sus celdas.</small>
              </v-card>
              <v-alert v-if="objectionPDFError" class="mt-4" type="error" variant="tonal" density="comfortable">
                <div class="objection-pdf-error-row"><span>No se pudo consultar la lista de PDFs. El espacio de Objeciones sí está creado y disponible; este problema es independiente de los documentos para completar el ZIP.</span><v-btn type="button" size="small" variant="tonal" :loading="objectionPDFLoading" :disabled="objectionPDFLoading" @click="loadObjectionPDFSelection(objectionCurrentWorkspace.job_id)">Reintentar</v-btn></div>
              </v-alert>
              <v-card class="planilla-card mt-4" rounded="xl" elevation="0">
                <div class="planilla-toolbar"><div><h3>¿Olvidaste seleccionar un paciente?</h3><p>Puedes añadirlo aquí al mismo espacio. Si ya cargaste la matriz, tendrás que subirla de nuevo con el trámite incluido.</p></div><v-btn type="button" variant="tonal" prepend-icon="mdi-account-search-outline" :loading="objectionBusy" :disabled="objectionBusy" @click="previewObjectionAddCandidates">Buscar pacientes</v-btn></div>
                <template v-if="objectionAddPreview">
                  <v-autocomplete v-model="objectionAddSelectedTramites" :items="objectionAddCandidateItems" label="Pacientes para agregar" placeholder="Busca por nombre, trámite o cédula" multiple chips closable-chips clearable hide-selected :disabled="objectionBusy || !objectionAddCandidateItems.length" no-data-text="No quedan trámites por agregar"/>
                  <div class="coverage-toolbar"><span class="coverage-footnote">Cada atención se agrega con su propia carpeta identificada por trámite.</span><v-btn type="button" color="primary" variant="tonal" prepend-icon="mdi-account-plus-outline" :loading="objectionBusy" :disabled="!objectionAddSelectedTramites.length || objectionBusy" @click="addObjectionPatients">Agregar seleccionados</v-btn></div>
                </template>
              </v-card>
              <div class="planilla-table-wrap mt-3"><v-table class="planilla-table" density="comfortable" fixed-header height="min(34vh, 330px)"><thead><tr><th>Paciente seleccionado</th><th>Trámite</th><th>Cédula</th><th>Postura</th><th>Cobertura</th></tr></thead><tbody><tr v-for="row in objectionRowsPage" :key="row.pdi_tramite"><td>{{ row.paciente }}</td><td>{{ row.pdi_tramite }}</td><td>{{ row.pdi_cedula || '—' }}</td><td>{{ row.postura || 'Pendiente' }}</td><td>{{ row.cobertura_adjunta ? 'Adjunta' : 'Pendiente' }}</td></tr></tbody></v-table></div>
              <div class="objection-patient-controls">
                <v-text-field v-model="objectionPatientSearch" label="Buscar paciente objetado" placeholder="Nombre, trámite o cédula" prepend-inner-icon="mdi-magnify" density="comfortable" variant="outlined" clearable hide-details/>
                <span>{{ filteredObjectionPatients.length }} paciente(s) coinciden</span>
              </div>
              <div class="objection-patient-list">
                <article v-for="patient in objectionPatientsPage" :key="patient.tramite" class="saved-workspace-card">
                  <div class="saved-workspace-card-main"><span class="saved-workspace-icon"><v-icon icon="mdi-account-alert-outline" size="22"/></span><div class="saved-workspace-card-copy"><h3>{{ patient.patient || 'Paciente' }}</h3><p class="saved-workspace-message">Trámite {{ patient.tramite }} · {{ patient.rows.length }} observación(es)</p></div></div>
                  <div class="objection-document-actions">
                    <v-btn type="button" size="small" variant="tonal" prepend-icon="mdi-file-eye-outline" :disabled="objectionBusy" @click="openObjectionPatientDocuments(patient)">Revisar y corregir expediente</v-btn>
                    <v-select :model-value="objectionPostures[String(patient.tramite)] || ''" :items="['ACEPTA','RECHAZA']" label="Postura" density="compact" hide-details @update:model-value="setObjectionPostureSelection(patient.tramite,$event)"/>
                    <v-btn type="button" size="small" color="primary" variant="tonal" :loading="objectionPostureSaving===String(patient.tramite)" :disabled="!['ACEPTA','RECHAZA'].includes(objectionPostures[String(patient.tramite)]) || Boolean(objectionPostureSaving)" @click="saveObjectionPosture(patient)">Guardar postura</v-btn>
                    <div class="objection-zip-documents">
                      <strong>PDFs que se incluirán en el ZIP</strong>
                      <p>Marca los documentos clínicos que correspondan al descargo. Solo los seleccionados entran en el ZIP; los obligatorios siempre se incluyen.</p>
                      <label v-for="document in objectionPatientPDFs(patient)" :key="document.path" class="objection-zip-document">
                        <input type="checkbox" :checked="document.incluido_en_zip" :disabled="document.obligatorio || objectionPDFSavingPath===document.path || Boolean(objectionPDFSavingPath)" @change="setObjectionPDFIncluded(document,$event.target.checked)"/>
                        <span><strong>{{ document.name }}</strong><small>{{ document.obligatorio ? 'Obligatorio' : prettySize(document.size_bytes) }}</small></span>
                        <v-progress-circular v-if="objectionPDFSavingPath===document.path" indeterminate size="16" width="2"/>
                      </label>
                      <div v-if="objectionPDFLoading && !objectionPatientPDFs(patient).length" class="workspace-empty">Cargando PDFs del trámite…</div>
                      <div v-else-if="!objectionPDFError && !objectionPatientPDFs(patient).length" class="workspace-empty">No hay PDFs copiados para este trámite.</div>
                    </div>
                    <v-select v-model="objectionAnnexTypeByPatient[patient.tramite]" :items="objectionAnnexTypes" label="Tipo de justificativo" density="compact" hide-details/>
                    <v-text-field v-if="objectionAnnexTypeByPatient[patient.tramite]==='OTRO'" :model-value="objectionAnnexNameByPatient[patient.tramite] || ''" label="Nombre descriptivo" placeholder="INFORME_COMPLEMENTARIO" density="compact" hide-details @update:model-value="setObjectionAnnexName(patient.tramite,$event)"/>
                    <label class="coverage-manual-upload">PDF para 5. ANEXOS<input type="file" accept=".pdf,application/pdf" :disabled="objectionBusy" @change="setObjectionUpload($event,patient.tramite,'anexo')"/><small>{{ objectionUploadFiles[`${patient.tramite}:anexo`]?.name || 'Seleccionar PDF' }}</small></label>
                    <small class="objection-scan-hint">Los anexos que guardes se incluirán automáticamente en el ZIP.</small>
                    <small class="objection-scan-hint">Escanea entre 72 y 300 DPI; el sistema valida PDF, pero no mide su DPI.</small>
                    <v-btn type="button" size="small" variant="tonal" :loading="objectionBusy" :disabled="!objectionUploadFiles[`${patient.tramite}:anexo`] || !objectionAnnexTypeByPatient[patient.tramite] || (objectionAnnexTypeByPatient[patient.tramite]==='OTRO' && !objectionAnnexNameByPatient[patient.tramite]?.trim()) || objectionBusy" @click="uploadObjectionDocument(patient,'anexo')">Guardar anexo</v-btn>
                  </div>
                </article>
              </div>
              <div v-if="!filteredObjectionPatients.length" class="workspace-empty">No hay pacientes con ese nombre, trámite o cédula. Borra la búsqueda para volver a ver todos.</div>
              <div class="planilla-pagination objection-patient-pagination">
                <span>Pacientes {{ objectionPatientRangeStart }}–{{ objectionPatientRangeEnd }} de {{ filteredObjectionPatients.length }}</span>
                <v-pagination v-if="objectionPatientPageCount > 1" v-model="objectionPatientPage" :length="objectionPatientPageCount" :total-visible="5" density="comfortable"/>
              </div>
            </template>
          </v-card>
        </section>
        <section v-else-if="activePage==='patient-documents'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">DOCUMENTOS DEL PACIENTE</div><h1>Revisar documentos del paciente<span class="title-period">.</span></h1><p class="subtitle">Elige un período guardado para revisar las carpetas de pacientes y sus PDFs.</p></div></div>
          <v-card class="planilla-card saved-workspaces-card" rounded="xl" elevation="0">
            <div class="patient-documents-picker-row">
              <WorkspacePicker
                v-model="selectedSavedWorkspace"
                :workspaces="orderedSavedWorkspaces"
                label="Período de los documentos"
                placeholder="Busca por tipo, servicio, período, estado, usuario o ID"
                icon="mdi-folder-clock-outline"
                :loading="savedWorkspacesLoading"
                :service-label="coverageServiceLabel"
                :month-label="coverageMonthLabel"
                :status-label="workspaceStatusLabel"
                :received-at="workspaceReceivedAt"
                empty-message="No hay períodos guardados para abrir documentos del paciente."
              />
              <v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar períodos guardados" :loading="savedWorkspacesLoading" :disabled="savedWorkspacesLoading" @click="loadSavedWorkspaces"/>
            </div>
            <v-alert v-if="savedWorkspacesError" class="mt-4" type="error" variant="tonal" density="comfortable">{{ savedWorkspacesError }}<v-btn type="button" size="small" variant="text" @click="loadSavedWorkspaces">Reintentar</v-btn></v-alert>
            <div v-else-if="savedWorkspacesLoading" class="planilla-state"><v-progress-circular indeterminate color="primary" size="22"/><span>Cargando períodos guardados…</span></div>
            <div v-else-if="!savedWorkspaces.length" class="planilla-state"><v-icon icon="mdi-folder-search-outline" size="25"/><span>No hay períodos guardados todavía.</span><v-btn type="button" variant="text" color="primary" @click="activePage='ingesta'">Recibir planillas</v-btn></div>
            <template v-else>
              <v-alert v-if="patientDocumentsError" class="mt-4" type="warning" variant="tonal" density="comfortable">{{ patientDocumentsError }}</v-alert>
              <div class="coverage-toolbar"><span class="coverage-footnote">Al abrir un período podrás elegir una carpeta de paciente y revisar sus PDFs.</span><v-btn type="button" color="primary" prepend-icon="mdi-file-eye-outline" :loading="openingSavedWorkspace" :disabled="!selectedSavedWorkspace || openingSavedWorkspace" @click="openPatientDocumentsWorkspace()">Abrir documentos</v-btn></div>
            </template>
          </v-card>
        </section>
        <section v-else-if="activePage==='zip-download'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">EXPORTAR EXPEDIENTE</div><h1>Descargar expediente ZIP<span class="title-period">.</span></h1><p class="subtitle">Elige por separado un período de recepción o un espacio de Objeciones. La descarga normal incluye todos los archivos disponibles.</p></div></div>
          <v-card class="planilla-card saved-workspaces-card" rounded="xl" elevation="0">
            <v-btn-toggle :model-value="zipDownloadType" class="ingest-mode-switch zip-download-kind-switch" color="primary" divided mandatory rounded="lg" aria-label="Tipo de espacio para descargar" @update:model-value="selectZipDownloadType">
              <v-btn type="button" value="RECEPCION" prepend-icon="mdi-folder-open-outline">Recepción de planillas <span class="ingest-mode-count">{{ zipDownloadTypeCounts.RECEPCION }}</span></v-btn>
              <v-btn type="button" value="OBJECIONES" prepend-icon="mdi-file-alert-outline">Objeciones <span class="ingest-mode-count">{{ zipDownloadTypeCounts.OBJECIONES }}</span></v-btn>
            </v-btn-toggle>
            <div class="zip-download-picker-row">
              <WorkspacePicker
                v-model="selectedZipWorkspace"
                :workspaces="zipDownloadWorkspaces"
                :label="zipDownloadType === 'OBJECIONES' ? 'Espacio de Objeciones para descargar' : 'Período de recepción para descargar'"
                placeholder="Busca por servicio, período, estado, usuario o ID"
                icon="mdi-folder-zip-outline"
                :loading="savedWorkspacesLoading"
                :disabled="zipDownloading"
                :service-label="coverageServiceLabel"
                :month-label="coverageMonthLabel"
                :status-label="workspaceStatusLabel"
                :received-at="workspaceReceivedAt"
                :empty-message="zipDownloadType === 'OBJECIONES' ? 'No hay espacios de Objeciones preparados para descargar.' : 'No hay períodos de recepción preparados para descargar.'"
              />
              <v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar expedientes preparados" :loading="savedWorkspacesLoading" :disabled="savedWorkspacesLoading || zipDownloading" @click="openZipDownloadPage"/>
            </div>
            <v-alert v-if="savedWorkspacesError" class="mt-4" type="error" variant="tonal" density="comfortable">{{ savedWorkspacesError }}<v-btn type="button" size="small" variant="text" @click="openZipDownloadPage">Reintentar</v-btn></v-alert>
            <div v-else-if="savedWorkspacesLoading" class="planilla-state"><v-progress-circular indeterminate color="primary" size="22"/><span>Cargando expedientes…</span></div>
            <div v-else-if="!zipDownloadWorkspaceItems.length" class="planilla-state"><v-icon icon="mdi-folder-search-outline" size="25"/><span v-if="zipDownloadType === 'OBJECIONES'">No hay espacios de Objeciones preparados para descargar.</span><span v-else>No hay períodos de recepción preparados todavía. Prepara el expediente desde Recepción de planillas; luego aparecerá aquí.</span><v-btn type="button" variant="text" color="primary" @click="activePage='ingesta'">Recibir planillas</v-btn></div>
            <template v-else>
              <v-alert v-if="zipDownloadError" class="mt-4" type="warning" variant="tonal" density="comfortable">{{ zipDownloadError }}<v-btn type="button" size="small" variant="text" @click="openPatientDocumentsPage">Revisar documentos del paciente</v-btn></v-alert>
              <v-alert v-if="zipDownloadNotice" class="mt-4" :type="zipDownloadArchiveState === 'INCOMPLETE' ? 'warning' : 'success'" variant="tonal" density="comfortable">{{ zipDownloadNotice }}</v-alert>
              <v-alert v-if="selectedZipWorkspaceRecord" class="zip-readiness-alert mt-4" :type="selectedZipWorkspaceReady ? 'success' : 'warning'" variant="tonal" density="comfortable" :title="selectedZipWorkspaceReady ? 'Listo para entrega' : 'Avance incompleto'">
                <p v-if="selectedZipWorkspaceReady">Este paquete cumple los requisitos actuales para preparar la entrega.</p>
                <p v-else>Puedes descargar ahora los archivos disponibles. El expediente seguirá abierto para completar estos requisitos:</p>
                <ul v-if="selectedZipWorkspaceMissing.length" class="zip-readiness-list">
                  <li v-for="item in selectedZipWorkspaceMissing" :key="item">{{ item }}</li>
                </ul>
              </v-alert>
              <div class="workspace-download-row"><span>{{ selectedZipWorkspaceRecord?.es_objeciones ? 'Se incluirán los PDFs clínicos marcados para Objeciones y los documentos obligatorios.' : 'Se incluirán todos los documentos disponibles del período de recepción.' }} {{ selectedZipWorkspaceReady ? 'El paquete cumple los requisitos actuales.' : 'La descarga se identificará como avance incompleto.' }}</span><v-btn type="button" color="primary" prepend-icon="mdi-folder-zip-outline" :loading="zipDownloading" :disabled="!selectedZipWorkspace || zipDownloading" @click="downloadWorkspaceZIP">{{ selectedZipWorkspaceRecord?.es_objeciones ? 'Descargar ZIP de Objeciones' : selectedZipWorkspaceReady ? 'Descargar ZIP listo para entrega' : 'Descargar ZIP de avance incompleto' }}</v-btn></div>
            </template>
          </v-card>
        </section>
        <section v-else class="content-wrap ingest-content" :class="{'ingest-workspace-active': !!ingestResult?.output}">
          <div v-if="!ingestResult?.output" class="welcome-line"><div><div class="eyebrow">RECEPCIÓN DE EXPEDIENTES</div><h1>Recibir lote de planillas<span class="title-period">.</span></h1><p class="subtitle">Carga el ZIP de planillas y añade los documentos habilitantes cuando los tengas.</p></div></div>
          <div v-else class="ingest-workspace-toolbar">
            <div><small>PERÍODO ABIERTO</small><strong>{{ coverageServiceLabel(ingestResult.tipo_servicio || ingestService) }} · {{ coverageMonthLabel(ingestResult.mes) }} {{ ingestResult.anio }}</strong><v-chip size="small" :color="workspaceStatusColor(ingestResult.status, ingestResult)" variant="tonal">{{ workspaceStatusLabel(ingestResult.status, ingestResult) }}</v-chip></div>
            <div class="ingest-workspace-actions"><v-btn type="button" variant="tonal" prepend-icon="mdi-information-outline" @click="showWorkspaceDetails = !showWorkspaceDetails">{{ showWorkspaceDetails ? 'Ocultar detalles' : 'Detalles del lote' }}</v-btn><v-btn type="button" color="primary" variant="text" prepend-icon="mdi-folder-clock-outline" @click="showSavedWorkspaceList">Cambiar período</v-btn></div>
          </div>
          <v-alert v-if="!ingestResult?.output" class="ingest-notice" type="info" variant="tonal" density="comfortable" prepend-icon="mdi-information-outline">{{ ingestMode === 'resume' ? 'Abre un período existente para revisar sus documentos o continuar el trabajo en el mismo espacio.' : 'Puedes empezar con el ZIP de planillas. Después añade la matriz, la planilla consolidada y el oficio al mismo expediente.' }}</v-alert>
          <v-btn-toggle v-if="!ingestResult?.output" :model-value="ingestMode" class="ingest-mode-switch" color="primary" divided mandatory rounded="lg" @update:model-value="switchIngestMode">
            <v-btn type="button" value="new" prepend-icon="mdi-plus-circle-outline">Crear período nuevo</v-btn>
            <v-btn type="button" value="resume" prepend-icon="mdi-folder-clock-outline">Abrir período guardado <span class="ingest-mode-count">{{ savedWorkspaceItems.length }}</span></v-btn>
          </v-btn-toggle>
          <form class="ingest-form" @submit.prevent="submitIngest">
            <section v-if="ingestMode === 'resume' && !ingestResult" class="ingest-card saved-workspaces-card">
              <div class="ingest-card-heading"><span class="ingest-step"><v-icon icon="mdi-folder-clock-outline" size="18"/></span><div><h2>Volver a un período guardado</h2><p>Busca un período compartido para abrirlo y continuar el trabajo, aunque lo haya recibido otra persona.</p></div><v-spacer/><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar expedientes guardados" :loading="savedWorkspacesLoading" @click="loadSavedWorkspaces"/></div>
              <div v-if="savedWorkspacesLoading" class="planilla-state saved-workspaces-loading"><v-progress-circular indeterminate color="primary" size="22"/><span>Cargando períodos guardados…</span></div>
              <v-alert v-else-if="savedWorkspacesError" type="error" variant="tonal" density="comfortable">{{ savedWorkspacesError }}<v-btn type="button" size="small" variant="text" @click="loadSavedWorkspaces">Reintentar</v-btn></v-alert>
              <div v-else-if="!savedWorkspaces.length" class="saved-workspaces-empty"><v-icon icon="mdi-folder-search-outline" size="24"/><span><strong>No hay períodos guardados todavía</strong><small>Crea un período nuevo y aparecerá aquí para que cualquier cuenta autorizada pueda continuarlo.</small></span><v-btn type="button" variant="text" color="primary" @click="switchIngestMode('new')">Crear período nuevo</v-btn></div>
              <template v-else>
                <div class="saved-workspace-filters">
                  <v-text-field v-model="savedWorkspaceSearch" label="Buscar períodos" placeholder="Mes, ID, usuario o tipo de espacio" prepend-inner-icon="mdi-magnify" density="comfortable" variant="outlined" hide-details clearable/>
                  <div class="saved-workspace-filter-grid">
                    <v-select v-model="savedWorkspaceTypeFilter" :items="savedWorkspaceTypeFilters" label="Tipo de espacio" density="comfortable" variant="outlined" hide-details :menu-props="{ contentClass: 'saved-workspace-filter-menu' }"/>
                    <v-select v-model="savedWorkspaceServiceFilter" :items="savedWorkspaceServiceFilters" label="Servicio" density="comfortable" variant="outlined" hide-details :menu-props="{ contentClass: 'saved-workspace-filter-menu' }"/>
                    <v-select v-model="savedWorkspaceYearFilter" :items="savedWorkspaceYearFilters" label="Año" density="comfortable" variant="outlined" hide-details :menu-props="{ contentClass: 'saved-workspace-filter-menu' }"/>
                    <v-select v-model="savedWorkspaceStatusFilter" :items="savedWorkspaceStatusFilters" item-title="title" item-value="value" label="Estado" density="comfortable" variant="outlined" hide-details :menu-props="{ contentClass: 'saved-workspace-filter-menu' }"/>
                  </div>
                  <div class="saved-workspace-results-toolbar">
                    <p>{{ filteredSavedWorkspaces.length }} de {{ savedWorkspaces.length }} espacios · período más reciente primero</p>
                    <v-btn v-if="hasSavedWorkspaceFilters" type="button" variant="text" prepend-icon="mdi-filter-remove-outline" @click="clearSavedWorkspaceFilters">Limpiar filtros</v-btn>
                  </div>
                </div>
                <div v-if="!filteredSavedWorkspaces.length" class="saved-workspaces-empty saved-workspaces-no-results"><v-icon icon="mdi-filter-remove-outline" size="24"/><span><strong>No hay períodos que coincidan</strong><small>Prueba otra búsqueda o combina otros filtros. Los períodos antiguos siguen disponibles al limpiar los filtros.</small></span><v-btn v-if="hasSavedWorkspaceFilters" type="button" variant="text" @click="clearSavedWorkspaceFilters">Limpiar filtros</v-btn></div>
                <div v-else class="saved-workspace-list">
                  <article v-for="workspace in filteredSavedWorkspaces" :key="workspace.job_id" class="saved-workspace-card">
                    <div class="saved-workspace-card-main">
                      <span class="saved-workspace-icon"><v-icon icon="mdi-folder-zip-outline" size="22"/></span>
                      <div class="saved-workspace-card-copy">
                        <div class="saved-workspace-title-row"><v-chip size="small" :color="workspace.es_objeciones ? 'deep-purple' : 'primary'" variant="tonal">{{ workspaceTypeLabel(workspace) }}</v-chip><h3>{{ coverageServiceLabel(workspace.tipo_servicio) }} · {{ coverageMonthLabel(workspaceMonthValue(workspace)) }} {{ workspaceYearLabel(workspace) }}</h3><v-chip size="small" :color="workspaceStatusColor(workspace.status, workspace)" variant="tonal">{{ workspaceStatusLabel(workspace.status, workspace) }}</v-chip></div>
                        <p v-if="workspace.missing_documents?.length" class="saved-workspace-missing"><v-icon icon="mdi-alert-circle-outline" size="16"/> Faltan: {{ workspace.missing_documents.join(', ') }}</p>
                        <p v-else-if="workspace.message" class="saved-workspace-message">{{ workspace.message }}</p>
                        <p v-else class="saved-workspace-message">Período listo para continuar.</p>
                        <div class="saved-workspace-meta"><span v-if="workspaceReceivedAt(workspace.received_at)"><v-icon icon="mdi-clock-outline" size="14"/> {{ workspaceReceivedAt(workspace.received_at) }}</span><span><v-icon icon="mdi-account-outline" size="14"/> {{ workspace.creado_por || 'Usuario anterior' }}</span><code>{{ workspace.job_id }}</code></div>
                      </div>
                    </div>
                    <div class="saved-workspace-card-actions"><v-btn type="button" color="primary" prepend-icon="mdi-folder-open-outline" :loading="openingSavedWorkspace && selectedSavedWorkspace === workspace.job_id" :disabled="openingSavedWorkspace || workspaceDeleting" @click="openSavedWorkspace(workspace.job_id)">Continuar</v-btn><v-btn v-if="canDeleteWorkspaces" type="button" icon="mdi-delete-outline" variant="text" color="error" :aria-label="`Eliminar período ${workspace.job_id}`" title="Eliminar período" :disabled="openingSavedWorkspace || workspaceDeleting" @click="requestDeleteWorkspace(workspace.job_id)"/></div>
                  </article>
                </div>
              </template>
            </section>
            <template v-if="ingestMode === 'new' && !ingestResult">
            <section class="ingest-card">
              <div class="ingest-card-heading"><span class="ingest-step">1</span><div><h2>Define el lote</h2><p>Selecciona el mes, año y tipo de atención del ZIP.</p></div></div>
              <div class="ingest-period-grid">
                <v-select v-model="ingestMonth" :items="ingestMonths" item-title="title" item-value="value" label="Mes" prepend-inner-icon="mdi-calendar-month-outline" required/>
                <v-text-field v-model="ingestYear" label="Año" type="number" min="2000" max="2100" prepend-inner-icon="mdi-calendar-outline" required/>
                <v-select v-model="ingestService" :items="ingestServices" label="Tipo de servicio" prepend-inner-icon="mdi-hospital-building" required/>
              </div>
            </section>
            <section class="ingest-card">
              <div class="ingest-card-heading"><span class="ingest-step">2</span><div><h2>Sube el lote de planillas</h2><p>Selecciona el ZIP con los expedientes organizados por número de trámite. Los documentos habilitantes se añaden después.</p></div></div>
              <div class="ingest-file-grid">
                <label v-for="item in ingestFileFields.filter(file => file.field === 'zip_file')" :key="item.field" class="ingest-file-card">
                  <input type="file" :accept="item.accept" :disabled="ingestSending" @change="selectIngestFile(item.field, $event)" />
                  <span class="ingest-file-icon"><v-icon :icon="item.icon" size="22"/></span>
                  <span class="ingest-file-copy"><strong>{{ item.title }}</strong><small>{{ ingestFiles[item.field]?.name || item.detail }}</small></span>
                  <v-icon :icon="ingestFiles[item.field] ? 'mdi-check-circle' : 'mdi-plus-circle-outline'" :color="ingestFiles[item.field] ? 'success' : 'grey'" size="20"/>
                </label>
              </div>
            </section>
            </template>
            <v-alert v-if="ingestError" type="error" variant="tonal" density="comfortable" role="alert">{{ ingestError }}</v-alert>
            <div v-if="['STAGED', 'PROCESSED', 'INCOMPLETE', 'REQUIERE_REVISION'].includes(ingestResult?.status)" class="ingest-card completion-card">
              <div class="ingest-card-heading"><span class="ingest-step">3</span><div><h2>Documentos habilitantes</h2><p>Los tres tipos quedan disponibles. Sube un archivo para añadirlo o reemplazar el que ya está guardado en este período.</p></div></div>
              <v-chip v-for="item in ingestResult.missing_documents" :key="item" class="missing-chip" size="small" color="warning" variant="tonal">Falta: {{ item }}</v-chip>
              <div class="ingest-file-grid">
                <label v-for="item in completionFields" :key="item.field" class="ingest-file-card">
                  <input type="file" :accept="item.accept" :disabled="ingestProcessing" @change="selectCompletionFile(item.field, $event)" />
                  <span class="ingest-file-icon"><v-icon :icon="item.icon" size="22"/></span>
                  <span class="ingest-file-copy"><strong>{{ item.title }}</strong><small>{{ completionFiles[item.field]?.name || currentCompletionFile(item.field)?.original_name || item.detail }}</small><small v-if="currentCompletionFile(item.field) && !completionFiles[item.field]" class="completion-file-status">Actual · selecciona otro archivo para reemplazarlo</small><small v-else-if="completionFiles[item.field] && currentCompletionFile(item.field)" class="completion-file-status">Se reemplazará el documento actual</small></span>
                  <v-icon :icon="completionFiles[item.field] ? 'mdi-check-circle' : 'mdi-plus-circle-outline'" :color="completionFiles[item.field] ? 'success' : 'grey'" size="20"/>
                </label>
              </div>
              <div class="ingest-submit-row"><span>Los cambios se guardan en el mismo expediente; el ZIP no se vuelve a subir.</span><v-btn color="primary" :loading="ingestProcessing" :disabled="!completionValid || !Object.values(completionFiles).some(Boolean)" prepend-icon="mdi-cloud-upload-outline" @click="addMissingDocuments">{{ Object.entries(completionFiles).some(([field, file]) => file && currentCompletionFile(field)) ? 'Guardar cambios' : 'Añadir documentos' }}</v-btn></div>
            </div>
          <section v-if="ingestResult && (!ingestResult.output || showWorkspaceDetails)" class="ingest-card current-job-card">
              <div class="ingest-card-heading"><span class="ingest-step">{{ ['STAGED','REQUIERE_REVISION'].includes(ingestResult.status) ? '3' : '✓' }}</span><div><h2>{{ ['STAGED','REQUIERE_REVISION'].includes(ingestResult.status) ? 'Lote de planillas recibido' : 'Espacio de trabajo del período' }}</h2><p>{{ ['STAGED','REQUIERE_REVISION'].includes(ingestResult.status) ? 'El ZIP está guardado. Revisa el cruce antes de preparar las carpetas por paciente.' : 'Este espacio se reutiliza para el mismo servicio y período.' }}</p></div></div>
              <div class="ingest-job-id"><span>ID del lote</span><code>{{ ingestResult.job_id }}</code></div>
              <v-alert v-if="ingestResult.reuse_notice" type="info" variant="tonal" density="comfortable">{{ ingestResult.reuse_notice }}</v-alert>
              <v-alert :type="ingestResult.status === 'PROCESSED' ? 'success' : ingestResult.status === 'REQUIERE_REVISION' ? 'warning' : 'info'" variant="tonal" density="comfortable" prepend-icon="mdi-information-outline">
                <strong>{{ ['STAGED','REQUIERE_REVISION'].includes(ingestResult.status) ? 'Siguiente paso: revisar y preparar expedientes' : ingestResult.status === 'INCOMPLETE' ? 'Expediente preparado; faltan documentos habilitantes.' : 'Expediente preparado para revisión.' }}</strong><div>{{ ingestResult.message }}</div>
              </v-alert>
              <div v-if="['STAGED','REQUIERE_REVISION'].includes(ingestResult.status)" class="ingest-next-step">
                <p v-if="!ingestPreview">Revisa la vista previa antes de preparar el expediente.</p>
                <p v-else-if="!ingestPreviewReadyToPrepare">Hay {{ ingestPreview.tramites_sin_oracle }} carpeta(s) del ZIP sin coincidencia en Oracle y {{ ingestPreview.entradas_invalidas }} ruta(s) inválida(s). Corrige estos problemas para continuar.</p>
                <p v-else>Se prepararán los {{ ingestPreview.pdfs }} PDFs de las {{ ingestPreview.carpetas_tramite }} carpetas incluidas en el ZIP.<span v-if="ingestPreview.tramites_oracle_sin_zip"> Las {{ ingestPreview.tramites_oracle_sin_zip }} planillas que solo aparecen en Oracle se omitirán y no bloquean la preparación.</span></p>
                <v-btn color="primary" size="large" prepend-icon="mdi-folder-cog-outline" :loading="ingestProcessing" :disabled="!ingestPreviewReadyToPrepare" @click="processIngest">Preparar expedientes</v-btn>
                <input ref="zipReplacementInput" type="file" accept=".zip,application/zip" hidden @change="replaceIngestZIP"/>
                <v-btn v-if="!ingestPreviewReadyToPrepare" type="button" variant="outlined" prepend-icon="mdi-file-replace-outline" :loading="zipReplacementSending" :disabled="ingestProcessing" @click="zipReplacementInput?.click()">Subir ZIP corregido</v-btn>
              </div>
            </section>
            <section v-if="ingestResult && (ingestPreview || ingestPreviewLoading || ingestPreviewError) && (!ingestResult.output || showWorkspaceDetails)" class="ingest-card ingest-preview-card">
              <div class="ingest-card-heading"><span class="ingest-step"><v-icon icon="mdi-folder-search-outline" size="19"/></span><div><h2>Vista previa del ZIP y cruce con Oracle</h2><p>Período seleccionado: {{ ingestPreview?.mes || ingestResult.mes }}/{{ ingestPreview?.anio || ingestResult.anio }} · Servicio: {{ ingestPreview?.tipo_servicio || ingestService }}</p></div><v-spacer/><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar vista previa del ZIP" :loading="ingestPreviewLoading" @click="loadIngestPreview(ingestResult.job_id)"/></div>
              <div v-if="ingestPreviewLoading" class="planilla-state"><v-progress-circular indeterminate color="primary"/><span>Contando carpetas y PDFs, y cruzando trámites con Oracle…</span></div>
              <v-alert v-else-if="ingestPreviewError" type="warning" variant="tonal" density="comfortable">{{ ingestPreviewError }}<v-btn type="button" size="small" variant="text" @click="loadIngestPreview(ingestResult.job_id)">Reintentar</v-btn></v-alert>
              <template v-else-if="ingestPreview">
                <v-alert v-if="ingestPreviewNotice" :type="ingestPreviewNotice.includes('no se pudo') || ingestPreviewNotice.includes('No se pudo') ? 'warning' : 'success'" variant="tonal" density="compact" class="ingest-preview-notice">{{ ingestPreviewNotice }}</v-alert>
                <v-alert v-if="ingestPreview.pdi_provisioning?.creados?.length" type="success" variant="tonal" density="comfortable" prepend-icon="mdi-database-check-outline">
                  Se crearon {{ ingestPreview.pdi_provisioning.creados.length }} registro(s) faltantes en PLANILLA_DIGITAL a partir de este ZIP. Las filas existentes no se modificaron.
                </v-alert>
                <v-alert v-if="ingestPreview.pdi_provisioning?.pendientes?.length" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-database-alert-outline">
                  No se pudo completar Oracle para estas planillas: <code>{{ ingestPreview.pdi_provisioning.pendientes.join(', ') }}</code>. Se requiere una coincidencia única en SIS para MSP, período y servicio.
                </v-alert>
                <v-alert v-if="ingestPreview.pdi_provisioning?.error" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-database-alert-outline">
                  El ZIP quedó guardado, pero no se pudieron completar los registros Oracle: {{ ingestPreview.pdi_provisioning.error }}
                </v-alert>
                <div class="ingest-preview-stats">
                  <div><strong>{{ ingestPreview.carpetas_tramite }}</strong><span>carpetas de trámites</span></div>
                  <div><strong>{{ ingestPreview.pdfs }}</strong><span>PDFs en el ZIP</span></div>
                  <div><strong>{{ ingestPreview.tramites_en_oracle }}</strong><span>trámites encontrados en Oracle</span></div>
                  <div><strong>{{ ingestPreview.tramites_sin_oracle }}</strong><span>trámites sin coincidencia</span></div>
                  <div><strong>{{ ingestPreview.tramites_oracle_sin_zip }}</strong><span>trámites Oracle sin carpeta en ZIP</span></div>
                  <div><strong>{{ ingestPreview.entradas_invalidas }}</strong><span>rutas o archivos inválidos</span></div>
                </div>
                <v-alert v-if="ingestPreview.tramites_sin_oracle || ingestPreview.entradas_invalidas || ingestPreview.tramites_sin_paciente_oracle" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-alert-outline">
                  Revisa la tabla antes de preparar. El cruce de cada carpeta se hace únicamente por su número <code>PDI_TRAMITE</code>. El servicio y las fechas que devuelve Oracle se muestran como referencia; no se excluye una planilla por diferencias en esos campos.
                  <span v-if="ingestPreview.tramites_sin_paciente_oracle"> {{ ingestPreview.tramites_sin_paciente_oracle }} trámites encontrados no tienen nombre de paciente en Oracle.</span>
                  <span v-if="ingestPreview.rutas_invalidas?.length"> Rutas con problemas: {{ ingestPreview.rutas_invalidas.join(', ') }}<span v-if="ingestPreview.entradas_invalidas > ingestPreview.rutas_invalidas.length"> y otras {{ ingestPreview.entradas_invalidas - ingestPreview.rutas_invalidas.length }}.</span></span>
                </v-alert>
                <v-alert v-if="ingestPreview.tramites_oracle_sin_zip" type="info" variant="tonal" density="comfortable" prepend-icon="mdi-folder-question-outline">
                  Oracle tiene {{ ingestPreview.tramites_oracle_sin_zip }} planillas para {{ ingestPreview.mes }}/{{ ingestPreview.anio }} que no aparecen en el ZIP. El ZIP define los expedientes del lote: esas planillas se omiten y no impiden continuar.
                  <details class="oracle-omitted-details"><summary>Ver trámites omitidos</summary><p><code>{{ ingestPreview.pdi_tramite_oracle_sin_zip.join(', ') }}</code></p></details>
                </v-alert>
                <div class="ingest-preview-table-wrap">
                  <v-table class="planilla-table ingest-preview-table" density="comfortable"><thead><tr><th>Carpeta ZIP / PDI_TRAMITE</th><th>PDFs</th><th>Paciente Oracle</th><th>PDI_SERVICIO</th><th>PDI_FECHA_DESDE – HASTA</th><th>Cruce</th><th>Corrección manual</th></tr></thead>
                    <tbody><tr v-for="folder in ingestPreviewRows" :key="folder.tramite"><td><code>{{ folder.tramite }}</code><div v-if="folder.vinculo_manual" class="ingest-manual-link">Oracle: <code>{{ folder.pdi_tramite_oracle }}</code></div></td><td>{{ folder.pdfs }}</td><td>{{ folder.paciente || (folder.coincide_oracle ? 'Sin nombre en Oracle' : '—') }}</td><td>{{ folder.servicio_oracle || '—' }}</td><td>{{ folder.fecha_desde || '—' }} – {{ folder.fecha_hasta || '—' }}</td><td><v-chip size="small" :color="folder.conflicto_vinculo ? 'error' : folder.coincide_oracle && !folder.paciente_faltante_oracle ? 'success' : 'warning'" variant="tonal">{{ folder.conflicto_vinculo ? 'Vínculo duplicado' : folder.vinculo_manual ? 'Vinculado a Oracle' : !folder.coincide_oracle ? 'No encontrado' : folder.paciente_faltante_oracle ? 'Falta paciente' : 'Coincide' }}</v-chip></td><td><div v-if="folder.vinculo_manual" class="ingest-manual-map-actions"><span>Usa PDI_TRAMITE {{ folder.pdi_tramite_oracle }}</span><v-btn size="small" variant="text" color="warning" :loading="ingestMappingSaving === folder.tramite" :disabled="Boolean(ingestMappingSaving)" @click="clearIngestTramiteMapping(folder)">Deshacer</v-btn></div><div v-else-if="!folder.coincide_oracle" class="ingest-manual-map-actions"><v-select v-model="ingestMappingSelections[folder.tramite]" :items="ingestPreview.planillas_oracle_disponibles || []" item-title="titulo" item-value="tramite" label="Trámite Oracle" density="compact" variant="outlined" hide-details :disabled="Boolean(ingestMappingSaving) || !(ingestPreview.planillas_oracle_disponibles || []).length"/><v-btn size="small" color="primary" :loading="ingestMappingSaving === folder.tramite" :disabled="Boolean(ingestMappingSaving) || !ingestMappingSelections[folder.tramite]" @click="saveIngestTramiteMapping(folder)">Emparejar</v-btn></div><span v-else>—</span></td></tr></tbody>
                  </v-table>
                </div>
                <div class="planilla-pagination"><span>Carpetas {{ ingestPreview.carpetas.length ? (ingestPreviewPage - 1) * ingestPreviewPageSize + 1 : 0 }}–{{ Math.min(ingestPreviewPage * ingestPreviewPageSize, ingestPreview.carpetas.length) }} de {{ ingestPreview.carpetas.length }}</span><v-pagination v-if="ingestPreviewPageCount > 1" v-model="ingestPreviewPage" :length="ingestPreviewPageCount" :total-visible="5" density="comfortable"/></div>
              </template>
            </section>
            <section v-if="ingestResult?.output" class="ingest-card review-card">
              <div class="ingest-card-heading"><span class="ingest-step">4</span><div><h2>Revisión de documentos</h2><p>La clasificación se ejecuta al preparar el expediente. Los pendientes requieren revisión documental antes del cierre.</p></div></div>
              <v-alert v-if="ingestResult.clasificacion" type="info" variant="tonal" density="comfortable">{{ ingestResult.clasificacion.clasificados }} PDFs identificados y {{ ingestResult.clasificacion.pendientes }} sin identificar, de {{ ingestResult.clasificacion.total }}. Lectura: {{ ingestResult.clasificacion.texto_vectorial }} con texto PDF, {{ ingestResult.clasificacion.ocr }} por OCR y {{ ingestResult.clasificacion.sin_texto_legible ?? 0 }} sin texto legible.</v-alert>
              <v-alert v-if="ingestResult.clasificacion?.fusiones_pendientes" type="warning" variant="tonal" density="comfortable">{{ ingestResult.clasificacion.fusiones_pendientes }} PDF(s) adicionales fueron reconocidos y se conservaron con nombre numerado. Revísalos y fusiónalos desde la carpeta del paciente cuando corresponda.</v-alert>
              <v-alert v-if="ingestResult.clasificacion?.documentos_fecha_fuera_periodo?.length" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-calendar-alert-outline">
                <strong>{{ ingestResult.clasificacion.documentos_fecha_fuera_periodo.length }} PDF contienen fechas fuera del período facturado de {{ billedPeriodLabel(ingestResult.clasificacion.documentos_fecha_fuera_periodo[0].periodo_facturado_oracle) }}.</strong>
                <p>SPD MSP compara las fechas reconocidas en el PDF con el mes y año del lote en Oracle (`PDI_MES` y `PDI_ANIO`). Revisa estos documentos; la alerta no bloquea ni modifica el expediente.</p>
                <ul class="period-date-alert-list"><li v-for="alert in ingestResult.clasificacion.documentos_fecha_fuera_periodo" :key="alert.documento"><code>{{ alert.documento }}</code>: {{ alert.fechas_detectadas.join(', ') }}</li></ul>
                <small>Las fechas se reconocen en el texto PDF o mediante OCR. Confirma el documento original antes de corregirlo.</small>
              </v-alert>
              <details class="ingest-secondary-action"><summary>Volver a analizar los PDFs</summary><p>Repite la clasificación de los archivos fuente y actualiza la carpeta de trabajo. Puede tardar varios minutos.</p><v-btn variant="outlined" prepend-icon="mdi-text-box-search-outline" :loading="ingestProcessing" @click="classifyIngest">Reanalizar PDFs</v-btn></details>
              <details v-if="ingestResult.workspace || ingestResult.output" class="ingest-secondary-action"><summary>Ver ubicaciones del expediente</summary><p v-if="ingestResult.workspace">Espacio permanente: <code>{{ ingestResult.workspace }}</code></p><p>Carpeta preparada: <code>{{ ingestResult.output }}</code></p></details>
            </section>
            <div v-if="ingestResult && !ingestResult.output" class="ingest-submit-row"><span>¿Necesitas recibir planillas de otro período o servicio?</span><v-btn type="button" variant="text" prepend-icon="mdi-plus" @click="startNewIngest">Recibir otro período</v-btn></div>
            <v-expansion-panels v-if="!ingestResult && ingestMode === 'new'" variant="accordion" class="ingest-resume-panel"><v-expansion-panel><v-expansion-panel-title>¿Tienes el ID de un lote?</v-expansion-panel-title><v-expansion-panel-text><p>Escribe el ID que apareció al recibir el lote.</p><div class="ingest-submit-row"><v-text-field v-model="existingIngestJobId" label="ID del lote" placeholder="JOB-…" density="comfortable" hide-details/><v-btn color="primary" prepend-icon="mdi-folder-cog-outline" :loading="ingestProcessing" :disabled="!existingIngestJobId.trim()" @click="processIngest">Abrir lote por ID</v-btn></div></v-expansion-panel-text></v-expansion-panel></v-expansion-panels>
            <div v-if="ingestSending || completionSending" class="ingest-progress"><div><span>{{ completionSending ? 'Subiendo documentos habilitantes…' : 'Subiendo lote de planillas…' }}</span><strong>{{ ingestProgress }}%</strong></div><v-progress-linear :model-value="ingestProgress" color="primary" rounded/></div>
            <div v-if="!ingestResult && ingestMode === 'new'" class="ingest-submit-row"><span>El ZIP se guardará en el espacio privado del servidor.</span><v-btn type="submit" color="primary" size="large" prepend-icon="mdi-cloud-upload-outline" :loading="ingestSending" :disabled="!ingestValid || ingestSending">{{ ingestSending ? 'Subiendo ZIP…' : 'Recibir lote de planillas' }}</v-btn></div>
          </form>
          <footer>Hecho con cuidado para tus documentos <span>✳</span></footer>
        </section>
      </main>
    </div>
    <div v-if="dragging" class="drop-overlay"><div class="drop-message"><v-icon icon="mdi-cloud-upload-outline" size="46"/><h2>Suelta tus archivos aquí</h2><p>Se guardarán en {{ activeFolder === folders[0] ? 'Mi espacio' : activeFolder }}</p></div></div>
    <v-dialog v-model="patientDocumentsDialog" class="patient-documents-dialog" width="96vw" max-width="1800" aria-labelledby="patient-documents-title">
      <v-card class="patient-documents-modal">
        <div class="patient-documents-header">
          <div><h2 id="patient-documents-title">Documentos del paciente</h2><p>{{ coverageServiceLabel(ingestResult?.tipo_servicio) }} · {{ coverageMonthLabel(ingestResult?.mes) }} {{ ingestResult?.anio }}</p></div>
          <div class="patient-documents-header-actions"><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar lista de PDFs" :loading="workspaceBusy" :disabled="workspaceBusy" @click="loadWorkspaceDocuments(ingestResult.job_id, selectedWorkspacePDF)"/><v-btn type="button" variant="tonal" prepend-icon="mdi-close" @click="patientDocumentsDialog = false">Cerrar</v-btn></div>
        </div>
        <v-card-text v-if="ingestResult?.output" class="patient-documents-body">
              <v-alert v-if="workspaceSending || workspaceMutationBusy || mergeSending || mergeAllSending" type="info" variant="tonal" density="comfortable" class="workspace-notice">
                Puedes cerrar este visor y seguir navegando. La operación ya enviada continuará en el servidor; si vence el tiempo de espera, revisa los PDFs antes de repetirla.
              </v-alert>
              <v-alert v-if="workspaceNotice" :type="workspaceNotice.includes('Se guardó') || workspaceNotice.includes('añadidos') || workspaceNotice.includes('reemplazado') || workspaceNotice.includes('quitado') || workspaceNotice.includes('descargó') || workspaceNotice.includes('fusionaron') ? 'success' : 'warning'" variant="tonal" density="compact" class="workspace-notice">{{ workspaceNotice }}</v-alert>
              <v-alert v-if="workspaceFusionGroups.length" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-content-copy">Hay {{ workspaceFusionGroups.length }} grupo(s) de documentos repetidos. Se conservaron con nombres numerados para revisarlos y fusionarlos después. Las fuentes originales se mantienen.</v-alert>
              <div v-if="workspaceFusionQueue.length" class="workspace-fusion-quick-access">
                <div class="workspace-fusion-quick-heading"><div class="workspace-fusion-quick-copy"><strong>Grupos pendientes de fusión</strong><span>Busca un paciente para revisar un grupo o fusiona todos en su orden actual.</span></div><v-btn type="button" color="warning" variant="outlined" prepend-icon="mdi-playlist-check" :disabled="mergeSending || mergeAllSending" @click="requestMergeAllPending">Fusionar todos sin revisar</v-btn></div>
                <div class="workspace-fusion-quick-controls"><v-autocomplete v-model="pendingFusionSelection" :items="workspaceFusionQueue" item-title="title" item-value="value" label="Paciente y tipo de documento" placeholder="Busca un grupo por revisar" prepend-inner-icon="mdi-magnify" density="comfortable" variant="outlined" hide-details clearable :disabled="mergeAllSending"/><v-btn type="button" color="warning" prepend-icon="mdi-file-document-multiple-outline" :disabled="!pendingFusionSelection || mergeSending || mergeAllSending" @click="reviewPendingFusion">Revisar</v-btn></div>
                <div v-if="selectedPendingFusionGroup" class="workspace-fusion-selected"><v-icon icon="mdi-account-box-outline" size="22"/><div><small>PACIENTE</small><strong>{{ selectedPendingFusionGroup.patient }}</strong></div><div><small>DOCUMENTO REPETIDO</small><strong>{{ selectedPendingFusionGroup.code }} · {{ selectedPendingFusionGroup.paths.length }} PDFs</strong></div></div>
                <div v-if="mergeAllSending" class="workspace-fusion-bulk-progress"><span>Fusionando grupos: {{ mergeAllProgress.done }} de {{ mergeAllProgress.total }}</span><v-progress-linear :model-value="mergeAllProgress.total ? mergeAllProgress.done / mergeAllProgress.total * 100 : 0" color="warning" rounded/></div>
              </div>
              <v-select v-model="workspacePatient" :items="workspacePatients" label="Carpeta del paciente" prepend-inner-icon="mdi-folder-account-outline" density="comfortable" class="workspace-patient-select" :disabled="!workspacePatients.length || replacePDFDialog" @update:model-value="selectWorkspacePatient"/>
              <div v-if="workspacePatientFusionGroups.length" class="workspace-patient-fusions">
                <div class="workspace-patient-fusions-heading"><strong>PDFs repetidos de este paciente</strong><span>Revisa y fusiona cada tipo por separado.</span></div>
                <div v-for="group in workspacePatientFusionGroups" :key="group.paths[0]" class="workspace-patient-fusion-item">
                  <div><strong>{{ group.code }}</strong><small>{{ group.paths.length }} PDFs · {{ group.paths.map(path => path.split('/').at(-1)).join(', ') }}</small></div>
                  <v-btn type="button" color="warning" variant="tonal" prepend-icon="mdi-file-document-multiple-outline" :disabled="workspaceBusy || mergeSending || mergeAllSending" @click="openWorkspaceFusion(group)">Revisar y fusionar</v-btn>
                </div>
              </div>
              <div class="workspace-file-layout">
                <div class="workspace-file-list">
                  <div class="workspace-file-count">{{ workspacePatientPDFs.length }} PDFs en esta carpeta</div>
                  <div v-for="document in workspacePatientPDFs" :key="document.path" class="workspace-file-entry">
                    <button type="button" class="workspace-file-row" :title="document.name" :aria-pressed="selectedWorkspacePDF === document.path" :class="{ selected: selectedWorkspacePDF === document.path }" @click.prevent="selectWorkspacePDF(document.path)">
                      <v-icon icon="mdi-file-pdf-box" color="error" size="20"/><span><strong>{{ document.name }}</strong><small>{{ prettySize(document.size) }}</small></span><v-chip v-if="isPendingWorkspaceDocument(document)" size="x-small" color="warning" variant="tonal">Sin nombre MSP</v-chip><v-chip v-else-if="workspaceFusionGroups.some(group => group.paths.includes(document.path))" size="x-small" color="warning" variant="tonal">Fusionar después</v-chip><v-icon v-if="selectedWorkspacePDF === document.path" icon="mdi-eye-outline" size="18"/>
                    </button>
                    <v-btn type="button" prepend-icon="mdi-delete-outline" variant="tonal" size="small" color="error" class="workspace-delete-pdf-btn" :aria-label="`Quitar ${document.name}`" title="Quitar PDF de esta carpeta" @click="requestDeleteWorkspacePDF(document.path)">Quitar</v-btn>
                  </div>
                  <div v-if="!workspacePatientPDFs.length && !workspaceBusy" class="workspace-empty">Esta carpeta todavía no tiene PDFs.</div>
                </div>
                <div class="workspace-preview">
                  <div v-if="selectedWorkspaceDocument" class="workspace-preview-heading"><strong>{{ selectedWorkspaceDocument.name }}</strong><span>{{ prettySize(selectedWorkspaceDocument.size) }}</span></div>
                  <iframe v-if="workspacePDFURL" :key="workspacePDFURL" :src="workspacePDFURL" :title="selectedWorkspaceDocument?.name || 'Vista previa del PDF'" />
                  <div v-else class="workspace-empty">Selecciona un PDF de la lista para verlo aquí.</div>
                  <div v-if="selectedWorkspaceDocument" class="workspace-rename-panel">
                    <div class="workspace-rename-copy"><strong>{{ isPendingWorkspaceDocument(selectedWorkspaceDocument) ? 'Asigna un nombre del catálogo MSP' : 'Cambiar nombre del documento' }}</strong><small>La vista previa ayuda a revisar el contenido. Asignar un nombre organiza el PDF, pero no confirma su clasificación.</small></div>
                    <div class="workspace-edit-row">
                      <v-autocomplete v-model="workspaceRename" :items="mspPDFOptions" item-title="title" item-value="value" :label="isPendingWorkspaceDocument(selectedWorkspaceDocument) ? 'Nombre MSP' : 'Nombre estándar MSP'" placeholder="Selecciona el tipo de documento" prepend-inner-icon="mdi-rename-box-outline" density="comfortable" variant="outlined" clearable hide-details/>
                      <v-btn type="button" color="primary" variant="tonal" prepend-icon="mdi-content-save-outline" :loading="workspaceBusy" :disabled="!workspaceRename" @click="renameWorkspacePDF">{{ isPendingWorkspaceDocument(selectedWorkspaceDocument) ? 'Asignar nombre' : 'Guardar nombre' }}</v-btn>
                    </div>
                  </div>
                </div>
              </div>
              <p v-if="selectedWorkspaceDocument" class="workspace-hint">Si añades un tipo que ya existe, se conserva como una copia numerada para revisar o fusionar después. Para sustituir una versión, usa “Reemplazar PDF seleccionado”.</p>
              <div class="workspace-add-row">
                <input ref="workspacePDFInput" type="file" accept=".pdf,application/pdf" multiple hidden @change="selectWorkspacePDFs" />
                <v-btn type="button" variant="outlined" prepend-icon="mdi-file-pdf-box" :disabled="workspaceSending" @click="workspacePDFInput?.click()">{{ workspaceUploadFiles.length ? 'Añadir más PDFs a la selección' : 'Seleccionar PDFs para añadir' }}</v-btn>
                <input ref="replacePDFInput" type="file" accept=".pdf,application/pdf" hidden @change="selectReplacementPDF" />
                <v-btn type="button" color="warning" variant="tonal" prepend-icon="mdi-file-replace-outline" :disabled="!selectedWorkspaceDocument || workspaceSending" @click="replacePDFInput?.click()">Reemplazar PDF seleccionado</v-btn>
              </div>
              <div v-if="workspaceUploadFiles.length" class="workspace-upload-queue">
                <div class="workspace-upload-queue-heading"><strong>{{ workspaceUploadFiles.length }} PDF{{ workspaceUploadFiles.length === 1 ? '' : 's' }} para añadir</strong><span>Asigna el paciente y nombre MSP. Si ya existe ese tipo de documento, se añadirá con el siguiente sufijo disponible para revisar o fusionar después.</span></div>
                <div v-for="(item, index) in workspaceUploadFiles" :key="item.id" class="workspace-upload-item">
                  <div class="workspace-upload-source"><v-icon icon="mdi-file-pdf-box" color="error"/><span><strong>{{ item.file.name }}</strong><small>{{ prettySize(item.file.size) }}</small></span></div>
                <v-autocomplete v-model="item.code" :items="mspPDFOptions" item-title="title" item-value="value" label="Nombre estándar MSP" placeholder="Selecciona el documento" density="compact" variant="outlined" clearable hide-details @update:model-value="item.replaceExisting = workspacePatientPDFs.some(document => document.name.toLowerCase() === (item.code || '').toLowerCase())"/>
                  <v-btn type="button" icon="mdi-close" variant="text" size="small" :aria-label="`Quitar ${item.file.name} de la selección`" @click="removeQueuedWorkspacePDF(index)"/>
                </div>
                <div class="workspace-upload-submit"><span>{{ queuedWorkspaceDuplicates ? `${queuedWorkspaceDuplicates} PDF(s) se añadirán con sufijo; el documento actual se conserva. ` : '' }}Para sustituir uno existente, usa “Reemplazar PDF seleccionado”.</span><v-btn type="button" color="primary" prepend-icon="mdi-cloud-upload-outline" :loading="workspaceSending" :disabled="!workspacePatient || workspaceUploadFiles.some(item => !item.code)" @click="addWorkspacePDFs">Añadir PDFs</v-btn></div>
              </div>
              <div v-if="workspaceSending" class="ingest-progress"><div><span>Guardando PDFs en el espacio de trabajo…</span><strong>{{ ingestProgress }}%</strong></div><v-progress-linear :model-value="ingestProgress" color="primary" rounded/></div>
              <v-dialog v-model="mergePDFDialog" max-width="1400">
                <v-card class="action-dialog">
                  <v-card-title>Revisar y fusionar PDFs</v-card-title>
                  <v-card-text>
                    <p>El orden inicial se conserva si no mueves los PDFs. Usa “Ver PDF” para revisarlos y las flechas para cambiar el orden; las páginas se unirán de arriba hacia abajo. El resultado conservará el nombre estándar y las fuentes originales permanecerán en el historial privado.</p>
                    <div class="workspace-merge-review">
                      <div><p class="workspace-fusion-drag-hint"><v-icon icon="mdi-cursor-move" size="18"/> Arrastra las filas desde el asa para ordenar. También puedes usar las flechas.</p><div class="workspace-fusion-list"><div v-for="(path, index) in mergePDFPaths" :key="path" class="workspace-fusion-item" :class="{ 'is-previewing': mergePreviewPath === path, 'is-dragging': mergeDraggingIndex === index, 'is-drop-target': mergeDropIndex === index && mergeDraggingIndex !== index }" @dragover="dragOverWorkspaceFusion($event, index)" @drop="dropWorkspaceFusion($event, index)"><button type="button" class="workspace-fusion-drag-handle" draggable="true" :aria-label="`Arrastrar ${path.split('/').at(-1)} para cambiar el orden`" title="Arrastrar para ordenar" @dragstart="startWorkspaceFusionDrag($event, index)" @dragend="finishWorkspaceFusionDrag"><v-icon icon="mdi-drag" size="21"/></button><span><strong>{{ index + 1 }}.</strong> {{ path.split('/').at(-1) }}</span><div class="workspace-fusion-item-actions"><v-btn type="button" prepend-icon="mdi-eye-outline" size="small" variant="tonal" :aria-pressed="mergePreviewPath === path" @click="mergePreviewPath = path">Ver PDF</v-btn><button type="button" class="workspace-fusion-order-btn" :disabled="index === 0 || mergeSending" :aria-label="`Mover ${path.split('/').at(-1)} arriba`" title="Mover arriba" @click="moveWorkspaceFusionPDF(index, -1)"><span aria-hidden="true">↑</span></button><button type="button" class="workspace-fusion-order-btn" :disabled="index === mergePDFPaths.length - 1 || mergeSending" :aria-label="`Mover ${path.split('/').at(-1)} abajo`" title="Mover abajo" @click="moveWorkspaceFusionPDF(index, 1)"><span aria-hidden="true">↓</span></button></div></div></div></div>
                      <div class="workspace-fusion-preview"><div class="workspace-fusion-preview-title">{{ mergePreviewPath.split('/').at(-1) || 'Vista previa del PDF' }}</div><iframe v-if="mergePreviewURL" :key="mergePreviewURL" :src="mergePreviewURL" :title="`Vista previa de ${mergePreviewPath.split('/').at(-1)}`"/><div v-else class="workspace-empty">Selecciona “Ver PDF” para revisar un documento.</div></div>
                    </div>
                  </v-card-text>
                  <v-alert v-if="mergeSending" type="info" variant="tonal" class="mx-6">La fusión ya fue enviada. Puedes cerrar esta ventana; se seguirá procesando y recibirás el resultado al volver.</v-alert>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="mergePDFDialog=false">{{ mergeSending ? 'Cerrar ventana' : 'Cancelar' }}</v-btn><v-btn type="button" color="warning" prepend-icon="mdi-file-document-multiple-outline" :loading="mergeSending" :disabled="mergeSending || mergeAllSending" @click="confirmWorkspaceFusion">Fusionar PDFs</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
              <v-dialog v-model="mergeAllConfirmDialog" max-width="600">
                <v-card class="action-dialog">
                  <v-card-title>Fusionar todos los grupos pendientes</v-card-title>
                  <v-card-text><p>Se fusionarán {{ workspaceFusionQueue.length }} grupos, uno por paciente y tipo de documento, usando el orden actual de cada lista. No tendrás que revisar cada PDF antes. Las fuentes originales se conservarán en el historial privado; si una fusión falla, las siguientes quedarán pendientes.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="mergeAllConfirmDialog=false">Cancelar</v-btn><v-btn type="button" color="warning" prepend-icon="mdi-playlist-check" @click="confirmMergeAllPending">Fusionar {{ workspaceFusionQueue.length }} grupos</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
              <v-dialog v-model="deletePDFDialog" max-width="440">
                <v-card class="action-dialog">
                  <v-card-title>Quitar PDF de la carpeta</v-card-title>
                  <v-card-text><p>Se quitará <strong>{{ deletePDFTarget.split('/').at(-1) }}</strong> de la carpeta de trabajo. La fuente del lote se conserva y no se volverá a incluir al reclasificar.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="deletePDFDialog=false;deletePDFTarget=''">Cancelar</v-btn><v-btn type="button" color="error" :loading="workspaceBusy" @click="deleteWorkspacePDF">Quitar PDF</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
              <v-dialog v-model="replacePDFDialog" max-width="460">
                <v-card class="action-dialog">
                  <v-card-title>Reemplazar PDF</v-card-title>
                  <v-alert v-if="replacePDFError" type="error" variant="tonal" role="alert" class="mx-6">{{ replacePDFError }}</v-alert>
                  <v-alert v-if="workspaceSending" type="info" variant="tonal" role="status" class="mx-6">El reemplazo ya fue enviado y seguirá en el servidor aunque cierres esta ventana.</v-alert>
                  <v-card-text><p>El archivo local <strong>{{ replacePDFFile?.name }}</strong> reemplazará a <strong>{{ selectedWorkspaceDocument?.name }}</strong> dentro de <strong>{{ workspacePatient }}</strong>. El archivo local adoptará automáticamente el nombre del PDF seleccionado; su nombre original se conservará solo como referencia. La versión anterior queda en las fuentes del expediente.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="replacePDFDialog=false;replacePDFFile=null;replacePDFTarget='';replacePDFError=''">{{ workspaceSending ? 'Cerrar ventana' : 'Cancelar' }}</v-btn><v-btn type="button" color="warning" :loading="workspaceSending" :disabled="workspaceSending" @click="confirmReplaceWorkspacePDF">Confirmar reemplazo</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
        </v-card-text>
      </v-card>
    </v-dialog>
    <v-dialog v-model="deleteWorkspaceDialog" max-width="600" :persistent="!workspaceDeleting">
      <v-card class="action-dialog">
        <v-card-title>{{ workspaceDeleteIsObjections ? 'Eliminar espacio de objeciones' : 'Eliminar período guardado' }}</v-card-title>
        <v-card-text>
          <p v-if="workspaceDeleteIsObjections">Se eliminará permanentemente el espacio derivado <strong>{{ workspaceDeleteTargetName }}</strong>. No se tocará el primer ingreso relacionado ni se cambiarán datos de Oracle. Esta acción no se puede deshacer.</p>
          <p v-else>Se eliminará permanentemente <strong>{{ workspaceDeleteTargetName }}</strong>, con el ZIP fuente, los documentos habilitantes, reportes y archivos preparados. También se quitará su asociación en Oracle y sus coberturas quedarán pendientes para poder generarlas nuevamente al recrear el período. Esta acción no se puede deshacer.</p>
          <p>Para confirmar, escribe o pega el ID completo del período:</p>
          <div class="workspace-delete-code">{{ workspaceDeleteTargetID }}</div>
          <v-text-field :model-value="workspaceDeleteInput" label="ID del período" autocomplete="off" autocapitalize="characters" spellcheck="false" prepend-inner-icon="mdi-keyboard-outline" @update:model-value="setWorkspaceDeleteInput"/>
          <v-alert v-if="workspaceDeleteError" type="error" variant="tonal" density="comfortable">{{ workspaceDeleteError }}</v-alert>
          <v-alert v-if="workspaceDeleting" type="info" variant="tonal" density="comfortable">La eliminación ya fue enviada. Cerrar esta ventana no la cancela; espera el resultado antes de volver a intentarlo.</v-alert>
        </v-card-text>
        <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="closeWorkspaceDeleteDialog">{{ workspaceDeleting ? 'Cerrar ventana' : 'Cancelar' }}</v-btn><v-btn v-if="canDeleteWorkspaces" type="button" color="error" prepend-icon="mdi-delete-forever-outline" :loading="workspaceDeleting" :disabled="!workspaceDeleteConfirmationMatches() || workspaceDeleting" @click="deleteWholeWorkspace">Eliminar definitivamente</v-btn></v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog v-model="createSpaceDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Crear un espacio</v-card-title><v-card-text><p>Organiza tus archivos en un espacio nuevo.</p><v-text-field v-model="newSpaceName" label="Nombre del espacio" placeholder="Ej. Clientes" prepend-inner-icon="mdi-folder-outline" maxlength="36" autofocus @keyup.enter="createSpace"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="createSpaceDialog=false">Cancelar</v-btn><v-btn color="primary" :disabled="!newSpaceName.trim()" @click="createSpace">Crear espacio</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="moveDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Mover {{ activeDoc ? 'documento' : `${selectedIds.length} documentos` }}</v-card-title><v-card-text><p>Elige el espacio de destino.</p><v-select v-model="moveTarget" :items="spaces" label="Espacio" prepend-inner-icon="mdi-folder-outline"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="moveDialog=false">Cancelar</v-btn><v-btn color="primary" @click="confirmMove">Mover</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="preview" max-width="720"><v-card v-if="preview" class="preview-card"><div class="preview-head"><div class="doc-type-icon" :class="'type-'+preview.color"><v-icon :icon="preview.icon" size="20"/></div><div class="preview-name"><b>{{ preview.name }}</b><small>{{ preview.ext }} · {{ preview.size }}</small></div><v-btn icon="mdi-download-outline" variant="text" aria-label="Descargar" @click="download(preview)"/><v-btn icon="mdi-delete-outline" variant="text" aria-label="Eliminar" @click="removeDoc(preview)"/><v-btn icon="mdi-close" variant="text" aria-label="Cerrar" @click="preview=null"/></div><div class="preview-body"><template v-if="preview.mime?.startsWith('image/')"><img :src="previewUrl" :alt="preview.name"/></template><template v-else-if="preview.mime==='application/pdf'"><iframe :src="previewUrl" :title="preview.name"/></template><div v-else class="preview-placeholder"><v-icon :icon="preview.icon" size="48"/><p>Vista previa disponible para imágenes y archivos PDF.</p><button class="primary-upload" @click="download(preview)"><v-icon icon="mdi-download-outline"/> Descargar archivo</button></div></div></v-card></v-dialog>
    <v-snackbar v-model="snackbar" timeout="2600" location="bottom end" color="primary">{{ snackbar }}<template #actions><v-btn variant="text" @click="snackbar=''">Cerrar</v-btn></template></v-snackbar>
  </v-app>
</template>
