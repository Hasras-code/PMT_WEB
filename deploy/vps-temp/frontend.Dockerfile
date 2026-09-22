# Temp VPS frontend image: Vite build + nginx static + API reverse proxy.
# Build context is the repo root (see docker-compose.vps.yml).
#   context: .
#   dockerfile: deploy/vps-temp/frontend.Dockerfile
ARG VITE_API_URL=http://localhost:8080

FROM node:20-alpine AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginx:1.27-alpine
COPY deploy/vps-temp/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
