FROM alpine:3.7

RUN apk add --no-cache ca-certificates e2fsprogs findmnt

ADD smb-csi-driver /bin/

ENTRYPOINT ["/bin/smb-csi-driver"]
