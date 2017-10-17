package ha

import (
	"flag"
	"fmt"
	"github.com/comail/colog"
	"io"
	"log"
)

const (
	ExitCodeOK = iota
	ExitCodeParserFlagError
	ExitCodeFatal
)

type CLI struct {
	OutStream, ErrStream io.Writer
	Version              string
	Name                 string
}

type arrayFlags []string

func (a *arrayFlags) String() string {
	return ""
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

type Option struct {
	version              bool
	verbose              bool
	color                bool
	dryRun               bool
	region               string
	elbName              string
	availabilityZone     string
	autoscalingGroupName arrayFlags
	span                 int
	upperCPUThreshold    float64
	middleCPUThreshold   float64
	lowerCPUThreshold    float64
	maxInstance          int64
	minInstance          int64
}

func (c *CLI) InitLogger(option *Option) {
	colog.SetDefaultLevel(colog.LInfo)
	if option.verbose {
		colog.SetMinLevel(colog.LDebug)
	} else {
		colog.SetMinLevel(colog.LInfo)
	}
	colog.SetFormatter(&colog.StdFormatter{
		Colors: option.color,
		Flag:   log.Ldate | log.Ltime,
	})
	colog.Register()
}

func (c *CLI) ParseArgs(args []string) *Option {
	option := &Option{}

	flags := flag.NewFlagSet(c.Name, flag.ContinueOnError)
	flags.SetOutput(c.ErrStream)
	flags.BoolVar(&option.version, "version", false, "print version infomation and quit")
	flags.BoolVar(&option.verbose, "verbose", false, "verbose mode")
	flags.BoolVar(&option.color, "color", false, "log color mode")
	flags.BoolVar(&option.dryRun, "dry-run", false, "dry-run mode (do not update AutoScaling Group values)")
	flags.StringVar(&option.region, "region", "", "AWS region")
	flags.StringVar(&option.elbName, "elb-name", "", "target ELB name")
	flags.StringVar(&option.availabilityZone, "availability-zone", "", "target availability zone")
	flags.Var(&option.autoscalingGroupName, "autoscaling-group-name", "target AutoScaling group names")
	flags.IntVar(&option.span, "span", 30, "target span")
	flags.Float64Var(&option.upperCPUThreshold, "upper-cpu-threshold", 0.65, "CPU upper threshold")
	flags.Float64Var(&option.lowerCPUThreshold, "lower-cpu-threshold", 0.45, "CPU lower threshold")

	flags.Int64Var(&option.maxInstance, "max-instance", 0, "maximum instance size")
	flags.Int64Var(&option.minInstance, "min-instance", 0, "minimum instance size")

	if err := flags.Parse(args[1:]); err != nil {
		return nil
	}

	if option == nil {
		return nil
	}

	c.InitLogger(option)

	if option.version {
		fmt.Fprintf(c.ErrStream, "%s version %s\n", c.Name, c.Version)
		return nil
	}

	if option.region == "" {
		log.Printf("error: specify `--region` option")
		return nil
	}

	if option.elbName == "" {
		log.Printf("error: specify `--elb-name` option")
		return nil
	}

	if len(option.autoscalingGroupName) == 0 {
		log.Printf("error: specify `--autoscaling-group-name` option")
		return nil
	}

	if option.span < 15 {
		log.Printf("error: specify `--span` option greater than 15")
		return nil
	}

	if option.maxInstance < 1 {
		log.Printf("error: specify `--max-instance` option greater than 1")
		return nil
	}

	if option.minInstance < 1 {
		log.Printf("error: specify `--min-instance` option greater than 1")
		return nil
	}

	option.middleCPUThreshold = (option.upperCPUThreshold + option.lowerCPUThreshold) / 2

	return option
}

func (c *CLI) Run(args []string) int {
	option := c.ParseArgs(args)
	if option == nil {
		return ExitCodeParserFlagError
	}

	result := c.run(option)
	if result == nil {
		return ExitCodeFatal
	}
	//
	log.Printf("info: ----- Result -----")
	log.Printf("info:    %s (current: %d, desirable: %d)", result.executionType, result.currentHostCount, result.desirableHostCount)

	if option.dryRun {
		log.Printf("info: dry-run mode; exit")
		return ExitCodeOK
	}

	if result.executionType == Stay {
		return ExitCodeOK
	}

	if option.maxInstance < result.desirableHostCount {
		log.Printf("warn: next desirable host count is greater than --max-instance")
		return ExitCodeOK
	}

	if option.minInstance > result.desirableHostCount {
		log.Printf("warn: next desirable host count is less than --min-instance")
		return ExitCodeOK
	}

	log.Printf("warn: Start updating AutoScaling values")
	for i := range option.autoscalingGroupName {
		client := Client{
			Region:               option.region,
			AvailabilityZone:     option.availabilityZone,
			ELBName:              option.elbName,
			AutoScalingGroupName: option.autoscalingGroupName[i],
			Span:                 option.span,
		}
		updateResult, err := client.UpdateAutoScalingGroupHostCount(result.desirableHostCount)
		if err != nil {
			log.Printf("error: %#v", err)
			return ExitCodeFatal
		}
		log.Printf("info: Update Result: %#v", updateResult)
	}

	return ExitCodeOK
}

func (c *CLI) run(option *Option) *JudgeResult {
	var rv *JudgeResult

	for i := range option.autoscalingGroupName {
		log.Printf("info: ----- %s -----", option.autoscalingGroupName[i])
		client := Client{
			Region:               option.region,
			AvailabilityZone:     option.availabilityZone,
			ELBName:              option.elbName,
			AutoScalingGroupName: option.autoscalingGroupName[i],
			Span:                 option.span,
		}

		metrics, err := client.GetCurrentMetrics()
		if err != nil {
			log.Printf("error: %s", err)
			return nil
		}

		pc := NewPointContainer(metrics)

		pretty := pc.Prettify(option)
		for i := range pretty {
			log.Printf("debug: %s", pretty[i])
		}

		maxRequest, err := client.GetMaxRequestCountTrackRecord()
		if err != nil {
			log.Printf("error: %s", err)
			return nil
		}

		log.Printf("debug: max request count track record: %.0f", *maxRequest)

		current := NewJudgement(pc, option, *maxRequest).Judge()
		if rv == nil {
			rv = current
		} else if rv.desirableHostCount < current.desirableHostCount {
			rv = current
		}
	}

	return rv
}
