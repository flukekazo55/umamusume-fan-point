# Deployment Guide

This application consists of a frontend (Angular) and backend (Go). They need to be deployed separately.

## Frontend Deployment on Vercel

Vercel hosts the Angular frontend in this project. The Go backend still needs to
be deployed separately on a platform that runs long-lived Go HTTP services, such
as Railway, Render, Fly.io, or Cloud Run.

### Prerequisites
- GitHub account with this repository connected
- Vercel account

### Steps

1. **Push to GitHub**
   ```powershell
   git push origin main
   ```

2. **Connect to Vercel**
   - Go to [vercel.com](https://vercel.com)
   - Click "New Project"
   - Import your GitHub repository
   - Select the project

3. **Configure Build Settings**
   - **Build Command**: `cd frontend && npm install && npm run build`
   - **Output Directory**: `frontend/dist/fan-point/browser`
   - **Install Command**: `npm ci`

4. **Set Environment Variables**
   - In Vercel Project Settings → Environment Variables
   - Add: `BACKEND_URL` = `https://your-backend-url.com`
   - This URL should be where your Go backend is deployed
   - Do not include a trailing slash

The frontend calls `/api/...`. Vercel routes those requests to `api/proxy.js`,
and that proxy forwards them to `${BACKEND_URL}/api/...`.

5. **Deploy**
   - Click "Deploy"
   - Your frontend will be live at `your-project.vercel.app`

## Backend Deployment Options

Your Go backend needs a separate platform that supports Go HTTP servers. Choose one:

### Option 1: Railway (Recommended - Simple)

1. Go to [railway.app](https://railway.app)
2. Create new project → GitHub repo
3. Select the backend folder
4. Railway auto-detects Go
5. Set environment variables:
   - `ADDR`: `:8080` (or omit for default)
   - `MONGO_URI`: Your MongoDB Atlas connection string (if using MongoDB)
   - `MONGO_DATABASE`: `umamusume_fan_point`
6. Your backend will be deployed at `your-app.railway.app`

### Option 2: Render

1. Go to [render.com](https://render.com)
2. Create new "Web Service"
3. Connect GitHub repository
4. Settings:
   - **Build Command**: `go mod download && go build -o server ./cmd/server`
   - **Start Command**: `./server`
   - **Root Directory**: `backend`
5. Set environment variables same as Railway
6. If the database is empty and you want the backend to seed from Excel, make sure `source.xlsx` is available to the backend and set `DATA_FILE` to that deployed file path. Otherwise, pre-import the Mongo data before first start.
7. Deploy

### Option 3: Fly.io

1. Install `flyctl`
2. In backend directory: `flyctl launch`
3. Follow prompts (sets up `fly.toml`)
4. Deploy: `flyctl deploy`

### Option 4: Google Cloud Run (Using Docker)

1. Build and push Docker image:
   ```bash
   cd backend
   docker build -t your-registry/fan-point-backend .
   docker push your-registry/fan-point-backend
   ```
2. Deploy to Cloud Run
3. Set environment variables in Cloud Run configuration

## MongoDB Setup

If you want to persist player data:

1. Create a MongoDB Atlas cluster (free tier available)
2. Get connection string from Atlas
3. Set `MONGO_URI` environment variable in your backend deployment
4. First run will seed data from `source.xlsx`

## Environment Variable Mapping

### Frontend (Vercel)
- `BACKEND_URL`: Full URL to your deployed backend (e.g., `https://fan-point-api.railway.app`)

### Backend (Railway/Render/Fly.io)
- `MONGO_URI`: MongoDB Atlas connection string (optional, defaults to in-memory)
- `MONGO_DATABASE`: Database name (optional, defaults to `umamusume_fan_point`)
- `ADDR`: Server address (optional, defaults to `:8080`)

## Testing the Deployment

After both are deployed:

1. Visit your Vercel frontend URL
2. Open browser DevTools → Network tab
3. Make an API call (click a button that fetches data)
4. Verify API calls go to `/api/...` on your Vercel domain and return backend data
5. Test CRUD operations (if using MongoDB)

## CORS Configuration

The deployed frontend normally calls the same Vercel origin, and `api/proxy.js`
forwards the request to the Go backend. The backend already sends permissive CORS
headers, which is useful if you also call the backend URL directly:

```go
mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Access-Control-Allow-Origin", "*")
  w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
  w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
  if r.Method == "OPTIONS" {
    w.WriteHeader(http.StatusOK)
    return
  }
  // Handle request
})
```

## Rollback

- **Frontend**: Vercel keeps deployment history. Click "Deployments" and redeploy any previous version
- **Backend**: Railway/Render also maintain history. Use their dashboards to rollback

## Monitoring

- **Frontend**: Vercel Analytics and Edge Functions metrics
- **Backend**: Check logs in Railway/Render dashboard

## Cost Estimates

- **Frontend (Vercel)**: Free tier covers most use cases
- **Backend (Railway)**: Free tier includes \$5 credit/month
- **MongoDB (Atlas)**: Free tier with 512MB storage
- **Total**: Often stays within free tier limits
