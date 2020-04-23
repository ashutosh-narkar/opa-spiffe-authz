FROM gcr.io/spiffe-io/spire-agent:0.9.3 as spire

FROM envoyproxy/envoy-alpine:v1.10.0
RUN mkdir -p /opt/spire/conf/agent
RUN mkdir -p /opt/spire/data/agent
COPY --from=spire /opt/spire/bin/spire-agent /opt/spire/bin/spire-agent
COPY conf/spire-agent.conf /opt/spire/conf/agent/agent.conf
COPY conf/agent.key.pem /opt/spire/conf/agent/agent.key.pem
COPY conf/agent.crt.pem /opt/spire/conf/agent/agent.crt.pem
COPY restricted /usr/local/bin/restricted
COPY opa/policy.rego /opt/spire

WORKDIR /opt/spire

RUN mkdir /code
COPY conf/start_service.sh /usr/local/bin/start_service.sh
RUN chmod u+x /usr/local/bin/start_service.sh
ENTRYPOINT /usr/local/bin/start_service.sh
