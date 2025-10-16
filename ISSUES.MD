1.Kafka robustness
Consumers are simple; no idempotency keys, no dead-letter strategy, no offset commit retries beyond happy path. That’s fine for PoC, risky in prod.