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

4. Post upload flow lacks bearer -> media-service

post-service UploadAndCreate → uploadToMediaService calls media-service without Authorization.

Impact: Works only when media-service JWT_SECRET is empty (dev). Fails in prod (401).

Fix:

In post/handler.UploadAndCreate, extract bearer (add httpx.BearerToken helper like in other services) and pass it into svc.UploadAndCreate.

Thread the bearer to uploadToMediaService and set Authorization: Bearer <token>.

5. Notification read API is public by path

GET /users/{user_id}/notifications is public (no auth), falling back to auth only if path param missing.

Impact: Anyone can fetch someone else’s notifications by user_id.

Fix: Require auth and ensure {user_id} equals sub from JWT (or drop the path variant and only serve /notifications).

6. feed service c2 lvl design with v1 and v2 what it correct?