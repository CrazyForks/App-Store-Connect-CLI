package asc

// GameCenterDetailAttributes represents a Game Center detail resource.
type GameCenterDetailAttributes struct {
	ArcadeEnabled                      bool `json:"arcadeEnabled,omitempty"`
	ChallengeEnabled                   bool `json:"challengeEnabled,omitempty"`
	LeaderboardSetEnabled              bool `json:"leaderboardSetEnabled,omitempty"`
	LeaderboardEnabled                 bool `json:"leaderboardEnabled,omitempty"`
	AchievementEnabled                 bool `json:"achievementEnabled,omitempty"`
	MultiplayerSessionEnabled          bool `json:"multiplayerSessionEnabled,omitempty"`
	MultiplayerTurnBasedSessionEnabled bool `json:"multiplayerTurnBasedSessionEnabled,omitempty"`
}

// GameCenterDetailResponse is the response from Game Center detail endpoints.
type GameCenterDetailResponse = SingleResponse[GameCenterDetailAttributes]

// Valid leaderboard formatters.
var ValidLeaderboardFormatters = []string{
	"INTEGER",
	"DECIMAL_POINT_1_PLACE",
	"DECIMAL_POINT_2_PLACE",
	"DECIMAL_POINT_3_PLACE",
	"ELAPSED_TIME_MILLISECOND",
	"ELAPSED_TIME_SECOND",
	"ELAPSED_TIME_MINUTE",
	"MONEY_WHOLE",
	"MONEY_POINT_2_PLACE",
}

// Valid leaderboard score sort types.
var ValidScoreSortTypes = []string{
	"ASC",
	"DESC",
}

// Valid leaderboard submission types.
var ValidSubmissionTypes = []string{
	"BEST_SCORE",
	"MOST_RECENT_SCORE",
}
