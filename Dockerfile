FROM node:20 AS ui
WORKDIR /rttys/ui
COPY ui .
RUN npm install && npm run build

FROM golang:latest AS rttys
WORKDIR /rttys
COPY . .
COPY --from=ui /rttys/internal/server/assets/dist internal/server/assets/dist
RUN CGO_ENABLED=0 \
    GitCommit=$(git log --pretty=format:"%h" -1) \
    BuildTime=$(date +%FT%T%z) \
    go build -ldflags="-s -w -X main.GitCommit=$GitCommit -X main.BuildTime=$BuildTime" -o rttys ./cmd/rttys

FROM alpine:latest
COPY --from=rttys /rttys/rttys /usr/bin/rttys
ENTRYPOINT ["/usr/bin/rttys"]
