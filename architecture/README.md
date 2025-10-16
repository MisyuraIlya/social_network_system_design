here my db diagrams 
feed service cache.dbml: 
// Service: feed-service (Redis-backed)
// Storage model (from code):
// - author_posts:{user_id}     -> LIST of FeedEntry (max 500)
// - users_feed:{user_id}       -> LIST of FeedEntry (max 1000)
// - celebrities_feed:{user_id} -> LIST of FeedEntry (max 500)
// - celebrities:set            -> SET of celebrity user IDs
//
// Notes:
// - We represent Redis lists/sets as logical tables for visualization.
// - No real SQL FKs (users live in user-service). Refs are omitted intentionally.

Table author_feeds {
  user_id    varchar(64) [pk, note: 'Redis key author_posts:{user_id}']
  entries    jsonb       [note: 'Array<FeedEntry> (LPUSH/LTRIM), newest first, cap=500']
  updated_at timestamp   [default: `now()`]
}

Table users_feeds {
  user_id    varchar(64) [pk, note: 'Redis key users_feed:{user_id}']
  entries    jsonb       [note: 'Array<FeedEntry> (R/W whole list on rebuild), cap=1000']
  updated_at timestamp   [default: `now()`]
}

Table celebrity_feeds {
  user_id    varchar(64) [pk, note: 'Redis key celebrities_feed:{user_id}']
  entries    jsonb       [note: 'Array<FeedEntry> for celebrity author, cap=500']
  updated_at timestamp   [default: `now()`]
}

Table celebrities {
  user_id     varchar(64) [pk, note: 'Redis set key celebrities:set']
  promoted_at timestamp    [default: `now()`]
}

// Optional: document the FeedEntry payload shape used in lists
// FeedEntry {
//   post_id   bigint
//   author_id varchar(64)
//   media_url varchar(512) | null
//   snippet   text | null
//   tags      text[] | null
//   created_at timestamp
//   score     double
// }


feedback service database.dbml
// Service: feedback-service
// Replication: master-slave (async), RF=3
// Sharding: key-based by post_id
// Notes: No cross-service FKs enforced (posts/users live elsewhere)

Table post_likes_sums {
  post_id     bigint [pk, note: 'Shard key; 1 row per post']
  likes_count bigint [not null, default: 0]
  updated_at  timestamp
}

Table post_comments_sums {
  post_id        bigint [pk, note: 'Shard key; 1 row per post']
  comments_count bigint [not null, default: 0]
  updated_at     timestamp
}

Table post_likes {
  post_id    bigint       [not null, note: 'Shard key']
  user_id    varchar(64)  [not null, note: 'User identifier (string/UUID)']
  created_at timestamp    [default: `now()`]

  indexes {
    (post_id, user_id) [pk, name: 'pk_post_likes'] // composite PK prevents dup likes
    post_id            [name: 'idx_post_likes_post']
    user_id            [name: 'idx_post_likes_user']
  }
  // cross-service references intentionally not enforced:
  // posts(id), users(id)
}

Table post_comments {
  id         bigint       [pk, increment]
  post_id    bigint       [not null, note: 'Shard key']
  user_id    varchar(64)  [not null]
  reply_id   bigint       [note: 'Self-reply; null for top-level']
  text       text         [not null]
  created_at timestamp    [default: `now()`]

  indexes {
    post_id          [name: 'idx_post_comments_post']
    user_id          [name: 'idx_post_comments_user']
    reply_id         [name: 'idx_post_comments_reply']
    (post_id, id)    [name: 'idx_post_comments_post_id']
  }
  // cross-service references intentionally not enforced:
  // posts(id), users(id)
}

/* Top-level refs (DBML requires this): */
Ref: post_comments.reply_id > post_comments.id

message service database.dbml:
// Service: message-service
// Replication: master-master (async slaves), RF=3
// Sharding: key-based by chat_id (conceptual; single DB in this service code)
// Notes: cross-service refs to users are intentionally not enforced here.

Table chats {
  id         bigint       [pk, increment]
  name       varchar(200) [not null]
  owner_id   varchar(64)  [not null, note: 'Creator user_id']
  created_at timestamp    [default: `now()`]

  indexes {
    owner_id [name: 'idx_chats_owner']
  }
}

Table chat_users {
  chat_id    bigint       [not null]
  user_id    varchar(64)  [not null]
  type       varchar(32)  [not null, note: 'member/admin/...']
  created_at timestamp    [default: `now()`]

  indexes {
    (chat_id, user_id) [pk, name: 'pk_chat_users']
    chat_id            [name: 'idx_chat_users_chat']
    user_id            [name: 'idx_chat_users_user']
  }
}

Table messages {
  id             bigint       [pk, increment]
  user_id        varchar(64)  [not null]
  chat_id        bigint       [not null]
  text           text
  media_url      varchar(512)
  is_seen        boolean      [not null, default: false]
  send_time      timestamp    [not null]             // set by service
  delivered_time timestamp                     

  indexes {
    chat_id           [name: 'idx_messages_chat']
    (chat_id, id)     [name: 'idx_messages_chat_id_desc'] // paging by chat, newest first
  }
}

