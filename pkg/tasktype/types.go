package tasktype

type Type string

const (
	// Demo 演示任务
	Demo Type = "demo"
)

func (t Type) String() string {
	return string(t)
}

func (t Type) Queue() string {
	return "default"
}

func (t Type) IsValid() bool {
	return t == Demo
}

var AllTypes = []Type{
	Demo,
}
