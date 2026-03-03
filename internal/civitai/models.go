package civitai

import "time"

// Model represents a Civitai model.
type Model struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Type          string         `json:"type"`
	NSFW          bool           `json:"nsfw"`
	Tags          []string       `json:"tags"`
	Mode          string         `json:"mode"`
	Creator       Creator        `json:"creator"`
	Stats         ModelStats     `json:"stats"`
	ModelVersions []ModelVersion `json:"modelVersions"`
}

type Creator struct {
	Username string `json:"username"`
	Image    string `json:"image"`
}

type ModelStats struct {
	DownloadCount int     `json:"downloadCount"`
	FavoriteCount int     `json:"favoriteCount"`
	CommentCount  int     `json:"commentCount"`
	RatingCount   int     `json:"ratingCount"`
	Rating        float64 `json:"rating"`
}

type ModelVersion struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	CreatedAt    time.Time         `json:"createdAt"`
	DownloadURL  string            `json:"downloadUrl"`
	TrainedWords []string          `json:"trainedWords"`
	Files        []File            `json:"files"`
	Images       []Image           `json:"images"`
	Stats        ModelVersionStats `json:"stats"`
}

type ModelVersionStats struct {
	DownloadCount int     `json:"downloadCount"`
	RatingCount   int     `json:"ratingCount"`
	Rating        float64 `json:"rating"`
}

type File struct {
	ID               int          `json:"id"`
	Name             string       `json:"name"`
	SizeKB           float64      `json:"sizeKb"`
	Type             string       `json:"type"`
	Metadata         FileMetadata `json:"metadata"`
	PickleScanResult string       `json:"pickleScanResult"`
	VirusScanResult  string       `json:"virusScanResult"`
	ScannedAt        *time.Time   `json:"scannedAt"`
	Hashes           FileHashes   `json:"hashes"`
	DownloadURL      string       `json:"downloadUrl"`
	Primary          bool         `json:"primary"`
}

type FileMetadata struct {
	FP     string `json:"fp"`
	Size   string `json:"size"`
	Format string `json:"format"`
}

type FileHashes struct {
	AutoV1 string `json:"AutoV1"`
	AutoV2 string `json:"AutoV2"`
	SHA256 string `json:"SHA256"`
	CRC32  string `json:"CRC32"`
	BLAKE3 string `json:"BLAKE3"`
}
