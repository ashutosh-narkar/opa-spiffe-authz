# opa-spiffe-authz

OPA-SPIFFE Authorization Example.

## Running the Example

### Step 1: Install Docker

Ensure that you have recent versions of `docker` and `docker-compose` installed.

### Step 2: Build

Build the binaries for the `db`(server) and `special`(client) service.

```bash
$ ./build.sh
```

### Step 3: Start containers

```bash
$ docker-compose up --build -d
$ docker-compose ps
                  Name                                 Command               State                 Ports
----------------------------------------------------------------------------------------------------------------------
opa-spiffe-demo_db_1             /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_special_1        /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_spire-server_1   /usr/bin/dumb-init /opt/sp ...   Up
```

### Step 4: Start SPIRE Infrastructure

Start the SPIRE Agents and register the `db` and `special` services with the SPIRE Server.

```bash
$ ./configure-spire.sh
```
