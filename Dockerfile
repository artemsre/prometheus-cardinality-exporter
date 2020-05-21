FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN make
CMD ["/app/prometheus-cardinality-exporter"]
