package components

import (
	"ebiten-rogue/ecs"
	"fmt"
	"reflect"
	"strings"
)

// componentNameMap maps string component names to their IDs
var componentNameMap = map[string]ecs.ComponentID{
	"Position":   Position,
	"Renderable": Renderable,
	"Collision":  Collision,
	"AI":         AI,
	"Map":        MapComponentID,
	"Appearance": Appearance,
	"Camera":     Camera,
	"Player":     Player,
	"Stats":      Stats,
	"MapType":    MapType,
	"Name":       Name,
	"MapContext": MapContext,
	"Inventory":  Inventory,
	"Item":       Item,
	"FOV":        FOV,
	"Equipment":  Equipment,
}

// GetComponentIDByName returns the ComponentID for a given component name string
// The lookup is case-insensitive
func GetComponentIDByName(name string) (ecs.ComponentID, bool) {
	// Try exact match first
	if id, exists := componentNameMap[name]; exists {
		return id, true
	}

	// Try case-insensitive match
	name = strings.ToLower(name)
	for compName, id := range componentNameMap {
		if strings.ToLower(compName) == name {
			return id, true
		}
	}

	return 0, false
}

// GetComponentProperty returns the value of a property in a component
// Uses reflection to access component properties dynamically
func GetComponentProperty(comp interface{}, propertyName string) (interface{}, error) {
	val := reflect.ValueOf(comp)

	// Handle pointer types
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check if the property exists
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("component is not a struct: %T", comp)
	}

	field := val.FieldByName(propertyName)
	if !field.IsValid() {
		return nil, fmt.Errorf("property not found: %s", propertyName)
	}

	// Return the property value
	return field.Interface(), nil
}

// SetComponentProperty sets the value of a property in a component
// Uses reflection to modify component properties dynamically
func SetComponentProperty(comp interface{}, propertyName string, value interface{}) error {
	val := reflect.ValueOf(comp)

	// Handle pointer types
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("component must be a pointer to struct: %T", comp)
	}
	val = val.Elem()

	// Check if the property exists
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("component is not a struct: %T", comp)
	}

	field := val.FieldByName(propertyName)
	if !field.IsValid() {
		return fmt.Errorf("property not found: %s", propertyName)
	}

	// Check if field is settable
	if !field.CanSet() {
		return fmt.Errorf("property cannot be set: %s", propertyName)
	}

	// Set the property value based on its type
	valueVal := reflect.ValueOf(value)
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle integer types
		var intVal int64
		switch v := value.(type) {
		case int:
			intVal = int64(v)
		case int64:
			intVal = v
		case float64:
			intVal = int64(v)
		default:
			return fmt.Errorf("cannot convert %T to int64 for property %s", value, propertyName)
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Handle unsigned integer types
		var uintVal uint64
		switch v := value.(type) {
		case uint:
			uintVal = uint64(v)
		case uint64:
			uintVal = v
		case int:
			if v < 0 {
				return fmt.Errorf("cannot convert negative value to uint for property %s", propertyName)
			}
			uintVal = uint64(v)
		case float64:
			if v < 0 {
				return fmt.Errorf("cannot convert negative value to uint for property %s", propertyName)
			}
			uintVal = uint64(v)
		default:
			return fmt.Errorf("cannot convert %T to uint64 for property %s", value, propertyName)
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		// Handle float types
		var floatVal float64
		switch v := value.(type) {
		case float64:
			floatVal = v
		case float32:
			floatVal = float64(v)
		case int:
			floatVal = float64(v)
		default:
			return fmt.Errorf("cannot convert %T to float64 for property %s", value, propertyName)
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		// Handle boolean type
		boolVal, ok := value.(bool)
		if !ok {
			return fmt.Errorf("cannot convert %T to bool for property %s", value, propertyName)
		}
		field.SetBool(boolVal)

	case reflect.String:
		// Handle string type
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("cannot convert %T to string for property %s", value, propertyName)
		}
		field.SetString(strVal)

	default:
		// Try to set directly if types match
		if field.Type() == valueVal.Type() {
			field.Set(valueVal)
		} else {
			return fmt.Errorf("unsupported property type: %s for %s", field.Kind(), propertyName)
		}
	}

	return nil
}
