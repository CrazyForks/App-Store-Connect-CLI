package asc

// GameCenterAchievementV2Relationships describes relationships for v2 achievements.
type GameCenterAchievementV2Relationships struct {
	GameCenterDetail *Relationship `json:"gameCenterDetail,omitempty"`
	GameCenterGroup  *Relationship `json:"gameCenterGroup,omitempty"`
}

// GameCenterAchievementV2CreateData is the data portion of a v2 achievement create request.
type GameCenterAchievementV2CreateData struct {
	Type          ResourceType                          `json:"type"`
	Attributes    GameCenterAchievementCreateAttributes `json:"attributes"`
	Relationships *GameCenterAchievementV2Relationships `json:"relationships,omitempty"`
}

// GameCenterAchievementV2CreateRequest is a request to create a v2 achievement.
type GameCenterAchievementV2CreateRequest struct {
	Data GameCenterAchievementV2CreateData `json:"data"`
}
