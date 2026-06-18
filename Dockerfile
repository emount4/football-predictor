# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ENV VITE_API_BASE_URL=
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /server ./server
COPY --from=backend /src/backend/internal/repository/schema.sql ./internal/repository/schema.sql
COPY --from=frontend /src/frontend/dist ./static
ENV PORT=8080
ENV DB_PATH=/data/football.db
ENV SCHEMA_PATH=internal/repository/schema.sql
ENV STATIC_DIR=static
ENV ENABLE_WORKER=true
EXPOSE 8080
VOLUME /data
CMD ["./server"]
