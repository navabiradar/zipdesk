// ═══════════════════════════
// Auth
// ═══════════════════════════
export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
  is_verified: boolean
  created_at: string
}

export interface Workspace {
  id: string
  name: string
  slug: string
  plan: string
  role: string
}

// ═══════════════════════════
// Links
// ═══════════════════════════
export interface Link {
  id: string
  workspace_id: string
  original_url: string
  short_code: string
  custom_slug?: string
  title: string
  total_clicks: number
  unique_clicks: number
  is_active: boolean
  expires_at?: string
  created_at: string
}

export interface LinkAnalytics {
  total_clicks: number
  unique_clicks: number
  clicks_by_day: Array<{
    date: string
    clicks: number
  }>
  clicks_by_country: Array<{
    country: string
    code: string
    clicks: number
  }>
  clicks_by_device: Array<{
    device: string
    clicks: number
  }>
}

// ═══════════════════════════
// Forms
// ═══════════════════════════
export interface Form {
  id: string
  title: string
  description: string
  slug: string
  is_published: boolean
  published_at?: string
  created_at: string
  fields?: FormField[]
}

export interface FormField {
  id: string
  form_id: string
  type: string
  label: string
  placeholder: string
  required: boolean
  options?: Array<{ id: string; label: string; value: string }>
  order: number
}

export interface FormResponse {
  id: string
  form_id: string
  data: Record<string, any>
  submitted_at: string
}

// ═══════════════════════════
// Mail
// ═══════════════════════════
export interface MailContact {
  id: string
  email: string
  first_name: string
  last_name: string
  company: string
  status: string
  source: string
  tags: string[]
  created_at: string
}

export interface Campaign {
  id: string
  name: string
  subject: string
  from_name: string
  from_email: string
  status: string
  sent_at?: string
  created_at: string
}

// ═══════════════════════════
// CRM
// ═══════════════════════════
export interface CRMContact {
  id: string
  first_name: string
  last_name: string
  email: string
  phone: string
  job_title: string
  lead_status: string
  lead_score: number
  tags: string[]
  created_at: string
}

export interface Deal {
  id: string
  title: string
  value: number
  currency: string
  probability: number
  created_at: string
}

// ═══════════════════════════
// Docs
// ═══════════════════════════
export interface Document {
  id: string
  title: string
  slug: string
  type: string
  status: string
  is_published: boolean
  pdf_url?: string
  created_at: string
}

// ═══════════════════════════
// Flow
// ═══════════════════════════
export interface FlowEvent {
  id: string
  type: string
  source: string
  workspace_id: string
  payload: Record<string, any>
  occurred_at: string
}

export interface Blueprint {
  id: string
  name: string
  description: string
  trigger_type: string
  actions: FlowAction[]
  is_active: boolean
  run_count: number
  created_at: string
}

export interface FlowAction {
  id: string
  type: string
  config: Record<string, any>
  order: number
}

export interface ServiceHealth {
  name: string
  status: "healthy" | "degraded" | "down"
  latency_ms: number
  quota_used?: number
  quota_max?: number
  quota_pct?: number
  error?: string
}

export interface HealthReport {
  timestamp: string
  services: Record<string, ServiceHealth>
  overall: "healthy" | "degraded" | "down"
}

// ═══════════════════════════
// API
// ═══════════════════════════
export interface ApiResponse<T> {
  success: boolean
  data: T
  meta?: {
    total: number
    page: number
    per_page: number
  }
  error?: {
    code: string
    message: string
    field?: string
  }
}

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  created_at: string
}
