FROM golang:1.13.0-stretch AS build-env
WORKDIR /app
ADD . /app
RUN cd /app && go build -mod=vendor -o udp_proxy

FROM busybox:glibc
WORKDIR /app
COPY --from=build-env /app/udp_proxy /app


# ENTRYPOINT ./udp_proxy

# docker build -t udp_proxy:1.0.0 .
# docker run -d -p <host local port>:<container local port>/udp --name my_udp_proxy udp_proxy:1.0.0 ./udp_proxy <container local port> <remote ip> <remote port>