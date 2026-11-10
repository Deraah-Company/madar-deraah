FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/backend/web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /trainer .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -g 10001 trainer && adduser -D -u 10001 -G trainer trainer && mkdir -p /app/data && chown trainer:trainer /app/data
WORKDIR /app
COPY --from=backend /trainer ./trainer
USER trainer
ENV HTTP_ADDR=:8080 DATA_DIR=/app/data TZ=Asia/Riyadh
EXPOSE 8080
CMD ["./trainer"]
