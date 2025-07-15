package metric

import (
	"context"
	"fmt"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDBStorage struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	org      string
	bucket   string
}

func NewInfluxDBStorage() *InfluxDBStorage {
	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	org := os.Getenv("INFLUXDB_ORG")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

	return &InfluxDBStorage{
		client:   client,
		writeAPI: writeAPI,
		org:      org,
		bucket:   bucket,
	}
}

func (s *InfluxDBStorage) QueryDiskHistory(device string, start string, stop string) ([]map[string]interface{}, error) {
	var flux string
	if device == "" {
		flux = fmt.Sprintf(`
        from(bucket: "%s")
        |> range(start: %s, stop: %s)
        |> filter(fn: (r) => r._measurement == "disk")
    `, s.bucket, start, stop)
	} else {
		flux = fmt.Sprintf(`
        from(bucket: "%s")
        |> range(start: %s, stop: %s)
        |> filter(fn: (r) => r._measurement == "disk" and r.device == "%s")
    `, s.bucket, start, stop, device)
	}

	queryAPI := s.client.QueryAPI(s.org)
	result, err := queryAPI.Query(context.Background(), flux)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	var records []map[string]interface{}
	for result.Next() {
		rec := result.Record()
		records = append(records, map[string]interface{}{
			"time":   rec.Time(),
			"field":  rec.Field(),
			"value":  rec.Value(),
			"device": rec.ValueByKey("device"),
		})
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return records, nil
}

func (s *InfluxDBStorage) WriteMetric(measurement string, tags map[string]string, fields map[string]interface{}, t time.Time) error {
	p := influxdb2.NewPoint(measurement, tags, fields, t)
	return s.writeAPI.WritePoint(context.Background(), p)
}

func (s *InfluxDBStorage) Close() {
	s.client.Close()
}
