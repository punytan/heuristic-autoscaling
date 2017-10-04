package ha

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"time"
)

type Point struct {
	Timestamp           time.Time
	ELBRequestCount     float64
	InServiceHostCount  float64
	AutoScalingGroupCPU float64
}

func NewPoint(r *cloudwatch.Datapoint) *Point {
	p := new(Point)
	p.Timestamp = *r.Timestamp
	return p
}

func (p Point) Efficiency() float64 {
	return p.ELBRequestCount / p.InServiceHostCount / p.AutoScalingGroupCPU
}

func (p Point) RequestCountPerHost() float64 {
	return p.ELBRequestCount / p.InServiceHostCount
}

func (p Point) RequiredHostCount(targetCPUUtilization float64) float64 {
	return p.ELBRequestCount / p.Efficiency() / 100 / targetCPUUtilization
}

func (p Point) EstimatedCurrentRequiredHostCount(efficiency float64, targetCPUUtilization float64) float64 {
	return p.ELBRequestCount / efficiency / 100 / targetCPUUtilization
}

func (p Point) EstimatedCurrentCPUUtilization(recentEfficiency float64) float64 {
	return p.ELBRequestCount / recentEfficiency / p.InServiceHostCount
}
