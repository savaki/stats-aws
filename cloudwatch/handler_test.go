package cloudwatch

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/segmentio/stats/v4"
)

func TestHandler(t *testing.T) {
	var (
		accessKeyID     = os.Getenv("AWS_ACCESS_KEY_ID")
		secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		sessionToken    = os.Getenv("AWS_SESSION_TOKEN")
	)

	if accessKeyID == "" || secretAccessKey == "" {
		t.SkipNow()
	}

	var (
		s = session.Must(session.NewSession(aws.NewConfig().
			WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, sessionToken)).
			WithRegion("us-east-2")))
		api     = cloudwatch.New(s)
		handler = New(api, "test", func(s string) { fmt.Println(s) })
	)

	stats.Register(handler)
	defer stats.Flush()

	func() {
		type funcMetrics struct {
			calls struct {
				count int           `metric:"count" type:"counter"`
				time  time.Duration `metric:"time"  type:"histogram"`
			} `metric:"func.calls"`
		}

		t := time.Now()
		time.Sleep(time.Millisecond * 250)
		callTime := time.Now().Sub(t)

		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime

		stats.Report(m)

		//// Increment counters.
		stats.Incr("user.login")
		defer stats.Incr("user.logout")
	}()
}
