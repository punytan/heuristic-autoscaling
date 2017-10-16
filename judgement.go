package ha

import (
	"log"
)

type Judgement struct {
	pc                      *PointContainer
	trackRecordRequestCount float64
	option                  *Option
}

type JudgeResult struct {
	currentHostCount   int64
	desirableHostCount int64
	executionType      ExecutionType
}

func NewJudgement(pc *PointContainer, option *Option, maxRequest float64) *Judgement {
	j := new(Judgement)
	j.pc = pc
	j.option = option
	j.trackRecordRequestCount = maxRequest
	return j
}

func (j *Judgement) Judge() *JudgeResult {
	nextHostCountMin := j.pc.EstimateNextRequiredHostCount(j.trackRecordRequestCount, j.option.upperCPUThreshold)
	nextHostCountMiddle := j.pc.EstimateNextRequiredHostCount(j.trackRecordRequestCount, j.option.middleCPUThreshold)
	nextHostCountMax := j.pc.EstimateNextRequiredHostCount(j.trackRecordRequestCount, j.option.lowerCPUThreshold)

	log.Printf("info: Estimated latest CPUUtilization")
	log.Printf("info:     %.2f%%", j.pc.EstimatedLatestCPUUtilization())
	log.Printf("info: Estimated Required Host Count")
	log.Printf("info:             %3.0f%% %3.0f%% %3.0f%%",
		j.option.upperCPUThreshold*100,
		j.option.middleCPUThreshold*100,
		j.option.lowerCPUThreshold*100)
	log.Printf("info:     Current  %3.0f  %3.0f  %3.0f",
		j.pc.EstimatedLatestRequiredHostCount(j.option.upperCPUThreshold),
		j.pc.EstimatedLatestRequiredHostCount(j.option.middleCPUThreshold),
		j.pc.EstimatedLatestRequiredHostCount(j.option.lowerCPUThreshold))
	log.Printf("info:     Next     %3.0f  %3.0f  %3.0f",
		nextHostCountMin, nextHostCountMiddle, nextHostCountMax)

	inServiceHostCount := j.pc.GetLatestPoint().InServiceHostCount
	result := &JudgeResult{
		currentHostCount:   int64(inServiceHostCount),
		desirableHostCount: int64(nextHostCountMiddle),
		executionType:      Stay,
	}

	var scaleInStatus string
	if nextHostCountMin <= inServiceHostCount && inServiceHostCount <= nextHostCountMax {
		scaleInStatus = "moderate"
	} else if inServiceHostCount <= nextHostCountMax {
		scaleInStatus = "maybe moderate"
	} else {
		scaleInStatus = "excess"
		result.executionType = Decrease
	}

	var scaleOutStatus string
	if nextHostCountMin <= inServiceHostCount {
		scaleOutStatus = "sufficient"
	} else {
		scaleOutStatus = "insufficient"
		result.executionType = Increase
	}

	var currentStatus string
	if j.pc.EstimatedLatestCPUUtilization() < j.option.upperCPUThreshold*100 {
		currentStatus = "OK"
	} else {
		currentStatus = "NG"
		result.executionType = Increase
	}

	log.Printf("info: Scaling strategy")
	log.Printf("    scale-in:  %s", scaleInStatus)
	log.Printf("    scale-out: %s", scaleOutStatus)
	log.Printf("    current:   %s", currentStatus)

	return result
}
