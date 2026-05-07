FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ADD _output/ship-hook /usr/bin/ship-hook

ENTRYPOINT ["/usr/bin/ship-hook"]
