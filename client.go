package ha

import (
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"math"
	"time"
)

type Client struct {
	Region               string
	AvailabilityZone     string
	ELBName              string
	AutoScalingGroupName string
	Span                 int
}

type Metrics struct {
	RequestCount, InServiceHostCount, CPUUtilization *cloudwatch.GetMetricStatisticsOutput
}

func (c *Client) GetCurrentMetrics() (*Metrics, error) { // TODO use goroutine
	from := time.Now().Add(-time.Hour * 1)
	to := time.Now()

	log.Printf("debug: start GetRequestCount")
	rc, err := c.GetRequestCount(from, to)
	if err != nil {
		return nil, err
	}

	log.Printf("debug: start GetInServiceHostCount")
	ish, err := c.GetInServiceHostCount(from, to)
	if err != nil {
		return nil, err
	}

	log.Printf("debug: start GetCPUUtilization")
	cu, err := c.GetCPUUtilization(from, to)
	if err != nil {
		return nil, err
	}

	return &Metrics{RequestCount: rc, InServiceHostCount: ish, CPUUtilization: cu}, nil
}

func (c *Client) GetMaxRequestCountTrackRecord() (*float64, error) {
	max := 0.0

	for i := 1; i <= 2; i++ {
		base := time.Hour * 24 * 7 * time.Duration(i)
		w, err := c.GetRequestCount(
			time.Now().Add(-base),
			time.Now().Add(-base+time.Minute*time.Duration(c.Span)),
		)
		if err != nil {
			return nil, err
		}

		for j := range w.Datapoints {
			log.Printf("debug: %s %.0f", w.Datapoints[j].Timestamp.In(time.FixedZone("Asia/Tokyo", 9*60*60)), *w.Datapoints[j].Sum)
			max = math.Max(max, *w.Datapoints[j].Sum)
		}
	}

	return &max, nil
}

func (c *Client) GetRequestCount(start time.Time, end time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(c.ELBName),
			},
			{
				Name:  aws.String("AvailabilityZone"),
				Value: aws.String(c.AvailabilityZone),
			},
		},
		MetricName: aws.String("RequestCount"),
		Namespace:  aws.String("AWS/ELB"),
		Period:     aws.Int64(int64(60)),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Statistics: []*string{aws.String("Sum")},
	}

	return c.get(params)
}

func (c *Client) GetInServiceHostCount(start time.Time, end time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String(c.AutoScalingGroupName),
			},
		},
		MetricName: aws.String("GroupInServiceInstances"),
		Namespace:  aws.String("AWS/AutoScaling"),
		Period:     aws.Int64(int64(60)),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Statistics: []*string{aws.String("Average")},
	}

	return c.get(params)
}

func (c *Client) GetCPUUtilization(start time.Time, end time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := &cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("AutoScalingGroupName"),
				Value: aws.String(c.AutoScalingGroupName),
			},
		},
		MetricName: aws.String("CPUUtilization"),
		Namespace:  aws.String("AWS/EC2"),
		Period:     aws.Int64(int64(60)),
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Statistics: []*string{aws.String("Average")},
	}

	return c.get(params)
}

func (c *Client) UpdateAutoScalingGroupHostCount(num int64) (*autoscaling.UpdateAutoScalingGroupOutput, error) {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(c.AutoScalingGroupName),
		DesiredCapacity:      aws.Int64(num),
		MaxSize:              aws.Int64(num),
		MinSize:              aws.Int64(num),
	}

	return autoscaling.New(sess, aws.NewConfig().WithRegion(c.Region)).UpdateAutoScalingGroup(params)

	/* TODO Error handling
	result, err := autoscaling.New(sess, aws.NewConfig().WithRegion(c.Region)).UpdateAutoScalingGroup(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeScalingActivityInProgressFault:
				log.Println(autoscaling.ErrCodeScalingActivityInProgressFault, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				log.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}

	return result, nil
	*/
}

func (c *Client) get(params *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	req, resp := cloudwatch.New(sess, aws.NewConfig().WithRegion(c.Region)).GetMetricStatisticsRequest(params)
	if err = req.Send(); err != nil {
		return nil, err
	}
	return resp, nil
}
