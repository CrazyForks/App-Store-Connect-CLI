package asc

// GameCenterLeaderboardV2Relationships describes relationships for v2 leaderboards.
type GameCenterLeaderboardV2Relationships struct {
	GameCenterDetail *Relationship `json:"gameCenterDetail,omitempty"`
	GameCenterGroup  *Relationship `json:"gameCenterGroup,omitempty"`
}

// GameCenterLeaderboardV2CreateData is the data portion of a v2 leaderboard create request.
type GameCenterLeaderboardV2CreateData struct {
	Type          ResourceType                          `json:"type"`
	Attributes    GameCenterLeaderboardCreateAttributes `json:"attributes"`
	Relationships *GameCenterLeaderboardV2Relationships `json:"relationships,omitempty"`
}

// GameCenterLeaderboardV2CreateRequest is a request to create a v2 leaderboard.
type GameCenterLeaderboardV2CreateRequest struct {
	Data GameCenterLeaderboardV2CreateData `json:"data"`
}
