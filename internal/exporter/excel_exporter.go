package exporter

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"reporting-service/internal/domain"
)

const sheetName = "Report"

type ExcelExporter struct{}

func NewExcelExporter() *ExcelExporter { return &ExcelExporter{} }

func (e *ExcelExporter) Format() domain.ExportFormat { return domain.FormatExcel }
func (e *ExcelExporter) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}
func (e *ExcelExporter) FileExtension() string { return "xlsx" }

func (e *ExcelExporter) Export(result *domain.ReportResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	for i, col := range result.Columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetCellValue(sheetName, cell, col); err != nil {
			return nil, fmt.Errorf("write header cell: %w", err)
		}
	}

	for r, row := range result.Rows {
		for c, col := range result.Columns {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return nil, err
			}
			if err := f.SetCellValue(sheetName, cell, row[col]); err != nil {
				return nil, fmt.Errorf("write data cell: %w", err)
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write xlsx buffer: %w", err)
	}
	return buf.Bytes(), nil
}
