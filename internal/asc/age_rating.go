package asc

import (
	"context"
	"encoding/json"
	"fmt"
)

// AgeRatingDeclarationAttributes describes the age rating declaration attributes.
type AgeRatingDeclarationAttributes struct {
	Gambling                                    *bool   `json:"gambling,omitempty"`
	SeventeenPlus                               *bool   `json:"seventeenPlus,omitempty"`
	UnrestrictedWebAccess                       *bool   `json:"unrestrictedWebAccess,omitempty"`
	AlcoholTobaccoOrDrugUseOrReferences         *string `json:"alcoholTobaccoOrDrugUseOrReferences,omitempty"`
	Contests                                    *string `json:"contests,omitempty"`
	GamblingSimulated                           *string `json:"gamblingSimulated,omitempty"`
	MedicalOrTreatmentInformation               *string `json:"medicalOrTreatmentInformation,omitempty"`
	ProfanityOrCrudeHumor                       *string `json:"profanityOrCrudeHumor,omitempty"`
	SexualContentGraphicAndNudity               *string `json:"sexualContentGraphicAndNudity,omitempty"`
	SexualContentOrNudity                       *string `json:"sexualContentOrNudity,omitempty"`
	HorrorOrFearThemes                          *string `json:"horrorOrFearThemes,omitempty"`
	MatureOrSuggestiveThemes                    *string `json:"matureOrSuggestiveThemes,omitempty"`
	ViolenceCartoonOrFantasy                    *string `json:"violenceCartoonOrFantasy,omitempty"`
	ViolenceRealistic                           *string `json:"violenceRealistic,omitempty"`
	ViolenceRealisticProlongedGraphicOrSadistic *string `json:"violenceRealisticProlongedGraphicOrSadistic,omitempty"`
	KidsAgeBand                                 *string `json:"kidsAgeBand,omitempty"`
}

// AgeRatingDeclarationResponse is the response from age rating declaration endpoints.
type AgeRatingDeclarationResponse = SingleResponse[AgeRatingDeclarationAttributes]

// AgeRatingDeclarationUpdateData is the data portion of an update request.
type AgeRatingDeclarationUpdateData struct {
	Type       ResourceType                   `json:"type"`
	ID         string                         `json:"id"`
	Attributes AgeRatingDeclarationAttributes `json:"attributes,omitempty"`
}

// AgeRatingDeclarationUpdateRequest is a request to update an age rating declaration.
type AgeRatingDeclarationUpdateRequest struct {
	Data AgeRatingDeclarationUpdateData `json:"data"`
}

// GetAgeRatingDeclarationForAppInfo retrieves the age rating declaration for an app info.
func (c *Client) GetAgeRatingDeclarationForAppInfo(ctx context.Context, appInfoID string) (*AgeRatingDeclarationResponse, error) {
	path := fmt.Sprintf("/v1/appInfos/%s/ageRatingDeclaration", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AgeRatingDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAgeRatingDeclarationForAppStoreVersion retrieves the age rating declaration for a version.
func (c *Client) GetAgeRatingDeclarationForAppStoreVersion(ctx context.Context, versionID string) (*AgeRatingDeclarationResponse, error) {
	path := fmt.Sprintf("/v1/appStoreVersions/%s/ageRatingDeclaration", versionID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AgeRatingDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateAgeRatingDeclaration updates an age rating declaration by ID.
func (c *Client) UpdateAgeRatingDeclaration(ctx context.Context, declarationID string, attributes AgeRatingDeclarationAttributes) (*AgeRatingDeclarationResponse, error) {
	request := AgeRatingDeclarationUpdateRequest{
		Data: AgeRatingDeclarationUpdateData{
			Type:       ResourceTypeAgeRatingDeclarations,
			ID:         declarationID,
			Attributes: attributes,
		},
	}

	body, err := BuildRequestBody(request)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, "PATCH", fmt.Sprintf("/v1/ageRatingDeclarations/%s", declarationID), body)
	if err != nil {
		return nil, err
	}

	var response AgeRatingDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
