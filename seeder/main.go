package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// User represents the registration payload.
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func main() {
	// Seed gofakeit for random data generation.
	gofakeit.Seed(time.Now().UnixNano())

	// --- USERS GROUP ---

	// 1. Register a new user.
	user := User{
		Email:    gofakeit.Email(),
		Password: "123456", // default password
		Name:     gofakeit.Name(),
	}
	registerUser(user)

	// 2. Login and retrieve the auth token.
	token := loginUser(user.Email, user.Password)
	if token == "" {
		log.Fatal("Could not obtain token, aborting seeding process")
	}

	// 3. Create a city.
	createCity(token)
	// 4. List cities.
	listCities(token)
	// 5. Get user data using query parameter.
	getUserData(token, gofakeit.Number(1, 100))
	// 6. Update user data.
	updateUserData(token)
	// 7. Create an interest.
	interestID := createInterest(token)
	// 8. List interests.
	listInterests(token)
	// 9. Add an interest to a user.
	addInterest(token, gofakeit.Number(1, 100), interestID)
	// 10. Create a follow relationship.
	createFollow(token, gofakeit.Number(1, 100), gofakeit.Number(1, 100))
	// 11. Create a relationship.
	createRelationship(token, gofakeit.Number(1, 100), gofakeit.Number(1, 100), gofakeit.Number(1, 3))
	// 12. Create a friend.
	createFriend(token, gofakeit.Number(1, 100), gofakeit.Number(1, 100))
	// 13. Get user details by ID.
	getUserByID(token, gofakeit.Number(1, 100))

	// --- POSTS GROUP ---

	// 14. Create a new post.
	postID := createPost(token, gofakeit.Number(1, 100))
	// 15. List all posts.
	listPosts(token)
	// 16. Get post details.
	getPost(token, postID)
	// 17. Update the post.
	updatePost(token, postID)
	// 18. Delete the post.
	// deletePost(token, postID)
	// 19. Create a comment on a post (using postID 1 for demonstration).
	createComment(token, 1)
	// 20. List comments on a post.
	listComments(token, 1)
	// 21. Like a post.
	likePost(token, postID)
	// 22. Create a new tag.
	tagID := createTag(token)
	log.Printf("Created tag with id: %d", tagID)
	// 23. List tags (GET request with no auth).
	listTags()

	// --- MEIDA (MEDIA) GROUP ---

	// 24. Check media service health.
	mediaHealthCheck()
	// 25. Upload a file (adjust file path as needed).
	uploadMedia(token)

	// --- FEED GROUP ---

	// 26. Feed service health check.
	feedHealthCheck()
	// 27. Create a new feed entry.
	createFeed(token)
	// 28. Get feed for a user.
	getFeed(token, gofakeit.Number(1, 100))

	// --- FEEDBACK GROUP ---

	// 29. Like feedback.
	feedbackLike(token)
	// 30. Comment on feedback.
	feedbackComment(token)
}

//
// USERS Group Functions
//

