1. when users create comment or like we set in this service to decrease from feed service this logic
can be here
post_like:
    post_id
    user_id
post_comment:
    post_id
    comment_id
    text
    created_at
    user_id
    reply_id
post_likes_summary:
    post_id
    likes_number
post_comments_summary:
    post_id
    comments_number

2. when feed service take the 10 feeds he go to feedback service fetch only the post_likes_summary and post_comments_summary because to scan post like and post comment for all this to much to scan
in future it can be different service 

3. we use
post_like:
    post_id
    user_id
post_comment:
    post_id
    comment_id
    text
    created_at
    user_id
    reply_id
in redis when feed service ask 
this one requests

4. if user want see who liked and see comments this second requests with this
post_likes_summary:
    post_id
    likes_number
post_comments_summary:
    post_id
    comments_number

5. preferably to create 2 replications of postgres 