import type { QueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'

export async function signOut(queryClient: QueryClient) {
  try {
    await unwrap(api.POST('/auth/sign-out'))
  } catch {
    // signing out locally anyway; the session cookie expires server-side
  }
  queryClient.clear()
  window.location.assign('/sign-in') // full reload drops all state
}
