# Stage 1: Build SvelteKit frontend
FROM node:24-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary with embedded frontend
FROM golang:1.25-alpine AS backend
WORKDIR /app/go
COPY go/go.mod ./
COPY go/*.go ./
COPY go/dictionary.txt go/rulesets.json ./
COPY go/config.json ./
COPY --from=frontend /app/web/build ./static/
RUN go build -o scrabble .

# Stage 3: Minimal runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/go/scrabble .
COPY --from=backend /app/go/dictionary.txt .
COPY --from=backend /app/go/rulesets.json .
COPY --from=backend /app/go/config.json .
RUN mkdir -p boards
EXPOSE 8080
CMD ["./scrabble", "serve"]
