#!/bin/bash
# This script starts the spire agent in the special, restricted external and db servers
# and creates the workload registration entries for them.

set -e

bb=$(tput bold)
nn=$(tput sgr0)

fingerprint() {
	cat $1 | openssl x509 -outform DER | openssl sha1 -r | awk '{print $1}'
}

SPECIAL_AGENT_FINGERPRINT=$(fingerprint docker/special/conf/agent.crt.pem)
RESTRICTED_AGENT_FINGERPRINT=$(fingerprint docker/restricted/conf/agent.crt.pem)
EXTERNAL_AGENT_FINGERPRINT=$(fingerprint docker/external/conf/agent.crt.pem)
DB_AGENT_FINGERPRINT=$(fingerprint docker/db/conf/agent.crt.pem)

# Bootstrap trust to the SPIRE server for each agent by copying over the
# trust bundle into each agent container. Alternatively, an upstream CA could
# be configured on the SPIRE server and each agent provided with the upstream
# trust bundle (see UpstreamCA under
# https://github.com/spiffe/spire/blob/master/doc/spire_server.md#plugin-types)
docker-compose exec -T spire-server bin/spire-server bundle show |
	docker-compose exec -T special tee conf/agent/bootstrap.crt > /dev/null
docker-compose exec -T spire-server bin/spire-server bundle show |
	docker-compose exec -T restricted tee conf/agent/bootstrap.crt > /dev/null
docker-compose exec -T spire-server bin/spire-server bundle show |
	docker-compose exec -T external tee conf/agent/bootstrap.crt > /dev/null
docker-compose exec -T spire-server bin/spire-server bundle show |
	docker-compose exec -T db tee conf/agent/bootstrap.crt > /dev/null

# Start up the special service SPIRE agent.
echo "${bb}Starting special service SPIRE agent...${nn}"
docker-compose exec -d special bin/spire-agent run

# Start up the restricted service SPIRE agent.
echo "${bb}Starting restricted service SPIRE agent...${nn}"
docker-compose exec -d restricted bin/spire-agent run

# Start up the external service SPIRE agent.
echo "${bb}Starting external service SPIRE agent...${nn}"
docker-compose exec -d external bin/spire-agent run

# Start up the db service SPIRE agent.
echo "${bb}Starting db service SPIRE agent...${nn}"
docker-compose exec -d db bin/spire-agent run

echo "${nn}"

echo "${bb}Creating registration entry for the special service...${nn}"
docker-compose exec spire-server bin/spire-server entry create \
	-selector unix:user:root \
	-spiffeID spiffe://domain.test/special \
	-parentID spiffe://domain.test/spire/agent/x509pop/${SPECIAL_AGENT_FINGERPRINT}

echo "${bb}Creating registration entry for the restricted service...${nn}"
docker-compose exec spire-server bin/spire-server entry create \
	-selector unix:user:root \
	-spiffeID spiffe://domain.test/restricted \
	-parentID spiffe://domain.test/spire/agent/x509pop/${RESTRICTED_AGENT_FINGERPRINT}

echo "${bb}Creating registration entry for the external service...${nn}"
docker-compose exec spire-server bin/spire-server entry create \
	-selector unix:user:root \
	-spiffeID spiffe://domain.test/external \
	-parentID spiffe://domain.test/spire/agent/x509pop/${EXTERNAL_AGENT_FINGERPRINT}

echo "${bb}Creating registration entry for the db service...${nn}"
docker-compose exec spire-server bin/spire-server entry create \
	-selector unix:user:root \
	-spiffeID spiffe://domain.test/db-server \
	-parentID spiffe://domain.test/spire/agent/x509pop/${DB_AGENT_FINGERPRINT}