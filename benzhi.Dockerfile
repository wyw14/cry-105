FROM golang:1.26.2
WORKDIR /src
ENV GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
RUN go build -mod=vendor -o /opt/orbitlink ./cmd/orbitlink
EXPOSE 19705
CMD ["/opt/orbitlink", "-listen", "0.0.0.0:19705", "-state-root", "/var/lib/orbitlink"]