Table message_seen {
  message_id bigint      [not null]
  user_id    varchar(64) [not null]
  seen_at    timestamp   [not null, default: `now()`]

  indexes {
    (message_id, user_id) [pk, name: 'pk_message_seen']
    message_id            [name: 'idx_message_seen_msg']
    user_id               [name: 'idx_message_seen_user']
  }
}

/* Top-level refs (diagram only; enforce locally, not cross-service) */
Ref: chats.id < chat_users.chat_id
Ref: chats.id < messages.chat_id
Ref: messages.id < message_seen.message_id

// Cross-service (users) – shown as comments to avoid enforcing external FKs:
// users.user_id > chats.owner_id
// users.user_id > chat_users.user_id
// users.user_id > messages.user_id
// users.user_id > message_seen.user_id


post service database.dbml:
// Service: post-service
// Replication: master-slave (async), RF=3
// Notes: Likes & comments live in feedback-service; media URL only

Table posts {
  id          bigint       [pk, increment]
  user_id     varchar(64)  [not null]
  description text         [not null]
  media_url   varchar(512) [note: 'URL from media-service']
  views       bigint       [not null, default: 0]
  created_at  timestamp    [default: `now()`]
  updated_at  timestamp    [default: `now()`]

  indexes {
    user_id          [name: 'idx_posts_user']
    (user_id, id)    [name: 'idx_posts_user_id'] // for "recent by user"
  }
  // cross-service reference to users(id) intentionally not enforced
}

Table tags {
  id         bigint       [pk, increment]
  name       varchar(120) [not null, unique]
  created_at timestamp    [default: `now()`]
}

Table posts_tags {
  post_id    bigint       [not null]
  tag_id     bigint       [not null]
  created_at timestamp    [default: `now()`]

  indexes {
    (post_id, tag_id) [pk, name: 'pk_posts_tags'] // upsert-friendly dedupe
    tag_id            [name: 'idx_posts_tags_tag']
  }
}

/* Top-level refs (DBML requires this): */
Ref: posts.id < posts_tags.post_id
Ref: tags.id  < posts_tags.tag_id


users service database.dbml
// Service: user-service
// Replication: master-slave (async), RF=3
// Sharding: key-based by user_id (user_id encodes shard id like "shard-uuid")
// Notes: FKs shown for clarity; in practice each shard DB holds its own partition.

Table users {
  id         bigint       [pk, increment]                     // internal numeric PK
  user_id    varchar(64)  [unique, note: 'Stable external ID with shard prefix']
  shard_id   int          [not null, note: 'For routing; also in JWT']
  email      varchar(120) [unique, not null]
  pass_hash  varchar(255) [not null]
  name       varchar(100) [not null]
  created_at timestamp    [default: `now()`]
  updated_at timestamp    [default: `now()`]

  indexes {
    shard_id                // read routing
    (email)                 // login path
    (user_id)               // lookups by external id
  }
}

Table profiles {
  user_id     varchar(64) [pk, note: 'Same as users.user_id']
  description text
  city_id     bigint
  education   jsonb
  hobby       jsonb
  updated_at  timestamp    [default: `now()`]

  indexes {
    city_id
  }
}

Table cities {
  id         bigint       [pk, increment]
  name       varchar(120) [unique, not null]
}

Table interests {
  id         bigint       [pk, increment]
  name       varchar(120) [unique, not null]
}

Table interest_users {
  user_id     varchar(64) [not null]
  interest_id bigint      [not null]

  indexes {
    (user_id, interest_id) [pk, name: 'pk_interest_users']  // composite PK
    user_id                 [name: 'idx_interest_users_user']
    interest_id             [name: 'idx_interest_users_interest']
  }
}

Table follows {
  user_id    varchar(64) [not null]  // follower
  target_id  varchar(64) [not null]  // followee
  created_at timestamp    [default: `now()`]

  indexes {
    (user_id, target_id) [pk, name: 'pk_follows']
    user_id              [name: 'idx_follows_user']
    target_id            [name: 'idx_follows_target']
  }
}

Table friends {
  user_id    varchar(64) [not null]
  friend_id  varchar(64) [not null]
  created_at timestamp    [default: `now()`]

  indexes {
    (user_id, friend_id) [pk, name: 'pk_friends']
    user_id              [name: 'idx_friends_user']
    friend_id            [name: 'idx_friends_friend']
  }
}

Table relationships {
  user_id    varchar(64) [not null]
  related_id varchar(64) [not null]
  type       int         [not null, note: '1=Follow, 2=Friend, 3=Block (see code)']
  created_at timestamp    [default: `now()`]

  indexes {
    (user_id, related_id, type) [pk, name: 'pk_relationships']
    user_id                     [name: 'idx_relationships_user']
    related_id                  [name: 'idx_relationships_related']
    (user_id, type)             [name: 'idx_relationships_user_type']
  }
}

