# Domains own their ports; cmd/main does all the wiring

Each domain declares both of its ports in its own `interface.go` — `Service`
inbound, `DB` (and any other need) outbound — and imports neither adapter. The
REST adapter takes the `Service` and registers routes against it directly;
`cmd/main` constructs DB adapter, then domain, then REST adapter, and is the
only place that knows the whole graph.

This replaced a `SetHandlerX` injection pattern, where the adapter was built
empty and handlers were pushed into it afterwards. That shape made a missing
wire a nil-pointer panic on the first request, and forced a nil guard into every
handler. Now a missing wire does not compile.

The trade-off is that `cmd/main` grows with each domain and its construction
order matters (domains with no cross-domain dependencies first). That is a
deliberate concentration: one long, explicit function that a reader can follow
beats a wiring mechanism they have to trust.
