# Posts

A small full-stack demo: a Go (Gin) JSON API for blog posts, backed by an
in-memory store and the public [jsonplaceholder](https://jsonplaceholder.typicode.com)
API, with a React (Vite) frontend.

## Structure

```
.
├── server/   Go + Gin backend (HTTP API on :8080)
└── client/   React + Vite frontend (dev server on :3000)
```

## Prerequisites

- Go 1.26+
- Node 18+ and npm

## Running

Start the backend:

```bash
cd server
go run .
```

It listens on `http://localhost:8080`.

In a second terminal, start the frontend:

```bash
cd client
npm install
npm run dev
```

Open `http://localhost:3000`. The Vite dev server proxies the API routes to the
backend, so no CORS configuration is needed.

## API

| Method | Path             | Auth | Description                                |
| ------ | ---------------- | ---- | ------------------------------------------ |
| GET    | `/all-posts`     | no   | All external posts plus all local posts    |
| GET    | `/posts/:userID` | no   | External and local posts for one user      |
| POST   | `/create-post`   | yes  | Create a post owned by the requesting user |
| PUT    | `/posts/:id`     | yes  | Edit one of your local posts               |
| DELETE | `/posts/:id`     | yes  | Delete one of your local posts             |

Authenticated routes require an `X-User-ID` header (an integer). Requests
without it get `401`. Editing or deleting an external post, or a post owned by
another user, returns `403`.

### Request bodies

`POST /create-post` and `PUT /posts/:id` take:

```json
{ "title": "My title", "body": "My body" }
```

### Example

```bash
# Create a post as user 5
curl -X POST http://localhost:8080/create-post \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 5' \
  -d '{"title":"Hello","body":"World"}'

# List that user's posts
curl http://localhost:8080/posts/5
```
