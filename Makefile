
.PHONY: run-backend run-frontend clean-data

# Run the backend server
run-backend:
	cd backend && go run cmd/server/main.go

# Run the frontend development server
run-frontend:
	cd frontend && npm run dev

# Clean up GitGym persistent data (remotes, sessions)
clean-data:
	@echo "Cleaning up .gitgym-data..."
	rm -rf backend/.gitgym-data
	rm -rf backend/.gitgym-data.bak*
	@echo "Done."

# Kill running processes on ports 8080 (backend) and 5173 (frontend)
kill:
	@echo "Killing processes on port 8080 (backend)..."
	-lsof -ti:8080 | xargs kill -9
	@echo "Killing processes on port 5173 (frontend)..."
	-lsof -ti:5173 | xargs kill -9
	@echo "Processes killed."

# Run both backend and frontend in development mode
dev: kill
	@echo "Starting backend and frontend..."
	$(MAKE) -j2 run-backend run-frontend
