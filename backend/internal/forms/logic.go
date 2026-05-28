package forms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LogicEngine evaluates conditional logic
type LogicEngine struct{}

// NewLogicEngine creates a new logic engine
func NewLogicEngine() *LogicEngine {
	return &LogicEngine{}
}

// EvaluateCondition checks if a condition is met
func (e *LogicEngine) EvaluateCondition(
	condition LogicCondition,
	data map[string]any,
) bool {
	fieldValue, exists := data[condition.Field]
	if !exists {
		return false
	}

	condValue := condition.Value

	switch condition.Operator {
	case "equals":
		return strings.EqualFold(fmt.Sprintf("%v", fieldValue), condValue)
	case "not_equals":
		return !strings.EqualFold(fmt.Sprintf("%v", fieldValue), condValue)
	case "contains":
		return strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(condValue),
		)
	case "not_contains":
		return !strings.Contains(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(condValue),
		)
	case "starts_with":
		return strings.HasPrefix(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(condValue),
		)
	case "ends_with":
		return strings.HasSuffix(
			strings.ToLower(fmt.Sprintf("%v", fieldValue)),
			strings.ToLower(condValue),
		)
	case "greater_than":
		return compareNumbers(fmt.Sprintf("%v", fieldValue), condValue) > 0
	case "less_than":
		return compareNumbers(fmt.Sprintf("%v", fieldValue), condValue) < 0
	case "in":
		return isInList(fieldValue, condValue)
	case "not_in":
		return !isInList(fieldValue, condValue)
	case "matches":
		return matchesRegex(fmt.Sprintf("%v", fieldValue), condValue)
	case "is_checked":
		return isTruthy(fieldValue)
	case "is_not_checked":
		return !isTruthy(fieldValue)
	case "is_empty":
		str := fmt.Sprintf("%v", fieldValue)
		return str == "" || str == "null"
	case "is_not_empty":
		str := fmt.Sprintf("%v", fieldValue)
		return str != "" && str != "null"
	default:
		return false
	}
}

// EvaluateConditionGroup evaluates a group of conditions
func (e *LogicEngine) EvaluateConditionGroup(
	group *ConditionGroup,
	data map[string]any,
) bool {
	if group == nil || len(group.Conditions) == 0 {
		return true
	}

	switch group.Match {
	case "any":
		for _, cond := range group.Conditions {
			if e.EvaluateCondition(cond, data) {
				return true
			}
		}
		return false
	default: // "all"
		for _, cond := range group.Conditions {
			if !e.EvaluateCondition(cond, data) {
				return false
			}
		}
		return true
	}
}

// GetVisibleFields returns fields visible given data
func (e *LogicEngine) GetVisibleFields(
	fields []FormField,
	data map[string]any,
) []FormField {
	visible := make([]FormField, 0, len(fields))

	for _, field := range fields {
		if e.isFieldVisible(field, data) {
			visible = append(visible, field)
		}
	}

	return visible
}

// isFieldVisible checks if field should show
func (e *LogicEngine) isFieldVisible(
	field FormField,
	data map[string]any,
) bool {
	if len(field.Logic) == 0 {
		return true
	}

	for _, logic := range field.Logic {
		if logic.Action != "show" && logic.Action != "hide" {
			continue
		}

		var conditionMet bool
		if logic.ConditionGroup != nil {
			conditionMet = e.EvaluateConditionGroup(logic.ConditionGroup, data)
		} else {
			conditionMet = e.EvaluateCondition(logic.Condition, data)
		}

		if logic.Action == "show" && !conditionMet {
			return false
		}
		if logic.Action == "hide" && conditionMet {
			return false
		}
	}

	return true
}

