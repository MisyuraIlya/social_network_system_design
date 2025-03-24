// Replication:
// - master-master (async slave)
// - replication factor 3
//
// Sharding:
// - key based by user_id

Table celebrities_feed {
  id integer
  user_id string [note: 'if follower counts more than 10000']
  posts list
  created_at datetime
}

// Replication:
// - master-slave (async)
// - replication factor 3
//
// Sharding:
// - key based by user_id

Table users_feed {
  id integer
  user_id string
  posts list
  created_at datetime
}