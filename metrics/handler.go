// Copyright 2021 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/segmentio/stats/v4"
)

type Handler struct {
	api       cloudwatchiface.CloudWatchAPI
	namespace string
	log       func(s string)
}

func New(api cloudwatchiface.CloudWatchAPI, namespace string, log func(s string)) *Handler {
	if log == nil {
		log = func(s string) {}
	}
	return &Handler{
		api:       api,
		log:       log,
		namespace: namespace,
	}
}

func (h *Handler) HandleMeasures(t time.Time, measures ...stats.Measure) {
	remain := makeDatum(t, measures...)
	for len(remain) > 0 {
		batch := remain
		if len(batch) > 20 {
			batch = batch[0:20]
		}
		remain = remain[len(batch):]

		input := cloudwatch.PutMetricDataInput{
			MetricData: batch,
			Namespace:  aws.String(h.namespace),
		}
		if _, err := h.api.PutMetricData(&input); err != nil {
			h.log(err.Error())
		}
	}
}

func getName(prefix string, field stats.Field) string {
	if prefix == "" {
		return field.Name
	}
	return fmt.Sprintf("%v.%v", prefix, field.Name)
}

func stripPrefix(name string, n int) string {
	for i := 0; i < n; i++ {
		index := strings.Index(name, ".")
		if index > 0 {
			name = name[index+1:]
		}
	}
	return name
}

func getValue(v stats.Value) (value float64, unit *string) {
	switch v.Type() {
	case stats.Null:
		return 0, nil
	case stats.Bool:
		if v.Bool() {
			return 1, nil
		}
		return 0, nil
	case stats.Int:
		return float64(v.Int()), nil
	case stats.Uint:
		return float64(v.Uint()), nil
	case stats.Float:
		return v.Float(), nil
	case stats.Duration:
		return float64(v.Duration() / time.Millisecond), aws.String(cloudwatch.StandardUnitMilliseconds)
	default:
		panic("unknown type found in a stats.Value")
	}
}

func makeDatum(t time.Time, mm ...stats.Measure) (datum []*cloudwatch.MetricDatum) {
	for _, m := range mm {
		var dimensions []*cloudwatch.Dimension
		for _, tag := range m.Tags {
			dimensions = append(dimensions, &cloudwatch.Dimension{
				Name:  aws.String(tag.Name),
				Value: aws.String(tag.Value),
			})
		}

		fmt.Println(m.Name)

		prefix := stripPrefix(m.Name, 2)
		for _, field := range m.Fields {
			name := getName(prefix, field)

			v := field.Value.Interface()
			fmt.Printf("%20s: %#v (%v:%T)\n", name, v, field.Value.Type(), v)

			value, unit := getValue(field.Value)
			d := cloudwatch.MetricDatum{
				Dimensions:        dimensions,
				MetricName:        aws.String(name),
				StatisticValues:   nil,
				StorageResolution: nil,
				Timestamp:         &t,
				Unit:              unit,
				Value:             aws.Float64(value),
				Values:            nil,
			}
			datum = append(datum, &d)
		}
	}
	return datum
}
