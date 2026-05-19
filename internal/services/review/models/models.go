package models

type CreateReviewRequest struct {
	Rating    int32    `json:"rating"`
	Comment   string   `json:"comment"`
	ImageURLs []string `json:"image_urls"`
}

type VoteReviewRequest struct {
	Vote string `json:"vote"`
}

type ReportReviewRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type ReplyReviewRequest struct {
	Comment string `json:"comment"`
}

type DisputeReviewRequest struct {
	Reason string `json:"reason"`
}

type ResolveReviewDisputeRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}
