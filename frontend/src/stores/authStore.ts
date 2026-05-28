import { create } from "zustand"
import { User, Workspace, getAuthState } from "@/lib/auth"

interface AuthStore {
  user: User | null
  workspace: Workspace | null
  token: string | null
  isLoaded: boolean
  setAuth: (
    user: User,
    workspace: Workspace,
    token: string
  ) => void
  clearAuth: () => void
  loadFromStorage: () => void
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  workspace: null,
  token: null,
  isLoaded: false,

  setAuth: (user, workspace, token) =>
    set({ user, workspace, token }),

  clearAuth: () =>
    set({ user: null, workspace: null, token: null }),

  loadFromStorage: () => {
    const state = getAuthState()
    set({
      user: state.user,
      workspace: state.workspace,
      token: state.token,
      isLoaded: true,
    })
  },
}))
