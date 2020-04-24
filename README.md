# opa-spiffe-authz

OPA-SPIFFE Authorization Example.

## Overview

This is a demo of integrating OPA and SPIRE. The goal of the demo is to show:

- How OPA can be used to authorize mTLS connections

- How OPA can be used to filter the data seen by the client

## Running the Example

### Step 1: Install Docker

Ensure that you have recent versions of `docker` and `docker-compose` installed.

### Step 2: Build

Build the binaries for all the services.

```bash
$ ./build.sh
```

### Step 3: Start containers

```bash
$ docker-compose up --build -d
$ docker-compose ps
             Name                           Command               State           Ports
------------------------------------------------------------------------------------------------
opa-spiffe-demo_api-server_1     flask run --host=0.0.0.0         Up      0.0.0.0:5000->5000/tcp
opa-spiffe-demo_db_1             /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_external_1       /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_privileged_1     /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_restricted_1     /bin/sh -c /usr/local/bin/ ...   Up      10000/tcp
opa-spiffe-demo_spire-server_1   /usr/bin/dumb-init /opt/sp ...   Up
```

The demo consists of a server(`opa-spiffe-demo_db_1`) which is a healthcare app that holds patient records.

A patient record contains the patient's `Firstname`, `Lastname`, `SSN` and `Enrollee_Type`(*primary/secondary*).

`opa-spiffe-demo_privileged_1`, `opa-spiffe-demo_restricted_1`
and `opa-spiffe-demo_external_1` are the clients trying to retrieve patient records from the server.

### Step 4: Start SPIRE Infrastructure

Start the SPIRE Agents and register the services with the SPIRE Server.

```bash
$ ./configure-spire.sh
```

### Step 4: Exercise mTLS Policy

SPIFFE's `v2` API provides methods (`Listen` and `Dial`) to create a mTLS connection using the X509-SVID obtained from the Workload API. These methods take an `Authorizer` which authorizes a workload given its `SPIFFE ID`. The library comes with some built-in authorizers that check a `SPIFFE ID`, a list of `SPIFFE IDs` etc. to make a decision.

For more complex scenarios, where the decision would depend on some dynamic properties or external context, the built-in authorizers wouldn't suffice.

OPA as an `Authorizer` would be perfect for such complex use-cases !

The policy we want to enforce says that:

> **Always** allow the `opa-spiffe-demo_privileged_1` and `opa-spiffe-demo_restricted_1` clients to form a mTLS
> connection with the server. The `opa-spiffe-demo_external_1` cannot connect to the sever on Monday, Wednesday and Friday.

Check the `opa-spiffe-demo_privileged_1` and `opa-spiffe-demo_restricted_1` can connect to the server.

```bash
$ curl -s localhost:5000/connect/privileged | jq .
{
  "client": "spiffe://domain.test/privileged",
  "connection_status": "Created",
  "reason": "OPA allowed request: Hello spiffe://domain.test/privileged"
}

$ curl -s localhost:5000/connect/restricted | jq .
{
  "client": "spiffe://domain.test/restricted",
  "connection_status": "Created",
  "reason": "OPA allowed request: Hello spiffe://domain.test/restricted"
}
```

Check that `opa-spiffe-demo_external_1` cannot connect to the server.

> Note: This is a time-based(UTC) policy.

```bash
$ curl -s localhost:5000/connect/external | jq .
{
  "client": "spiffe://domain.test/external",
  "connection_status": "Not Created",
  "reason": "OPA denied request: unexpected peer ID spiffe://domain.test/external"
}
```

OPA made these decisions in-part using the `SPIFFE ID` provided by the `Authorizer` callback included in the `Listen` and `Dial` methods.

### Step 5: Exercise Filter Policy

This policy attempts to demonstrate that OPA decisions can be more than boolean.

As mentioned before, the patient records contain information like `Firstname` as well as sensitive information like `SSN`.

The policy we want to enforce says that:

> `opa-spiffe-demo_privileged_1` should be able to see all the fields in the patient record while
> `opa-spiffe-demo_restricted_1` shouldn't be able to see the sensitive fields in the patient record.

```bash
$ curl -s localhost:5000/getdata/privileged | jq .
{
  "client": "spiffe://domain.test/privileged",
  "patients": [
    {
      "id": "1",
      "firstname": "Iron",
      "lastname": "Man",
      "ssn": "111-11-1111",
      "enrollee_type": "Primary"
    },
    {
      "id": "2",
      "firstname": "Thor",
      "lastname": "Odinson",
      "ssn": "222-22-2222",
      "enrollee_type": "Primary"
    },
    {
      "id": "3",
      "firstname": "Peter",
      "lastname": "Parker",
      "ssn": "333-33-3333",
      "enrollee_type": "Secondary"
    },
    {
      "id": "4",
      "firstname": "Nick",
      "lastname": "Fury",
      "ssn": "333-33-3333",
      "enrollee_type": "Secondary"
    }
  ]
}
```

```bash
$ curl -s localhost:5000/getdata/restricted | jq .
{
  "client": "spiffe://domain.test/restricted",
  "patients": [
    {
      "id": "1",
      "firstname": "Iron",
      "lastname": "Man",
      "ssn": "***********",
      "enrollee_type": "***********"
    },
    {
      "id": "2",
      "firstname": "Thor",
      "lastname": "Odinson",
      "ssn": "***********",
      "enrollee_type": "***********"
    },
    {
      "id": "3",
      "firstname": "Peter",
      "lastname": "Parker",
      "ssn": "***********",
      "enrollee_type": "***********"
    },
    {
      "id": "4",
      "firstname": "Nick",
      "lastname": "Fury",
      "ssn": "***********",
      "enrollee_type": "***********"
    }
  ]
}
```

In the above response, the `SSN` and `Enrollee_Type` fields are hidden. The `pii` rule in the below
OPA policy snippet returns a list of fields that are considered "*sensitive*" and then the server application hides them
from the final output. The policy can also be extended to return a filtered list instead of returning the "*sensitive*"
fields. The `SPIFFE ID` (`input.peerID`) is obtained from the server connection after a successful TLS handshake.

```ruby
pii = ["SSN", "EnrolleeType"] {
    input.peerID == "spiffe://domain.test/restricted"
}
```

All the policies used in the demo can be modified by exec'ing into the server/client container and changing
the `policy.rego` file. For example try modifying the `pii` rule as below by exec'ing into the `opa-spiffe-demo_db_1`
container and then run ```$ curl -s localhost:5000/getdata/restricted | jq .``` again:

```ruby
pii = ["EnrolleeType"] {
    input.peerID == "spiffe://domain.test/restricted"
}
```

This time the `SSN` should be exposed !
