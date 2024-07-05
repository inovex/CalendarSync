package sync

import (
	"fmt"
	"reflect"

	"github.com/inovex/CalendarSync/internal/config"
)

func autoConfigure(object any, config config.CustomMap) {
	ps := reflect.ValueOf(object)
	s := ps.Elem()
	if s.Kind() == reflect.Struct {
		for key, value := range config {
			field := s.FieldByName(key)
			if field.IsValid() && field.CanSet() {
				switch field.Kind() {
				case reflect.Int,
					reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64:
					if v, ok := value.(int); ok {
						field.SetInt(int64(v))
					} else {
						field.SetInt(value.(int64))
					}
				case reflect.Bool:
					field.SetBool(value.(bool))
				case reflect.String:
					field.SetString(value.(string))
				default:
					panic(fmt.Sprintf("autoConfigure(): unknown kind '%s' for field '%s'", key, field.Kind().String()))
				}
			}
		}
	}
}
