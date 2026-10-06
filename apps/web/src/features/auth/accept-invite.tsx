import { Navigate } from '@tanstack/react-router'

// The token is not used by the web: acceptance is by verified email at sign-in (P1-L1 §8).
export function AcceptInvite({ email }: { email?: string }) {
  return <Navigate to="/sign-in" search={email ? { email } : {}} replace />
}
