import { http } from './client'

export function fetchMRs() {
  return http.get('/meta/mrs')
}

export function fetchHospitals() {
  return http.get('/meta/hospitals')
}

export function fetchHCPs(hospitalId) {
  return http.get('/meta/hcps', { hospital_id: hospitalId })
}

export function fetchProducts() {
  return http.get('/meta/products')
}

export function fetchVisits(params) {
  return http.get('/visits', params)
}

export function fetchVisit(id) {
  return http.get(`/visits/${id}`)
}

export function createVisit(payload) {
  return http.post('/visits', payload)
}

export function checkInVisit(id, payload) {
  return http.post(`/visits/${id}/check-in`, payload)
}

export function checkOutVisit(id, payload) {
  return http.post(`/visits/${id}/check-out`, payload)
}

export function submitVisitReport(id, payload) {
  return http.put(`/visits/${id}/report`, payload)
}

export function cancelVisit(id, reason) {
  return http.post(`/visits/${id}/cancel`, { reason: reason || '' })
}

export function fetchDashboard(params) {
  return http.get('/dashboard/products', params)
}
