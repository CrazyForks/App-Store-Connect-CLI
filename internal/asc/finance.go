package asc

import (
	"context"
	"net/url"
	"strings"
)

// FinanceReportType represents the finance report type.
type FinanceReportType string

const (
	FinanceReportTypeFinancial     FinanceReportType = "FINANCIAL"
	FinanceReportTypeFinanceDetail FinanceReportType = "FINANCE_DETAIL"
)

// FinanceReportParams describes finance report query parameters.
type FinanceReportParams struct {
	VendorNumber string
	ReportType   FinanceReportType
	RegionCode   string
	ReportDate   string
}

func buildFinanceReportQuery(params FinanceReportParams) string {
	values := url.Values{}
	if strings.TrimSpace(params.VendorNumber) != "" {
		values.Set("filter[vendorNumber]", strings.TrimSpace(params.VendorNumber))
	}
	if params.ReportType != "" {
		values.Set("filter[reportType]", string(params.ReportType))
	}
	if strings.TrimSpace(params.RegionCode) != "" {
		values.Set("filter[regionCode]", strings.TrimSpace(params.RegionCode))
	}
	if strings.TrimSpace(params.ReportDate) != "" {
		values.Set("filter[reportDate]", strings.TrimSpace(params.ReportDate))
	}
	return values.Encode()
}

// DownloadFinanceReport retrieves a finance report as a gzip stream.
func (c *Client) DownloadFinanceReport(ctx context.Context, params FinanceReportParams) (*ReportDownload, error) {
	path := "/v1/financeReports"
	if queryString := buildFinanceReportQuery(params); queryString != "" {
		path += "?" + queryString
	}

	resp, err := c.doStream(ctx, "GET", path, nil, "application/a-gzip")
	if err != nil {
		return nil, err
	}
	return &ReportDownload{Body: resp.Body, ContentLength: resp.ContentLength}, nil
}
