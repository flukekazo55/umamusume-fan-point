# Deployment Quick Start Guide

## 5-Minute Setup

### Frontend on Vercel (2 minutes)

1. **Push to GitHub**
   ```powershell
   git add .
   git commit -m "chore: prepare for deployment"
   git push origin main
   ```

2. **Deploy to Vercel**
   - Go to [vercel.com](https://vercel.com)
   - Click "New Project"
   - Import your GitHub repo
   - Click "Deploy"
   - Vercel will automatically detect the build configuration

3. **Add Backend URL**
   - In Vercel Project Settings → Environment Variables
   - Add `BACKEND_URL` = `https://your-backend-url.railway.app` (you'll get this URL after backend deployment)

### Backend on Railway (2 minutes)

1. **Deploy to Railway**
   - Go to [railway.app](https://railway.app)
   - Click "New Project" → "Deploy from GitHub repo"
   - Select your repository
   - Railway auto-detects Go

2. **Set Environment Variables (optional)**
   - In Railway → Variables
   - Add `MONGO_URI` if using MongoDB persistence
   - Add `MONGO_DATABASE` if not using default name

3. **Get Backend URL**
   - In Railway dashboard, find your deployment
   - Copy the public URL (e.g., `https://fan-point-api-production.railway.app`)
   - Update Vercel environment variable with this URL

## Verification

After both deployments complete:

```bash
# Test frontend (from browser)
https://your-project.vercel.app

# Test backend health (from browser or curl)
https://your-backend-url.railway.app/api/health

# Test API call (from browser console)
fetch('/api/months')
  .then(r => r.json())
  .then(console.log)
```

## Environment Variables Reference

### For Vercel (Frontend)
```
BACKEND_URL=https://your-backend.railway.app
```

### For Railway/Render (Backend)
```
MONGO_URI=mongodb+srv://username:password@cluster.mongodb.net/
MONGO_DATABASE=umamusume_fan_point
ADDR=:8080
```

## Troubleshooting

### "Cannot GET /" on Vercel
- Check that Angular build completed successfully
- Verify `vercel.json` has correct `outputDirectory`

### CORS errors in browser console
- Backend needs to allow requests from Vercel domain
- Check that `withCORS` middleware is applied (it is by default)

### API calls return 503
- Backend might not be running
- Check Railway/Render dashboard for errors
- Verify environment variables are set

### MongoDB connection failed
- Verify connection string is correct
- Check MongoDB Atlas IP allowlist includes Railway server IP

## Next Steps

1. **Monitor**: Check Vercel Analytics and Railway logs
2. **Custom Domain** (optional): Point domain to Vercel
3. **Backups** (if using MongoDB): Configure automated backups in MongoDB Atlas
4. **Error Tracking** (optional): Add Sentry or similar service

## Support

- [Vercel Docs](https://vercel.com/docs)
- [Railway Docs](https://docs.railway.app)
- [MongoDB Atlas Docs](https://docs.atlas.mongodb.com)
