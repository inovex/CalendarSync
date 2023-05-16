package sync

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/inovex/CalendarSync/internal/config"
	"github.com/inovex/CalendarSync/internal/models"
	"github.com/inovex/CalendarSync/internal/transformation"
)

// Transformer applies a well-defined transformation to an event. Multiple transformers may be concatenated.
type Transformer interface {
	NamedComponent
	// Transform applies the Transformer logic to the event and returns the changed Event.
	Transform(source models.Event, sink models.Event) (models.Event, error)
}

// TransformEvent will transform the given event by applying every transformer given.
// The final transformed event is returned.
func TransformEvent(event models.Event, transformers ...Transformer) models.Event {
	transformedEvent := models.NewSyncEvent(event)

	for i := 0; i < len(transformers); i++ {
		transformedEvent, _ = transformers[i].Transform(event, transformedEvent)
	}
	return transformedEvent
}

var (
	// transformerConfigMapping maps "name" values from the config to a default object of the matching Transformer.
	transformerConfigMapping = map[string]Transformer{
		"ReplaceTitle":    &transformation.ReplaceTitle{NewTitle: "[CalendarSync Event]"},
		"PrefixTitle":     &transformation.PrefixTitle{Prefix: ""},
		"KeepTitle":       &transformation.KeepTitle{},
		"KeepMeetingLink": &transformation.KeepMeetingLink{},
		"KeepDescription": &transformation.KeepDescription{},
		"KeepLocation":    &transformation.KeepLocation{},
		"KeepAttendees":   &transformation.KeepAttendees{UseEmailAsDisplayName: false},
		"KeepReminders":   &transformation.KeepReminders{},
	}

	// this is the order of the transformers in which they get evaluated
	// from first transformer to last
	transformerOrder = []string{
		"KeepAttendees",
		"KeepLocation",
		"KeepReminders",
		"KeepDescription",
		"KeepMeetingLink",
		"KeepTitle",
		"PrefixTitle",
		"ReplaceTitle",
	}
)

// TransformerFactory can build all configured transformers from the config file
func TransformerFactory(configuredTransformers []config.Transformer) (loadedTransformers []Transformer) {
	for _, configuredTransformer := range configuredTransformers {
		if _, nameExists := transformerConfigMapping[configuredTransformer.Name]; !nameExists {
			// todo: handle properly
			panic(fmt.Sprintf("unknown transformer: %s", configuredTransformer.Name))
		}
		// load the default Transformer for the configured name and initialize it based on the config
		transformerDefault := transformerConfigMapping[configuredTransformer.Name]
		loadedTransformers = append(loadedTransformers, TransformerFromConfig(transformerDefault, configuredTransformer.Config))
	}

	var sortedAndLoadedTransformer []Transformer
	for _, name := range transformerOrder {
		for _, v := range loadedTransformers {
			if strings.EqualFold(name, v.Name()) {
				sortedAndLoadedTransformer = append(sortedAndLoadedTransformer, v)
			}
		}
	}

	return sortedAndLoadedTransformer
}

// autoConfigure can automatically map keys of the config.CustomMap to fields of a given Transformer implementation.
// TODO: I kinda wrote this in a hurry. Re-visit this.
func autoConfigure(transformer Transformer, config config.CustomMap) {
	ps := reflect.ValueOf(transformer)
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
					field.SetInt(value.(int64))
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

func TransformerFromConfig(transformer Transformer, config config.CustomMap) Transformer {
	autoConfigure(transformer, config)
	return transformer
}
