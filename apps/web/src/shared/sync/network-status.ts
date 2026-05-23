import { create } from 'zustand'

interface NetworkState {
  isOnline: boolean
  setOnline: (v: boolean) => void
}

export const useNetworkStatus = create<NetworkState>((set) => ({
  isOnline: typeof navigator !== 'undefined' ? navigator.onLine : true,
  setOnline: (v) => set({ isOnline: v }),
}))

// Initialize listeners (call once at app startup)
export function initNetworkListeners() {
  const setOnline = useNetworkStatus.getState().setOnline
  window.addEventListener('online', () => setOnline(true))
  window.addEventListener('offline', () => setOnline(false))
}