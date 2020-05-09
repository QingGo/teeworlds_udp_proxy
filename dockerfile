FROM golang:1.11-alpine3.10 AS build-env
WORKDIR /app
ADD . /app
# avoid dynamic linking of net package 
ENV CGO_ENABLED=0
# if no vendor folder, install git and build by go.mod(may have a network problem)
RUN cd /app && go build -mod=vendor -o udp_proxy || apk add git && go build -o udp_proxy

FROM scratch
WORKDIR /app
COPY --from=build-env /app/udp_proxy /app


# ENTRYPOINT ./udp_proxy

# docker build -t udp_proxy:1.0.0 .
# docker run -d -p <host local port>:<container local port>/udp --name my_udp_proxy udp_proxy:1.0.0 ./udp_proxy <container local port> <remote ip> <remote port>