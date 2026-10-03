package domain

type PrintIndex struct {
	Snapshot       string `json:"snapshot_sha256"`
	OriginalSHA256 string `json:"original_sha256"`
	Pages          int    `json:"pages"`
	TextStatus     string `json:"text_status"`
	Tool           string `json:"tool"`
	TextFile       string `json:"-"`
	TextSHA256     string `json:"-"`
	CoverPath      string `json:"-"`
	CoverSHA256    string `json:"-"`
	CoverBytes     int64  `json:"-"`
}
type PrintHit struct {
	Page    int    `json:"page"`
	Snippet string `json:"snippet"`
}
