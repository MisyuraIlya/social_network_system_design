// Replication:
// - master-slave (async)
// - replication factor 3
//
// Sharding:
// - key based by user_id

Table users {
  id integer [primary key]
  name varchar
  photo url
  created_at timestamp
}

Table user_data {
  user_id integer [primary key]
  description text
  city_id integer
  education object
  hobby object
}
Table cities {
  id integer [primary key]
  name varchar
}
Table interests {
  id integer [primary key]
  name varchar
}
Table interests_users {
  interest_id integer
  user_id integer
}

Table follows {
  user_id integer
  followed_id integer
  created_at timestamp
}
Table relationship {
  user_id integer
  related_id integer
  relationship_type integer
  created_at timestamp
}
Table friends {
  user_id integer
  friend_id integer
  created_at timestamp
}

Ref: users.id < user_data.user_id
Ref: cities.id < user_data.city_id
Ref: users.id < interests_users.user_id
Ref: interests.id < interests_users.interest_id

Ref: users.id < follows.user_id
Ref: users.id < relationship.user_id
Ref: users.id < friends.user_id