package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterGroups retrieves the list of Game Center groups.
func (c *Client) GetGameCenterGroups(ctx context.Context, opts ...GCGroupsOption) (*GameCenterGroupsResponse, error) {
	query := &gcGroupsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterGroups"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-groups: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCGroupsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroup retrieves a Game Center group by ID.
func (c *Client) GetGameCenterGroup(ctx context.Context, groupID string) (*GameCenterGroupResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterGroup creates a new Game Center group.
func (c *Client) CreateGameCenterGroup(ctx context.Context, referenceName *string) (*GameCenterGroupResponse, error) {
	var attrs *GameCenterGroupCreateAttributes
	if referenceName != nil {
		value := strings.TrimSpace(*referenceName)
		attrs = &GameCenterGroupCreateAttributes{
			ReferenceName: &value,
		}
	}

	payload := GameCenterGroupCreateRequest{
		Data: GameCenterGroupCreateData{
			Type:       ResourceTypeGameCenterGroups,
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterGroups", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterGroup updates an existing Game Center group.
func (c *Client) UpdateGameCenterGroup(ctx context.Context, groupID string, referenceName *string) (*GameCenterGroupResponse, error) {
	var attrs *GameCenterGroupUpdateAttributes
	if referenceName != nil {
		value := strings.TrimSpace(*referenceName)
		attrs = &GameCenterGroupUpdateAttributes{
			ReferenceName: &value,
		}
	}

	payload := GameCenterGroupUpdateRequest{
		Data: GameCenterGroupUpdateData{
			Type:       ResourceTypeGameCenterGroups,
			ID:         strings.TrimSpace(groupID),
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterGroup deletes a Game Center group.
func (c *Client) DeleteGameCenterGroup(ctx context.Context, groupID string) error {
	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// UpdateGameCenterGroupAchievements replaces the group's achievements.
func (c *Client) UpdateGameCenterGroupAchievements(ctx context.Context, groupID string, achievementIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterAchievements, achievementIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterAchievements", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupAchievementsV2 replaces the group's v2 achievements.
func (c *Client) UpdateGameCenterGroupAchievementsV2(ctx context.Context, groupID string, achievementIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterAchievements, achievementIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterAchievementsV2", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupLeaderboards replaces the group's leaderboards.
func (c *Client) UpdateGameCenterGroupLeaderboards(ctx context.Context, groupID string, leaderboardIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterLeaderboards", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupLeaderboardsV2 replaces the group's v2 leaderboards.
func (c *Client) UpdateGameCenterGroupLeaderboardsV2(ctx context.Context, groupID string, leaderboardIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterLeaderboardsV2", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupChallenges replaces the group's challenges.
func (c *Client) UpdateGameCenterGroupChallenges(ctx context.Context, groupID string, challengeIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterChallenges, challengeIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterChallenges", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}
