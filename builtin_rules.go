package validation

import (
	"reflect"
	"strconv"
	"strings"
)

// registerBuiltInRules registers all built-in validation rules
func (v *Validator) registerBuiltInRules() {
	// Basic validation rules
	v.customRules["required"] = isRequired
	v.customRules["omitempty"] = isOmitEmpty
	
	// String validation rules
	v.customRules["min"] = hasMinOf
	v.customRules["max"] = hasMaxOf
	v.customRules["len"] = hasLengthOf
	v.customRules["eq"] = isEq
	v.customRules["ne"] = isNe
	v.customRules["oneof"] = isOneOf
	
	// String format rules
	v.customRules["alpha"] = isAlpha
	v.customRules["alphanum"] = isAlphaNumeric
	v.customRules["numeric"] = isNumeric
	v.customRules["email"] = isEmail
	v.customRules["url"] = isURL
	v.customRules["uri"] = isURI
	
	// Network validation rules
	v.customRules["ip"] = isIP
	v.customRules["ipv4"] = isIPv4
	v.customRules["ipv6"] = isIPv6
	v.customRules["cidr"] = isCIDR
	v.customRules["mac"] = isMAC
	v.customRules["hostname"] = isHostname
	
	// UUID validation
	v.customRules["uuid"] = isUUID
	v.customRules["uuid4"] = isUUIDv4
	
	// Date/time validation
	v.customRules["datetime"] = isDateTime
	v.customRules["date"] = isDate
	v.customRules["time"] = isTime
	
	// Other format validation
	v.customRules["json"] = isJSON
	v.customRules["base64"] = isBase64
	v.customRules["creditcard"] = isCreditCard
	v.customRules["phone"] = isPhone
	
	// Cross-field validation
	v.customRules["eqfield"] = isEqField
	v.customRules["nefield"] = isNeField
	v.customRules["gtfield"] = isGtField
	v.customRules["gtefiled"] = isGteField
	v.customRules["ltfield"] = isLtField
	v.customRules["ltefield"] = isLteField
	
	// Conditional validation
	v.customRules["required_if"] = isRequiredIf
	v.customRules["required_unless"] = isRequiredUnless
	v.customRules["required_with"] = isRequiredWith
	v.customRules["required_without"] = isRequiredWithout
}

// validateBuiltInRule validates using built-in rules that need special handling
func (v *Validator) validateBuiltInRule(fl *fieldLevel) error {
	switch fl.tag {
	case "ip":
		return ValidateIP(fl.fieldName, getString(fl.field))
	case "ipv4":
		return ValidateIPv4(fl.fieldName, getString(fl.field))
	case "ipv6":
		return ValidateIPv6(fl.fieldName, getString(fl.field))
	case "cidr":
		return ValidateCIDR(fl.fieldName, getString(fl.field))
	case "mac":
		return ValidateMAC(fl.fieldName, getString(fl.field))
	case "uuid":
		return ValidateUUID(fl.fieldName, getString(fl.field))
	case "uuid4":
		return ValidateUUIDv4(fl.fieldName, getString(fl.field))
	case "email":
		return ValidateEmail(fl.fieldName, getString(fl.field))
	case "url", "uri":
		return ValidateURL(fl.fieldName, getString(fl.field))
	case "hostname":
		return ValidateHostname(fl.fieldName, getString(fl.field))
	case "datetime":
		return ValidateDateTime(fl.fieldName, getString(fl.field))
	case "date":
		return ValidateDate(fl.fieldName, getString(fl.field))
	case "time":
		return ValidateTime(fl.fieldName, getString(fl.field))
	case "json":
		return ValidateJSON(fl.fieldName, getString(fl.field))
	case "base64":
		return ValidateBase64(fl.fieldName, getString(fl.field))
	case "creditcard":
		return ValidateCreditCard(fl.fieldName, getString(fl.field))
	case "phone":
		return ValidatePhone(fl.fieldName, getString(fl.field))
	}
	return nil
}

// Built-in validation functions

// isRequired validates that the field is not empty
func isRequired(fl FieldLevel) bool {
	return HasValue(fl)
}

// isOmitEmpty allows empty values to pass validation
func isOmitEmpty(fl FieldLevel) bool {
	return true // Always passes, used to skip validation on empty values
}

// hasMinOf validates minimum value/length
func hasMinOf(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()
	
	min, err := ParseIntParam(param)
	if err != nil {
		return false
	}
	
	switch field.Kind() {
	case reflect.String:
		return int64(len(field.String())) >= min
	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) >= min
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() >= min
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(field.Uint()) >= min
	case reflect.Float32, reflect.Float64:
		return int64(field.Float()) >= min
	}
	
	return false
}

// hasMaxOf validates maximum value/length
func hasMaxOf(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()
	
	max, err := ParseIntParam(param)
	if err != nil {
		return false
	}
	
	switch field.Kind() {
	case reflect.String:
		return int64(len(field.String())) <= max
	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) <= max
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() <= max
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(field.Uint()) <= max
	case reflect.Float32, reflect.Float64:
		return int64(field.Float()) <= max
	}
	
	return false
}

