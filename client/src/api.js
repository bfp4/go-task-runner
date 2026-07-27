// Thin wrapper around the Go backend. Paths are relative so the Vite dev proxy
// (see vite.config.js) forwards them to http://localhost:8080.

function authHeaders(userId) {
  return {
    'Content-Type': 'application/json',
    'X-User-ID': String(userId),
  }
}

async function handle(res) {
  if (res.status === 204) return null
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(data?.error || `Request failed (${res.status})`)
  }
  return data
}

export function getAllPosts() {
  return fetch('/all-posts').then(handle)
}

export function getPostsByUser(userId) {
  return fetch(`/posts/${userId}`).then(handle)
}

export function createPost(userId, { title, body }) {
  return fetch('/create-post', {
    method: 'POST',
    headers: authHeaders(userId),
    body: JSON.stringify({ title, body }),
  }).then(handle)
}

export function updatePost(userId, id, { title, body }) {
  return fetch(`/posts/${id}`, {
    method: 'PUT',
    headers: authHeaders(userId),
    body: JSON.stringify({ title, body }),
  }).then(handle)
}

export function deletePost(userId, id) {
  return fetch(`/posts/${id}`, {
    method: 'DELETE',
    headers: authHeaders(userId),
  }).then(handle)
}

// The Go server treats IDs below this as read-only "external" posts.
export const LOCAL_POST_ID_START = 101
