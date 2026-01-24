package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printTerritoriesTable(resp *TerritoriesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCurrency")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Attributes.Currency)
	}
	return w.Flush()
}

func printTerritoriesMarkdown(resp *TerritoriesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Currency |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Currency),
		)
	}
	return nil
}

func printAppPricePointsTable(resp *AppPricePointsV3Response) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCustomer Price\tProceeds")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			item.ID,
			item.Attributes.CustomerPrice,
			item.Attributes.Proceeds,
		)
	}
	return w.Flush()
}

func printAppPricePointsMarkdown(resp *AppPricePointsV3Response) error {
	fmt.Fprintln(os.Stdout, "| ID | Customer Price | Proceeds |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.CustomerPrice),
			escapeMarkdown(item.Attributes.Proceeds),
		)
	}
	return nil
}

func printAppPriceScheduleTable(resp *AppPriceScheduleResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID")
	fmt.Fprintf(w, "%s\n", resp.Data.ID)
	return w.Flush()
}

func printAppPriceScheduleMarkdown(resp *AppPriceScheduleResponse) error {
	fmt.Fprintln(os.Stdout, "| ID |")
	fmt.Fprintln(os.Stdout, "| --- |")
	fmt.Fprintf(os.Stdout, "| %s |\n", escapeMarkdown(resp.Data.ID))
	return nil
}

func printAppAvailabilityTable(resp *AppAvailabilityV2Response) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tAvailable In New Territories")
	fmt.Fprintf(w, "%s\t%t\n", resp.Data.ID, resp.Data.Attributes.AvailableInNewTerritories)
	return w.Flush()
}

func printAppAvailabilityMarkdown(resp *AppAvailabilityV2Response) error {
	fmt.Fprintln(os.Stdout, "| ID | Available In New Territories |")
	fmt.Fprintln(os.Stdout, "| --- | --- |")
	fmt.Fprintf(os.Stdout, "| %s | %t |\n",
		escapeMarkdown(resp.Data.ID),
		resp.Data.Attributes.AvailableInNewTerritories,
	)
	return nil
}
