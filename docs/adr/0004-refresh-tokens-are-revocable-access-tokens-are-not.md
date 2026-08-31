# Only refresh tokens are revocable

Logout has to mean something, but checking a denylist on every request would put
a network round-trip in front of every authenticated call and make the token
store a hard dependency of the request path. We split the two token types:
access tokens are short-lived (15m default) and validated by signature alone,
never looked up; refresh tokens carry a `jti`, are denylisted on logout, and are
single-use — `/auth/refresh` claims the presented token atomically, which both
rejects an already-used one and revokes it in a single operation.

So revocation is not immediate. A stolen access token stays valid until it
expires, and the access TTL is therefore the real exposure window — shorten it
before adding revocation machinery.

The `Revoker` port is required, not optional, precisely because a no-op here
would turn logout into a lie. A build without Redis gets `adapter/memory`, which
is a real denylist that is process-local and does not survive a restart; that is
a correctness limit to respect when running more than one replica, not a
disabled feature.
