@echo off
setlocal enabledelayedexpansion

echo Building Go backend...
cd backend
go build -o server.exe ./cmd/server
if errorlevel 1 (
    echo Error building backend
    exit /b 1
)
echo OK Backend built successfully
cd ..

echo Building Angular frontend...
cd frontend
call npm install
if errorlevel 1 (
    echo Error installing dependencies
    exit /b 1
)
call npm run build
if errorlevel 1 (
    echo Error building frontend
    exit /b 1
)
echo OK Frontend built successfully
cd ..

echo OK Both projects built! Ready for deployment.
