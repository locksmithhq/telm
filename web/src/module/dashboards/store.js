const KEY = 'telm-dashboards-v2'

export function loadAll() {
  try {
    return JSON.parse(localStorage.getItem(KEY) || '[]')
  } catch {
    return []
  }
}

export function saveAll(list) {
  localStorage.setItem(KEY, JSON.stringify(list))
}

export function newDashboard(name) {
  return {
    id: crypto.randomUUID(),
    name: name || 'Untitled Dashboard',
    createdAt: new Date().toISOString(),
    panels: [],
  }
}

export function newPanel(query = '') {
  return {
    id: crypto.randomUUID(),
    title: '',
    query,
    cols: 2,  // grid column span 1–4
    rows: 3,  // grid row span  (each row = 80px)
  }
}
