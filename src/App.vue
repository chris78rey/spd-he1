<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { openDB } from 'idb'
import mspPdfCodes from '../catalogos/codigos_msp.json'

const dbPromise = openDB('folio-documentos', 1, { upgrade(db) { db.createObjectStore('files', { keyPath: 'id' }) } })
const folders = ['Todos los documentos', 'Mi espacio', 'Trabajo', 'Personal', 'Compartido conmigo', 'Favoritos', 'Papelera']
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
const activePage = ref('documents')
const planillaPage = ref(1)
const planillaData = ref({ columns: [], rows: [], total: 0, totalPages: 1 })
const planillaLoading = ref(false)
const planillaError = ref('')
const ingestMonth = ref('')
const ingestYear = ref(String(new Date().getFullYear()))
const ingestService = ref('')
const ingestServices = [
  { title: 'Hospitalización / Internación / Hospital del Día', value: 'HOSPITALIZACION' },
  { title: 'Emergencia', value: 'EMERGENCIA' },
  { title: 'Ambulatorio / Laboratorio Clínico', value: 'AMBULATORIO_LABORATORIO_CLINICO' },
  { title: 'Ambulatorio / Procedimientos', value: 'AMBULATORIO_PROCEDIMIENTOS' },
  { title: 'Ambulatorio / Consulta Externa', value: 'AMBULATORIO_CONSULTA_EXTERNA' },
  { title: 'Hemodiálisis', value: 'HEMODIALISIS' },
  { title: 'Diálisis Peritoneal', value: 'DIALISIS_PERITONEAL' },
  { title: 'Componentes Sanguíneos', value: 'COMPONENTES_SANGUINEOS' },
  { title: 'Transporte Sanitario', value: 'TRANSPORTE_SANITARIO' },
  { title: 'Trasplante', value: 'TRASPLANTE' },
  { title: 'Coberturas Compartidas', value: 'COBERTURAS_COMPARTIDAS' },
]
const ingestMonths = ['Enero','Febrero','Marzo','Abril','Mayo','Junio','Julio','Agosto','Septiembre','Octubre','Noviembre','Diciembre'].map((title, index) => ({ title, value: String(index + 1).padStart(2, '0') }))
const ingestFileFields = [
  { field: 'zip_file', title: 'Lote clínico', detail: 'Archivo ZIP con las carpetas numéricas de planillas', accept: '.zip', icon: 'mdi-folder-zip-outline' },
  { field: 'matriz_file', title: 'Matriz de planillaje', detail: 'Libro Excel habilitado para macros', accept: '.xlsm', icon: 'mdi-file-excel-outline' },
  { field: 'consolidada_file', title: 'Planilla consolidada', detail: 'PDF firmado por responsables médico y financiero', accept: '.pdf,application/pdf', icon: 'mdi-file-pdf-box' },
  { field: 'oficio_file', title: 'Oficio de pago', detail: 'PDF firmado por representante legal', accept: '.pdf,application/pdf', icon: 'mdi-file-document-outline' },
]
const ingestFiles = ref({ zip_file: null, matriz_file: null, consolidada_file: null, oficio_file: null })
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
const ingestPreviewPage = ref(1)
const ingestPreviewPageSize = 10
const savedWorkspaces = ref([])
const selectedSavedWorkspace = ref('')
const savedWorkspacesLoading = ref(false)
const openingSavedWorkspace = ref(false)
const workspacePDFs = ref([])
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
const replacePDFTarget = ref('')
const deletePDFTarget = ref('')
const deletePDFDialog = ref(false)
const deleteWorkspaceDialog = ref(false)
const workspaceDeleteCode = ref('')
const workspaceDeleteInput = ref('')
const workspaceDeleteError = ref('')
const workspaceDeleting = ref(false)
const selectedWorkspaceDocument = computed(() => workspacePDFs.value.find(document => document.path === selectedWorkspacePDF.value) || null)
const workspacePatientPDFs = computed(() => {
  const prefix = workspacePatient.value ? `4. EXPEDIENTES/${workspacePatient.value}/` : ''
  return prefix ? workspacePDFs.value.filter(document => document.path.startsWith(prefix)) : []
})
const mspPDFOptions = mspPdfCodes
function standardCodeForFilename(filename = '') {
  const canonical = filename.replace(/_\d+(?=\.pdf$)/i, '').toLowerCase()
  return mspPDFOptions.find(option => option.value.toLowerCase() === canonical)?.value || ''
}
const workspacePDFURL = computed(() => selectedWorkspacePDF.value && ingestResult.value?.job_id
  ? `/api/v1/expedientes/documentos/archivo/${encodeURIComponent(ingestResult.value.job_id)}?path=${encodeURIComponent(selectedWorkspacePDF.value)}`
  : '')
