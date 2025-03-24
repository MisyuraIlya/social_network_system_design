// Replication:
// - master-slave (async)
// - replication factor 3
//
// Sharding:
// - key based by post_id

Table posts {
  id integer [primary key]
  user_id integer
  description text
  media url [note: 'Link to content']
  likes integer
  views integer
  created_at timestamp
}
Table tags {
  id integer [primary key]
  name varchar
  created_at timestamp
}
Table posts_tags {
  post_id integer
  tag_id integer
  created_at timestamp
}
Table comments {
  id integer [primary key]
  user_id integer
  post_id integer
  name varchar
  text text
  created_at timestamp
}

Table likes {
  id integer [primary key]
  user_id integer
  post_id integer
  comment_id integer
}

table s3_media_store {
  id integer [primary key]
  data media
}


Ref: posts.id < posts_tags.post_id
Ref: tags.id < posts_tags.tag_id
Ref: comments.post_id < posts.id
Ref: posts.id < likes.post_id