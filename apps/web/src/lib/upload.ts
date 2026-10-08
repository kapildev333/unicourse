import { unwrap } from '../api/client'

/**
 * POSTs multipart form data through XHR, because fetch reports no upload bytes.
 * Resolves with the parsed JSON body and rejects with the same ApiError as the fetch client for non-2xx;
 * a network failure rejects with a plain Error. The browser sets the multipart boundary and Sec-Fetch-Site.
 */
export function sendWithProgress<T>(url: string, form: FormData, onBytes: (sent: number, total: number) => void): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)
    xhr.withCredentials = true // same as the client's credentials: 'include'
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onBytes(e.loaded, e.total)
    }
    xhr.onerror = xhr.onabort = xhr.ontimeout = () => reject(new Error('Network error'))
    xhr.onload = () => {
      let body: unknown
      try {
        body = JSON.parse(xhr.responseText)
      } catch {
        body = undefined // empty or non-JSON body: unwrap falls back to the status text
      }
      const ok = xhr.status >= 200 && xhr.status < 300
      const headers = new Headers()
      const id = xhr.getResponseHeader('X-Request-Id')
      if (id) headers.set('X-Request-Id', id)
      const response = new Response(null, { status: xhr.status, statusText: xhr.statusText, headers })
      unwrap<T>(Promise.resolve({ data: ok ? (body as T) : undefined, error: ok ? undefined : body, response })).then(resolve, reject)
    }
    xhr.send(form)
  })
}
