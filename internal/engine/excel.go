package engine

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ExtractExcel(path, sheet string, hasHeader bool) (*Table, error) {
	sheet = strings.TrimSpace(sheet)
	if err := ValidateSheet(sheet); err != nil {
		return nil, err
	}
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("打开 Excel 失败: %w", err)
	}
	defer file.Close()
	if sheet == "" {
		list := file.GetSheetList()
		if len(list) == 0 {
			return &Table{}, nil
		}
		sheet = list[0]
	}
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}
	return tableFromRows(rows, hasHeader)
}

func ValidateSheet(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if len([]rune(name)) > 31 || strings.ContainsAny(name, `:\/?*[]`) {
		return fmt.Errorf("工作表名称无效")
	}
	return nil
}
