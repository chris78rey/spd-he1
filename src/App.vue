<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { openDB } from 'idb'
import mspPdfCodes from '../catalogos/codigos_msp.json'

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
const zipDownloading = ref(false)
const zipDownloadError = ref('')
const zipDownloadNotice = ref('')
const workspacePDFs = ref([])
const workspacePDFRevision = ref(0)
const workspacePatients = ref([])
const selectedWorkspacePDF = ref('')
const workspaceRename = ref('')
const workspacePatient = ref('')
const workspaceUploadFiles = ref([])
const workspaceBusy = ref(false)
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
const patientDocumentsLocked = computed(() => workspaceSending.value || workspaceBusy.value || mergeSending.value || mergeAllSending.value || replacePDFDialog.value || deletePDFDialog.value || mergePDFDialog.value || mergeAllConfirmDialog.value)
watch(() => ingestResult.value?.job_id, () => { patientDocumentsDialog.value = false })
const deleteWorkspaceDialog = ref(false)
const workspaceDeleteTargetID = ref('')
const workspaceDeleteTargetName = ref('')
const workspaceDeleteInput = ref('')
const workspaceDeleteError = ref('')
const workspaceDeleting = ref(false)
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
    if (!response.ok) { authStatus.value = 'anonymous'; return }
    const data = await response.json()
    currentUser.value = data.username
    authStatus.value = 'authenticated'
    await loadDocuments()
    await validateCachedIngest()
    await loadSavedWorkspaces()
  } catch {
    authStatus.value = 'anonymous'
    authError.value = 'No se pudo conectar con el servicio de inicio de sesión.'
  }
}
async function signIn() {
  authError.value = ''
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
    oraclePassword.value = ''
    authStatus.value = 'authenticated'
    await loadDocuments()
    await validateCachedIngest()
    await loadSavedWorkspaces()
  } catch (error) {
    authError.value = error.message || 'No se pudo conectar con Oracle.'
  } finally {
    signingIn.value = false
  }
}
async function signOut() {
  try { await fetch('/api/logout', { method: 'POST', credentials: 'same-origin' }) } catch { /* La sesión local se cierra aunque falle la petición. */ }
  authStatus.value = 'anonymous'
  currentUser.value = ''
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
  const eligible = savedWorkspaces.value.filter(workspace => ['PROCESSED', 'INCOMPLETE'].includes(workspace.status))
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
  workspaceBusy.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/documentos/${encodeURIComponent(jobId)}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron cargar los PDFs del expediente.')
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
    workspaceNotice.value = error.message || 'No se pudieron cargar los PDFs del expediente.'
  } finally { workspaceBusy.value = false }
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
  if (!selectedWorkspaceDocument.value || !workspaceRename.value || workspaceBusy.value) return
  workspaceNotice.value = ''; workspaceBusy.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/documentos/renombrar/${encodeURIComponent(ingestResult.value.job_id)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: selectedWorkspaceDocument.value.path, name: workspaceRename.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo cambiar el nombre del PDF.')
    const path = data.path
    await loadWorkspaceDocuments(ingestResult.value.job_id, path)
    workspaceNotice.value = `Se guardó como ${data.name}.`
  } catch (error) { workspaceNotice.value = error.message || 'No se pudo cambiar el nombre del PDF.' }
  finally { workspaceBusy.value = false }
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
      const response = await fetch(`/api/v1/expedientes/documentos/fusionar/${encodeURIComponent(jobId)}`, {
        method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rutas: group.paths }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) throw new Error(data.error || `Falló la fusión de ${group.patient} · ${group.code}.`)
      merged++
      mergeAllProgress.value = { done: merged, total: groups.length }
    } catch (error) {
      failure = error.message || `Falló la fusión de ${group.patient} · ${group.code}.`
      break
    }
  }
  mergeAllSending.value = false
  await loadWorkspaceDocuments(jobId, selectedWorkspacePDF.value)
  await loadSavedWorkspaces()
  if (failure) workspaceNotice.value = merged
    ? `Se detuvo después de fusionar ${merged} de ${groups.length} grupos. Los demás siguen pendientes. ${failure}`
    : `No se completó la fusión general. Los grupos siguen pendientes. ${failure}`
  else workspaceNotice.value = `Se fusionaron ${merged} grupos pendientes en el orden actual de cada lista. Las fuentes originales se conservaron.`
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
  if (!ingestResult.value?.job_id || mergePDFPaths.value.length < 2 || mergeSending.value || mergeAllSending.value) return
  mergeSending.value = true
  workspaceNotice.value = ''
  try {
    const response = await fetch(`/api/v1/expedientes/documentos/fusionar/${encodeURIComponent(ingestResult.value.job_id)}`, {
      method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rutas: mergePDFPaths.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudieron fusionar los PDFs.')
    mergePDFDialog.value = false
    mergePDFPaths.value = []
    mergePreviewPath.value = ''
    pendingFusionSelection.value = ''
    await loadWorkspaceDocuments(data.job_id, data.fused)
    workspaceNotice.value = `${data.merged_count} PDFs fusionados en ${data.fused.split('/').at(-1)}. Las fuentes originales se conservaron.`
    await loadSavedWorkspaces()
  } catch (error) { workspaceNotice.value = error.message || 'No se pudieron fusionar los PDFs.' }
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
  if (!workspaceUploadFiles.value.length || !workspacePatient.value || workspaceSending.value) return
  workspaceNotice.value = ''; workspaceSending.value = true; ingestProgress.value = 0
  const form = new FormData()
  form.append('paciente', workspacePatient.value)
  for (const item of workspaceUploadFiles.value) {
    form.append('pdf_files', item.file)
    form.append('pdf_codes', item.code)
  }
  try {
    const data = await new Promise((resolve, reject) => {
      const request = new XMLHttpRequest()
      request.open('POST', `/api/v1/expedientes/documentos/${encodeURIComponent(ingestResult.value.job_id)}`)
      request.withCredentials = true
      request.upload.onprogress = event => { if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100) }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch { /* La API debe responder JSON. */ }
        if (request.status === 200) resolve(response)
        else reject(new Error(response.error || `No se pudieron añadir los PDFs (HTTP ${request.status}).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante la carga.'))
      request.send(form)
    })
    workspaceUploadFiles.value = []
    workspacePDFs.value = data.documents || []
    workspacePDFRevision.value++
    workspacePatients.value = data.patients || []
    const replacedCount = data.replaced_count || 0
    const addedCount = data.added_count ?? (data.added?.length || 0)
    workspaceNotice.value = replacedCount
      ? `${addedCount} PDF${addedCount === 1 ? '' : 's'} añadido${addedCount === 1 ? '' : 's'} y ${replacedCount} reemplazado${replacedCount === 1 ? '' : 's'}. Las versiones anteriores se conservan como fuentes.`
      : `${addedCount} PDF${addedCount === 1 ? '' : 's'} añadido${addedCount === 1 ? '' : 's'} al expediente.`
    selectedWorkspacePDF.value = data.added?.at(-1)?.relative_path || selectedWorkspacePDF.value
    workspaceRename.value = standardCodeForFilename(selectedWorkspaceDocument.value?.name || '')
    ingestProgress.value = 100
  } catch (error) { workspaceNotice.value = error.message || 'No se pudieron añadir los PDFs.' }
  finally { workspaceSending.value = false }
}
async function confirmReplaceWorkspacePDF() {
  const current = workspacePDFs.value.find(document => document.path === replacePDFTarget.value)
  if (!current || !replacePDFFile.value || workspaceSending.value) return
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
      request.open('POST', `/api/v1/expedientes/documentos/${encodeURIComponent(ingestResult.value.job_id)}`)
      request.withCredentials = true
      request.upload.onprogress = event => { if (event.lengthComputable) ingestProgress.value = Math.round(event.loaded / event.total * 100) }
      request.onload = () => {
        let response = {}
        try { response = JSON.parse(request.responseText) } catch { /* La API debe responder JSON. */ }
        if (request.status === 200) resolve(response)
        else reject(new Error(response.error || `No se pudo reemplazar el PDF (HTTP ${request.status}).`))
      }
      request.onerror = () => reject(new Error('Se interrumpió la conexión durante el reemplazo.'))
      request.send(form)
    })
    workspacePDFs.value = data.documents || []
    workspacePDFRevision.value++
    replacePDFFile.value = null
    selectedWorkspacePDF.value = current.path
    workspaceNotice.value = 'PDF reemplazado. La versión fuente se conserva en el expediente.'
    ingestProgress.value = 100
    replacePDFDialog.value = false
  } catch (error) { replacePDFError.value = error.message || 'No se pudo reemplazar el PDF.' }
  finally { workspaceSending.value = false }
}
function requestDeleteWorkspacePDF(path) {
  deletePDFTarget.value = path
  deletePDFDialog.value = true
}
function requestDeleteWorkspace(jobId = ingestResult.value?.job_id) {
  if (!jobId || workspaceDeleting.value) return
  workspaceDeleteTargetID.value = jobId
  workspaceDeleteTargetName.value = jobId === ingestResult.value?.job_id
    ? ingestResult.value.workspace?.split('/').at(-1) || jobId
    : savedWorkspaceItems.value.find(item => item.value === jobId)?.title || jobId
  workspaceDeleteInput.value = ''
  workspaceDeleteError.value = ''
  deleteWorkspaceDialog.value = true
}
function setWorkspaceDeleteInput(value) { workspaceDeleteInput.value = String(value || '').trim().toUpperCase() }
async function deleteWholeWorkspace() {
  const jobId = workspaceDeleteTargetID.value
  if (!jobId || workspaceDeleteInput.value !== jobId || workspaceDeleting.value) return
  workspaceDeleteError.value = ''
  workspaceDeleting.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/eliminar/${encodeURIComponent(jobId)}`, {
      method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ confirmacion: workspaceDeleteInput.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo eliminar el espacio completo.')
    const deletedFolder = workspaceDeleteTargetName.value
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
    deleteWorkspaceDialog.value = false
    selectedSavedWorkspace.value = ''
    await loadSavedWorkspaces()
    workspaceDeleteTargetID.value = ''
    workspaceDeleteTargetName.value = ''
    workspaceDeleteInput.value = ''
    snackbar.value = `Se eliminó ${deletedFolder} y sus archivos.`
  } catch (error) { workspaceDeleteError.value = error.message || 'No se pudo eliminar el espacio completo.' }
  finally { workspaceDeleting.value = false }
}
async function deleteWorkspacePDF() {
  if (!deletePDFTarget.value || workspaceBusy.value) return
  workspaceNotice.value = ''; workspaceBusy.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/documentos/${encodeURIComponent(ingestResult.value.job_id)}`, {
      method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: deletePDFTarget.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo quitar el PDF.')
    workspacePDFs.value = data.documents || []
    const previousPath = deletePDFTarget.value
    deletePDFTarget.value = ''
    if (selectedWorkspacePDF.value === previousPath) selectWorkspacePDF(workspacePatientPDFs.value[0]?.path || '')
    workspaceNotice.value = 'PDF quitado de la carpeta del paciente.'
    deletePDFDialog.value = false
  } catch (error) { workspaceNotice.value = error.message || 'No se pudo quitar el PDF.' }
  finally { workspaceBusy.value = false }
}
async function downloadWorkspaceZIP() {
  if (!selectedZipWorkspace.value || zipDownloading.value) return
  zipDownloadError.value = ''
  zipDownloadNotice.value = ''
  zipDownloading.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/descargar/${encodeURIComponent(selectedZipWorkspace.value)}`, { credentials: 'same-origin' })
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `No se pudo descargar el ZIP (HTTP ${response.status}).`)
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = response.headers.get('Content-Disposition')?.match(/filename="?([^";]+)"?/i)?.[1] || `${selectedZipWorkspace.value}.zip`
    link.click()
    URL.revokeObjectURL(url)
    zipDownloadNotice.value = 'Se descargó el ZIP con los archivos actuales del expediente.'
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
    await loadIngestPreview(data.job_id)
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
const savedWorkspaceItems = computed(() => savedWorkspaces.value.map(workspace => {
  const monthName = ingestMonths.find(month => month.value === String(workspace.period || '').slice(5, 7))?.title || workspace.mes
  const serviceName = ingestServices.find(service => service.value === workspace.tipo_servicio)?.title || workspace.tipo_servicio
  const state = workspace.status === 'STAGED' ? 'pendiente de preparar' : workspace.status === 'INCOMPLETE' ? 'incompleto' : 'preparado'
  return { title: `${serviceName} · ${monthName} ${workspace.anio} · ${state} · recibido por ${workspace.creado_por || 'usuario anterior'}`, value: workspace.job_id }
}))
const zipDownloadWorkspaceItems = computed(() => {
  const eligibleIDs = new Set(savedWorkspaces.value.filter(workspace => ['PROCESSED', 'INCOMPLETE'].includes(workspace.status)).map(workspace => workspace.job_id))
  return savedWorkspaceItems.value.filter(item => eligibleIDs.has(item.value))
})
const filteredSavedWorkspaces = computed(() => {
  const normalizedQuery = String(savedWorkspaceSearch.value || '').trim().toLocaleLowerCase('es').normalize('NFD').replace(/[\u0300-\u036f]/g, '')
  return savedWorkspaces.value.filter(workspace => {
    if (savedWorkspaceStatusFilter.value !== 'ALL' && workspace.status !== savedWorkspaceStatusFilter.value) return false
    if (!normalizedQuery) return true
    const haystack = [coverageServiceLabel(workspace.tipo_servicio), workspace.tipo_servicio, coverageMonthLabel(workspace.mes), workspace.mes, workspace.anio, workspace.period, workspace.job_id, workspace.creado_por, workspace.message, workspaceStatusLabel(workspace.status), ...(workspace.missing_documents || [])]
      .filter(Boolean).join(' ').toLocaleLowerCase('es').normalize('NFD').replace(/[\u0300-\u036f]/g, '')
    return haystack.includes(normalizedQuery)
  })
})
const savedWorkspaceStatusFilters = computed(() => [
  { title: `Todos (${savedWorkspaces.value.length})`, value: 'ALL' },
  { title: `Incompletos (${savedWorkspaces.value.filter(item => item.status === 'INCOMPLETE').length})`, value: 'INCOMPLETE' },
  { title: `Preparados (${savedWorkspaces.value.filter(item => item.status === 'PROCESSED').length})`, value: 'PROCESSED' },
  { title: `Por preparar (${savedWorkspaces.value.filter(item => item.status === 'STAGED').length})`, value: 'STAGED' },
  { title: `Revisión (${savedWorkspaces.value.filter(item => item.status === 'REQUIERE_REVISION').length})`, value: 'REQUIERE_REVISION' },
])
function workspaceStatusLabel(status) {
  return ({ STAGED: 'Pendiente de preparar', INCOMPLETE: 'Incompleto', PROCESSED: 'Preparado', REQUIERE_REVISION: 'Requiere revisión' })[status] || status || 'Estado desconocido'
}
function workspaceStatusColor(status) {
  return ({ STAGED: 'info', INCOMPLETE: 'warning', PROCESSED: 'success', REQUIERE_REVISION: 'error' })[status] || 'secondary'
}
function workspaceReceivedAt(value) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('es-EC', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
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
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
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
  const title = computed(() => activePage.value === 'planilla' ? 'Planilla digital' : activePage.value === 'coberturas' ? 'Hojas de cobertura' : activePage.value === 'patient-documents' ? 'Abrir documentos del paciente' : activePage.value === 'zip-download' ? 'Descargar expediente ZIP' : activePage.value === 'ingesta' ? 'Recepción de planillas' : activeFolder.value === 'Todos los documentos' ? 'Mis documentos' : activeFolder.value)
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
        <div class="nav-label">DATOS</div>
        <button class="nav-item" :class="{selected:activePage==='planilla'}" @click="openPlanilla"><v-icon icon="mdi-table-large" size="19"/><span>Planilla digital</span></button>
        <div class="nav-label">PROCESOS</div>
        <button class="nav-item" :class="{selected:activePage==='ingesta'}" @click="activePage='ingesta'"><v-icon icon="mdi-cloud-upload-outline" size="19"/><span>Recibir planillas</span></button>
        <button class="nav-item" :class="{selected:activePage==='coberturas'}" aria-label="Descargar hojas de cobertura" @click="openCoverageDownloads"><v-icon icon="mdi-file-download-outline" size="19"/><span>Descargar hojas de cobertura</span></button>
        <button class="nav-item" :class="{selected:activePage==='patient-documents'}" @click="openPatientDocumentsPage"><v-icon icon="mdi-folder-account-outline" size="19"/><span>Abrir documentos del paciente</span></button>
        <button class="nav-item" :class="{selected:activePage==='zip-download'}" @click="openZipDownloadPage"><v-icon icon="mdi-folder-zip-outline" size="19"/><span>Descargar expediente ZIP</span></button>
        <div class="sidebar-bottom"><button class="profile" @click="signOut"><span class="avatar">{{ displayUser.slice(0,1) }}</span><span class="profile-copy"><b>{{ displayUser }}</b><small>Cerrar sesión</small></span><v-icon icon="mdi-logout" size="18"/></button></div>
      </aside>
      <main class="main-area">
        <header class="topbar"><div class="breadcrumbs"><span>SPD MSP</span><v-icon icon="mdi-chevron-right" size="16"/><b>{{ title }}</b></div><div class="top-actions"><span class="avatar top-avatar">{{ displayUser.slice(0,1) }}</span></div></header>
        <section v-if="activePage==='documents'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">BIBLIOTECA PERSONAL</div><h1>{{ title }}<span class="title-period">.</span></h1><p class="subtitle">Busca, organiza y abre tus documentos guardados.</p></div><button class="primary-upload" @click="uploadInput?.click()"><v-icon icon="mdi-upload" size="18"/> Subir documento</button></div>
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
            <div class="coverage-toolbar"><v-select :model-value="selectedCoverageWorkspace" :items="savedWorkspaces.filter(workspace => ['PROCESSED','INCOMPLETE'].includes(workspace.status)).map(workspace => ({ title: `${coverageServiceLabel(workspace.tipo_servicio)} · ${coverageMonthLabel(workspace.mes)} ${workspace.anio}`, value: workspace.job_id }))" label="Servicio y período del ZIP" placeholder="Selecciona un lote preparado" prepend-inner-icon="mdi-folder-zip-outline" density="comfortable" hide-details :disabled="savedWorkspacesLoading || coverageLoading || coverageGenerating" @update:model-value="loadCoveragePlanillas"/><v-btn color="primary" prepend-icon="mdi-content-save-outline" :loading="coverageGenerating" :disabled="!coverageSelectedIds.length || coverageGenerating" @click="generateCoverageSheets">Generar y guardar ({{ coverageSelectedIds.length }})</v-btn><v-btn variant="tonal" prepend-icon="mdi-download-outline" :loading="coverageGenerating" :disabled="!coverageSelectedIds.length || coverageSelectedIds.length > 500 || coverageGenerating" @click="downloadSelectedCoverageSheets">Descargar ZIP ({{ coverageSelectedIds.length }})</v-btn></div>
            <div v-if="coverageProgress" class="coverage-notice"><v-progress-circular indeterminate color="primary" size="18"/><span>{{ coverageProgress }}</span></div>
            <div v-if="coverageNotice" class="coverage-notice"><v-icon icon="mdi-information-outline"/><span>{{ coverageNotice }}</span></div>
            <div v-if="coverageError" class="planilla-state planilla-error"><v-icon icon="mdi-alert-circle-outline"/><span>{{ coverageError }}</span><v-btn size="small" variant="text" @click="loadCoveragePlanillas()">Reintentar</v-btn></div>
            <div v-else-if="coverageLoading" class="planilla-state"><v-progress-circular indeterminate color="primary" size="22"/><span>Consultando planillas MSP del lote…</span></div>
            <div v-else-if="!selectedCoverageWorkspace" class="planilla-state"><v-icon icon="mdi-folder-search-outline"/><span>Prepara primero el ZIP en “Recibir planillas” y selecciona aquí ese mismo servicio y período.</span></div>
            <div v-else-if="!coveragePlanillas.length" class="planilla-state"><v-icon icon="mdi-file-search-outline"/><span>No se encontraron planillas MSP de ese período en las carpetas del ZIP.</span></div>
            <div v-else class="planilla-table-wrap"><v-table class="planilla-table" density="comfortable" fixed-header height="min(62vh, 620px)"><thead><tr><th><label title="Seleccionar todas las pendientes"><input type="checkbox" :checked="coverageAllSelected" :disabled="!coverageSelectableRows.length || coverageGenerating" aria-label="Seleccionar todas las planillas pendientes" @change="toggleAllCoveragePlanillas"/> Todas</label></th><th>Trámite del ZIP</th><th>Paciente</th><th>Fecha hasta</th><th>Estado de cobertura</th><th>Acción manual</th></tr></thead><tbody><tr v-for="row in coveragePlanillas" :key="row.pdi_id"><td><input type="checkbox" :checked="coverageSelectedIds.includes(row.pdi_id)" :disabled="(row.pdi_cobertura === 'S' && !row.hoja_generada) || coverageGenerating" :aria-label="`Seleccionar planilla ${row.pdi_tramite}`" @change="toggleCoveragePlanilla(row)"/></td><td>{{ row.pdi_tramite }}</td><td>{{ row.paciente || '—' }}</td><td>{{ row.fecha_hasta || '—' }}</td><td><v-chip size="small" :color="row.pdi_cobertura === 'S' ? 'success' : row.hoja_generada ? 'info' : (row.descarga_manual || row.motivo_manual) ? 'error' : 'warning'" variant="tonal">{{ row.pdi_cobertura === 'S' ? 'Generada' : row.hoja_generada ? 'PDF en expediente · Oracle pendiente' : row.descarga_manual ? 'Descarga manual' : row.motivo_manual ? 'Revisar datos' : 'Pendiente' }}</v-chip><small v-if="row.motivo_manual" class="coverage-failure">{{ row.motivo_manual }}</small></td><td><div v-if="row.descarga_manual" class="coverage-manual"><a href="https://coberturasalud.msp.gob.ec/" target="_blank" rel="noopener noreferrer">Abrir portal MSP</a><small v-if="row.coberturas_manual?.length">Descarga un PDF por cédula: {{ row.coberturas_manual.map(member => member.cedula).join(', ') }}</small><label class="coverage-manual-upload"><input type="file" accept="application/pdf,.pdf" multiple :disabled="coverageManualUploadingId === row.pdi_id" @change="uploadManualCoverageSheets(row,$event)"/>{{ coverageManualUploadingId === row.pdi_id ? 'Adjuntando…' : 'Adjuntar PDFs descargados' }}</label></div><span v-else>—</span></td></tr></tbody></v-table></div>
            <div v-if="coveragePlanillas.length" class="coverage-footnote">Marca la casilla del encabezado para seleccionar todas las pendientes. Se procesan en grupos de 10; las hojas generadas se guardan en cada expediente. El ZIP admite hasta 500 planillas.</div>
          </v-card>
        </section>
        <section v-else-if="activePage==='patient-documents'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">DOCUMENTOS DEL PACIENTE</div><h1>Abrir documentos del paciente<span class="title-period">.</span></h1><p class="subtitle">Elige un período guardado para revisar las carpetas de pacientes y sus PDFs.</p></div></div>
          <v-card class="planilla-card saved-workspaces-card" rounded="xl" elevation="0">
            <div class="coverage-toolbar">
              <v-autocomplete v-model="selectedSavedWorkspace" :items="savedWorkspaceItems" label="Período y servicio" placeholder="Busca un período guardado" prepend-inner-icon="mdi-folder-clock-outline" density="comfortable" variant="outlined" hide-details clearable :loading="savedWorkspacesLoading" :disabled="savedWorkspacesLoading || !savedWorkspaceItems.length"/>
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
          <div class="welcome-line"><div><div class="eyebrow">EXPORTAR EXPEDIENTE</div><h1>Descargar expediente ZIP<span class="title-period">.</span></h1><p class="subtitle">Elige un expediente preparado para descargar su estructura y los archivos actuales.</p></div></div>
          <v-card class="planilla-card saved-workspaces-card" rounded="xl" elevation="0">
            <div class="coverage-toolbar">
              <v-autocomplete v-model="selectedZipWorkspace" :items="zipDownloadWorkspaceItems" label="Expediente y período" placeholder="Busca un expediente preparado" prepend-inner-icon="mdi-folder-zip-outline" density="comfortable" variant="outlined" hide-details clearable :loading="savedWorkspacesLoading" :disabled="savedWorkspacesLoading || !zipDownloadWorkspaceItems.length || zipDownloading"/>
              <v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar expedientes preparados" :loading="savedWorkspacesLoading" :disabled="savedWorkspacesLoading || zipDownloading" @click="openZipDownloadPage"/>
            </div>
            <v-alert v-if="savedWorkspacesError" class="mt-4" type="error" variant="tonal" density="comfortable">{{ savedWorkspacesError }}<v-btn type="button" size="small" variant="text" @click="openZipDownloadPage">Reintentar</v-btn></v-alert>
            <div v-else-if="savedWorkspacesLoading" class="planilla-state"><v-progress-circular indeterminate color="primary" size="22"/><span>Cargando expedientes…</span></div>
            <div v-else-if="!zipDownloadWorkspaceItems.length" class="planilla-state"><v-icon icon="mdi-folder-search-outline" size="25"/><span>No hay expedientes preparados para descargar todavía.</span><v-btn type="button" variant="text" color="primary" @click="activePage='ingesta'">Recibir planillas</v-btn></div>
            <template v-else>
              <v-alert v-if="zipDownloadError" class="mt-4" type="warning" variant="tonal" density="comfortable">{{ zipDownloadError }}<v-btn type="button" size="small" variant="text" @click="openPatientDocumentsPage">Abrir documentos del paciente</v-btn></v-alert>
              <v-alert v-if="zipDownloadNotice" class="mt-4" type="success" variant="tonal" density="comfortable">{{ zipDownloadNotice }}</v-alert>
              <div class="workspace-download-row"><span>Los grupos de PDFs pendientes de fusionar deben resolverse antes de descargar el ZIP.</span><v-btn type="button" color="primary" prepend-icon="mdi-folder-zip-outline" :loading="zipDownloading" :disabled="!selectedZipWorkspace || zipDownloading" @click="downloadWorkspaceZIP">Descargar expediente ZIP</v-btn></div>
            </template>
          </v-card>
        </section>
        <section v-else class="content-wrap ingest-content" :class="{'ingest-workspace-active': !!ingestResult?.output}">
          <div v-if="!ingestResult?.output" class="welcome-line"><div><div class="eyebrow">RECEPCIÓN DE EXPEDIENTES</div><h1>Recibir lote de planillas<span class="title-period">.</span></h1><p class="subtitle">Carga el ZIP de planillas y añade los documentos habilitantes cuando los tengas.</p></div></div>
          <div v-else class="ingest-workspace-toolbar">
            <div><small>PERÍODO ABIERTO</small><strong>{{ coverageServiceLabel(ingestResult.tipo_servicio || ingestService) }} · {{ coverageMonthLabel(ingestResult.mes) }} {{ ingestResult.anio }}</strong><v-chip size="small" :color="workspaceStatusColor(ingestResult.status)" variant="tonal">{{ workspaceStatusLabel(ingestResult.status) }}</v-chip></div>
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
                  <v-text-field v-model="savedWorkspaceSearch" label="Buscar período" placeholder="Servicio, mes, ID o usuario" prepend-inner-icon="mdi-magnify" density="comfortable" variant="outlined" hide-details clearable/>
                  <v-chip-group v-model="savedWorkspaceStatusFilter" selected-class="filter-chip-selected" mandatory color="primary" class="saved-workspace-status-filters">
                    <v-chip v-for="filter in savedWorkspaceStatusFilters" :key="filter.value" :value="filter.value" size="small" variant="outlined">{{ filter.title }}</v-chip>
                  </v-chip-group>
                </div>
                <div v-if="!filteredSavedWorkspaces.length" class="saved-workspaces-empty saved-workspaces-no-results"><v-icon icon="mdi-filter-remove-outline" size="24"/><span><strong>No hay períodos que coincidan</strong><small>Cambia el texto de búsqueda o el estado seleccionado.</small></span><v-btn type="button" variant="text" @click="savedWorkspaceSearch='';savedWorkspaceStatusFilter='ALL'">Limpiar filtros</v-btn></div>
                <div v-else class="saved-workspace-list">
                  <article v-for="workspace in filteredSavedWorkspaces" :key="workspace.job_id" class="saved-workspace-card">
                    <div class="saved-workspace-card-main">
                      <span class="saved-workspace-icon"><v-icon icon="mdi-folder-zip-outline" size="22"/></span>
                      <div class="saved-workspace-card-copy">
                        <div class="saved-workspace-title-row"><h3>{{ coverageServiceLabel(workspace.tipo_servicio) }} · {{ coverageMonthLabel(workspace.mes) }} {{ workspace.anio }}</h3><v-chip size="small" :color="workspaceStatusColor(workspace.status)" variant="tonal">{{ workspaceStatusLabel(workspace.status) }}</v-chip></div>
                        <p v-if="workspace.missing_documents?.length" class="saved-workspace-missing"><v-icon icon="mdi-alert-circle-outline" size="16"/> Faltan: {{ workspace.missing_documents.join(', ') }}</p>
                        <p v-else-if="workspace.message" class="saved-workspace-message">{{ workspace.message }}</p>
                        <p v-else class="saved-workspace-message">Período listo para continuar.</p>
                        <div class="saved-workspace-meta"><span v-if="workspaceReceivedAt(workspace.received_at)"><v-icon icon="mdi-clock-outline" size="14"/> {{ workspaceReceivedAt(workspace.received_at) }}</span><span><v-icon icon="mdi-account-outline" size="14"/> {{ workspace.creado_por || 'Usuario anterior' }}</span><code>{{ workspace.job_id }}</code></div>
                      </div>
                    </div>
                    <div class="saved-workspace-card-actions"><v-btn type="button" color="primary" prepend-icon="mdi-folder-open-outline" :loading="openingSavedWorkspace && selectedSavedWorkspace === workspace.job_id" :disabled="openingSavedWorkspace || workspaceDeleting" @click="openSavedWorkspace(workspace.job_id)">Continuar</v-btn><v-btn type="button" icon="mdi-delete-outline" variant="text" color="error" :aria-label="`Eliminar período ${workspace.job_id}`" title="Eliminar período" :disabled="openingSavedWorkspace || workspaceDeleting" @click="requestDeleteWorkspace(workspace.job_id)"/></div>
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
                <div class="ingest-preview-stats">
                  <div><strong>{{ ingestPreview.carpetas_tramite }}</strong><span>carpetas de trámites</span></div>
                  <div><strong>{{ ingestPreview.pdfs }}</strong><span>PDFs en el ZIP</span></div>
                  <div><strong>{{ ingestPreview.tramites_en_oracle }}</strong><span>trámites encontrados en Oracle</span></div>
                  <div><strong>{{ ingestPreview.tramites_sin_oracle }}</strong><span>trámites sin coincidencia</span></div>
                  <div><strong>{{ ingestPreview.tramites_oracle_sin_zip }}</strong><span>trámites Oracle sin carpeta en ZIP</span></div>
                  <div><strong>{{ ingestPreview.entradas_invalidas }}</strong><span>rutas o archivos inválidos</span></div>
                </div>
                <v-alert v-if="ingestPreview.tramites_sin_oracle || ingestPreview.entradas_invalidas || ingestPreview.tramites_sin_paciente_oracle" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-alert-outline">
                  Revisa la tabla antes de preparar. El cruce se hace con <code>PDI_TRAMITE</code> dentro de las planillas marcadas para {{ ingestPreview.mes }}/{{ ingestPreview.anio }}. Las fechas de atención y <code>PDI_SERVICIO</code> se muestran para que confirmes que el contenido corresponde al lote; no se excluyen automáticamente por esas columnas.
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
    <v-dialog v-model="patientDocumentsDialog" class="patient-documents-dialog" width="96vw" max-width="1800" :persistent="patientDocumentsLocked" aria-labelledby="patient-documents-title">
      <v-card class="patient-documents-modal">
        <div class="patient-documents-header">
          <div><h2 id="patient-documents-title">Documentos del paciente</h2><p>{{ coverageServiceLabel(ingestResult?.tipo_servicio) }} · {{ coverageMonthLabel(ingestResult?.mes) }} {{ ingestResult?.anio }}</p></div>
          <div class="patient-documents-header-actions"><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar lista de PDFs" :loading="workspaceBusy" :disabled="patientDocumentsLocked" @click="loadWorkspaceDocuments(ingestResult.job_id, selectedWorkspacePDF)"/><v-btn type="button" variant="tonal" prepend-icon="mdi-close" :disabled="patientDocumentsLocked" @click="patientDocumentsDialog = false">Cerrar</v-btn></div>
        </div>
        <v-card-text v-if="ingestResult?.output" class="patient-documents-body">
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
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" :disabled="mergeSending || mergeAllSending" @click="mergePDFDialog=false">Cancelar</v-btn><v-btn type="button" color="warning" prepend-icon="mdi-file-document-multiple-outline" :loading="mergeSending" :disabled="mergeAllSending" @click="confirmWorkspaceFusion">Fusionar PDFs</v-btn></v-card-actions>
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
              <v-dialog v-model="replacePDFDialog" max-width="460" :persistent="workspaceSending">
                <v-card class="action-dialog">
                  <v-card-title>Reemplazar PDF</v-card-title>
                  <v-alert v-if="replacePDFError" type="error" variant="tonal" role="alert" class="mx-6">{{ replacePDFError }}</v-alert>
                  <v-card-text><p>El archivo local <strong>{{ replacePDFFile?.name }}</strong> reemplazará a <strong>{{ selectedWorkspaceDocument?.name }}</strong> dentro de <strong>{{ workspacePatient }}</strong>. El archivo local adoptará automáticamente el nombre del PDF seleccionado; su nombre original se conservará solo como referencia. La versión anterior queda en las fuentes del expediente.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" :disabled="workspaceSending" @click="replacePDFDialog=false;replacePDFFile=null;replacePDFTarget='';replacePDFError=''">Cancelar</v-btn><v-btn type="button" color="warning" :loading="workspaceSending" @click="confirmReplaceWorkspacePDF">Confirmar reemplazo</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
        </v-card-text>
      </v-card>
    </v-dialog>
    <v-dialog v-model="deleteWorkspaceDialog" max-width="600" persistent>
      <v-card class="action-dialog">
        <v-card-title>Eliminar período guardado</v-card-title>
        <v-card-text>
          <p>Se eliminará permanentemente <strong>{{ workspaceDeleteTargetName }}</strong>, con el ZIP fuente, los documentos habilitantes, reportes y archivos preparados. También se quitará su asociación en Oracle y sus coberturas quedarán pendientes para poder generarlas nuevamente al recrear el período. Esta acción no se puede deshacer.</p>
          <p>Para confirmar, escribe o pega el ID completo del período:</p>
          <div class="workspace-delete-code">{{ workspaceDeleteTargetID }}</div>
          <v-text-field :model-value="workspaceDeleteInput" label="ID del período" autocomplete="off" autocapitalize="characters" spellcheck="false" prepend-inner-icon="mdi-keyboard-outline" @update:model-value="setWorkspaceDeleteInput"/>
          <v-alert v-if="workspaceDeleteError" type="error" variant="tonal" density="comfortable">{{ workspaceDeleteError }}</v-alert>
        </v-card-text>
        <v-card-actions><v-spacer/><v-btn type="button" variant="text" :disabled="workspaceDeleting" @click="deleteWorkspaceDialog=false;workspaceDeleteInput='';workspaceDeleteError='';workspaceDeleteTargetID='';workspaceDeleteTargetName=''">Cancelar</v-btn><v-btn type="button" color="error" prepend-icon="mdi-delete-forever-outline" :loading="workspaceDeleting" :disabled="workspaceDeleteInput !== workspaceDeleteTargetID" @click="deleteWholeWorkspace">Eliminar período definitivamente</v-btn></v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog v-model="createSpaceDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Crear un espacio</v-card-title><v-card-text><p>Organiza tus archivos en un espacio nuevo.</p><v-text-field v-model="newSpaceName" label="Nombre del espacio" placeholder="Ej. Clientes" prepend-inner-icon="mdi-folder-outline" maxlength="36" autofocus @keyup.enter="createSpace"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="createSpaceDialog=false">Cancelar</v-btn><v-btn color="primary" :disabled="!newSpaceName.trim()" @click="createSpace">Crear espacio</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="moveDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Mover {{ activeDoc ? 'documento' : `${selectedIds.length} documentos` }}</v-card-title><v-card-text><p>Elige el espacio de destino.</p><v-select v-model="moveTarget" :items="spaces" label="Espacio" prepend-inner-icon="mdi-folder-outline"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="moveDialog=false">Cancelar</v-btn><v-btn color="primary" @click="confirmMove">Mover</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="preview" max-width="720"><v-card v-if="preview" class="preview-card"><div class="preview-head"><div class="doc-type-icon" :class="'type-'+preview.color"><v-icon :icon="preview.icon" size="20"/></div><div class="preview-name"><b>{{ preview.name }}</b><small>{{ preview.ext }} · {{ preview.size }}</small></div><v-btn icon="mdi-download-outline" variant="text" aria-label="Descargar" @click="download(preview)"/><v-btn icon="mdi-delete-outline" variant="text" aria-label="Eliminar" @click="removeDoc(preview)"/><v-btn icon="mdi-close" variant="text" aria-label="Cerrar" @click="preview=null"/></div><div class="preview-body"><template v-if="preview.mime?.startsWith('image/')"><img :src="previewUrl" :alt="preview.name"/></template><template v-else-if="preview.mime==='application/pdf'"><iframe :src="previewUrl" :title="preview.name"/></template><div v-else class="preview-placeholder"><v-icon :icon="preview.icon" size="48"/><p>Vista previa disponible para imágenes y archivos PDF.</p><button class="primary-upload" @click="download(preview)"><v-icon icon="mdi-download-outline"/> Descargar archivo</button></div></div></v-card></v-dialog>
    <v-snackbar v-model="snackbar" timeout="2600" location="bottom end" color="primary">{{ snackbar }}<template #actions><v-btn variant="text" @click="snackbar=''">Cerrar</v-btn></template></v-snackbar>
  </v-app>
</template>
