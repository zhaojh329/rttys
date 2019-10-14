FROM node
WORKDIR /build
COPY ./html ./
RUN  npm install && npm run build 

FROM golang:alpine
WORKDIR /build
COPY . ./
COPY --from=0 /build/dist /build/dist
RUN apk update && \
    apk add git && \
    go get -v github.com/rakyll/statik && \
    statik -src=/build/dist && \
    go build -ldflags "-w -s -extldflags "-static""

FROM alpine
WORKDIR /rttys
RUN apk --no-cache add ca-certificates 
COPY  --from=1 /build/rttys /rttys/rttys
ENTRYPOINT ["/rttys/rttys"]