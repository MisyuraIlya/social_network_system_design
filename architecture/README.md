services:
feed service:
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

feedback service:
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

message service:
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

post service:
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

user service:
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

c4 design lvl 1:
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

c2 level design:
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