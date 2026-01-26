package asc

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printDevicesTable(resp *DevicesResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tUDID\tPlatform\tStatus\tClass\tModel\tAdded")
	for _, item := range resp.Data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.UDID),
			compactWhitespace(string(item.Attributes.Platform)),
			compactWhitespace(string(item.Attributes.Status)),
			compactWhitespace(string(item.Attributes.DeviceClass)),
			compactWhitespace(item.Attributes.Model),
			compactWhitespace(item.Attributes.AddedDate),
		)
	}
	return w.Flush()
}

func printDevicesMarkdown(resp *DevicesResponse) error {
	fmt.Fprintln(os.Stdout, "| ID | Name | UDID | Platform | Status | Class | Model | Added |")
	fmt.Fprintln(os.Stdout, "| --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, item := range resp.Data {
		fmt.Fprintf(os.Stdout, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdown(item.ID),
			escapeMarkdown(item.Attributes.Name),
			escapeMarkdown(item.Attributes.UDID),
			escapeMarkdown(string(item.Attributes.Platform)),
			escapeMarkdown(string(item.Attributes.Status)),
			escapeMarkdown(string(item.Attributes.DeviceClass)),
			escapeMarkdown(item.Attributes.Model),
			escapeMarkdown(item.Attributes.AddedDate),
		)
	}
	return nil
}