let dragDepth = 0
const spaces = computed(() => ['Trabajo','Personal','Mi espacio',...customSpaces.value])
const samples = [
  { id:'demo-1', name:'Propuesta de marca — Q3', ext:'PDF', size:'2.4 MB', date:'Hoy, 10:42', addedAt:Date.now()-1000*60*45, folder:'Trabajo', color:'pdf', icon:'mdi-file-pdf-box', author:'Tú', favorite:true },
  { id:'demo-2', name:'Presupuesto familiar', ext:'XLSX', size:'840 KB', date:'Ayer, 16:20', addedAt:Date.now()-1000*60*60*28, folder:'Personal', color:'sheet', icon:'mdi-file-excel', author:'Tú' },
  { id:'demo-3', name:'Notas de reunión · Equipo', ext:'DOCX', size:'1.2 MB', date:'Ayer, 09:15', addedAt:Date.now()-1000*60*60*33, folder:'Trabajo', color:'word', icon:'mdi-file-word', author:'Valentina R.' },
  { id:'demo-4', name:'Ideas para el proyecto', ext:'PDF', size:'3.1 MB', date:'18 sep, 14:30', addedAt:Date.now()-1000*60*60*24*11, folder:'Mi espacio', color:'pdf', icon:'mdi-file-pdf-box', author:'Tú' },
  { id:'demo-5', name:'Contrato de servicios', ext:'DOCX', size:'560 KB', date:'16 sep, 11:08', addedAt:Date.now()-1000*60*60*24*13, folder:'Trabajo', color:'word', icon:'mdi-file-word', author:'Tú' },
  { id:'demo-6', name:'Referencias visuales', ext:'JPG', size:'4.8 MB', date:'12 sep, 17:55', addedAt:Date.now()-1000*60*60*24*18, folder:'Personal', color:'image', icon:'mdi-image-outline', author:'Tú' },
]
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
    workspacePatients.value = data.patients || []
    workspaceNotice.value = 'PDFs añadidos al espacio persistente del expediente.'
    selectedWorkspacePDF.value = data.added?.at(-1)?.relative_path || selectedWorkspacePDF.value
    workspaceRename.value = standardCodeForFilename(selectedWorkspaceDocument.value?.name || '')
    ingestProgress.value = 100
  } catch (error) { workspaceNotice.value = error.message || 'No se pudieron añadir los PDFs.' }
  finally { workspaceSending.value = false }
}
async function confirmReplaceWorkspacePDF() {
  const current = workspacePDFs.value.find(document => document.path === replacePDFTarget.value)
  if (!current || !replacePDFFile.value || workspaceSending.value) return
  workspaceNotice.value = ''; workspaceSending.value = true; ingestProgress.value = 0
  const form = new FormData()
  const patient = current.path.split('/')[2]
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
    replacePDFFile.value = null
    selectedWorkspacePDF.value = current.path
    workspaceNotice.value = 'PDF reemplazado. La versión fuente se conserva en el expediente.'
    ingestProgress.value = 100
    replacePDFDialog.value = false
  } catch (error) { workspaceNotice.value = error.message || 'No se pudo reemplazar el PDF.' }
  finally { workspaceSending.value = false }
}
function requestDeleteWorkspacePDF(path) {
  deletePDFTarget.value = path
  deletePDFDialog.value = true
}
function requestDeleteWorkspace() {
  const random = new Uint32Array(1)
  crypto.getRandomValues(random)
  workspaceDeleteCode.value = String(10000000 + (random[0] % 90000000))
  workspaceDeleteInput.value = ''
  workspaceDeleteError.value = ''
  deleteWorkspaceDialog.value = true
}
function preventWorkspaceDeleteClipboard(event) { event.preventDefault() }
function preventWorkspaceDeletePaste(event) {
  if (event.inputType === 'insertFromPaste' || event.inputType === 'insertFromDrop') event.preventDefault()
}
function blockWorkspaceDeleteClipboardKeys(event) {
  if (((event.ctrlKey || event.metaKey) && ['v', 'c', 'x'].includes(event.key.toLowerCase())) || (event.shiftKey && event.key === 'Insert')) event.preventDefault()
}
function setWorkspaceDeleteInput(value) { workspaceDeleteInput.value = String(value || '').replace(/\D/g, '').slice(0, 8) }
async function deleteWholeWorkspace() {
  if (!ingestResult.value?.job_id || workspaceDeleteInput.value !== workspaceDeleteCode.value || workspaceDeleting.value) return
  workspaceDeleteError.value = ''
  workspaceDeleting.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/eliminar/${encodeURIComponent(ingestResult.value.job_id)}`, {
      method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ confirmacion: workspaceDeleteInput.value }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo eliminar el espacio completo.')
    const deletedFolder = ingestResult.value.workspace?.split('/').at(-1) || 'período'
    ingestResult.value = null
    ingestPreview.value = null
    ingestPreviewError.value = ''
    workspacePDFs.value = []
    workspacePatients.value = []
    workspacePatient.value = ''
    selectedWorkspacePDF.value = ''
    workspaceUploadFiles.value = []
    localStorage.removeItem('folio-ingest-result')
    deleteWorkspaceDialog.value = false
    await loadSavedWorkspaces()
    snackbar.value = `Se eliminó la carpeta completa ${deletedFolder} y sus fuentes.`
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
  if (!ingestResult.value?.job_id || workspaceBusy.value) return
  workspaceNotice.value = ''; workspaceBusy.value = true
  try {
    const response = await fetch(`/api/v1/expedientes/descargar/${encodeURIComponent(ingestResult.value.job_id)}`, { credentials: 'same-origin' })
    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `No se pudo descargar el ZIP (HTTP ${response.status}).`)
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${ingestResult.value.output.split('/').at(-1)}.zip`
    link.click()
    URL.revokeObjectURL(url)
    workspaceNotice.value = 'Se descargó el ZIP con la estructura actual del expediente.'
  } catch (error) { workspaceNotice.value = error.message || 'No se pudo descargar el ZIP.' }
  finally { workspaceBusy.value = false }
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
const savedWorkspaceItems = computed(() => savedWorkspaces.value.map(workspace => {
  const monthName = ingestMonths.find(month => month.value === String(workspace.period || '').slice(5, 7))?.title || workspace.mes
  const serviceName = ingestServices.find(service => service.value === workspace.tipo_servicio)?.title || workspace.tipo_servicio
  const state = workspace.status === 'STAGED' ? 'pendiente de preparar' : workspace.status === 'INCOMPLETE' ? 'incompleto' : 'preparado'
  return { title: `${serviceName} · ${monthName} ${workspace.anio} · ${state}`, value: workspace.job_id }
}))
async function loadSavedWorkspaces() {
  savedWorkspacesLoading.value = true
  try {
    const response = await fetch('/api/v1/expedientes', { credentials: 'same-origin' })
    if (!response.ok) return
    const data = await response.json()
    savedWorkspaces.value = data.workspaces || []
    if (!savedWorkspaces.value.some(workspace => workspace.job_id === selectedSavedWorkspace.value)) selectedSavedWorkspace.value = savedWorkspaces.value[0]?.job_id || ''
  } catch { /* Se puede continuar con la carga manual si no responde la lista. */ }
  finally { savedWorkspacesLoading.value = false }
}
async function openSavedWorkspace() {
  if (!selectedSavedWorkspace.value || openingSavedWorkspace.value) return
  openingSavedWorkspace.value = true
  ingestError.value = ''
  try {
    const response = await fetch(`/api/v1/ingesta/estado/${encodeURIComponent(selectedSavedWorkspace.value)}`, { credentials: 'same-origin' })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'No se pudo abrir el espacio guardado.')
    ingestResult.value = data
    localStorage.setItem('folio-ingest-result', JSON.stringify(data))
    if (data.output) await loadWorkspaceDocuments(data.job_id)
    else { workspacePDFs.value = []; workspacePatients.value = [] }
    await loadIngestPreview(data.job_id)
  } catch (error) { ingestError.value = error.message || 'No se pudo abrir el espacio guardado.' }
  finally { openingSavedWorkspace.value = false }
}
function startNewIngest() {
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
const completionFields = computed(() => ingestFileFields.filter(item => item.field !== 'zip_file' && (ingestResult.value?.missing_documents || []).some(label => label.startsWith(item.title === 'Matriz de planillaje' ? 'Matriz' : item.title === 'Planilla consolidada' ? 'Planilla consolidada' : 'Oficio'))))
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
    if (savedIngest?.job_id) ingestResult.value = savedIngest
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
  let items = [...samples, ...docs.value]
  if (activeFolder.value === 'Favoritos') items = items.filter(d => d.favorite)
  else if (activeFolder.value === 'Compartido conmigo') items = items.filter(d => d.author !== 'Tú')
  else if (activeFolder.value === 'Papelera') items = []
  else if (activeFolder.value !== 'Todos los documentos') items = items.filter(d => d.folder === activeFolder.value)
  if (query.value.trim()) { const q = query.value.toLowerCase(); items = items.filter(d => (d.name+' '+d.ext+' '+d.folder).toLowerCase().includes(q)) }
  if (typeFilter.value !== 'Todo') items = items.filter(d => d.color === typeFilter.value)
  if (dateFilter.value !== 'Cualquier fecha') { const days = dateFilter.value === 'Últimos 7 días' ? 7 : 30; const since = Date.now()-days*24*60*60*1000; items = items.filter(d => d.addedAt && d.addedAt >= since) }
  if (sortBy.value === 'Nombre') items.sort((a,b) => a.name.localeCompare(b.name))
  else if (sortBy.value === 'Tamaño') items.sort((a,b) => (b.bytes||0)-(a.bytes||0))
  else items.sort((a,b) => (b.addedAt||0)-(a.addedAt||0))
  return items
})
  const title = computed(() => activePage.value === 'planilla' ? 'Planilla digital' : activePage.value === 'ingesta' ? 'Recepción de planillas' : activeFolder.value === 'Todos los documentos' ? 'Mis documentos' : activeFolder.value)
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
  for (const doc of moving) { if (doc.id.startsWith('demo-')) continue; doc.folder = moveTarget.value; await (await dbPromise).put('files', doc); count++ }
  if (moving.some(doc => doc.id.startsWith('demo-'))) snackbar.value = 'Los archivos de ejemplo no se pueden mover.'
  else snackbar.value = `${count} ${count === 1 ? 'documento movido' : 'documentos movidos'} a «${moveTarget.value}»`
  moveDialog.value = false; activeDoc.value = null; selectedIds.value = []
}
async function bulkFavorite() { const moving = docs.value.filter(doc => selectedIds.value.includes(doc.id) && !doc.id.startsWith('demo-')); for (const doc of moving) { doc.favorite = true; await (await dbPromise).put('files', doc) }; snackbar.value = `${moving.length} ${moving.length === 1 ? 'documento añadido' : 'documentos añadidos'} a favoritos`; selectedIds.value = [] }
async function bulkDelete() { const moving = docs.value.filter(doc => selectedIds.value.includes(doc.id) && !doc.id.startsWith('demo-')); for (const doc of moving) await (await dbPromise).delete('files', doc.id); docs.value = docs.value.filter(doc => !selectedIds.value.includes(doc.id) || doc.id.startsWith('demo-')); snackbar.value = `${moving.length} ${moving.length === 1 ? 'documento eliminado' : 'documentos eliminados'}`; selectedIds.value = [] }
async function upload(event) {
  const files = [...(event.target.files || [])]
  if (!files.length) return
  const db = await dbPromise
  for (const file of files) {
    const ext = (file.name.split('.').pop() || 'FILE').toUpperCase()
    const typeMap = { PDF:['pdf','mdi-file-pdf-box'], DOC:['word','mdi-file-word'], DOCX:['word','mdi-file-word'], XLS:['sheet','mdi-file-excel'], XLSX:['sheet','mdi-file-excel'], PNG:['image','mdi-image-outline'], JPG:['image','mdi-image-outline'], JPEG:['image','mdi-image-outline'] }
    const [color, icon] = typeMap[ext] || ['other','mdi-file-outline']
    const doc = { id: crypto.randomUUID(), name:file.name, ext, size:prettySize(file.size), bytes:file.size, date:'Ahora', addedAt:Date.now(), folder:activeFolder.value === 'Todos los documentos' ? 'Mi espacio' : activeFolder.value, color, icon, author:'Tú', blob:file, mime:file.type, favorite:false }
    await db.put('files', doc)
    docs.value.unshift(doc)
  }
  snackbar.value = `${files.length} ${files.length === 1 ? 'documento añadido' : 'documentos añadidos'} a tu espacio`
  event.target.value = ''
}
async function toggleFavorite(doc) {
  if (doc.id.startsWith('demo-')) { doc.favorite = !doc.favorite; snackbar.value = doc.favorite ? 'Añadido a favoritos' : 'Quitado de favoritos'; return }
  doc.favorite = !doc.favorite; await (await dbPromise).put('files', doc); snackbar.value = doc.favorite ? 'Añadido a favoritos' : 'Quitado de favoritos'
}
function openPreview(doc) { if (!doc.blob) { snackbar.value = 'Sube un archivo para ver su contenido.'; return }; preview.value = doc }
function download(doc) { if (!doc.blob) { snackbar.value = 'Los documentos de ejemplo son solo una vista previa.'; return }; const url=URL.createObjectURL(doc.blob); const a=document.createElement('a'); a.href=url; a.download=doc.name; a.click(); URL.revokeObjectURL(url) }
async function removeDoc(doc) { if (doc.id.startsWith('demo-')) { snackbar.value='Los documentos de ejemplo no se pueden eliminar.'; return }; await (await dbPromise).delete('files', doc.id); docs.value=docs.value.filter(d=>d.id!==doc.id); preview.value=null; snackbar.value='Documento eliminado' }
</script>

