<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { openDB } from 'idb'

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
onMounted(async () => {
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
const title = computed(() => activePage.value === 'planilla' ? 'Planilla digital' : activeFolder.value === 'Todos los documentos' ? 'Mis documentos' : activeFolder.value)
function prettySize(n) { return n < 1024*1024 ? `${Math.max(1, Math.round(n/1024))} KB` : `${(n/1024/1024).toFixed(1)} MB` }
function handleShortcut(event) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); searchInput.value?.focus() }
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'u') { event.preventDefault(); uploadInput.value?.click() }
  if (event.key === 'Escape') { dragging.value = false; selectedIds.value = [] }
}
function onDragEnter(event) { if (event.dataTransfer?.types?.includes('Files')) { event.preventDefault(); dragDepth++; dragging.value = true } }
function onDragLeave(event) { if (event.dataTransfer?.types?.includes('Files')) { event.preventDefault(); dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) dragging.value = false } }
function onDrop(event) { event.preventDefault(); dragDepth = 0; dragging.value = false; if (event.dataTransfer?.files?.length) upload({ target: { files: event.dataTransfer.files, value: '' } }) }
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
        <section v-else class="content-wrap planilla-content">
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
      </main>
    </div>
    <div v-if="dragging" class="drop-overlay"><div class="drop-message"><v-icon icon="mdi-cloud-upload-outline" size="46"/><h2>Suelta tus archivos aquí</h2><p>Se guardarán en {{ activeFolder === folders[0] ? 'Mi espacio' : activeFolder }}</p></div></div>
    <v-dialog v-model="createSpaceDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Crear un espacio</v-card-title><v-card-text><p>Organiza tus archivos en un espacio nuevo.</p><v-text-field v-model="newSpaceName" label="Nombre del espacio" placeholder="Ej. Clientes" prepend-inner-icon="mdi-folder-outline" maxlength="36" autofocus @keyup.enter="createSpace"/></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="createSpaceDialog=false">Cancelar</v-btn><v-btn color="primary" :disabled="!newSpaceName.trim()" @click="createSpace">Crear espacio</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="moveDialog" max-width="420"><v-card class="action-dialog"><v-card-title>Mover {{ activeDoc ? 'documento' : `${selectedIds.length} documentos` }}</v-card-title><v-card-text><p>Elige el espacio de destino.</p><v-select v-model="moveTarget" :items="spaces" label="Espacio" prepend-inner-icon="mdi-folder-outline"/><div v-if="activeDoc?.id.startsWith('demo-')" class="demo-note"><v-icon icon="mdi-information-outline" size="17"/> Los archivos de ejemplo no se pueden mover.</div></v-card-text><v-card-actions><v-spacer/><v-btn variant="text" @click="moveDialog=false">Cancelar</v-btn><v-btn color="primary" @click="confirmMove">Mover</v-btn></v-card-actions></v-card></v-dialog>
    <v-dialog v-model="preview" max-width="720"><v-card v-if="preview" class="preview-card"><div class="preview-head"><div class="doc-type-icon" :class="'type-'+preview.color"><v-icon :icon="preview.icon" size="20"/></div><div class="preview-name"><b>{{ preview.name }}</b><small>{{ preview.ext }} · {{ preview.size }}</small></div><v-btn icon="mdi-download-outline" variant="text" aria-label="Descargar" @click="download(preview)"/><v-btn icon="mdi-delete-outline" variant="text" aria-label="Eliminar" @click="removeDoc(preview)"/><v-btn icon="mdi-close" variant="text" aria-label="Cerrar" @click="preview=null"/></div><div class="preview-body"><template v-if="preview.mime?.startsWith('image/')"><img :src="previewUrl" :alt="preview.name"/></template><template v-else-if="preview.mime==='application/pdf'"><iframe :src="previewUrl" :title="preview.name"/></template><div v-else class="preview-placeholder"><v-icon :icon="preview.icon" size="48"/><p>Vista previa disponible para imágenes y archivos PDF.</p><button class="primary-upload" @click="download(preview)"><v-icon icon="mdi-download-outline"/> Descargar archivo</button></div></div></v-card></v-dialog>
    <v-snackbar v-model="snackbar" timeout="2600" location="bottom end" color="primary">{{ snackbar }}<template #actions><v-btn variant="text" @click="snackbar=''">Cerrar</v-btn></template></v-snackbar>
  </v-app>
</template>
