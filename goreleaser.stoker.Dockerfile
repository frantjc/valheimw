FROM scratch
COPY stoker /stoker
ENTRYPOINT ["/stoker"]