func registerUser(user User) {
	url := "http://localhost:8090/auth/register"
	data, _ := json.Marshal(user)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println("Error in registerUser:", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("registerUser: %s status: %s", user.Email, resp.Status)
}

func loginUser(email, password string) string {
	url := "http://localhost:8090/auth/login"
	credentials := map[string]string{"email": email, "password": password}
	data, _ := json.Marshal(credentials)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println("Error in loginUser:", err)
		return ""
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	token, _ := result["token"].(string)
	log.Printf("loginUser: %s logged in. Token: %s", email, token)
	return token
}

func createCity(token string) {
	url := "http://localhost:8090/users/cities/create"
	payload := map[string]string{"name": gofakeit.City()}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createCity:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createCity status:", resp.Status)
}

func listCities(token string) {
	url := "http://localhost:8081/users/cities/list"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in listCities:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("listCities status:", resp.Status)
}

func getUserData(token string, userID int) {
	url := fmt.Sprintf("http://localhost:8090/users/userdata/get?user_id=%d", userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in getUserData:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("getUserData status:", resp.Status)
}

func updateUserData(token string) {
	url := "http://localhost:8090/users/userdata/update"
	payload := map[string]interface{}{
		"userID":      gofakeit.Number(1, 100),
		"description": "Updated description " + gofakeit.Sentence(5),
		"cityID":      gofakeit.Number(1, 100),
		"education":   fmt.Sprintf("{\"school\":\"%s\"}", gofakeit.Company()),
		"hobby":       fmt.Sprintf("{\"likes\":[\"%s\"]}", gofakeit.Hobby()),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in updateUserData:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("updateUserData status:", resp.Status)
}

func createInterest(token string) int {
	url := "http://localhost:8090/users/interests/create"
	payload := map[string]string{"name": gofakeit.Hobby()}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createInterest:", err)
		return 0
	}
	defer resp.Body.Close()
	log.Println("createInterest status:", resp.Status)
	// For demonstration, return a fake interest ID.
	return gofakeit.Number(1, 100)
}

func listInterests(token string) {
	url := "http://localhost:8090/users/interests/list"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in listInterests:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("listInterests status:", resp.Status)
}

func addInterest(token string, userID, interestID int) {
	url := "http://localhost:8090/users/interests/add"
	payload := map[string]interface{}{
		"user_id":     userID,
		"interest_id": interestID,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in addInterest:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("addInterest status:", resp.Status)
}

func createFollow(token string, userID, followedID int) {
	url := "http://localhost:8090/users/follows/create"
	payload := map[string]interface{}{
		"user_id":     userID,
		"followed_id": followedID,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createFollow:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createFollow status:", resp.Status)
}

func createRelationship(token string, userID, relatedID, relationshipType int) {
	url := "http://localhost:8090/users/relationships/create"
	payload := map[string]interface{}{
		"user_id":           userID,
		"related_id":        relatedID,
		"relationship_type": relationshipType,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createRelationship:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createRelationship status:", resp.Status)
}

func createFriend(token string, userID, friendID int) {
	url := "http://localhost:8090/users/friends/create"
	payload := map[string]interface{}{
		"user_id":   userID,
		"friend_id": friendID,
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createFriend:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createFriend status:", resp.Status)
}

func getUserByID(token string, userID int) {
	url := fmt.Sprintf("http://localhost:8081/users/%d", userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in getUserByID:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("getUserByID status:", resp.Status)
}

//
// POSTS Group Functions
//

func createPost(token string, userID int) int {
	url := "http://localhost:8090/posts/"
	payload := map[string]interface{}{
		"user_id":     userID,
		"description": gofakeit.Sentence(10),
		"media":       gofakeit.ImageURL(640, 480),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createPost:", err)
		return 0
	}
	defer resp.Body.Close()
	log.Println("createPost status:", resp.Status)
	// For demonstration, return a fake post ID.
	return gofakeit.Number(1, 100)
}

func listPosts(token string) {
	url := "http://localhost:8090/posts/"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in listPosts:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("listPosts status:", resp.Status)
}

func getPost(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/%d", postID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in getPost:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("getPost status:", resp.Status)
}

func updatePost(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/%d", postID)
	payload := map[string]string{
		"description": "Updated post description " + gofakeit.Sentence(5),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in updatePost:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("updatePost status:", resp.Status)
}

func deletePost(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/%d", postID)
	// DELETE request; here we send an empty body.
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in deletePost:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("deletePost status:", resp.Status)
}

func createComment(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/comments/%d", postID)
	payload := map[string]interface{}{
		"user_id": gofakeit.Number(1, 100),
		"name":    gofakeit.FirstName(),
		"text":    gofakeit.Sentence(5),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createComment:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createComment status:", resp.Status)
}

func listComments(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/comments/%d", postID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in listComments:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("listComments status:", resp.Status)
}

func likePost(token string, postID int) {
	url := fmt.Sprintf("http://localhost:8090/posts/like/%d", postID)
	payload := map[string]interface{}{
		"user_id":    gofakeit.Number(1, 100),
		"comment_id": 0, // Dummy value for demonstration.
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in likePost:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("likePost status:", resp.Status)
}

func createTag(token string) int {
	url := "http://localhost:8090/posts/tags"
	payload := map[string]string{"name": gofakeit.Word()}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createTag:", err)
		return 0
	}
	defer resp.Body.Close()
	log.Println("createTag status:", resp.Status)
	// Return a fake tag ID for demonstration.
	return gofakeit.Number(1, 100)
}

func listTags() {
	url := "http://localhost:8082/posts/tags"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error in listTags:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("listTags status:", resp.Status)
}

//
// MEIDA (MEDIA) Group Functions
//

func mediaHealthCheck() {
	url := "http://localhost:8084/health"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error in mediaHealthCheck:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("mediaHealthCheck status:", resp.Status)
}

func uploadMedia(token string) {
	url := "http://localhost:8090/media/upload"
	// Path to the file to be uploaded; adjust as necessary.
	filePath := "testfile.png"
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file for upload:", err)
		return
	}
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		log.Println("Error creating form file:", err)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		log.Println("Error copying file data:", err)
		return
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		log.Println("Error creating upload request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in uploadMedia:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("uploadMedia status:", resp.Status)
}

//
// FEED Group Functions
//

func feedHealthCheck() {
	url := "http://localhost:8083/health"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error in feedHealthCheck:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("feedHealthCheck status:", resp.Status)
}

func createFeed(token string) {
	url := "http://localhost:8090/feed"
	payload := map[string]string{
		"user_id": "123",
		"post_id": gofakeit.Word(),
		"content": "Hello World " + gofakeit.Sentence(3),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in createFeed:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("createFeed status:", resp.Status)
}

func getFeed(token string, userID int) {
	url := fmt.Sprintf("http://localhost:8090/feed?user_id=%d", userID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in getFeed:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("getFeed status:", resp.Status)
}

//
// FEEDBACK Group Functions
//

func feedbackLike(token string) {
	url := "http://localhost:8090/feedback/like"
	payload := map[string]string{
		"userId": "user123",
		"postId": "post456",
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in feedbackLike:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("feedbackLike status:", resp.Status)
}

func feedbackComment(token string) {
	url := "http://localhost:8090/feedback/comment"
	payload := map[string]string{
		"userId":  "user123",
		"postId":  "post456",
		"content": "This is amazing! " + gofakeit.Sentence(3),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in feedbackComment:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("feedbackComment status:", resp.Status)
}
