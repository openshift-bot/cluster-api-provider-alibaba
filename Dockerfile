FROM registry.ci.openshift.org/openshift/release:golang-1.18 AS builder
WORKDIR /go/src/github.com/openshift/cluster-api-provider-alibaba
COPY . .
# VERSION env gets set in the openshift/release image and refers to the golang version, which interfers with our own
RUN unset VERSION \
  && GOPROXY=off NO_DOCKER=1 make build

FROM registry.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift/cluster-api-provider-alibaba/bin/machine-controller-manager /
