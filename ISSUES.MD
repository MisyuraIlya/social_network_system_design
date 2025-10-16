1.Kafka robustness
Consumers are simple; no idempotency keys, no dead-letter strategy, no offset commit retries beyond happy path. That’s fine for PoC, risky in prod.
2.Auth inconsistency (JWT vs HMAC)

Most services use standard JWT (golang-jwt), but notification-service uses a custom HMAC token (raw.sig with verifyToken).

Impact: Any real JWT from your gateway will be rejected by notification-service; protected endpoints will 401.

Fix: Switch notification-service internal/shared/httpx to the same JWT parser you use elsewhere (HS256; sub claim). Keep env JWT_SECRET consistent.

3. Message privacy & integrity

message-service doesn’t verify membership for:

sending messages (Send, SendWithUpload)

reading chat history (ListByChat)

marking seen (MarkSeen)

Impact: Any authenticated user can read or post in any chat by guessing chat_id.

Fix (service layer):

Add IsMember(chatID, userID) (bool, error) to chat.Repository/Service (or reuse a SELECT 1 FROM chat_users …).

In message.Service:

Send/SendWithUpload: require membership.

ListByChat: require membership.

MarkSeen: require membership (and optionally that the message belongs to that chat).

Optional: store sender’s membership at send-time; double check on read.