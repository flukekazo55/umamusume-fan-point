#!/bin/bash
set -e

echo "Building Go backend..."
cd backend
CGO_ENABLED=0 go build -a -installsuffix cgo -o server ./cmd/server
echo "✓ Backend built successfully"

echo "Building Angular frontend..."
cd ../frontend
npm install
npm run build
echo "✓ Frontend built successfully"

echo "✓ Both projects built! Ready for deployment."
