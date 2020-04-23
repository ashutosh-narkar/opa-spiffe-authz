package example

default allow = false
default pii = []

# workload with identity "special" and "restricted" can access the server on all days
# workload with identity "external" CANNOT access server on Monday, Wednesday and Friday

restricted_days := {"Monday", "Wednesday", "Friday"}

allow {
    input.peerID == "spiffe://domain.test/special"
}

allow {
    input.peerID == "spiffe://domain.test/restricted"
}

allow {
    input.peerID == "spiffe://domain.test/external"
    not is_day_restricted
}

is_day_restricted {
    day := time.weekday(time.now_ns())
    restricted_days[day]
}

pii = ["SSN", "EnrolleeType"] {
    input.peerID == "spiffe://domain.test/restricted"
}
