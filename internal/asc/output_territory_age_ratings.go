package asc

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

func printTerritoryAgeRatingsTable(resp *TerritoryAgeRatingsResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTerritory\tApp Store Age Rating")
	for _, item := range resp.Data {
		territoryID, err := territoryAgeRatingTerritoryID(item.Relationships)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", item.ID, territoryID, string(item.Attributes.AppStoreAgeRating))
	}
	return w.Flush()
}

func printTerritoryAgeRatingsMarkdown(resp *TerritoryAgeRatingsResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Territory | App Store Age Rating |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		territoryID, err := territoryAgeRatingTerritoryID(item.Relationships)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(territoryID),
			escapeMarkdown(string(item.Attributes.AppStoreAgeRating)),
		)
	}
	return nil
}

func territoryAgeRatingTerritoryID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var relationships TerritoryAgeRatingRelationships
	if err := json.Unmarshal(raw, &relationships); err != nil {
		return "", fmt.Errorf("decode territory age rating relationships: %w", err)
	}
	return relationships.Territory.Data.ID, nil
}