// ValidateResponse validates submission data
func (e *LogicEngine) ValidateResponse(
	fields []FormField,
	data map[string]any,
) []FieldError {
	var errors []FieldError

	for _, field := range fields {
		value, exists := data[field.ID]

		// Check required
		if field.Required {
			if !exists || isEmpty(value) {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: field.Label + " is required",
				})
				continue
			}
		}

		if !exists || isEmpty(value) {
			continue
		}

		strVal := fmt.Sprintf("%v", value)

		// Validate email
		if field.Type == FieldTypeEmail {
			if !isValidEmail(strVal) {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: "please enter a valid email address",
				})
			}
		}

		// Validate number range
		if field.Type == FieldTypeNumber {
			num, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: "must be a valid number",
				})
				continue
			}
			if field.Validation.Min != nil &&
				num < *field.Validation.Min {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: fmt.Sprintf(
						"must be at least %.0f",
						*field.Validation.Min,
					),
				})
			}
			if field.Validation.Max != nil &&
				num > *field.Validation.Max {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: fmt.Sprintf(
						"must be at most %.0f",
						*field.Validation.Max,
					),
				})
			}
		}

		// Validate text length
		if field.Type == FieldTypeText ||
			field.Type == FieldTypeLongText {
			if field.Validation.MinLen != nil &&
				len(strVal) < *field.Validation.MinLen {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: fmt.Sprintf(
						"must be at least %d characters",
						*field.Validation.MinLen,
					),
				})
			}
			if field.Validation.MaxLen != nil &&
				len(strVal) > *field.Validation.MaxLen {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: fmt.Sprintf(
						"must be at most %d characters",
						*field.Validation.MaxLen,
					),
				})
			}
		}

		// Validate pattern
		if field.Validation.Pattern != "" {
			matched, err := regexp.MatchString(field.Validation.Pattern, strVal)
			if err == nil && !matched {
				errors = append(errors, FieldError{
					FieldID: field.ID,
					Message: field.Label + " format is invalid",
				})
			}
		}
	}

	return errors
}

// ExtractEmail finds email value from response data
func ExtractEmail(
	fields []FormField,
	data map[string]any,
) string {
	for _, field := range fields {
		if field.Type == FieldTypeEmail {
			if val, ok := data[field.ID]; ok {
				if email, ok := val.(string); ok {
					return strings.TrimSpace(email)
				}
			}
		}
	}
	// Try common field names
	for _, key := range []string{
		"email", "email_address",
		"your_email", "contact_email",
	} {
		if val, ok := data[key]; ok {
			if email, ok := val.(string); ok &&
				isValidEmail(email) {
				return strings.TrimSpace(email)
			}
		}
	}
	return ""
}

// ExtractName finds name value from response data
func ExtractName(
	fields []FormField,
	data map[string]any,
) string {
	for _, field := range fields {
		label := strings.ToLower(field.Label)
		if strings.Contains(label, "name") {
			if val, ok := data[field.ID]; ok {
				if name, ok := val.(string); ok {
					return strings.TrimSpace(name)
				}
			}
		}
	}
	return ""
}

// FieldError holds field validation error
type FieldError struct {
	FieldID string `json:"field_id"`
	Message string `json:"message"`
}

// Helpers

func compareNumbers(a, b string) float64 {
	numA, errA := strconv.ParseFloat(a, 64)
	numB, errB := strconv.ParseFloat(b, 64)
	if errA != nil || errB != nil {
		return 0
	}
	return numA - numB
}

func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	str := fmt.Sprintf("%v", val)
	return strings.TrimSpace(str) == ""
}

func isValidEmail(email string) bool {
	return strings.Contains(email, "@") &&
		strings.Contains(email, ".") &&
		len(email) >= 5
}

func isInList(val any, list string) bool {
	items := strings.Split(list, ",")
	strVal := strings.ToLower(fmt.Sprintf("%v", val))

	switch v := val.(type) {
	case []any:
		for _, item := range v {
			itemStr := strings.ToLower(fmt.Sprintf("%v", item))
			for _, listItem := range items {
				if strings.TrimSpace(itemStr) == strings.TrimSpace(strings.ToLower(listItem)) {
					return true
				}
			}
		}
		return false
	default:
		for _, item := range items {
			if strVal == strings.TrimSpace(strings.ToLower(item)) {
				return true
			}
		}
		return false
	}
}

func matchesRegex(val, pattern string) bool {
	matched, err := regexp.MatchString(pattern, val)
	if err != nil {
		return false
	}
	return matched
}

func isTruthy(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		return v == "true" || v == "yes" || v == "1" || v == "on"
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return fmt.Sprintf("%v", val) != ""
	}
}
