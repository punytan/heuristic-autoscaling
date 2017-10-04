package ha

type ExecutionType int

const (
	Stay ExecutionType = iota
	Increase
	Decrease
)

func (t ExecutionType) String() string {
	switch t {
	case Increase:
		return "Increase"
	case Decrease:
		return "Decrease"
	case Stay:
		return "Stay"
	}
	panic("Unknown value")
}
