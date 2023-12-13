FROM golang:1.20.8-bullseye as builder
COPY . .
RUN bash build.sh arc-storage /arc-storage

FROM golang:1.20.8-bullseye
COPY arc-storage.toml /arc-storage.toml
COPY swagger /swagger
WORKDIR /
EXPOSE  80
HEALTHCHECK --interval=30s --timeout=15s \
    CMD curl --fail http://localhost:80/health || exit 1
COPY --from=builder /arc-storage /arc-storage
ENTRYPOINT [ "/arc-storage" ]
CMD ["run"]