// hasLengthOf validates exact length
func hasLengthOf(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()
	
	length, err := ParseIntParam(param)
	if err != nil {
		return false
	}
	
	switch field.Kind() {
	case reflect.String:
		return int64(len(field.String())) == length
	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) == length
	}
	
	return false
}

// isEq validates equality
func isEq(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()
	
	switch field.Kind() {
	case reflect.String:
		return field.String() == param
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(param, 10, 64)
		return err == nil && field.Int() == p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		p, err := strconv.ParseUint(param, 10, 64)
		return err == nil && field.Uint() == p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(param, 64)
		return err == nil && field.Float() == p
	case reflect.Bool:
		p, err := strconv.ParseBool(param)
		return err == nil && field.Bool() == p
	}
	
	return false
}

// isNe validates inequality
func isNe(fl FieldLevel) bool {
	return !isEq(fl)
}

// isOneOf validates that value is one of the allowed values
func isOneOf(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()
	
	values := strings.Split(param, " ")
	fieldStr := getString(field)
	
	for _, v := range values {
		if fieldStr == strings.TrimSpace(v) {
			return true
		}
	}
	
	return false
}

// isAlpha validates alphabetic characters only
func isAlpha(fl FieldLevel) bool {
	field := getString(fl.Field())
	for _, r := range field {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return field != ""
}

// isAlphaNumeric validates alphanumeric characters only
func isAlphaNumeric(fl FieldLevel) bool {
	field := getString(fl.Field())
	for _, r := range field {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return field != ""
}

// isNumeric validates numeric characters only
func isNumeric(fl FieldLevel) bool {
	field := getString(fl.Field())
	for _, r := range field {
		if !(r >= '0' && r <= '9') {
			return false
		}
	}
	return field != ""
}

// isEmail validates email format
func isEmail(fl FieldLevel) bool {
	return ValidateEmail(fl.FieldName(), getString(fl.Field())) == nil
}

// isURL validates URL format
func isURL(fl FieldLevel) bool {
	return ValidateURL(fl.FieldName(), getString(fl.Field())) == nil
}

// isURI validates URI format (alias for URL)
func isURI(fl FieldLevel) bool {
	return ValidateURL(fl.FieldName(), getString(fl.Field())) == nil
}

// isIP validates IP address
func isIP(fl FieldLevel) bool {
	return ValidateIP(fl.FieldName(), getString(fl.Field())) == nil
}

// isIPv4 validates IPv4 address
func isIPv4(fl FieldLevel) bool {
	return ValidateIPv4(fl.FieldName(), getString(fl.Field())) == nil
}

// isIPv6 validates IPv6 address
func isIPv6(fl FieldLevel) bool {
	return ValidateIPv6(fl.FieldName(), getString(fl.Field())) == nil
}

// isCIDR validates CIDR notation
func isCIDR(fl FieldLevel) bool {
	return ValidateCIDR(fl.FieldName(), getString(fl.Field())) == nil
}

// isMAC validates MAC address
func isMAC(fl FieldLevel) bool {
	return ValidateMAC(fl.FieldName(), getString(fl.Field())) == nil
}

// isHostname validates hostname
func isHostname(fl FieldLevel) bool {
	return ValidateHostname(fl.FieldName(), getString(fl.Field())) == nil
}

// isUUID validates UUID
func isUUID(fl FieldLevel) bool {
	return ValidateUUID(fl.FieldName(), getString(fl.Field())) == nil
}

// isUUIDv4 validates UUID v4
func isUUIDv4(fl FieldLevel) bool {
	return ValidateUUIDv4(fl.FieldName(), getString(fl.Field())) == nil
}

// isDateTime validates datetime format
func isDateTime(fl FieldLevel) bool {
	return ValidateDateTime(fl.FieldName(), getString(fl.Field())) == nil
}

// isDate validates date format
func isDate(fl FieldLevel) bool {
	return ValidateDate(fl.FieldName(), getString(fl.Field())) == nil
}

// isTime validates time format
func isTime(fl FieldLevel) bool {
	return ValidateTime(fl.FieldName(), getString(fl.Field())) == nil
}

// isJSON validates JSON format
func isJSON(fl FieldLevel) bool {
	return ValidateJSON(fl.FieldName(), getString(fl.Field())) == nil
}

// isBase64 validates base64 format
func isBase64(fl FieldLevel) bool {
	return ValidateBase64(fl.FieldName(), getString(fl.Field())) == nil
}

// isCreditCard validates credit card using Luhn algorithm
func isCreditCard(fl FieldLevel) bool {
	return ValidateCreditCard(fl.FieldName(), getString(fl.Field())) == nil
}

// isPhone validates phone number
func isPhone(fl FieldLevel) bool {
	return ValidatePhone(fl.FieldName(), getString(fl.Field())) == nil
}

// Cross-field validation functions

// isEqField validates that field equals another field
func isEqField(fl FieldLevel) bool {
	field, kind, found := fl.GetStructFieldOK()
	if !found || kind != fl.Field().Kind() {
		return false
	}
	
	return field.Interface() == fl.Field().Interface()
}

// isNeField validates that field does not equal another field
func isNeField(fl FieldLevel) bool {
	return !isEqField(fl)
}

// isGtField validates that field is greater than another field
func isGtField(fl FieldLevel) bool {
	field, kind, found := fl.GetStructFieldOK()
	if !found {
		return false
	}
	
	return compareFields(fl.Field(), field, kind, 1)
}

// isGteField validates that field is greater than or equal to another field
func isGteField(fl FieldLevel) bool {
	field, kind, found := fl.GetStructFieldOK()
	if !found {
		return false
	}
	
	return compareFields(fl.Field(), field, kind, 0)
}

// isLtField validates that field is less than another field
func isLtField(fl FieldLevel) bool {
	field, kind, found := fl.GetStructFieldOK()
	if !found {
		return false
	}
	
	return compareFields(fl.Field(), field, kind, -1)
}

// isLteField validates that field is less than or equal to another field
func isLteField(fl FieldLevel) bool {
	field, kind, found := fl.GetStructFieldOK()
	if !found {
		return false
	}
	
	return compareFields(fl.Field(), field, kind, 0) || compareFields(fl.Field(), field, kind, -1)
}

// Conditional validation functions

// isRequiredIf validates that field is required if another field has a specific value
func isRequiredIf(fl FieldLevel) bool {
	params, err := ParseParam(fl.Param())
	if err != nil || len(params) < 2 {
		return false
	}
	
	fieldName := params[0]
	expectedValue := params[1]
	
	field, _, found := fl.(*fieldLevel).getStructFieldOK(fl.Parent(), fieldName)
	if !found {
		return true // If comparison field doesn't exist, this field is not required
	}
	
	if getString(field) == expectedValue {
		return HasValue(fl) // Field is required
	}
	
	return true // Field is not required
}

// isRequiredUnless validates that field is required unless another field has a specific value
func isRequiredUnless(fl FieldLevel) bool {
	params, err := ParseParam(fl.Param())
	if err != nil || len(params) < 2 {
		return false
	}
	
	fieldName := params[0]
	expectedValue := params[1]
	
	field, _, found := fl.(*fieldLevel).getStructFieldOK(fl.Parent(), fieldName)
	if !found {
		return HasValue(fl) // If comparison field doesn't exist, this field is required
	}
	
	if getString(field) != expectedValue {
		return HasValue(fl) // Field is required
	}
	
	return true // Field is not required
}

// isRequiredWith validates that field is required if another field has any value
func isRequiredWith(fl FieldLevel) bool {
	fieldName := fl.Param()
	field, _, found := fl.(*fieldLevel).getStructFieldOK(fl.Parent(), fieldName)
	if !found {
		return true // If comparison field doesn't exist, this field is not required
	}
	
	if !IsEmpty(&fieldLevel{field: field}) {
		return HasValue(fl) // Field is required
	}
	
	return true // Field is not required
}

// isRequiredWithout validates that field is required if another field is empty
func isRequiredWithout(fl FieldLevel) bool {
	fieldName := fl.Param()
	field, _, found := fl.(*fieldLevel).getStructFieldOK(fl.Parent(), fieldName)
	if !found {
		return HasValue(fl) // If comparison field doesn't exist, this field is required
	}
	
	if IsEmpty(&fieldLevel{field: field}) {
		return HasValue(fl) // Field is required
	}
	
	return true // Field is not required
}

// Helper functions

// getString safely converts a reflect.Value to string
func getString(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	}
	return ""
}

// compareFields compares two fields based on their type
func compareFields(field1, field2 reflect.Value, kind reflect.Kind, expected int) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val1, val2 := field1.Int(), field2.Int()
		if expected == 1 {
			return val1 > val2
		} else if expected == -1 {
			return val1 < val2
		}
		return val1 >= val2
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val1, val2 := field1.Uint(), field2.Uint()
		if expected == 1 {
			return val1 > val2
		} else if expected == -1 {
			return val1 < val2
		}
		return val1 >= val2
		
	case reflect.Float32, reflect.Float64:
		val1, val2 := field1.Float(), field2.Float()
		if expected == 1 {
			return val1 > val2
		} else if expected == -1 {
			return val1 < val2
		}
		return val1 >= val2
		
	case reflect.String:
		val1, val2 := field1.String(), field2.String()
		if expected == 1 {
			return val1 > val2
		} else if expected == -1 {
			return val1 < val2
		}
		return val1 >= val2
		
	default:
		return false
	}
}

// getStructFieldOK helper for fieldLevel
func (fl *fieldLevel) getStructFieldOK(val reflect.Value, fieldName string) (reflect.Value, reflect.Kind, bool) {
	val, kind, ok := fl.ExtractType(val)
	if !ok || kind != reflect.Struct {
		return reflect.Value{}, kind, false
	}
	
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return reflect.Value{}, reflect.Invalid, false
	}
	
	return fl.ExtractType(field)
}