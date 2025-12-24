#!/bin/bash
set -e

echo "🔍 Starting Agent Verification Workflow..."

# 1. Backend Verification
echo "---------------------------------------"
echo "🐹 Verifying Backend (Go)..."
cd backend
echo "   > Running Unit Tests..."
go test ./...
echo "   > Running Linters (golangci-lint)..."
# Check if golangci-lint is installed, otherwise skip or warn
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./...
else
    echo "   [INFO] golangci-lint not found in PATH, running via 'go run' (v1.62.2)..."
    go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run ./...
fi
cd ..

# 2. Frontend Verification
echo "---------------------------------------"
echo "⚛️  Verifying Frontend (React)..."
cd frontend
echo "   > Running Type Check..."
npm run build # This runs tsc -b
echo "   > Running ESLint..."
npm run lint
cd ..

echo "---------------------------------------"
echo "✅ Verification Complete! All checks passed."
