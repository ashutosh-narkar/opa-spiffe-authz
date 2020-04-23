package example

default allow = false

allow {
    input.peerID == "spiffe://domain.test/db-server"
}
