import { useEffect, useState } from 'react'
import * as api from './api.js'

const USER_KEY = 'posts.userId'

export default function App() {
  // The backend has no real auth — it just trusts the X-User-ID header — so we
  // fake a login: on arrival the visitor picks which user they are. The choice
  // is persisted so a reload keeps them "logged in".
  const [userId, setUserId] = useState(() => {
    const saved = localStorage.getItem(USER_KEY)
    return saved ? Number(saved) : null
  })

  function login(id) {
    localStorage.setItem(USER_KEY, String(id))
    setUserId(id)
  }

  function logout() {
    localStorage.removeItem(USER_KEY)
    setUserId(null)
  }

  if (userId == null) {
    return <Login onLogin={login} />
  }

  return <Posts userId={userId} onLogout={logout} />
}

function Login({ onLogin }) {
  const [value, setValue] = useState('1')

  function submit(e) {
    e.preventDefault()
    const id = Number(value)
    if (id > 0) onLogin(id)
  }

  return (
    <div className="app login">
      <header>
        <h1>Posts</h1>
        <p className="subtitle">Pick a user to continue</p>
      </header>

      <form className="card" onSubmit={submit}>
        <label>
          I am user ID
          <input
            type="number"
            min="1"
            autoFocus
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        </label>
        <p className="hint">
          The demo backend has no real accounts — this just sets who you act as.
          Users 1–10 have sample posts from the external API.
        </p>
        <button type="submit" disabled={!(Number(value) > 0)}>
          Continue
        </button>
      </form>
    </div>
  )
}

function Posts({ userId, onLogout }) {
  const [filterByUser, setFilterByUser] = useState(false)

  const [posts, setPosts] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const data = filterByUser
        ? await api.getPostsByUser(userId)
        : await api.getAllPosts()
      setPosts(data || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  // Reload whenever the view (all vs. filtered) or logged-in user changes.
  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterByUser, userId])

  return (
    <div className="app">
      <header className="topbar">
        <div>
          <h1>Posts</h1>
          <p className="subtitle">React client for the Go backend</p>
        </div>
        <div className="session">
          <span>
            Signed in as <strong>user {userId}</strong>
          </span>
          <button className="ghost" onClick={onLogout}>
            Switch user
          </button>
        </div>
      </header>

      <section className="controls">
        <label className="checkbox">
          <input
            type="checkbox"
            checked={filterByUser}
            onChange={(e) => setFilterByUser(e.target.checked)}
          />
          Only show my posts
        </label>
        <button onClick={load} disabled={loading}>
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </section>

      <NewPostForm userId={userId} onCreated={load} />

      {error && <p className="error">Error: {error}</p>}

      <PostList
        posts={posts}
        userId={userId}
        loading={loading}
        onChanged={load}
      />
    </div>
  )
}

function NewPostForm({ userId, onCreated }) {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState(null)

  async function submit(e) {
    e.preventDefault()
    if (!title.trim()) return
    setSubmitting(true)
    setError(null)
    try {
      await api.createPost(userId, { title, body })
      setTitle('')
      setBody('')
      onCreated()
    } catch (err) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="card new-post" onSubmit={submit}>
      <h2>New post</h2>
      <input
        placeholder="Title"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
      />
      <textarea
        placeholder="Body"
        rows="3"
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      {error && <p className="error">{error}</p>}
      <button type="submit" disabled={submitting || !title.trim()}>
        {submitting ? 'Posting…' : 'Post'}
      </button>
    </form>
  )
}

function PostList({ posts, userId, loading, onChanged }) {
  if (loading && posts.length === 0) return <p>Loading posts…</p>
  if (posts.length === 0) return <p>No posts yet.</p>

  return (
    <ul className="posts">
      {posts.map((post) => (
        <PostCard
          key={post.id}
          post={post}
          userId={userId}
          onChanged={onChanged}
        />
      ))}
    </ul>
  )
}

function PostCard({ post, userId, onChanged }) {
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState(post.title)
  const [body, setBody] = useState(post.body)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  // The backend rejects edits to external posts (id < 101) and to posts owned
  // by a different user, so only offer controls when they'd actually work.
  const isLocal = post.id >= api.LOCAL_POST_ID_START
  const isOwner = post.userId === userId
  const canEdit = isLocal && isOwner

  async function save() {
    setBusy(true)
    setError(null)
    try {
      await api.updatePost(userId, post.id, { title, body })
      setEditing(false)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!confirm('Delete this post?')) return
    setBusy(true)
    setError(null)
    try {
      await api.deletePost(userId, post.id)
      onChanged()
    } catch (err) {
      setError(err.message)
      setBusy(false)
    }
  }

  return (
    <li className="card post">
      {editing ? (
        <>
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
          <textarea
            rows="3"
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </>
      ) : (
        <>
          <h3>{post.title}</h3>
          <p>{post.body}</p>
        </>
      )}

      <div className="meta">
        <span>#{post.id}</span>
        <span>user {post.userId}</span>
        {isOwner && isLocal && <span className="tag mine">you</span>}
        {!isLocal && <span className="tag">external</span>}
      </div>

      {error && <p className="error">{error}</p>}

      {canEdit && (
        <div className="actions">
          {editing ? (
            <>
              <button onClick={save} disabled={busy}>
                Save
              </button>
              <button
                className="ghost"
                onClick={() => {
                  setEditing(false)
                  setTitle(post.title)
                  setBody(post.body)
                }}
                disabled={busy}
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              <button onClick={() => setEditing(true)}>Edit</button>
              <button className="danger" onClick={remove} disabled={busy}>
                Delete
              </button>
            </>
          )}
        </div>
      )}
    </li>
  )
}
