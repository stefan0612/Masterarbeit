FROM golang:1.15.2 AS build-env
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

RUN apt-get update && apt-get install -y git gcc
ADD . /driver
WORKDIR /driver

## Running these in separate steps gives a better error
## output indicating which one actually failed.
RUN make
RUN go test

FROM busybox:1.32.0

ENV VENDOR=seitenbau
ENV DRIVER=smb

COPY --from=build-env /driver/smb-flex-driver /
COPY install.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/install.sh

CMD ["/usr/local/bin/install.sh"]
