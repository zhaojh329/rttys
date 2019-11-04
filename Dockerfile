FROM node
WORKDIR /build
COPY ./html ./
RUN  npm install && npm run build

FROM golang:alpine
WORKDIR /build
COPY . ./
COPY --from=0 /build/dist /build/dist
RUN apk update && \
    apk add git gcc linux-pam-dev libc-dev && \
    go get -v github.com/rakyll/statik && \
    statik -src=/build/dist && \
    go build -ldflags "-w -s"

FROM alpine
WORKDIR /rttys
RUN apk update && \
    apk add --no-cache linux-pam-dev
COPY  --from=1 /build/rttys /rttys/rttys
ENTRYPOINT ["/rttys/rttys"]