<template>
  <v-app>
    <main v-if="authStatus !== 'authenticated'" class="login-page">
      <div class="login-orbit orbit-one"></div><div class="login-orbit orbit-two"></div>
      <section class="login-card" aria-labelledby="login-title">
        <a class="login-brand" href="#" aria-label="Folio"><span class="brand-mark"><v-icon icon="mdi-book-open-page-variant" size="21" /></span><span>folio<span class="brand-dot">.</span></span></a>
        <div v-if="authStatus === 'checking'" class="login-loading"><v-progress-circular indeterminate color="primary"/><p>Comprobando tu sesión…</p></div>
        <template v-else>
          <div class="login-heading"><span class="login-kicker">TU BIBLIOTECA, EN UN SOLO LUGAR</span><h1 id="login-title">Bienvenido a Folio</h1><p>Inicia sesión con tu usuario de Oracle. Necesitas el rol <strong>SPD_EXTERNOS</strong>.</p></div>
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
        <a class="brand" href="#" @click.prevent="activePage='documents';activeFolder=folders[0]"><span class="brand-mark"><v-icon icon="mdi-book-open-page-variant" size="21" /></span><span>folio<span class="brand-dot">.</span></span></a>
        <button class="upload-btn" @click="uploadInput?.click()"><v-icon icon="mdi-plus" size="20" /> Subir documento <v-icon class="upload-arrow" icon="mdi-chevron-down" size="17" /></button>
        <input ref="uploadInput" type="file" multiple hidden @change="upload" />
        <div class="nav-label">BIBLIOTECA</div>
        <button v-for="(folder,i) in folders.slice(0,6)" :key="folder" class="nav-item" :class="{selected:activePage==='documents'&&activeFolder===folder}" @click="activePage='documents';activeFolder=folder"><v-icon :icon="['mdi-view-grid-outline','mdi-folder-outline','mdi-briefcase-outline','mdi-account-outline','mdi-account-multiple-outline','mdi-star-outline'][i]" size="19"/><span>{{ folder }}</span><span v-if="folder==='Todos los documentos'" class="nav-count">{{ samples.length + docs.length }}</span></button>
        <div class="sidebar-divider"></div>
        <div class="nav-label">ESPACIOS</div>
        <button v-for="(f,i) in spaces" :key="f+'space'" class="nav-item space-item" :class="{selected:activePage==='documents'&&activeFolder===f}" @click="activePage='documents';activeFolder=f"><span class="space-dot" :class="'dot-'+(i%3)"></span><span>{{ f }}</span></button>
        <div class="sidebar-divider"></div>
        <div class="nav-label">DATOS</div>
        <button class="nav-item" :class="{selected:activePage==='planilla'}" @click="openPlanilla"><v-icon icon="mdi-table-large" size="19"/><span>Planilla digital</span></button>
        <div class="nav-label">PROCESOS</div>
        <button class="nav-item" :class="{selected:activePage==='ingesta'}" @click="activePage='ingesta'"><v-icon icon="mdi-cloud-upload-outline" size="19"/><span>Recibir planillas</span></button>
        <button class="add-space" @click="createSpaceDialog=true"><v-icon icon="mdi-plus" size="17"/> Crear espacio</button>
          <div class="sidebar-bottom"><div class="storage-row"><span>Almacenamiento</span><span>{{ docs.length ? prettySize(docs.reduce((a,d)=>a+(d.bytes||0),0)) : '0 MB' }} / 5 GB</span></div><div class="storage-track"><span :style="{width:Math.max(3,Math.min(100,docs.reduce((a,d)=>a+(d.bytes||0),0)/(5*1024*1024*1024)*100))+'%'}"></span></div><button class="profile" @click="signOut"><span class="avatar">{{ currentUser.slice(0,1).toUpperCase() }}</span><span class="profile-copy"><b>{{ currentUser }}</b><small>Cerrar sesión</small></span><v-icon icon="mdi-logout" size="18"/></button></div>
      </aside>
      <main class="main-area">
        <header class="topbar"><div class="breadcrumbs"><span>Espacios</span><v-icon icon="mdi-chevron-right" size="16"/><b>{{ title }}</b></div><div class="top-actions"><button class="icon-btn" aria-label="Ayuda" @click="snackbar='Tus documentos se guardan en este navegador.'"><v-icon icon="mdi-help-circle-outline"/></button><button class="icon-btn notification-btn" aria-label="Notificaciones" @click="snackbar='No tienes notificaciones nuevas.'"><v-icon icon="mdi-bell-outline"/><i></i></button><span class="avatar top-avatar">{{ currentUser.slice(0,1).toUpperCase() }}</span></div></header>
        <section v-if="activePage==='documents'" class="content-wrap">
          <div class="welcome-line"><div><div class="eyebrow">LUNES, 22 DE SEPTIEMBRE</div><h1>{{ title }}<span class="title-period">.</span></h1><p class="subtitle">Todo lo que necesitas, en un solo lugar.</p></div><button class="primary-upload" @click="uploadInput?.click()"><v-icon icon="mdi-upload" size="18"/> Subir documento</button></div>
          <div class="stats-row"><div class="stat-card"><span class="stat-icon green"><v-icon icon="mdi-file-multiple-outline"/></span><div><span class="stat-label">Documentos</span><strong>{{ samples.length + docs.length }} <small>archivos</small></strong></div></div><div class="stat-card"><span class="stat-icon peach"><v-icon icon="mdi-folder-multiple-outline"/></span><div><span class="stat-label">Espacios</span><strong>3 <small>activos</small></strong></div></div><div class="stat-card storage-stat"><span class="stat-icon lavender"><v-icon icon="mdi-cloud-outline"/></span><div class="stat-flex"><div><span class="stat-label">Almacenamiento</span><strong>{{ docs.length ? prettySize(docs.reduce((a,d)=>a+(d.bytes||0),0)) : '0 MB' }} <small>de 5 GB</small></strong></div><div class="storage-track mini"><span :style="{width:Math.max(3,Math.min(100,docs.reduce((a,d)=>a+(d.bytes||0),0)/(5*1024*1024*1024)*100))+'%'}"></span></div></div></div></div>
          <section class="recent-section"><div class="section-heading"><div><h2>{{ activeFolder==='Todos los documentos' ? 'Tus archivos' : 'Archivos' }} <span class="muted-count">{{ visibleDocs.length }}</span></h2><p>Organiza y encuentra lo que buscas.</p></div><button class="text-action" @click="activeFolder=folders[0]">Ver todo <v-icon icon="mdi-arrow-right" size="16"/></button></div>
            <div class="toolbar"><div class="search-wrap"><v-icon icon="mdi-magnify" size="19"/><input ref="searchInput" v-model="query" placeholder="Buscar documentos..." aria-label="Buscar documentos"/><kbd>⌘ K</kbd></div><div class="toolbar-right"><v-select v-model="sortBy" :items="['Recientes','Nombre','Tamaño']" prepend-inner-icon="mdi-sort" class="sort-select" aria-label="Ordenar documentos"/><div class="view-switch"><button :class="{active:view==='grid'}" aria-label="Vista de cuadrícula" @click="view='grid'"><v-icon icon="mdi-view-grid-outline" size="18"/></button><button :class="{active:view==='list'}" aria-label="Vista de lista" @click="view='list'"><v-icon icon="mdi-view-list-outline" size="19"/></button></div></div></div>
            <div class="filter-row"><v-chip-group v-model="typeFilter" selected-class="filter-chip-selected" mandatory color="primary"><v-chip value="Todo" size="small" variant="outlined" filter>Todos los tipos</v-chip><v-chip value="pdf" size="small" variant="outlined" prepend-icon="mdi-file-pdf-box">PDF</v-chip><v-chip value="word" size="small" variant="outlined" prepend-icon="mdi-file-word">Word</v-chip><v-chip value="sheet" size="small" variant="outlined" prepend-icon="mdi-file-excel">Hojas de cálculo</v-chip><v-chip value="image" size="small" variant="outlined" prepend-icon="mdi-image-outline">Imágenes</v-chip></v-chip-group><v-select v-model="dateFilter" class="date-filter" :items="['Cualquier fecha','Últimos 7 días','Últimos 30 días']" prepend-inner-icon="mdi-calendar-blank-outline" aria-label="Filtrar por fecha"/><v-chip v-if="query" size="small" closable variant="tonal" color="secondary" @click:close="query=''">“{{ query }}”</v-chip></div>
            <div v-if="selectedIds.length" class="selection-bar"><v-chip size="small" color="primary" variant="tonal" prepend-icon="mdi-check-circle-outline">{{ selectedIds.length }} seleccionados</v-chip><v-btn size="small" variant="text" @click="selectAll">{{ selectedIds.length === visibleDocs.length ? 'Quitar selección' : 'Seleccionar todos' }}</v-btn><v-spacer/><v-tooltip text="Mover a un espacio"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-folder-move-outline" size="small" variant="text" aria-label="Mover seleccionados" @click="startMove()"/></template></v-tooltip><v-tooltip text="Añadir a favoritos"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-star-outline" size="small" variant="text" aria-label="Favoritos" @click="bulkFavorite"/></template></v-tooltip><v-tooltip text="Eliminar"><template #activator="{ props }"><v-btn v-bind="props" icon="mdi-delete-outline" size="small" variant="text" color="error" aria-label="Eliminar seleccionados" @click="bulkDelete"/></template></v-tooltip><v-btn size="small" variant="text" @click="selectedIds=[]">Cancelar</v-btn></div>
            <div v-if="loading" class="empty-state">Cargando tus documentos…</div>
            <div v-else-if="!visibleDocs.length" class="empty-state"><span class="empty-icon"><v-icon icon="mdi-folder-search-outline" size="32"/></span><h3>No encontramos documentos</h3><p>Prueba con otra búsqueda o añade un documento a tu espacio.</p><button class="primary-upload" @click="uploadInput?.click()"><v-icon icon="mdi-upload" size="18"/> Subir documento</button></div>
            <div v-else class="document-grid" :class="{'list-view':view==='list'}"><article v-for="doc in visibleDocs" :key="doc.id" class="document-card" :class="{'is-selected':selectedIds.includes(doc.id)}" @click="openPreview(doc)"><div class="doc-cover" :class="'cover-'+doc.color"><div class="paper-sheet"><div class="sheet-header"><span class="file-badge" :class="'badge-'+doc.color">{{ doc.ext }}</span><v-menu location="bottom end"><template #activator="{ props }"><button v-bind="props" class="more-btn" aria-label="Opciones del documento" @click.stop><v-icon icon="mdi-dots-horizontal" size="19"/></button></template><v-list density="compact" min-width="185" class="doc-menu"><v-list-item prepend-icon="mdi-eye-outline" title="Vista previa" @click="openPreview(doc)"/><v-list-item :prepend-icon="doc.favorite ? 'mdi-star' : 'mdi-star-outline'" :title="doc.favorite ? 'Quitar favorito' : 'Añadir a favoritos'" @click="toggleFavorite(doc)"/><v-list-item prepend-icon="mdi-folder-move-outline" title="Mover a…" @click="startMove(doc)"/><v-list-item prepend-icon="mdi-download-outline" title="Descargar" @click="download(doc)"/><v-divider class="my-1"/><v-list-item prepend-icon="mdi-delete-outline" title="Eliminar" class="text-error" @click="removeDoc(doc)"/></v-list></v-menu></div><div v-if="doc.color==='image'" class="image-art"><div></div><span>✳</span><i></i></div><div v-else class="fake-lines"><span class="line-title"></span><span></span><span></span><span class="line-short"></span><span class="line-gap"></span><span></span><span class="line-mid"></span></div><span class="page-corner"></span></div><v-btn class="select-doc" :class="{checked:selectedIds.includes(doc.id)}" :icon="selectedIds.includes(doc.id) ? 'mdi-check-circle' : 'mdi-checkbox-blank-circle-outline'" size="small" variant="flat" aria-label="Seleccionar documento" @click.stop="toggleSelected(doc)"/></div><div class="doc-info"><div class="doc-type-icon" :class="'type-'+doc.color"><v-icon :icon="doc.icon" size="19"/></div><div class="doc-detail"><h3>{{ doc.name }}</h3><p>{{ doc.ext }} <i>·</i> {{ doc.size }} <i>·</i> {{ doc.date }}</p></div><v-tooltip :text="doc.favorite ? 'Favorito' : 'Más opciones'"><template #activator="{ props }"><v-btn v-bind="props" class="favorite-btn" :icon="doc.favorite ? 'mdi-star' : 'mdi-star-outline'" size="small" variant="text" :color="doc.favorite ? 'amber-darken-2' : 'grey'" aria-label="Añadir a favoritos" @click.stop="toggleFavorite(doc)"/></template></v-tooltip></div></article></div>
          </section>
          <section class="shared-banner"><div class="shared-ornament"><v-icon icon="mdi-folder-heart-outline" size="29"/></div><div><h3>Comparte tus ideas con el equipo</h3><p>Crea un espacio compartido y trabajen juntos en sus documentos.</p></div><button @click="snackbar='Pronto podrás invitar a tu equipo.'">Invitar a alguien <v-icon icon="mdi-arrow-right" size="16"/></button></section>
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
        <section v-else class="content-wrap ingest-content">
          <div class="welcome-line"><div><div class="eyebrow">RECEPCIÓN DE EXPEDIENTES</div><h1>Recibir lote de planillas<span class="title-period">.</span></h1><p class="subtitle">Carga el ZIP de planillas y añade los documentos habilitantes cuando los tengas.</p></div></div>
          <v-alert class="ingest-notice" type="info" variant="tonal" density="comfortable" prepend-icon="mdi-information-outline">Puedes empezar con el ZIP de planillas. Después añade la matriz, la planilla consolidada y el oficio al mismo expediente.</v-alert>
          <form class="ingest-form" @submit.prevent="submitIngest">
            <section v-if="savedWorkspaceItems.length || savedWorkspacesLoading" class="ingest-card saved-workspaces-card">
              <div class="ingest-card-heading"><span class="ingest-step"><v-icon icon="mdi-folder-clock-outline" size="18"/></span><div><h2>Volver a un período guardado</h2><p>Abre el mismo expediente para continuar donde lo dejaste, sin subir el ZIP otra vez.</p></div><v-spacer/><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar expedientes guardados" :loading="savedWorkspacesLoading" @click="loadSavedWorkspaces"/></div>
              <div class="saved-workspaces-row"><v-select v-model="selectedSavedWorkspace" :items="savedWorkspaceItems" label="Servicio y período" prepend-inner-icon="mdi-folder-open-outline" density="comfortable" hide-details/><v-btn type="button" color="primary" prepend-icon="mdi-folder-open-outline" :loading="openingSavedWorkspace" :disabled="!selectedSavedWorkspace" @click="openSavedWorkspace">Abrir período</v-btn></div>
            </section>
            <template v-if="!ingestResult">
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
          <section v-if="ingestResult" class="ingest-card current-job-card">
              <div class="ingest-card-heading"><span class="ingest-step">{{ ingestResult.status === 'STAGED' ? '3' : '✓' }}</span><div><h2>{{ ingestResult.status === 'STAGED' ? 'Lote de planillas recibido' : 'Espacio de trabajo del período' }}</h2><p>{{ ingestResult.status === 'STAGED' ? 'El ZIP ya está guardado. El siguiente paso prepara las carpetas por paciente.' : 'Este espacio se reutiliza para el mismo servicio y período.' }}</p></div></div>
              <div class="ingest-job-id"><span>ID del lote</span><code>{{ ingestResult.job_id }}</code></div>
              <v-alert v-if="ingestResult.reuse_notice" type="info" variant="tonal" density="comfortable">{{ ingestResult.reuse_notice }}</v-alert>
              <v-alert :type="ingestResult.status === 'PROCESSED' ? 'success' : 'info'" variant="tonal" density="comfortable" prepend-icon="mdi-information-outline">
                <strong>{{ ingestResult.status === 'STAGED' ? 'Siguiente paso: preparar expedientes' : ingestResult.status === 'INCOMPLETE' ? 'Expediente preparado; faltan documentos habilitantes.' : 'Expediente preparado para revisión.' }}</strong><div>{{ ingestResult.message }}</div>
              </v-alert>
              <div v-if="ingestResult.status === 'STAGED'" class="ingest-next-step"><p v-if="!ingestPreview">Revisa la vista previa antes de preparar el expediente.</p><p v-else-if="!ingestPreviewReadyToPrepare">Corrige el ZIP o el período seleccionado: hay carpetas que no se pueden emparejar o entradas fuera de la estructura esperada.</p><p v-else>Los {{ ingestPreview.pdfs }} PDFs están dentro de {{ ingestPreview.carpetas_tramite }} carpetas y todos sus trámites se encuentran en Oracle para el período seleccionado.</p><v-btn color="primary" size="large" prepend-icon="mdi-folder-cog-outline" :loading="ingestProcessing" :disabled="!ingestPreviewReadyToPrepare" @click="processIngest">Preparar expedientes</v-btn></div>
            </section>
            <section v-if="ingestResult && (ingestPreview || ingestPreviewLoading || ingestPreviewError)" class="ingest-card ingest-preview-card">
              <div class="ingest-card-heading"><span class="ingest-step"><v-icon icon="mdi-folder-search-outline" size="19"/></span><div><h2>Vista previa del ZIP y cruce con Oracle</h2><p>Período seleccionado: {{ ingestPreview?.mes || ingestResult.mes }}/{{ ingestPreview?.anio || ingestResult.anio }} · Servicio: {{ ingestPreview?.tipo_servicio || ingestService }}</p></div><v-spacer/><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar vista previa del ZIP" :loading="ingestPreviewLoading" @click="loadIngestPreview(ingestResult.job_id)"/></div>
              <div v-if="ingestPreviewLoading" class="planilla-state"><v-progress-circular indeterminate color="primary"/><span>Contando carpetas y PDFs, y cruzando trámites con Oracle…</span></div>
              <v-alert v-else-if="ingestPreviewError" type="warning" variant="tonal" density="comfortable">{{ ingestPreviewError }}<v-btn type="button" size="small" variant="text" @click="loadIngestPreview(ingestResult.job_id)">Reintentar</v-btn></v-alert>
              <template v-else-if="ingestPreview">
                <div class="ingest-preview-stats">
                  <div><strong>{{ ingestPreview.carpetas_tramite }}</strong><span>carpetas de trámites</span></div>
                  <div><strong>{{ ingestPreview.pdfs }}</strong><span>PDFs en el ZIP</span></div>
                  <div><strong>{{ ingestPreview.tramites_en_oracle }}</strong><span>trámites encontrados en Oracle</span></div>
                  <div><strong>{{ ingestPreview.tramites_sin_oracle }}</strong><span>trámites sin coincidencia</span></div>
                  <div><strong>{{ ingestPreview.entradas_invalidas }}</strong><span>rutas o archivos inválidos</span></div>
                </div>
                <v-alert v-if="ingestPreview.tramites_sin_oracle || ingestPreview.entradas_invalidas || ingestPreview.tramites_sin_paciente_oracle" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-alert-outline">
                  Revisa la tabla antes de preparar. El cruce se hace con <code>PDI_TRAMITE</code> dentro de las planillas marcadas para {{ ingestPreview.mes }}/{{ ingestPreview.anio }}. Las fechas de atención y <code>PDI_SERVICIO</code> se muestran para que confirmes que el contenido corresponde al lote; no se excluyen automáticamente por esas columnas.
                  <span v-if="ingestPreview.tramites_sin_paciente_oracle"> {{ ingestPreview.tramites_sin_paciente_oracle }} trámites encontrados no tienen nombre de paciente en Oracle.</span>
                  <span v-if="ingestPreview.rutas_invalidas?.length"> Rutas con problemas: {{ ingestPreview.rutas_invalidas.join(', ') }}<span v-if="ingestPreview.entradas_invalidas > ingestPreview.rutas_invalidas.length"> y otras {{ ingestPreview.entradas_invalidas - ingestPreview.rutas_invalidas.length }}.</span></span>
                </v-alert>
                <div class="ingest-preview-table-wrap">
                  <v-table class="planilla-table ingest-preview-table" density="comfortable"><thead><tr><th>Carpeta / PDI_TRAMITE</th><th>PDFs</th><th>Paciente Oracle</th><th>PDI_SERVICIO</th><th>PDI_FECHA_DESDE – HASTA</th><th>Cruce</th></tr></thead>
                    <tbody><tr v-for="folder in ingestPreviewRows" :key="folder.tramite"><td><code>{{ folder.tramite }}</code></td><td>{{ folder.pdfs }}</td><td>{{ folder.paciente || (folder.coincide_oracle ? 'Sin nombre en Oracle' : '—') }}</td><td>{{ folder.servicio_oracle || '—' }}</td><td>{{ folder.fecha_desde || '—' }} – {{ folder.fecha_hasta || '—' }}</td><td><v-chip size="small" :color="folder.coincide_oracle && !folder.paciente_faltante_oracle ? 'success' : 'warning'" variant="tonal">{{ !folder.coincide_oracle ? 'No encontrado' : folder.paciente_faltante_oracle ? 'Falta paciente' : 'Coincide' }}</v-chip></td></tr></tbody>
                  </v-table>
                </div>
                <div class="planilla-pagination"><span>Carpetas {{ ingestPreview.carpetas.length ? (ingestPreviewPage - 1) * ingestPreviewPageSize + 1 : 0 }}–{{ Math.min(ingestPreviewPage * ingestPreviewPageSize, ingestPreview.carpetas.length) }} de {{ ingestPreview.carpetas.length }}</span><v-pagination v-if="ingestPreviewPageCount > 1" v-model="ingestPreviewPage" :length="ingestPreviewPageCount" :total-visible="5" density="comfortable"/></div>
              </template>
            </section>
            <div v-if="ingestResult?.status !== 'STAGED' && ingestResult?.missing_documents?.length" class="ingest-card completion-card">
              <div class="ingest-card-heading"><span class="ingest-step">3</span><div><h2>Completar documentos</h2><p>Se guardarán en el mismo expediente, sin volver a subir el ZIP de planillas.</p></div></div>
              <v-chip v-for="item in ingestResult.missing_documents" :key="item" class="missing-chip" size="small" color="warning" variant="tonal">Falta: {{ item }}</v-chip>
              <div class="ingest-file-grid">
                <label v-for="item in completionFields" :key="item.field" class="ingest-file-card">
                  <input type="file" :accept="item.accept" :disabled="ingestProcessing" @change="selectCompletionFile(item.field, $event)" />
                  <span class="ingest-file-icon"><v-icon :icon="item.icon" size="22"/></span>
                  <span class="ingest-file-copy"><strong>{{ item.title }}</strong><small>{{ completionFiles[item.field]?.name || item.detail }}</small></span>
                  <v-icon :icon="completionFiles[item.field] ? 'mdi-check-circle' : 'mdi-plus-circle-outline'" :color="completionFiles[item.field] ? 'success' : 'grey'" size="20"/>
                </label>
              </div>
              <div class="ingest-submit-row"><span>Los documentos enviados se incorporan al espacio permanente de este lote.</span><v-btn color="primary" :loading="ingestProcessing" :disabled="!completionValid || !Object.values(completionFiles).some(Boolean)" prepend-icon="mdi-cloud-upload-outline" @click="addMissingDocuments">Añadir documentos</v-btn></div>
            </div>
            <section v-if="ingestResult?.output" class="ingest-card review-card">
              <div class="ingest-card-heading"><span class="ingest-step">4</span><div><h2>Revisión de documentos</h2><p>La clasificación se ejecuta al preparar el expediente. Los pendientes requieren revisión documental antes del cierre.</p></div></div>
              <v-alert v-if="ingestResult.clasificacion" type="info" variant="tonal" density="comfortable">{{ ingestResult.clasificacion.clasificados }} PDFs identificados y {{ ingestResult.clasificacion.pendientes }} pendientes, de {{ ingestResult.clasificacion.total }}. Lectura: {{ ingestResult.clasificacion.texto_vectorial }} con texto PDF, {{ ingestResult.clasificacion.ocr }} por OCR y {{ ingestResult.clasificacion.sin_texto_legible ?? 0 }} sin texto legible.</v-alert>
              <v-alert v-if="ingestResult.clasificacion?.documentos_fecha_fuera_atencion?.length" type="warning" variant="tonal" density="comfortable" prepend-icon="mdi-calendar-alert-outline">
                <strong>{{ ingestResult.clasificacion.documentos_fecha_fuera_atencion.length }} PDF con fechas fuera del intervalo de atención de Oracle.</strong>
                <p>Se compara el texto de las primeras páginas con PDI_FECHA_DESDE y PDI_FECHA_HASTA del PDI_TRAMITE correspondiente. Son alertas de revisión: no bloquean ni cambian el expediente.</p>
                <ul class="period-date-alert-list"><li v-for="alert in ingestResult.clasificacion.documentos_fecha_fuera_atencion" :key="alert.documento"><code>{{ alert.documento }}</code>: {{ alert.fechas_detectadas.join(', ') }}<span v-if="alert.intervalo_oracle"> (atención Oracle: {{ alert.intervalo_oracle }})</span></li></ul>
                <small>Las fechas se reconocen por texto u OCR; revisa el PDF original antes de decidir.</small>
              </v-alert>
              <v-alert v-if="ingestResult.clasificacion?.sin_intervalo_oracle" type="info" variant="tonal" density="comfortable" prepend-icon="mdi-calendar-question-outline">No se pudo comparar la fecha de {{ ingestResult.clasificacion.sin_intervalo_oracle }} PDF porque Oracle no tiene PDI_FECHA_DESDE o PDI_FECHA_HASTA para su trámite. Esto no bloquea la preparación.</v-alert>
              <details class="ingest-secondary-action"><summary>Volver a analizar los PDFs</summary><p>Repite la clasificación de los archivos fuente y actualiza la carpeta de trabajo. Puede tardar varios minutos.</p><v-btn variant="outlined" prepend-icon="mdi-text-box-search-outline" :loading="ingestProcessing" @click="classifyIngest">Reanalizar PDFs</v-btn></details>
              <details v-if="ingestResult.workspace || ingestResult.output" class="ingest-secondary-action"><summary>Ver ubicaciones del expediente</summary><p v-if="ingestResult.workspace">Espacio permanente: <code>{{ ingestResult.workspace }}</code></p><p>Carpeta preparada: <code>{{ ingestResult.output }}</code></p></details>
            </section>
            <section v-if="ingestResult?.output" class="ingest-card workspace-files-card">
              <div class="ingest-card-heading"><span class="ingest-step">5</span><div><h2>Documentos del paciente</h2><p>Elige una carpeta para ver, añadir o reemplazar sus PDFs.</p></div><v-spacer/><v-btn type="button" icon="mdi-refresh" variant="text" aria-label="Actualizar lista de PDFs" :loading="workspaceBusy" @click="loadWorkspaceDocuments(ingestResult.job_id, selectedWorkspacePDF)"/></div>
              <v-alert v-if="workspaceNotice" :type="workspaceNotice.includes('Se guardó') || workspaceNotice.includes('añadidos') || workspaceNotice.includes('reemplazado') || workspaceNotice.includes('quitado') || workspaceNotice.includes('descargó') ? 'success' : 'warning'" variant="tonal" density="compact" class="workspace-notice">{{ workspaceNotice }}</v-alert>
              <v-select v-model="workspacePatient" :items="workspacePatients" label="Carpeta del paciente" prepend-inner-icon="mdi-folder-account-outline" density="comfortable" class="workspace-patient-select" :disabled="!workspacePatients.length || replacePDFDialog" @update:model-value="selectWorkspacePatient"/>
              <div class="workspace-file-layout">
                <div class="workspace-file-list">
                  <div class="workspace-file-count">{{ workspacePatientPDFs.length }} PDFs en esta carpeta</div>
                  <div v-for="document in workspacePatientPDFs" :key="document.path" class="workspace-file-entry">
                    <button type="button" class="workspace-file-row" :class="{ selected: selectedWorkspacePDF === document.path }" @click.prevent="selectWorkspacePDF(document.path)">
                      <v-icon icon="mdi-file-pdf-box" color="error" size="20"/><span><strong>{{ document.name }}</strong><small>{{ document.path }}</small></span><v-icon v-if="selectedWorkspacePDF === document.path" icon="mdi-eye-outline" size="18"/>
                    </button>
                    <v-btn type="button" icon="mdi-delete-outline" variant="text" size="small" color="error" :aria-label="`Quitar ${document.name}`" title="Quitar PDF de esta carpeta" @click="requestDeleteWorkspacePDF(document.path)"/>
                  </div>
                  <div v-if="!workspacePatientPDFs.length && !workspaceBusy" class="workspace-empty">Esta carpeta todavía no tiene PDFs.</div>
                </div>
                <div class="workspace-preview">
                  <div v-if="selectedWorkspaceDocument" class="workspace-preview-heading"><strong>{{ selectedWorkspaceDocument.name }}</strong><span>{{ prettySize(selectedWorkspaceDocument.size) }}</span></div>
                  <iframe v-if="workspacePDFURL" :key="workspacePDFURL" :src="workspacePDFURL" :title="selectedWorkspaceDocument?.name || 'Vista previa del PDF'" />
                  <div v-else class="workspace-empty">Selecciona un PDF de la lista para verlo aquí.</div>
                </div>
              </div>
              <div v-if="selectedWorkspaceDocument" class="workspace-edit-row">
                <v-autocomplete v-model="workspaceRename" :items="mspPDFOptions" item-title="title" item-value="value" label="Nombre estándar MSP" placeholder="Selecciona el tipo de documento" prepend-inner-icon="mdi-rename-box-outline" density="comfortable" variant="outlined" clearable hide-details/>
                <v-btn type="button" color="primary" variant="tonal" prepend-icon="mdi-content-save-outline" :loading="workspaceBusy" :disabled="!workspaceRename" @click="renameWorkspacePDF">Guardar nombre</v-btn>
              </div>
              <p v-if="selectedWorkspaceDocument" class="workspace-hint">El nombre se elige del catálogo y siempre termina en .pdf. Si ya existe otro documento con ese código, se asigna una secuencia (_1, _2…).</p>
              <div class="workspace-add-row">
                <input ref="workspacePDFInput" type="file" accept=".pdf,application/pdf" multiple hidden @change="selectWorkspacePDFs" />
                <v-btn type="button" variant="outlined" prepend-icon="mdi-file-pdf-box" :disabled="workspaceSending" @click="workspacePDFInput?.click()">{{ workspaceUploadFiles.length ? 'Añadir más PDFs a la selección' : 'Seleccionar PDFs para añadir' }}</v-btn>
                <input ref="replacePDFInput" type="file" accept=".pdf,application/pdf" hidden @change="selectReplacementPDF" />
                <v-btn type="button" color="warning" variant="tonal" prepend-icon="mdi-file-replace-outline" :disabled="!selectedWorkspaceDocument || workspaceSending" @click="replacePDFInput?.click()">Reemplazar PDF seleccionado</v-btn>
              </div>
              <div v-if="workspaceUploadFiles.length" class="workspace-upload-queue">
                <div class="workspace-upload-queue-heading"><strong>{{ workspaceUploadFiles.length }} PDF{{ workspaceUploadFiles.length === 1 ? '' : 's' }} para añadir</strong><span>Asigna a cada archivo el nombre con el que quedará en esta carpeta.</span></div>
                <div v-for="(item, index) in workspaceUploadFiles" :key="item.id" class="workspace-upload-item">
                  <div class="workspace-upload-source"><v-icon icon="mdi-file-pdf-box" color="error"/><span><strong>{{ item.file.name }}</strong><small>{{ prettySize(item.file.size) }}</small></span></div>
                  <v-autocomplete v-model="item.code" :items="mspPDFOptions" item-title="title" item-value="value" label="Nombre estándar MSP" placeholder="Selecciona el documento" density="compact" variant="outlined" clearable hide-details/>
                  <v-btn type="button" icon="mdi-close" variant="text" size="small" :aria-label="`Quitar ${item.file.name} de la selección`" @click="removeQueuedWorkspacePDF(index)"/>
                </div>
                <div class="workspace-upload-submit"><span>El nombre original del archivo solo se conserva como referencia; no se usa en la carpeta de entrega.</span><v-btn type="button" color="primary" prepend-icon="mdi-cloud-upload-outline" :loading="workspaceSending" :disabled="!workspacePatient || workspaceUploadFiles.some(item => !item.code)" @click="addWorkspacePDFs">Añadir a esta carpeta</v-btn></div>
              </div>
              <div v-if="workspaceSending" class="ingest-progress"><div><span>Guardando PDFs en el espacio de trabajo…</span><strong>{{ ingestProgress }}%</strong></div><v-progress-linear :model-value="ingestProgress" color="primary" rounded/></div>
              <div class="workspace-download-row"><span>Descarga la carpeta madre con la estructura y los archivos actuales. Revisa pendientes y documentos obligatorios antes de entregar al MSP.</span><v-btn type="button" color="primary" prepend-icon="mdi-folder-zip-outline" :loading="workspaceBusy" @click="downloadWorkspaceZIP">Descargar expediente ZIP</v-btn></div>
              <v-dialog v-model="deletePDFDialog" max-width="440">
                <v-card class="action-dialog">
                  <v-card-title>Quitar PDF de la carpeta</v-card-title>
                  <v-card-text><p>Se quitará <strong>{{ deletePDFTarget.split('/').at(-1) }}</strong> de la carpeta de trabajo. La fuente del lote se conserva y no se volverá a incluir al reclasificar.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="deletePDFDialog=false;deletePDFTarget=''">Cancelar</v-btn><v-btn type="button" color="error" :loading="workspaceBusy" @click="deleteWorkspacePDF">Quitar PDF</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
              <v-dialog v-model="deleteWorkspaceDialog" max-width="540" persistent>
                <v-card class="action-dialog">
                  <v-card-title>Eliminar carpeta completa</v-card-title>
                  <v-card-text>
                    <p>Se eliminará permanentemente <strong>{{ ingestResult?.workspace?.split('/').at(-1) }}</strong>, con el ZIP fuente, los documentos habilitantes, reportes y archivos preparados. Esta acción no se puede deshacer.</p>
                    <p>Para confirmar, escribe a mano este código de 8 dígitos:</p>
                    <div class="workspace-delete-code">{{ workspaceDeleteCode }}</div>
                    <v-text-field :model-value="workspaceDeleteInput" label="Código de confirmación" inputmode="numeric" autocomplete="off" autocorrect="off" spellcheck="false" maxlength="8" counter="8" prepend-inner-icon="mdi-keyboard-outline" @update:model-value="setWorkspaceDeleteInput" @paste.prevent="preventWorkspaceDeleteClipboard" @copy.prevent="preventWorkspaceDeleteClipboard" @cut.prevent="preventWorkspaceDeleteClipboard" @drop.prevent="preventWorkspaceDeleteClipboard" @beforeinput="preventWorkspaceDeletePaste" @keydown="blockWorkspaceDeleteClipboardKeys"/>
                    <v-alert v-if="workspaceDeleteError" type="error" variant="tonal" density="comfortable">{{ workspaceDeleteError }}</v-alert>
                  </v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" :disabled="workspaceDeleting" @click="deleteWorkspaceDialog=false;workspaceDeleteInput='';workspaceDeleteError=''">Cancelar</v-btn><v-btn type="button" color="error" prepend-icon="mdi-delete-forever-outline" :loading="workspaceDeleting" :disabled="workspaceDeleteInput !== workspaceDeleteCode" @click="deleteWholeWorkspace">Eliminar definitivamente</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
              <v-dialog v-model="replacePDFDialog" max-width="460">
                <v-card class="action-dialog">
                  <v-card-title>Reemplazar PDF</v-card-title>
                  <v-card-text><p>El archivo local <strong>{{ replacePDFFile?.name }}</strong> reemplazará a <strong>{{ selectedWorkspaceDocument?.name }}</strong> dentro de <strong>{{ workspacePatient }}</strong>. El archivo local adoptará automáticamente el nombre del PDF seleccionado; su nombre original se conservará solo como referencia. La versión anterior queda en las fuentes del expediente.</p></v-card-text>
                  <v-card-actions><v-spacer/><v-btn type="button" variant="text" @click="replacePDFDialog=false;replacePDFFile=null;replacePDFTarget=''">Cancelar</v-btn><v-btn type="button" color="warning" :loading="workspaceSending" @click="confirmReplaceWorkspacePDF">Confirmar reemplazo</v-btn></v-card-actions>
                </v-card>
              </v-dialog>
            </section>
            <div v-if="ingestResult" class="ingest-submit-row"><span>¿Necesitas recibir planillas de otro período o servicio?</span><v-btn type="button" variant="text" prepend-icon="mdi-plus" @click="startNewIngest">Recibir otro período</v-btn></div>
            <div v-if="ingestResult" class="ingest-submit-row workspace-danger-row"><span>Elimina permanentemente toda la carpeta de este período, incluidas fuentes, reportes y PDFs.</span><v-btn type="button" color="error" variant="tonal" prepend-icon="mdi-folder-remove-outline" @click="requestDeleteWorkspace">Eliminar carpeta completa</v-btn></div>
            <v-expansion-panels v-if="!ingestResult" variant="accordion" class="ingest-resume-panel"><v-expansion-panel><v-expansion-panel-title>¿Ya recibiste un lote y quieres continuar?</v-expansion-panel-title><v-expansion-panel-text><p>Escribe el ID que apareció cuando subiste el lote.</p><div class="ingest-submit-row"><v-text-field v-model="existingIngestJobId" label="ID del lote" placeholder="JOB-…" density="comfortable" hide-details/><v-btn color="primary" prepend-icon="mdi-folder-cog-outline" :loading="ingestProcessing" :disabled="!existingIngestJobId.trim()" @click="processIngest">Abrir lote</v-btn></div></v-expansion-panel-text></v-expansion-panel></v-expansion-panels>
            <div v-if="ingestSending || completionSending" class="ingest-progress"><div><span>{{ completionSending ? 'Subiendo documentos habilitantes…' : 'Subiendo lote de planillas…' }}</span><strong>{{ ingestProgress }}%</strong></div><v-progress-linear :model-value="ingestProgress" color="primary" rounded/></div>
            <div v-if="!ingestResult" class="ingest-submit-row"><span>El ZIP se guardará en el espacio privado del servidor.</span><v-btn type="submit" color="primary" size="large" prepend-icon="mdi-cloud-upload-outline" :loading="ingestSending" :disabled="!ingestValid || ingestSending">{{ ingestSending ? 'Subiendo ZIP…' : 'Recibir lote de planillas' }}</v-btn></div>
          </form>
          <footer>Hecho con cuidado para tus documentos <span>✳</span></footer>
        </section>
      </main>
    </div>
    <div v-if="dragging" class="drop-overlay"><div class="drop-message"><v-icon icon="mdi-cloud-upload-outline" size="46"/><h2>Suelta tus archivos aquí</h2><p>Se guardarán en {{ activeFolder === folders[0] ? 'Mi espacio' : activeFolder }}</p></div></div>
    <v-dialog v-model="createSpaceDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Crear un espacio</v-card-title><v-card-text><p>Organiza tus archivos en un espacio nuevo.</p><v-text-field v-model="newSpaceName" label="Nombre del espacio" placeholder="Ej. Clientes" prepend-inner-icon="mdi-folder-outline" maxlength="36" autofocus @keyup.enter="createSpace"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="createSpaceDialog=false">Cancelar</v-btn><v-btn color="primary" :disabled="!newSpaceName.trim()" @click="createSpace">Crear espacio</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="moveDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Mover {{ activeDoc ? 'documento' : `${selectedIds.length} documentos` }}</v-card-title><v-card-text><p>Elige el espacio de destino.</p><v-select v-model="moveTarget" :items="spaces" label="Espacio" prepend-inner-icon="mdi-folder-outline"/><div v-if="activeDoc?.id.startsWith('demo-')" class="demo-note"><v-icon icon="mdi-information-outline" size="17"/> Los archivos de ejemplo no se pueden mover.</div></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="moveDialog=false">Cancelar</v-btn><v-btn color="primary" @click="confirmMove">Mover</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="preview" max-width="720"><v-card v-if="preview" class="preview-card"><div class="preview-head"><div class="doc-type-icon" :class="'type-'+preview.color"><v-icon :icon="preview.icon" size="20"/></div><div class="preview-name"><b>{{ preview.name }}</b><small>{{ preview.ext }} · {{ preview.size }}</small></div><v-btn icon="mdi-download-outline" variant="text" aria-label="Descargar" @click="download(preview)"/><v-btn icon="mdi-delete-outline" variant="text" aria-label="Eliminar" @click="removeDoc(preview)"/><v-btn icon="mdi-close" variant="text" aria-label="Cerrar" @click="preview=null"/></div><div class="preview-body"><template v-if="preview.mime?.startsWith('image/')"><img :src="previewUrl" :alt="preview.name"/></template><template v-else-if="preview.mime==='application/pdf'"><iframe :src="previewUrl" :title="preview.name"/></template><div v-else class="preview-placeholder"><v-icon :icon="preview.icon" size="48"/><p>Vista previa disponible para imágenes y archivos PDF.</p><button class="primary-upload" @click="download(preview)"><v-icon icon="mdi-download-outline"/> Descargar archivo</button></div></div></v-card></v-dialog>
    <v-snackbar v-model="snackbar" timeout="2600" location="bottom end" color="primary">{{ snackbar }}<template #actions><v-btn variant="text" @click="snackbar=''">Cerrar</v-btn></template></v-snackbar>
  </v-app>
</template>
