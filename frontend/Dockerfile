# Build Stage
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# Type check skip for faster build in docker if desired, but tsc is good.
# However, we often skip tsc in docker to be permissive. 
# Vite build runs tsc by default in template.
RUN npm run build

# Production Stage
FROM nginxinc/nginx-unprivileged:alpine

# Remove default nginx static assets (though unprivileged image might have different path, usually safe to keep, 
# but unprivileged uses /usr/share/nginx/html by default too)
# Note: unprivileged user cannot rm -rf system dirs easily if not owned, 
# but usually html dir is writable or we overwrite. 
# Let's just COPY --chown

# Copy static assets from builder stage
COPY --from=builder --chown=nginx:nginx /app/dist /usr/share/nginx/html

# Copy custom nginx config
COPY --chown=nginx:nginx nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