/* Top-level refs (diagram only; enforce per-shard in practice) */
Ref: users.user_id  < profiles.user_id
Ref: cities.id      < profiles.city_id

Ref: users.user_id  < interest_users.user_id
Ref: interests.id   < interest_users.interest_id

Ref: users.user_id  < follows.user_id
Ref: users.user_id  < follows.target_id

Ref: users.user_id  < friends.user_id
Ref: users.user_id  < friends.friend_id

Ref: users.user_id  < relationships.user_id
Ref: users.user_id  < relationships.related_id


here now c4 designs
c1 level:
@startuml
!include <C4/C4_Container>

Person(user, "User")
Container(loadBalancer, "Load Balancer", "Nginx")
Container(apiGateway, "API Gateway")
Container(cdn, "CDN")

Container(postService, "Post Service", "Handling posts")
Container(feedService, "Feed Service", "Сollects a feed of posts")
Container(messageService, "Message Service", "Handling messages")
Container(userService, "User Service", "Handling actions with user")
Container(feedbackService, "Feedback Service", "Handling comments, likes")

System_Boundary(mediaSystem, "Media Service") {
    Container(mediaService, "Media Service", "Handling media files")
    ContainerDb(s3, "S3", "Blob storage")
}

Rel(user, loadBalancer, "Request", "REST")
Rel(loadBalancer, apiGateway, "Request", "REST")
Rel(user, cdn, "Downloads media")
Rel(apiGateway, postService, "Send post", )
Rel(apiGateway, mediaService, "Uploads media files")
Rel(apiGateway, feedService, "Get feed")
Rel(apiGateway, messageService, "Get/Send message")
Rel(apiGateway, userService, "Get user_data/relation")
Rel(apiGateway, feedbackService, "Get\Set comment/like")

Rel(mediaService, s3, "Uploads media files")
Rel(cdn, s3, "Downloads media from origin s3")
@enduml

here the c2 level services:
feed service:
@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")
Container(kafka, "Kafka", "message queue", "includes posts for creating a home and user feed")
Container(postService, "Post Service", "")
Container(userService, "User Service", "")

System_Boundary(feedSystem, "Feed Service") {
    Container(feedService, "Feed Service", "Processes posts")
    ContainerDb(redis, "redis", "store posts")
}

Rel(apiGateway, feedService, "request", "REST")
Rel(feedService, redis, "store feeds")
Rel(feedService, kafka, "get posts")
Rel(feedService, postService, "get more posts")
Rel(feedService, userService, "get followers, friedns...")


@enduml

feedback service:
@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")

System_Boundary(feedbackSystem, "Feedback Service") {
    Container(feedbackService, "Feedback Service", "Processes likes, comments")
    ContainerDb(pgSQL, "postgreSQL", "store likes, comments")
    ContainerDb(redis, "redis", "store popular likes, comments")
}

Rel(apiGateway, feedbackService, "request", "REST")
Rel(feedbackService, pgSQL, "")
Rel(feedbackService, redis, "")

@enduml

media service:
@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")
Container(cdn, "CDN")

System_Boundary(mediaSystem, "Media Service") {
    Container(mediaService, "Media Service", "Handling media files")
    ContainerDb(s3, "S3", "Blob storage")
}

Rel(apiGateway, mediaService, "Uploads media files", "REST")
Rel(mediaService, s3, "Uploads media files")
Rel(cdn, s3, "Downloads media from origin s3")
@enduml

message service:
@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")
Container(kafka, "Kafka", "")
Container(s3, "MediaService", "")
Container(notifycationService, "Notification service", "")


System_Boundary(MessageSystem, "Message Service") {
    Container(MessageService, "Message Service", "Processes messages")
    ContainerDb(redis, "redis")
    ContainerDb(pgSQL, "pgSQL")
}

Rel(apiGateway, MessageService, "request", "REST")
Rel(MessageService, redis, "store popular chats")
Rel(MessageService, pgSQL, "store messages")
Rel(MessageService, s3, "upload media")
Rel(MessageService, kafka, "new message")
Rel(kafka, notifycationService, "")


@enduml

post service:
@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")
Container(kafka, "Kafka", "message queue", "includes posts for creating a home and user feed")
Container(mediaService, "Media Service", "blob storage")

System_Boundary(postSystem, "Post Service") {
    Container(postService, "Post Service", "Processes posts")
    ContainerDb(pgSQL, "postgreSQL", "store posts")
}

Rel(apiGateway, postService, "request", "REST")
Rel(postService, pgSQL, "")
Rel(postService, kafka, "add post")
Rel(postService, mediaService, "upload media")
@enduml

user service:

@startuml
!include <C4/C4_Container>

Container(apiGateway, "API Gateway")

System_Boundary(userSystem, "User Service") {
    Container(userService, "User Service", "Handling users data")
    ContainerDb(pgSQL, "postgreSQL", "store data, relations, follow...")
}

Rel(apiGateway, userService, "request", "REST")
Rel(userService, pgSQL, "")
@enduml