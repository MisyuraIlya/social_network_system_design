1. to prepare the user homepage feeds need to preapre before - we use kafka for this
2. we store in redis in the memory the data not in disk then we need to use cahce db for example redis
3. we store last 10 data per user for example json
{
    user_id
    posts [
        {
            title:..
            description:..
            ...
        }
    ],
    home_feed: 
     user_id
     posts [

     ]
}
4.the calulation how much we need for this data per 50 000 000 users if we save for him the feed posts
memory need:
50 000 000  * 10 posts & 2000 kb = 1TB for cached feeds

5. if we add home feed then we can approximately for 2 TB
6. how we get the feeds friends? or subscibers?
we have in users service relation service wich say "who is my friends or subscribers"
then i go to my redis and update the feeds delete the last it its 10 and push the new one 

ISSUE can be: "Celebrity issue"
if we have like ronnaldo users he have 1 000 000 subscribers one his post can crash and do 1 000 000 operations wich need to do and this issue we can solve with add fields "celebrity" in users if he have more then 10000 subsribers he will be celebrity
and for this situation we dont update for all 1 000 000 users there feeds we just update his feed
and when user A wants his feeds he go only for his celebrity friends and check if there any new update in the celebrity feeds
if there new one we push to him to redis

7. preferably to create 2 replications of redis because if the redis down all load go opun post service