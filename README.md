# System Design for a Social Network

## Functional Requirements
- Support text and image posts
- Provide a home feed of posts and allow viewing other users’ posts
- Display posts in chronological order
- Enable likes and comments

## Non-Functional Requirements
- **Daily Active Users (DAU):** 50,000,000
- **Availability:** 99.95%
- **Data Persistence:** All posts must be saved reliably
- **Comments:** No limit on the number of comments
- **Maximum Friends per User:** 1,000,000
- **Maximum Words per Post:** 2,000
- **Images per Post:** Only one
- **Posting Frequency:** On average, each user creates one post every 5 days
- **Post Views:** Each user views posts 10 times per day
- **Time Zones:** The system must handle different time zones
- **Sessions:** No session management is required

## Approximate Requirements Calculations

1. **Write RPS (Requests Per Second)**  
   \[
   \frac{50{,}000{,}000 \text{ users}}{5 \text{ days}} \div 86{,}400 \text{ seconds/day} 
   \approx 115 \text{ writes/second}
   \]

2. **Read RPS**  
   \[
   \frac{50{,}000{,}000 \text{ users} \times 10 \text{ views/day}}{86{,}400 \text{ seconds/day}} 
   \approx 5{,}787 \text{ reads/second}
   \]

3. **Write Traffic**  
   \[
   115 \text{ writes/second} \times 4 \text{ KB/post} 
   = 460 \text{ KB/second}
   \]

4. **Read Traffic**  
   \[
   5{,}787 \text{ reads/second} \times 4 \text{ KB/post} 
   \approx 23 \text{ MB/second}
   \]

5. **Feed Storage**  
   - Storing 10 feeds per user:  
     \[
     50{,}000{,}000 \text{ users} \times 10 \text{ feeds/user} \times 2 \text{ KB/post} \times 2{,}000 \text{ words/post (approx.)}
     \approx 1 \text{ TB total}
     \]

## Issues and Proposed Solutions

- **Issue:**  
  Retrieving the feeds of all subscribed users plus the user’s own posts can require scanning a huge amount of data on disk, making it difficult to scale.

- **Proposed Solution:**  
  - Use a data pipeline tool like **Kafka** to pre-collect and aggregate feed data.
  - Maintain both a **user feed** and a **home feed**. Storing each feed might require around **1 TB** each.
  - Use a **caching database** (e.g., **Redis**, **Tarantool**) to reduce disk reads and improve read performance.
