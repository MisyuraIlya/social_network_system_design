1.Kafka robustness
Consumers are simple; no idempotency keys, no dead-letter strategy, no offset commit retries beyond happy path. That’s fine for PoC, risky in prod.
2.Auth inconsistency (JWT vs HMAC)

Most services use standard JWT (golang-jwt), but notification-service uses a custom HMAC token (raw.sig with verifyToken).

Impact: Any real JWT from your gateway will be rejected by notification-service; protected endpoints will 401.

Fix: Switch notification-service internal/shared/httpx to the same JWT parser you use elsewhere (HS256; sub claim). Keep env JWT_SECRET consistent.