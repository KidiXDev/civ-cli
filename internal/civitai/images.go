package civitai

import "time"

// Image represents a Civitai image.
type Image struct {
	ID        int         `json:"id"`
	URL       string      `json:"url"`
	Hash      string      `json:"hash"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	NSFW      interface{} `json:"nsfw"` // sometimes bool, sometimes string in API doc... safer to be generic or unmarshal manually
	NSFWLevel interface{} `json:"nsfwLevel"`
	CreatedAt time.Time   `json:"createdAt"`
	PostID    int         `json:"postId"`
	Stats     ImageStats  `json:"stats"`
	Meta      ImageMeta   `json:"meta"`
	Username  string      `json:"username"`
}

type ImageStats struct {
	CryCount     int `json:"cryCount"`
	LaughCount   int `json:"laughCount"`
	LikeCount    int `json:"likeCount"`
	DislikeCount int `json:"dislikeCount"`
	HeartCount   int `json:"heartCount"`
	CommentCount int `json:"commentCount"`
}

// ImageMeta is a map since parameters can be dynamic
type ImageMeta map[string]interface{}
