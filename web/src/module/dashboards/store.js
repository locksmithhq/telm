import api from "@/plugins/axios"

export async function loadAll() {
  try {
    const { data } = await api.get("/dashboards")
    return data || []
  } catch {
    return []
  }
}

export async function create(dashboard) {
  const { data } = await api.post("/dashboards", dashboard)
  return data
}

export async function update(dashboard) {
  const { data } = await api.put(`/dashboards/${dashboard.id}`, dashboard)
  return data
}

export async function remove(id) {
  await api.delete(`/dashboards/${id}`)
}

export function newDashboard(name) {
  return {
    id: crypto.randomUUID(),
    name: name || "Untitled Dashboard",
    createdAt: new Date().toISOString(),
    panels: [],
  }
}

export function newPanel(query = "") {
  return {
    id: crypto.randomUUID(),
    title: "",
    query,
    cols: 2,
    rows: 3,
  }
}
