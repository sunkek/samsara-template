# SQL is hand-written, run through pgx

The postgresql adapters hold SQL as string literals and scan rows explicitly.
There is no ORM and no query builder, and none is planned.

The adapter layer exists to be the one place that knows about the database, so
its whole job is the translation an ORM would hide. Keeping the SQL visible
means the query a use case runs is readable next to the use case, `EXPLAIN`
works on text copied straight from the source, and there is no mapping layer to
learn or fight when a query stops being simple.

The cost is real: writing out columns, scanning by hand, and no compile-time
check that a query matches its struct. Migrations in `infra/postgresql/migration`
are the only schema authority, and a mismatch surfaces at run time.
