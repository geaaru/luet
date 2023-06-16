FROM macaronios/terragon-minimal:latest-amd64 as builder
RUN luet i -y --sync-repos make upx-bin go ca-certificates git && \
      luet cleanup --purge-repos && mkdir /tmp
ADD . /luet
RUN cd /luet && make build-small

FROM scratch
ENV LUET_NOLOCK=true
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /luet/luet /usr/bin/luet

ENTRYPOINT ["/usr/bin/luet"]
