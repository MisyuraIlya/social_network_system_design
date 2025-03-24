// Replication:
// - master-master (async slave)
// - replication factor 3
//
// Sharding:
// - key based by chat_id

Table chats {
  id integer [primary key]
  user_id integer
  name varchar
  created_at timestamp
}
Table chats_users {
  chat_id integer
  user_id integer
  type string
  created_at timestamp
}
Table messages {
  id integer [primary key]
  user_id integer
  chat_id integer
  text text
  is_seen bool
  send_time timestamp
  delivered_time timestamp
}

Ref: chats.id < chats_users.chat_id
Ref: chats.id < messages.chat_id