# Uma Musume Fan Point Tracker - Deployment Checklist

## Pre-Deployment Checklist

### Code & Testing
- [ ] All tests passing
- [ ] No console errors in development
- [ ] No hardcoded localhost URLs (check environment configuration)
- [ ] API service uses relative URLs or environment variables
- [ ] CORS configuration verified for cross-origin requests

### Backend (Go)
- [ ] `go.mod` and `go.sum` are committed
- [ ] `source.xlsx` is in repo root (needed for seeding)
- [ ] Dockerfile is created and tested locally
- [ ] Environment variables documented:
  - [ ] `MONGO_URI` (optional, for persistence)
  - [ ] `MONGO_DATABASE` (optional, defaults to `umamusume_fan_point`)
  - [ ] `ADDR` (optional, defaults to `:8080`)

### Frontend (Angular)
- [ ] Environment files created (`environment.ts`, `environment.prod.ts`)
- [ ] `vercel.json` configured correctly
- [ ] Build command works: `npm run build`
- [ ] Output directory exists: `dist/fan-point`
- [ ] No relative imports causing issues in production

### Configuration Files
- [ ] `vercel.json` - ✓ Created
- [ ] `Dockerfile` - ✓ Created (backend)
- [ ] `.dockerignore` - ✓ Created (backend)
- [ ] `environment.ts` - ✓ Created (frontend)
- [ ] `environment.prod.ts` - ✓ Created (frontend)
- [ ] `.env.local.example` - ✓ Created
- [ ] `.env.production.example` - ✓ Created
- [ ] `DEPLOYMENT.md` - ✓ Created

## Frontend Deployment (Vercel)

1. [ ] **Push code to GitHub**
   ```powershell
   git add .
   git commit -m "chore: prepare for Vercel deployment"
   git push origin main
   ```

2. [ ] **Connect to Vercel**
   - [ ] Visit vercel.com
   - [ ] Import GitHub repository
   - [ ] Select project

3. [ ] **Configure Build Settings in Vercel**
   - **Framework**: Other (not Next.js)
   - **Build Command**: `cd frontend && npm install && npm run build`
   - **Output Directory**: `frontend/dist/fan-point/browser`
   - **Install Command**: `npm ci`

4. [ ] **Add Environment Variables**
   - [ ] `BACKEND_URL` = `https://your-backend-deployment-url.com`
   - [ ] Backend URL has no trailing slash
   - [ ] Vercel project uses the repo root, or if it uses `frontend` as root, `frontend/api/proxy.js` is included

5. [ ] **Deploy**
   - [ ] Click Deploy button
   - [ ] Wait for build to complete
   - [ ] Test the deployed app

## Backend Deployment

### Choose One Platform:

#### Railway
- [ ] Login to railway.app
- [ ] Create new project from GitHub
- [ ] Select backend folder
- [ ] Set environment variables
- [ ] Deploy

#### Render
- [ ] Login to render.com
- [ ] Create Web Service from GitHub
- [ ] Configure build and start commands
- [ ] Set environment variables
- [ ] Deploy

#### Fly.io
- [ ] Install `flyctl`
- [ ] Run `flyctl launch` in backend directory
- [ ] Configure `fly.toml`
- [ ] Run `flyctl deploy`

#### Google Cloud Run (Docker)
- [ ] Build Docker image
- [ ] Push to container registry
- [ ] Deploy to Cloud Run
- [ ] Set environment variables

## Post-Deployment Testing

- [ ] [ ] Frontend loads at `https://your-domain.vercel.app`
- [ ] [ ] API calls successfully reach backend
- [ ] [ ] CORS errors don't appear in console
- [ ] [ ] Data loads correctly
- [ ] [ ] Create/Read/Update/Delete operations work
- [ ] [ ] MongoDB persistence works (if configured)
- [ ] [ ] Mobile responsiveness verified
- [ ] [ ] Performance is acceptable

## Domain & DNS (Optional)

- [ ] [ ] Custom domain registered
- [ ] [ ] Vercel domain connected to frontend
- [ ] [ ] Backend URL updated in Vercel environment variables
- [ ] [ ] SSL certificates generated automatically by Vercel

## Monitoring & Maintenance

- [ ] [ ] Vercel Analytics enabled
- [ ] [ ] Backend logs accessible
- [ ] [ ] Error tracking configured (Sentry, etc.)
- [ ] [ ] Database backups configured (if using MongoDB)
- [ ] [ ] Alerts configured for failures

## Documentation

- [ ] [ ] `DEPLOYMENT.md` shared with team
- [ ] [ ] Backend URL documented
- [ ] [ ] Environment variables documented
- [ ] [ ] Rollback procedures documented
- [ ] [ ] Team members can deploy independently
