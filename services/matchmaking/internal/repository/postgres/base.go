package postgres

// getOperator converts the filter operator to SQL operator
func getOperator(op string) string {
	switch op {
	case "eq":
		return "="
	case "neq":
		return "!="
	case "gt":
		return ">"
	case "lt":
		return "<"
	case "gte":
		return ">="
	case "lte":
		return "<="
	case "like":
		return "LIKE"
	case "not_like":
		return "NOT ILIKE"
	case "in":
		return "IN"
	case "not_in":
		return "NOT IN"
	default:
		return "="
	}
}

func getOperatorArray(op string) string {
	switch op {
	case "contains": // массив содержит значение или массив
		return "@>"
	case "contained": // значение содержится в массиве
		return "<@"
	case "overlap": // массивы пересекаются
		return "&&"
	default:
		return "@>"
	}
}

func isScalarOperator(op string) bool {
	switch op {
	case "eq", "neq", "gt", "lt", "gte", "lte", "in", "not_in":
		return true
	default:
		return false
	}
}

func isStringOperator(op string) bool {
	switch op {
	case "eq", "neq", "like", "not_like", "in", "not_in":
		return true
	default:
		return false
	}
}

func isBoolOperator(op string) bool {
	switch op {
	case "eq", "neq", "in", "not_in":
		return true
	default:
		return false
	}
}

func isArrayOperator(op string) bool {
	switch op {
	case "contains", "contained", "overlap":
		return true
	default:
		return false
	}
}
