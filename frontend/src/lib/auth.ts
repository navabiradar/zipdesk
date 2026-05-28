import { api } from "./api/client"

export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
  is_verified: boolean
}

export interface Workspace {
  id: string
  name: string
  slug: string
  plan: string
  role: string
}

export interface AuthState {
  user: User | null
  workspace: Workspace | null
  token: string | null
}

export function getAuthState(): AuthState {
  if (typeof window === "undefined") {
    return { user: null, workspace: null, token: null }
  }
  try {
    const user = JSON.parse(
      localStorage.getItem("user") || "null"
    )
    const workspace = JSON.parse(
      localStorage.getItem("workspace") || "null"
    )
    const token = localStorage.getItem("access_token")
    return { user, workspace, token }
  } catch {
    return { user: null, workspace: null, token: null }
  }
}

export function setAuthState(
  user: User,
  workspace: Workspace,
  token: string
) {
  localStorage.setItem("user", JSON.stringify(user))
  localStorage.setItem(
    "workspace", JSON.stringify(workspace)
  )
  localStorage.setItem("access_token", token)
}

export function clearAuthState() {
  localStorage.removeItem("user")
  localStorage.removeItem("workspace")
  localStorage.removeItem("access_token")
}

export function isAuthenticated(): boolean {
  const { token } = getAuthState()
  return !!token
}

export async function login(
  email: string,
  password: string
) {
  const { data } = await api.post("/auth/login", {
    email,
    password,
  })
  const { user, workspace, access_token } = data.data
  setAuthState(user, workspace, access_token)
  return { user, workspace }
}

export async function register(
  name: string,
  email: string,
  password: string,
  workspaceName: string
) {
  const { data } = await api.post("/auth/register", {
    name,
    email,
    password,
    workspace_name: workspaceName,
  })
  const { user, workspace, access_token } = data.data
  setAuthState(user, workspace, access_token)
  return { user, workspace }
}

export function logout() {
  clearAuthState()
  window.location.href = "/login"
}
