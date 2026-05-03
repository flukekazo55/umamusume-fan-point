# Backend Configuration for Production

When deploying the Go backend to production, use one of these approaches:

## Using Railway (Recommended)

Railway auto-detects Go projects and handles deployment automatically.

### Process:
1. Go to [railway.app](https://railway.app)
2. Connect your GitHub repository
3. Railway will auto-detect Go from `go.mod`
4. Set environment variables in Railway dashboard:
   ```
   ADDR=:8080
   MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/
   MONGO_DATABASE=umamusume_fan_point
   ```
5. Deploy with a click

### Cost:
- Free tier includes \$5/month credit
- \$0.50/hour for active services

## Using Docker

Build and run locally to test:

```bash
docker build -f backend/Dockerfile -t fan-point-backend .
docker run -p 8080:8080 -e MONGO_URI=your_mongo_url fan-point-backend
```

Then deploy to:
- Google Cloud Run
- AWS ECR + ECS
- Fly.io (docker deploy)
- Any container registry

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `ADDR` | `:8080` | No | Server bind address and port |
| `MONGO_URI` | Empty | No | MongoDB Atlas connection string |
| `MONGO_DATABASE` | `umamusume_fan_point` | No | MongoDB database name |
| `DATA_FILE` | `../source.xlsx` | No | Path to Excel data file |
| `STATIC_DIR` | Empty | No | Static files directory (for serving frontend) |

## Troubleshooting

### "source.xlsx not found"
Ensure the file is deployed with the backend or set `DATA_FILE` to the deployed
file path. If MongoDB is already populated, the backend can run from Mongo data,
but an empty database needs either `source.xlsx` for seeding or a manual import.

### MongoDB connection fails
- Verify connection string in `MONGO_URI`
- Check MongoDB Atlas IP allowlist includes deployment server IP
- Ensure network connectivity (test with `telnet` or `mongo` CLI)

### Port conflicts
Use the `ADDR` environment variable to change the port (e.g., `ADDR=:3000`)

## Health Check

Test your deployment:
```bash
curl https://your-backend-url.herokuapp.com/api/health
```

Should return `200 OK` if healthy.
