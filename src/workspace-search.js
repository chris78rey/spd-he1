export function normalizeWorkspaceSearch(value) {
  return String(value ?? '')
    .trim()
    .toLocaleLowerCase('es')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
}

export function workspaceTypeLabel(workspace) {
  return workspace?.es_objeciones ? 'Objeciones · espacio separado' : 'Recepción de planillas'
}

export function workspaceYearLabel(workspace) {
  return String(workspace?.anio || String(workspace?.period || '').slice(0, 4))
}

export function workspaceMonthValue(workspace) {
  return String(workspace?.mes || String(workspace?.period || '').slice(5, 7)).padStart(2, '0')
}

export function workspaceFilterStatusValue(workspace) {
  if (workspace?.ready_for_delivery === true) return 'READY_FOR_DELIVERY'
  if (['PROCESSED', 'INCOMPLETE'].includes(workspace?.status) && workspace?.ready_for_delivery === false) return 'INCOMPLETE_PACKAGE'
  return workspace?.status || ''
}

export function orderWorkspaceRecords(workspaces = []) {
  return [...workspaces].sort((left, right) => {
    const leftYear = Number(workspaceYearLabel(left)) || 0
    const rightYear = Number(workspaceYearLabel(right)) || 0
    if (leftYear !== rightYear) return rightYear - leftYear
    const leftMonth = Number(workspaceMonthValue(left)) || 0
    const rightMonth = Number(workspaceMonthValue(right)) || 0
    if (leftMonth !== rightMonth) return rightMonth - leftMonth
    const receivedDifference = new Date(right.received_at || 0).getTime() - new Date(left.received_at || 0).getTime()
    if (Number.isFinite(receivedDifference) && receivedDifference !== 0) return receivedDifference
    return String(right.job_id || '').localeCompare(String(left.job_id || ''), 'es')
  })
}

export function filterWorkspaceRecords(workspaces = [], filters = {}, labels = {}) {
  const queryTerms = normalizeWorkspaceSearch(filters.search).split(/\s+/).filter(Boolean)
  return orderWorkspaceRecords(workspaces).filter(workspace => {
    if (filters.type === 'RECEPCION' && workspace.es_objeciones) return false
    if (filters.type === 'OBJECIONES' && !workspace.es_objeciones) return false
    if (filters.service && filters.service !== 'ALL' && workspace.tipo_servicio !== filters.service) return false
    const year = workspaceYearLabel(workspace)
    if (filters.year && filters.year !== 'ALL' && year !== filters.year) return false
    if (filters.status && filters.status !== 'ALL') {
      if (filters.status === 'READY_FOR_DELIVERY' && workspace.ready_for_delivery !== true) return false
      if (filters.status === 'INCOMPLETE_PACKAGE' && (!['PROCESSED', 'INCOMPLETE'].includes(workspace.status) || workspace.ready_for_delivery === true)) return false
      if (!['READY_FOR_DELIVERY', 'INCOMPLETE_PACKAGE'].includes(filters.status) && workspace.status !== filters.status) return false
    }
    if (!queryTerms.length) return true

    const searchText = [
      (labels.typeLabel || workspaceTypeLabel)(workspace),
      (labels.serviceLabel || (value => value || ''))(workspace.tipo_servicio),
      workspace.tipo_servicio,
      (labels.monthLabel || (value => value))(workspaceMonthValue(workspace)),
      workspaceMonthValue(workspace),
      year,
      workspace.period,
      workspace.job_id,
      workspace.creado_por,
      workspace.message,
      (labels.statusLabel || (value => value || ''))(workspace.status),
      workspace.ready_for_delivery === true ? 'Listo para entrega' : workspace.ready_for_delivery === false ? 'Avance incompleto' : '',
      (labels.receivedAt || (value => value || ''))(workspace.received_at),
      ...(workspace.missing_documents || []),
      ...(workspace.delivery_missing || []),
    ].filter(Boolean).join(' ')
    const normalizedText = normalizeWorkspaceSearch(searchText)
    return queryTerms.every(term => normalizedText.includes(term))
  })
}
