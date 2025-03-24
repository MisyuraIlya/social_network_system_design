// Replication:
// - master-slave (async)
// - replication factor 3
//
// Sharding:
// - key based by post_id

Table post_likes_sum {
  post_id integer [primary key]
  likes_count integer
}

Table post_comments_sum {
  post_id integer [primary key]
  comments_count integer
}

Table post_like {
  post_id integer [primary key]
  user_id integer
}

Table post_comment {
  post_id integer [primary key]
  user_id integer
  comment_id integer
  reply_id integer
  text text
  created_at timestamp
}