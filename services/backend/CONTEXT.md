# My Project

The vocabulary of this codebase. Three different words here mean "a unit of the
system" and they are not interchangeable; most of this glossary exists to keep
them apart.

## Language

### Units of the system

**Component**:
Something the samsara supervisor starts, watches and restarts — the HTTP server
and each piece of infra. A component has a tier and a restart policy.
_Avoid_: service, module

**Service** (compose):
A container in a Docker Compose stack. Only ever means this.
_Avoid_: app, container name — services address each other by network alias

<!-- feat:if backend -->
**Domain**:
A bounded context inside the backend: one package under `internal/domain/`
owning one area of the problem, with its own model, ports and adapters.
_Avoid_: module, package, service, feature

**Service** (port):
The inbound port of a domain — the set of use cases callers may invoke,
implemented by `*Domain`. Written `Service` only inside `interface.go`; say
"the inbound port" in prose where compose services are also in play.

**Use case**:
One thing a domain can do, one verb, one file (`usecase_<verb>.go`), one method
on `*Domain`.
_Avoid_: handler, action, command, operation, service method

**Adapter**:
The translation between a domain and the outside world. Inbound adapters call a
domain's use cases; outbound adapters implement a domain's outbound ports.
_Avoid_: repository, gateway, client, DAO

**Port**:
An interface a domain declares in its own `interface.go`, naming what it offers
(inbound) or needs (outbound). A domain owns its ports; adapters satisfy them.
_Avoid_: interface (too general), contract

**Model**:
The plain structs a domain is about, in `<domain>/model/`. No framework
imports, so both adapters may depend on it.
_Avoid_: entity, DTO, schema

<!-- feat:if rabbitmq -->
**Read model**:
A domain whose state is projected from events rather than written directly, and
which serves queries only. `notestats` is the example.
_Avoid_: cache, view, materialized view, aggregate

**Projection**:
The act of folding an event into a read model's stored state, and the store that
holds the result.
_Avoid_: sync, replication, handler
<!-- feat:end -->
<!-- feat:end -->

### Builds and environments

<!-- feat:if template -->
**Feature**:
A slice of this template a fork may keep or drop at bootstrap time — `backend`,
`frontend`, `postgresql`, `redis`, `rabbitmq`. Selection happens once, when the
fork is created, and deletes what is not chosen. Nothing about it is a runtime
switch.
_Avoid_: feature flag, toggle, option — those imply a decision made at run time

**Marker**:
A `feat:if` / `feat:else` / `feat:end` comment that gates a block of a file for
one feature set.
_Avoid_: directive, pragma, ifdef

**Render**:
Turning the marked template into one chosen feature set, by pruning whole files
and collapsing markers in the rest.
_Avoid_: build, generate, compile
<!-- feat:end -->

**Environment**:
One of `local`, `dev`, `stage`, `prod` — a named set of env files, compose
overrides and host ports. `local` runs on the host against `dev`'s infra, which
is why the two share a secret pool.
_Avoid_: stage as a verb, deployment, profile (`profile` is a Compose term used
here only for the `app` profile)
