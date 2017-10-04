package ha

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type PointContainer struct {
	Points map[string]*Point
}

func NewPointContainer(metrics *Metrics) *PointContainer {
	points := map[string]*Point{}

	for i := range metrics.RequestCount.Datapoints {
		key := metrics.RequestCount.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(metrics.RequestCount.Datapoints[i])
		}
		points[key].ELBRequestCount = *metrics.RequestCount.Datapoints[i].Sum
	}

	for i := range metrics.InServiceHostCount.Datapoints {
		key := metrics.InServiceHostCount.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(metrics.InServiceHostCount.Datapoints[i])
		}
		points[key].InServiceHostCount = *metrics.InServiceHostCount.Datapoints[i].Average
	}

	for i := range metrics.CPUUtilization.Datapoints {
		key := metrics.CPUUtilization.Datapoints[i].Timestamp.String()
		if _, ok := points[key]; !ok {
			points[key] = NewPoint(metrics.CPUUtilization.Datapoints[i])
		}
		points[key].AutoScalingGroupCPU = *metrics.CPUUtilization.Datapoints[i].Average
	}

	pc := &PointContainer{
		Points: points,
	}

	keys := pc.Keys()
	delete(pc.Points, keys[len(keys)-1]) // dismiss ambiguous datapoint

	return pc
}

func (pc *PointContainer) Keys() []string {
	sortedTimes := make([]time.Time, 0, len(pc.Points))

	for _, value := range pc.Points {
		sortedTimes = append(sortedTimes, value.Timestamp)
	}

	sort.Slice(sortedTimes, func(i, j int) bool {
		return sortedTimes[i].Before(sortedTimes[j])
	})

	sortedKeys := make([]string, 0, len(sortedTimes))
	for i := range sortedTimes {
		sortedKeys = append(sortedKeys, sortedTimes[i].String())
	}

	return sortedKeys
}

func (pc *PointContainer) RecentEfficiency() float64 {
	keys := pc.Keys()
	count := 0
	total := 0.0
	for i := len(keys) - 1; i >= 0; i-- {
		if math.IsInf(pc.Points[keys[i]].Efficiency(), 0) {
			continue
		}

		if count > 5 {
			break
		}

		total += pc.Points[keys[i]].Efficiency()
		count++
	}

	return total / float64(count)
}

func (pc *PointContainer) GetLatestPoint() *Point {
	keys := pc.Keys()
	return pc.Points[keys[len(keys)-1]]
}

func (pc *PointContainer) EstimateNextRequiredHostCount(requestCount float64, ratio float64) float64 {
	return requestCount / pc.RecentEfficiency() / 100 / ratio
}

func (pc *PointContainer) EstimatedLatestRequiredHostCount(ratio float64) float64 {
	return pc.GetLatestPoint().EstimatedCurrentRequiredHostCount(pc.RecentEfficiency(), ratio)
}

func (pc *PointContainer) EstimatedLatestCPUUtilization() float64 {
	return pc.GetLatestPoint().EstimatedCurrentCPUUtilization(pc.RecentEfficiency())
}

func (pc *PointContainer) Prettify(option *Option) []string {
	var rv []string
	sortedKeys := pc.Keys()
	for i := range sortedKeys {
		k := sortedKeys[i]
		v := pc.Points[k]
		l := fmt.Sprintf("[%s] ReqCount: %.0f, HostCount: %.0f, CPU: %6.2f(%6.2f), ReqPerHost: %.2f, Required: %3.0f, %3.0f, %3.0f; E-Host %.2f; E-CPURAvg: %.2f, RecentAgv: %.2f",
			v.Timestamp.In(time.FixedZone("Asia/Tokyo", 9*60*60)),
			v.ELBRequestCount,
			v.InServiceHostCount,
			v.AutoScalingGroupCPU,
			v.Efficiency(),
			v.RequestCountPerHost(),
			v.RequiredHostCount(option.upperCPUThreshold),
			v.RequiredHostCount(option.middleCPUThreshold),
			v.RequiredHostCount(option.lowerCPUThreshold),
			v.EstimatedCurrentRequiredHostCount(pc.RecentEfficiency(), option.middleCPUThreshold),
			pc.EstimatedLatestCPUUtilization(),
			pc.RecentEfficiency(),
		)
		rv = append(rv, l)
	}

	return rv
}
